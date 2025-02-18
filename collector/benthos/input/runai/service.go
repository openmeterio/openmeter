package runai

import (
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/redpanda-data/benthos/v4/public/service"
)

type Service struct {
	logger    *service.Logger
	client    *resty.Client
	appID     string
	appSecret string
	token     string
}

type HTTPRequestConfig struct {
	Timeout          time.Duration
	RetryWaitTime    time.Duration
	RetryMaxWaitTime time.Duration
	RetryCount       int
}

func NewService(baseURL, appID, appSecret string, logger *service.Logger, requestConfig HTTPRequestConfig) (*Service, error) {
	service := &Service{
		logger:    logger,
		appID:     appID,
		appSecret: appSecret,
	}

	client := resty.New().
		SetBaseURL(baseURL).
		SetLogger(logger).
		SetTimeout(requestConfig.Timeout).
		SetRetryCount(requestConfig.RetryCount).
		SetRetryWaitTime(requestConfig.RetryWaitTime).
		SetRetryMaxWaitTime(requestConfig.RetryMaxWaitTime).
		OnBeforeRequest(func(client *resty.Client, request *resty.Request) error {
			service.logger.Tracef("request: %s", request.URL)

			// Skip token request.
			if request.URL == "/api/v1/token" {
				return nil
			}

			err := service.RefreshToken(request.Context())
			if err != nil {
				return err
			}

			request.SetAuthToken(service.GetToken())

			return nil
		}).
		OnAfterResponse(func(client *resty.Client, response *resty.Response) error {
			if response.StatusCode() == http.StatusUnauthorized {
				service.SetToken("")
			}

			return nil
		})

	service.client = client

	return service, nil
}
