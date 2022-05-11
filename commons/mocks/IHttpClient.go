package mocks

import (
	"context"
	"io"
	"net/http"

	"github.com/stretchr/testify/mock"
)

type MockHTTPClient struct {
	mock.Mock
}

func (h *MockHTTPClient) Post(ctx context.Context, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	args := h.Mock.Called()
	return args.Get(0).(*http.Response), args.Error(1)
}

func (h *MockHTTPClient) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	args := h.Mock.Called()
	return args.Get(0).(*http.Response), args.Error(1)
}

func (h *MockHTTPClient) Delete(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	args := h.Mock.Called()
	return args.Get(0).(*http.Response), args.Error(1)
}
func (h *MockHTTPClient) Put(ctx context.Context, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	args := h.Mock.Called()
	return args.Get(0).(*http.Response), args.Error(1)
}

func (h *MockHTTPClient) Getwithbody(ctx context.Context, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	args := h.Mock.Called()
	return args.Get(0).(*http.Response), args.Error(1)
}
