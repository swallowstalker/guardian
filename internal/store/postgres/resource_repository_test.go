package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/odpf/guardian/core/resource"
	"github.com/odpf/guardian/domain"
	"github.com/odpf/guardian/internal/store/postgres"
	"github.com/odpf/salt/log"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"
)

type ResourceRepositoryTestSuite struct {
	suite.Suite
	store         *postgres.Store
	pool          *dockertest.Pool
	resource      *dockertest.Resource
	dummyProvider *domain.Provider
	repository    *postgres.ResourceRepository
}

func (s *ResourceRepositoryTestSuite) SetupSuite() {
	var err error

	logger := log.NewLogrus(log.LogrusWithLevel("debug"))
	s.store, s.pool, s.resource, err = newTestStore(logger)
	if err != nil {
		s.T().Fatal(err)
	}

	s.repository = postgres.NewResourceRepository(s.store.DB())

	s.dummyProvider = &domain.Provider{
		Type: "provider_test",
		URN:  "provider_urn_test",
	}
	providerRepository := postgres.NewProviderRepository(s.store.DB())
	err = providerRepository.Create(context.Background(), s.dummyProvider)
	s.Require().NoError(err)
}

func (s *ResourceRepositoryTestSuite) AfterTest(suiteName, testName string) {
	if err := truncateTable(s.store, "resources"); err != nil {
		s.T().Fatal(err)
	}
}

func (s *ResourceRepositoryTestSuite) TearDownSuite() {
	// Clean tests
	db, err := s.store.DB().DB()
	if err != nil {
		s.T().Fatal(err)
	}
	err = db.Close()
	if err != nil {
		s.T().Fatal(err)
	}

	err = purgeTestDocker(s.pool, s.resource)
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *ResourceRepositoryTestSuite) TestFind() {
	s.Run("should pass conditions based on filters", func() {
		dummyResources := []*domain.Resource{
			{
				ProviderType: s.dummyProvider.Type,
				ProviderURN:  s.dummyProvider.URN,
				Type:         "test_type",
				URN:          "test_urn_1",
				Name:         "test_name_1",
				Details: map[string]interface{}{
					"foo": "bar",
				},
			},
			{
				ProviderType: s.dummyProvider.Type,
				ProviderURN:  s.dummyProvider.URN,
				Type:         "test_type",
				URN:          "test_urn_2",
				Name:         "test_name_2",
			},
		}
		err := s.repository.BulkUpsert(context.Background(), dummyResources)
		s.Require().NoError(err)

		testCases := []struct {
			name           string
			filters        domain.ListResourcesFilter
			expectedResult []*domain.Resource
		}{
			{
				name:           "empty filter",
				filters:        domain.ListResourcesFilter{},
				expectedResult: dummyResources,
			},
			{
				name: "filter by ids",
				filters: domain.ListResourcesFilter{
					IDs: []string{dummyResources[0].ID},
				},
				expectedResult: []*domain.Resource{dummyResources[0]},
			},
			{
				name: "filter by type",
				filters: domain.ListResourcesFilter{
					ResourceType: "test_type",
				},
				expectedResult: dummyResources,
			},
			{
				name: "filter by name",
				filters: domain.ListResourcesFilter{
					Name: "test_name_1",
				},
				expectedResult: []*domain.Resource{dummyResources[0]},
			},
			{
				name: "filter by provider type",
				filters: domain.ListResourcesFilter{
					ProviderType: s.dummyProvider.Type,
				},
				expectedResult: dummyResources,
			},
			{
				name: "filter by provider urn",
				filters: domain.ListResourcesFilter{
					ProviderURN: s.dummyProvider.URN,
				},
				expectedResult: dummyResources,
			},
			{
				name: "filter by urn",
				filters: domain.ListResourcesFilter{
					ResourceURN: "test_urn_1",
				},
				expectedResult: []*domain.Resource{dummyResources[0]},
			},
			{
				name: "filter by details",
				filters: domain.ListResourcesFilter{
					Details: map[string]string{
						"foo": "bar",
					},
				},
				expectedResult: []*domain.Resource{dummyResources[0]},
			},
		}

		for _, tc := range testCases {
			s.Run(tc.name, func() {
				actualResult, actualError := s.repository.Find(context.Background(), tc.filters)

				s.NoError(actualError)
				options := []cmp.Option{
					cmpopts.EquateApproxTime(time.Microsecond),
					cmpopts.SortSlices(func(a, b *domain.Resource) bool { return a.ID < b.ID }),
				}
				if diff := cmp.Diff(tc.expectedResult, actualResult, options...); diff != "" {
					s.T().Errorf("result not match, diff: %v", diff)
				}
			})
		}
	})

	s.Run("should return error if filters validation returns an error", func() {
		invalidFilters := domain.ListResourcesFilter{
			IDs: []string{},
		}
		actualRecords, actualError := s.repository.Find(context.Background(), invalidFilters)

		s.Error(actualError)
		s.Nil(actualRecords)
	})

	s.Run("should return error if db returns error", func() {
		invalidFilters := domain.ListResourcesFilter{
			IDs: []string{"invalid-uuid"},
		}
		actualRecords, actualError := s.repository.Find(context.Background(), invalidFilters)

		s.Error(actualError)
		s.Nil(actualRecords)
	})
}

func (s *ResourceRepositoryTestSuite) TestGetOne() {
	s.Run("should return error if id is empty", func() {
		expectedError := resource.ErrEmptyIDParam

		actualResult, actualError := s.repository.GetOne(context.Background(), "")

		s.Nil(actualResult)
		s.EqualError(actualError, expectedError.Error())
	})

	s.Run("should return error if record not found", func() {
		expectedError := resource.ErrRecordNotFound

		sampleUUID := uuid.New().String()
		actualResult, actualError := s.repository.GetOne(context.Background(), sampleUUID)

		s.Nil(actualResult)
		s.EqualError(actualError, expectedError.Error())
	})

	s.Run("should return record and nil error on success", func() {
		resources := s.getTestResources()
		err := s.repository.BulkUpsert(context.Background(), resources)
		s.Nil(err)

		expectedResource := resources[0]

		r, actualError := s.repository.GetOne(context.Background(), expectedResource.ID)
		s.Nil(actualError)
		s.Equal(expectedResource.URN, r.URN)
	})
}

func (s *ResourceRepositoryTestSuite) TestBulkUpsert() {
	s.Run("should return records with existing or new IDs on insertion", func() {
		resources := s.getTestResources()

		err := s.repository.BulkUpsert(context.Background(), resources)

		actualIDs := make([]string, 0)
		for _, r := range resources {
			if r.ID != "" {
				actualIDs = append(actualIDs, r.ID)
			}
		}

		s.Nil(err)
		s.Equal(len(resources), len(actualIDs))
	})

	s.Run("should update specified fields on existing records", func() {
		resources := s.getTestResources()
		err := s.repository.BulkUpsert(context.Background(), resources)
		s.Require().NoError(err)

		toBeUpdatedResource := &domain.Resource{}
		*toBeUpdatedResource = *resources[0]
		toBeUpdatedResource.Name = "updated_name"
		toBeUpdatedResource.Details = map[string]interface{}{"foo": "updated_bar"}
		toBeUpdatedResource.IsDeleted = true
		toBeUpdatedResource.ParentID = &resources[1].ID
		ctx := context.Background()
		err = s.repository.BulkUpsert(ctx, []*domain.Resource{toBeUpdatedResource})
		s.NoError(err)

		newResource, err := s.repository.GetOne(ctx, toBeUpdatedResource.ID)
		s.NoError(err)
		s.Equal(toBeUpdatedResource.Name, newResource.Name)
		s.Equal(toBeUpdatedResource.Details, newResource.Details)
		s.Equal(toBeUpdatedResource.IsDeleted, newResource.IsDeleted)
		s.Equal(toBeUpdatedResource.ParentID, newResource.ParentID)
	})

	s.Run("should return nil error if resources input is empty", func() {
		var resources []*domain.Resource

		err := s.repository.BulkUpsert(context.Background(), resources)

		s.Nil(err)
	})

	s.Run("should return error if resources is invalid", func() {
		invalidResources := []*domain.Resource{
			{
				Details: map[string]interface{}{
					"foo": make(chan int), // invalid value
				},
			},
		}

		actualError := s.repository.BulkUpsert(context.Background(), invalidResources)

		s.EqualError(actualError, "json: unsupported type: chan int")
	})
}

func (s *ResourceRepositoryTestSuite) TestUpdate() {
	s.Run("should return error if id is empty", func() {
		expectedError := resource.ErrEmptyIDParam

		actualError := s.repository.Update(context.Background(), &domain.Resource{})

		s.EqualError(actualError, expectedError.Error())
	})

	s.Run("should return error if resource is invalid", func() {
		invalidResource := &domain.Resource{
			ID: uuid.New().String(),
			Details: map[string]interface{}{
				"foo": make(chan int), // invalid value
			},
		}
		actualError := s.repository.Update(context.Background(), invalidResource)

		s.EqualError(actualError, "json: unsupported type: chan int")
	})

	s.Run("should update record", func() {
		dummyResource := &domain.Resource{
			ProviderType: s.dummyProvider.Type,
			ProviderURN:  s.dummyProvider.URN,
			Type:         "test_type",
			URN:          "test_urn",
			Name:         "test_name",
		}
		err := s.repository.BulkUpsert(context.Background(), []*domain.Resource{dummyResource})
		s.Require().NoError(err)
		expectedID := dummyResource.ID
		payload := &domain.Resource{
			ID:   expectedID,
			Name: "test_new_name",
		}

		err = s.repository.Update(context.Background(), payload)

		actualID := payload.ID

		s.NoError(err)
		s.Equal(expectedID, actualID)
		s.NotEqual(dummyResource.Name, payload.Name)
	})
}

func (s *ResourceRepositoryTestSuite) TestDelete() {
	s.Run("should return error if ID param is empty", func() {
		err := s.repository.Delete(context.Background(), "")

		s.Error(err)
		s.ErrorIs(err, resource.ErrEmptyIDParam)
	})

	s.Run("should return error if resource not found", func() {
		sampleUUID := uuid.New().String()
		err := s.repository.Delete(context.Background(), sampleUUID)

		s.Error(err)
		s.ErrorIs(err, resource.ErrRecordNotFound)
	})

	s.Run("should return nil on success", func() {
		dummyResource := &domain.Resource{
			ProviderType: s.dummyProvider.Type,
			ProviderURN:  s.dummyProvider.URN,
			Type:         "test_type",
			URN:          "test_urn_deletion",
		}
		err := s.repository.BulkUpsert(context.Background(), []*domain.Resource{dummyResource})
		s.Require().NoError(err)

		toBeDeletedID := dummyResource.ID
		err = s.repository.Delete(context.Background(), toBeDeletedID)
		s.Nil(err)
	})
}

func (s *ResourceRepositoryTestSuite) TestBatchDelete() {
	s.Run("should return error if ID param is empty", func() {
		err := s.repository.BatchDelete(context.Background(), nil)

		s.Error(err)
		s.ErrorIs(err, resource.ErrEmptyIDParam)
	})

	s.Run("should return error if resource(s) not found", func() {
		sampleUUID := uuid.New().String()
		err := s.repository.BatchDelete(context.Background(), []string{sampleUUID})

		s.Error(err)
		s.ErrorIs(err, resource.ErrRecordNotFound)
	})

	s.Run("should return nil on success", func() {
		dummyResource := &domain.Resource{
			ProviderType: s.dummyProvider.Type,
			ProviderURN:  s.dummyProvider.URN,
			Type:         "test_type",
			URN:          "test_urn_batch_deletion",
		}
		err := s.repository.BulkUpsert(context.Background(), []*domain.Resource{dummyResource})
		s.Require().NoError(err)

		expectedIDs := []string{dummyResource.ID}

		err = s.repository.BatchDelete(context.Background(), expectedIDs)
		s.NoError(err)
	})
}

func (s *ResourceRepositoryTestSuite) getTestResources() []*domain.Resource {
	return []*domain.Resource{
		{
			ProviderType: "provider_test",
			ProviderURN:  "provider_urn_test",
			Type:         "resource_type",
			URN:          "resource_type.resource_name",
			Name:         "resource_name",
		},
		{
			ProviderType: "provider_test",
			ProviderURN:  "provider_urn_test",
			Type:         "resource_type",
			URN:          "resource_type.resource_name_2",
			Name:         "resource_name_2",
		},
	}
}

func TestResourceRepository(t *testing.T) {
	suite.Run(t, new(ResourceRepositoryTestSuite))
}
