package httpdriver_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	channels "github.com/openmeterio/openmeter/api/v3/handlers/notification/channels"
	rules "github.com/openmeterio/openmeter/api/v3/handlers/notification/rules"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
)

func TestWebhookErrorResponses(t *testing.T) {
	const (
		validationMessage = "Webhook URL must use HTTPS"
		sensitiveDetail   = "provider detail: credential=example-sensitive-value provider_app_id=example-app server=http://provider.invalid/internal"
		resourceID        = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	)

	resolveNamespace := func(context.Context) (string, error) {
		return "test", nil
	}

	routes := []struct {
		name    string
		method  string
		body    string
		handler func(notification.Service) http.Handler
	}{
		{
			name:   "v1 create channel",
			method: http.MethodPost,
			body:   `{"type":"WEBHOOK","name":"test","url":"http://example.com/webhook"}`,
			handler: func(service notification.Service) http.Handler {
				return httpdriver.New(namespacedriver.StaticNamespaceDecoder("test"), service, nil).CreateChannel()
			},
		},
		{
			name:   "v1 update channel",
			method: http.MethodPut,
			body:   `{"type":"WEBHOOK","name":"test","url":"http://example.com/webhook"}`,
			handler: func(service notification.Service) http.Handler {
				return httpdriver.New(namespacedriver.StaticNamespaceDecoder("test"), service, nil).UpdateChannel().With(resourceID)
			},
		},
		{
			name:   "v1 delete rule",
			method: http.MethodDelete,
			handler: func(service notification.Service) http.Handler {
				return httpdriver.New(namespacedriver.StaticNamespaceDecoder("test"), service, nil).DeleteRule().With(resourceID)
			},
		},
		{
			name:   "v3 create channel",
			method: http.MethodPost,
			body:   `{"type":"webhook","name":"test","url":"http://example.com/webhook"}`,
			handler: func(service notification.Service) http.Handler {
				return channels.New(resolveNamespace, service).CreateNotificationChannel()
			},
		},
		{
			name:   "v3 update channel",
			method: http.MethodPut,
			body:   `{"type":"webhook","name":"test","url":"http://example.com/webhook"}`,
			handler: func(service notification.Service) http.Handler {
				return channels.New(resolveNamespace, service).UpdateNotificationChannel().With(resourceID)
			},
		},
		{
			name:   "v3 delete rule",
			method: http.MethodDelete,
			handler: func(service notification.Service) http.Handler {
				return rules.New(resolveNamespace, service, nil).DeleteNotificationRule().With(resourceID)
			},
		},
	}

	tests := []struct {
		name   string
		err    error
		status int
		detail string
	}{
		{
			name:   "validation",
			err:    webhook.NewValidationError(webhook.ErrURLRequiresHTTPS),
			status: http.StatusBadRequest,
			detail: validationMessage,
		},
		{
			name:   "wrapped validation",
			err:    webhook.NewUnrecoverableError(webhook.NewValidationError(fmt.Errorf("%s: %w", sensitiveDetail, webhook.ErrURLRequiresHTTPS))),
			status: http.StatusBadRequest,
			detail: validationMessage,
		},
		{
			name:   "unknown validation",
			err:    webhook.NewValidationError(errors.New(sensitiveDetail)),
			status: http.StatusBadRequest,
			detail: "Invalid webhook configuration",
		},
		{
			name:   "unknown wrapped validation",
			err:    webhook.NewUnrecoverableError(webhook.NewValidationError(errors.New(sensitiveDetail))),
			status: http.StatusBadRequest,
			detail: "Invalid webhook configuration",
		},
		{
			name:   "channel limit",
			err:    webhook.NewValidationError(webhook.ErrMaxChannelsPerWebhookExceeded),
			status: http.StatusBadRequest,
			detail: "Webhook channel limit exceeded",
		},

		{
			name:   "provider unavailable",
			err:    webhook.NewRetryableError(errors.New("provider unavailable"), 0),
			status: http.StatusInternalServerError,
		},
	}

	for _, route := range routes {
		for _, tt := range tests {
			t.Run(route.name+"/"+tt.name, func(t *testing.T) {
				// given a provider failure after the request passes input validation
				service := &failingNotificationService{err: tt.err}
				request := httptest.NewRequest(route.method, "/", strings.NewReader(route.body)).WithContext(t.Context())
				request.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()

				// when the notification handler processes the request
				route.handler(service).ServeHTTP(response, request)

				// then validation is a client error and outages remain server errors
				require.True(t, service.called)
				require.Equal(t, tt.status, response.Code, response.Body.String())

				var problem struct {
					Status int    `json:"status"`
					Detail string `json:"detail"`
				}

				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
				require.Equal(t, tt.status, problem.Status)

				require.Equal(t, tt.detail, problem.Detail)
				require.NotContains(t, response.Body.String(), "example-sensitive-value")
				require.NotContains(t, response.Body.String(), "example-app")
				require.NotContains(t, response.Body.String(), "provider.invalid")

				if route.method != http.MethodDelete {
					require.Equal(t, "http://example.com/webhook", service.url)
				}
			})
		}
	}
}

type failingNotificationService struct {
	notification.Service
	err    error
	url    string
	called bool
}

func (s *failingNotificationService) CreateChannel(_ context.Context, input notification.CreateChannelInput) (*notification.Channel, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	s.called = true
	s.url = input.Config.WebHook.URL

	return nil, fmt.Errorf("failed to create webhook for channel: %w", s.err)
}

func (s *failingNotificationService) UpdateChannel(_ context.Context, input notification.UpdateChannelInput) (*notification.Channel, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	s.called = true
	s.url = input.Config.WebHook.URL

	return nil, fmt.Errorf("failed to update webhook for channel: %w", s.err)
}

func (s *failingNotificationService) DeleteRule(_ context.Context, input notification.DeleteRuleInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	s.called = true

	return fmt.Errorf("failed to update webhook for channel: %w", s.err)
}
