package utils

import (
	"context"
	"os"

	"github.eagleview.com/engineering/assess-platform-library/auth_client"
	"github.eagleview.com/engineering/assess-platform-library/httpservice"
	"github.eagleview.com/engineering/symphony-service/commons/error_codes"
	"github.eagleview.com/engineering/symphony-service/commons/error_handler"
)

type AuthTokenInterface interface {
	AddAuthorizationTokenHeader(ctx context.Context, httpClient httpservice.IHTTPClientV2, headers map[string]string, appCode, clientID, clientSecret string) error
}

type AuthTokenUtil struct {
}

const (
	AuthEndpoint = "AuthEndpoint"
)

func (authToken *AuthTokenUtil) AddAuthorizationTokenHeader(ctx context.Context, httpClient httpservice.IHTTPClientV2, headers map[string]string, appCode, clientID, clientSecret string) error {
	endpoint := os.Getenv(AuthEndpoint)
	generateFreshToken := false
	token, err := auth_client.GetAccessToken(ctx, httpClient, appCode, endpoint, clientID, clientSecret, generateFreshToken)
	if err != nil {
		return error_handler.NewServiceError(error_codes.ErrorGettingAccessToken, err.Error())
	}
	headers["Authorization"] = "Bearer " + token
	return nil
}
