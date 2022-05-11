// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	legacy_client "github.eagleview.com/engineering/symphony-service/commons/legacy_client"
)

// ILegacyClient is an autogenerated mock type for the ILegacyClient type
type ILegacyClient struct {
	mock.Mock
}

// GetLegacyBaseUrlAndAuthToken provides a mock function with given fields: ctx
func (_m *ILegacyClient) GetLegacyBaseUrlAndAuthToken(ctx context.Context) (string, string) {
	ret := _m.Called(ctx)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context) string); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func(context.Context) string); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Get(1).(string)
	}

	return r0, r1
}

// UpdateReportStatus provides a mock function with given fields: ctx, req
func (_m *ILegacyClient) UpdateReportStatus(ctx context.Context, req *legacy_client.LegacyUpdateRequest) error {
	ret := _m.Called(ctx, req)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *legacy_client.LegacyUpdateRequest) error); ok {
		r0 = rf(ctx, req)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
