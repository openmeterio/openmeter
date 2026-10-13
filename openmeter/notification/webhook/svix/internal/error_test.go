package internal

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	svix "github.com/svix/svix-webhooks/v2/go"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
)

func TestWrapSvixErrorPublicValidationMessage(t *testing.T) {
	const httpsMessage = "Endpoint URL schemes must be https when endpoint_https_only is set."

	tests := []struct {
		name    string
		status  int
		body    string
		message string
	}{
		{
			name:    "known HTTPS constraint",
			status:  http.StatusUnprocessableEntity,
			body:    `{"detail":[{"loc":["body","url"],"type":"value_error","msg":"` + httpsMessage + `","input":"example-sensitive-value"}]}`,
			message: "Webhook URL must use HTTPS",
		},
		{
			name:    "unknown provider message",
			status:  http.StatusUnprocessableEntity,
			body:    `{"detail":[{"loc":["body","secret"],"type":"value_error","msg":"provider credential=example-sensitive-value app=example-app server=http://provider.invalid"}]}`,
			message: "Invalid webhook configuration",
		},
		{
			name:    "known text on a different field",
			status:  http.StatusUnprocessableEntity,
			body:    `{"detail":[{"loc":["body","secret"],"type":"value_error","msg":"` + httpsMessage + `"}]}`,
			message: "Invalid webhook configuration",
		},
		{
			name:    "known text with a different type",
			status:  http.StatusUnprocessableEntity,
			body:    `{"detail":[{"loc":["body","url"],"type":"custom_error","msg":"` + httpsMessage + `"}]}`,
			message: "Invalid webhook configuration",
		},
		{
			name:    "multiple validation failures",
			status:  http.StatusUnprocessableEntity,
			body:    `{"detail":[{"loc":["body","url"],"type":"value_error","msg":"` + httpsMessage + `"},{"loc":["body","secret"],"type":"value_error","msg":"example-sensitive-value"}]}`,
			message: "Invalid webhook configuration",
		},
		{
			name:    "bad request detail",
			status:  http.StatusBadRequest,
			body:    `{"code":"invalid_request","detail":"credential=example-sensitive-value app=example-app"}`,
			message: "Invalid webhook configuration",
		},
		{
			name:    "conflict detail",
			status:  http.StatusConflict,
			body:    `{"code":"conflict","detail":"provider identifier=example-sensitive-value"}`,
			message: "Invalid webhook configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given a provider response containing validation or diagnostic text
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			client, err := svix.New("test", &svix.SvixOptions{ServerUrl: serverURL})
			require.NoError(t, err)

			// when the SDK failure is translated into a webhook validation error
			_, err = client.Endpoint().Create(t.Context(), "app-test", svix.EndpointIn{Url: "https://example.com/webhook"}, nil)
			require.Error(t, err)

			err = WrapSvixError(err)

			// then only an explicitly mapped public message is exposed
			var validationErr webhook.ValidationError
			require.True(t, errors.As(err, &validationErr))
			require.Equal(t, tt.message, validationErr.PublicMessage())

			var providerErr SvixError
			require.ErrorAs(t, err, &providerErr)
			require.Equal(t, tt.status, providerErr.HTTPStatus)
			require.NotEmpty(t, providerErr.Details)
		})
	}
}
