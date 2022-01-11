// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	domain "github.com/odpf/guardian/domain"
	mock "github.com/stretchr/testify/mock"
)

// AppealRepository is an autogenerated mock type for the AppealRepository type
type AppealRepository struct {
	mock.Mock
}

// BulkUpsert provides a mock function with given fields: _a0
func (_m *AppealRepository) BulkUpsert(_a0 []*domain.Appeal) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func([]*domain.Appeal) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Find provides a mock function with given fields: _a0
func (_m *AppealRepository) Find(_a0 *domain.ListAppealsFilter) ([]*domain.Appeal, error) {
	ret := _m.Called(_a0)

	var r0 []*domain.Appeal
	if rf, ok := ret.Get(0).(func(*domain.ListAppealsFilter) []*domain.Appeal); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*domain.Appeal)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*domain.ListAppealsFilter) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByID provides a mock function with given fields: _a0
func (_m *AppealRepository) GetByID(_a0 uint) (*domain.Appeal, error) {
	ret := _m.Called(_a0)

	var r0 *domain.Appeal
	if rf, ok := ret.Get(0).(func(uint) *domain.Appeal); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.Appeal)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update provides a mock function with given fields: _a0
func (_m *AppealRepository) Update(_a0 *domain.Appeal) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*domain.Appeal) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
