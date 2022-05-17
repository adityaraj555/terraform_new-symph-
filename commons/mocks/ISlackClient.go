// Code generated by mockery v2.12.2. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	testing "testing"
)

// ISlackClient is an autogenerated mock type for the ISlackClient type
type ISlackClient struct {
	mock.Mock
}

// SendErrorMessage provides a mock function with given fields: reportId, workflowId, lambdaName, msg
func (_m *ISlackClient) SendErrorMessage(reportId string, workflowId string, lambdaName string, msg string) {
	_m.Called(reportId, workflowId, lambdaName, msg)
}

// NewISlackClient creates a new instance of ISlackClient. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewISlackClient(t testing.TB) *ISlackClient {
	mock := &ISlackClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
