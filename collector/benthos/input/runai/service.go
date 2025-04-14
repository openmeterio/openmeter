package runai

import (
	"fmt"
	"net/http"
	"regexp"
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
	pageSize  int

	resourceTypeMetrics *service.MetricGauge
}

type ServiceConfig struct {
	Timeout          time.Duration
	RetryWaitTime    time.Duration
	RetryMaxWaitTime time.Duration
	RetryCount       int
	PageSize         int

	TimingMetrics       *service.MetricTimer
	ResourceTypeMetrics *service.MetricGauge
}

func NewService(baseURL, appID, appSecret string, logger *service.Logger, config ServiceConfig) (*Service, error) {
	service := &Service{
		logger:    logger,
		appID:     appID,
		appSecret: appSecret,
		pageSize:  config.PageSize,

		resourceTypeMetrics: config.ResourceTypeMetrics,
	}

	client := resty.New().
		SetBaseURL(baseURL).
		SetLogger(logger).
		SetTimeout(config.Timeout).
		SetRetryCount(config.RetryCount).
		SetRetryWaitTime(config.RetryWaitTime).
		SetRetryMaxWaitTime(config.RetryMaxWaitTime).
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

			if config.TimingMetrics != nil {
				path := response.Request.RawRequest.URL.Path
				if matched, err := regexp.MatchString("/api/v1/workloads/[0-9a-f-]+/pods/[0-9a-f-]+/metrics", path); err == nil && matched {
					path = "/api/v1/workloads/:workloadID/pods/:podID/metrics"
				} else if matched, err := regexp.MatchString("/api/v1/workloads/[0-9a-f-]+/metrics", path); err == nil && matched {
					path = "/api/v1/workloads/:workloadID/metrics"
				} else if matched, err := regexp.MatchString("/api/v1/pods", path); err == nil && matched {
					path = "/api/v1/pods"
				} else if matched, err := regexp.MatchString("/api/v1/workloads", path); err == nil && matched {
					path = "/api/v1/workloads"
				}
				config.TimingMetrics.Timing(response.Time().Nanoseconds(), path, fmt.Sprintf("%d", response.StatusCode()))
			}

			return nil
		})

	service.client = client

	return service, nil
}
