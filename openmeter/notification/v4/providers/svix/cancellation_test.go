package svix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	svix "github.com/svix/svix-webhooks/v2/go"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/notification/v4"
	webhooksecret "github.com/openmeterio/openmeter/openmeter/notification/webhook/secret"
)

func TestProviderOperationCancellation(t *testing.T) {
	namespaceID := ulid.Make().String()
	channelID := ulid.Make().String()

	desiredSecret, err := webhooksecret.NewSigningSecretWithDefaultSize()
	require.NoError(t, err)

	existingSecret, err := webhooksecret.NewSigningSecretWithDefaultSize()
	require.NoError(t, err)
	require.NotEqual(t, desiredSecret, existingSecret)

	endpointJSON, err := json.Marshal(map[string]string{
		"id":  "ep_test",
		"uid": channelID,
	})
	require.NoError(t, err)

	secretJSON, err := json.Marshal(map[string]string{"key": existingSecret})
	require.NoError(t, err)

	namespace := v4.ProviderNamespace{ID: namespaceID, Name: "Cancellation test"}
	config := v4.WebhookConfig{
		URL:           "https://example.com/webhook",
		SigningSecret: desiredSecret,
	}

	createChannel := func(ctx context.Context, provider v4.WebhookProvider) (any, error) {
		return provider.CreateChannel(ctx, v4.CreateChannelInput[v4.WebhookConfig]{
			Namespace: namespaceID,
			ChannelID: channelID,
			Name:      "Cancellation test",
			Config:    config,
		})
	}

	updateChannel := func(ctx context.Context, provider v4.WebhookProvider) (any, error) {
		return provider.UpdateChannel(ctx, v4.UpdateChannelInput[v4.WebhookConfigUpdate]{
			Namespace: namespaceID,
			ChannelID: channelID,
			Name:      new("Cancellation test"),
			Config: v4.WebhookConfigUpdate{
				URL:           &config.URL,
				CustomHeaders: &config.CustomHeaders,
				SigningSecret: &config.SigningSecret,
			},
		})
	}

	type testCase struct {
		name           string
		run            func(context.Context, v4.WebhookProvider) (any, error)
		responses      []cancellationResponse
		wantMessage    string
		wantChannel    bool
		wantEndpointID bool
	}

	tests := []testCase{
		{
			name: "get namespace",
			run: func(ctx context.Context, provider v4.WebhookProvider) (any, error) {
				return provider.GetNamespace(ctx, v4.GetNamespaceInput{ID: namespaceID})
			},
			wantMessage: "failed to get application",
		},
		{
			name: "create namespace",
			run: func(ctx context.Context, provider v4.WebhookProvider) (any, error) {
				return provider.CreateNamespace(ctx, v4.CreateNamespaceInput{ProviderNamespace: namespace})
			},
			wantMessage: "failed to get or create application",
		},
		{
			name: "update namespace",
			run: func(ctx context.Context, provider v4.WebhookProvider) (any, error) {
				return provider.UpdateNamespace(ctx, v4.UpdateNamespaceInput{ID: namespace.ID, Name: &namespace.Name})
			},
			wantMessage: "failed to update application",
		},
		{
			name: "delete namespace",
			run: func(ctx context.Context, provider v4.WebhookProvider) (any, error) {
				return nil, provider.DeleteNamespace(ctx, v4.DeleteNamespaceInput{ID: namespaceID})
			},
			wantMessage: "failed to delete application",
		},
		{
			name: "get channel",
			run: func(ctx context.Context, provider v4.WebhookProvider) (any, error) {
				return provider.GetChannel(ctx, v4.GetChannelInput{Namespace: namespaceID, ChannelID: channelID})
			},
			wantMessage: "failed to get endpoint",
			wantChannel: true,
		},
		{
			name:        "create channel/create endpoint",
			run:         createChannel,
			wantMessage: "failed to create endpoint",
			wantChannel: true,
		},
		{
			name:           "create channel/set headers",
			run:            createChannel,
			responses:      []cancellationResponse{{status: http.StatusCreated, body: string(endpointJSON)}},
			wantMessage:    "failed to set endpoint headers",
			wantChannel:    true,
			wantEndpointID: true,
		},
		{
			name: "delete channel",
			run: func(ctx context.Context, provider v4.WebhookProvider) (any, error) {
				return nil, provider.DeleteChannel(ctx, v4.DeleteChannelInput{Namespace: namespaceID, ChannelID: channelID})
			},
			wantMessage: "failed to delete endpoint",
			wantChannel: true,
		},
	}

	stages := []struct {
		name        string
		responses   []cancellationResponse
		wantMessage string
	}{
		{
			name:        "patch",
			wantMessage: "failed to update endpoint",
		},
		{
			name:        "set headers",
			responses:   []cancellationResponse{{status: http.StatusOK, body: string(endpointJSON)}},
			wantMessage: "failed to set endpoint headers",
		},
		{
			name: "get secret",
			responses: []cancellationResponse{
				{status: http.StatusOK, body: string(endpointJSON)},
				{status: http.StatusOK, body: `{}`},
			},
			wantMessage: "failed to get endpoint secret",
		},
		{
			name: "rotate secret",
			responses: []cancellationResponse{
				{status: http.StatusOK, body: string(endpointJSON)},
				{status: http.StatusOK, body: `{}`},
				{status: http.StatusOK, body: string(secretJSON)},
			},
			wantMessage: "failed to rotate endpoint secret",
		},
	}

	for _, stage := range stages {
		tests = append(tests, testCase{
			name:           "update channel/" + stage.name,
			run:            updateChannel,
			responses:      stage.responses,
			wantMessage:    stage.wantMessage,
			wantChannel:    true,
			wantEndpointID: len(stage.responses) > 0,
		})

		// Create conflicts delegate to UpdateChannel; preserve cancellation from
		// every recovery stage rather than reporting a completed provisioning.
		responses := []cancellationResponse{{status: http.StatusConflict, body: `{}`}}
		responses = append(responses, stage.responses...)

		tests = append(tests, testCase{
			name:           "create channel/conflict recovery/" + stage.name,
			run:            createChannel,
			responses:      responses,
			wantMessage:    stage.wantMessage,
			wantChannel:    true,
			wantEndpointID: len(stage.responses) > 0,
		})
	}

	events := newEventCancellationFixture(t, namespaceID)

	getEvent := func(ctx context.Context, provider v4.WebhookProvider) (any, error) {
		return provider.GetEvent(ctx, v4.GetEventInput{Event: events.event, Destinations: events.destinations})
	}

	sendEvent := func(ctx context.Context, provider v4.WebhookProvider) (any, error) {
		return provider.SendEvent(ctx, v4.SendEventInput{
			Event: events.event,
			Destinations: []v4.EventDestination{
				{ChannelID: events.destinations[0].ChannelID},
			},
		})
	}

	tests = append(tests,
		testCase{
			name:        "send event/create message",
			run:         sendEvent,
			wantMessage: "failed to create message",
		},
		testCase{
			name:        "send event/conflict recovery",
			run:         sendEvent,
			responses:   []cancellationResponse{{status: http.StatusConflict, body: `{}`}},
			wantMessage: "failed to recover existing message",
		},
		testCase{
			name:        "get event/get message",
			run:         getEvent,
			wantMessage: "failed to get message",
		},
		testCase{
			name:        "get event/list destinations",
			run:         getEvent,
			responses:   []cancellationResponse{events.message},
			wantMessage: "failed to list message destinations",
		},
		testCase{
			name: "get event/list destinations on a later page",
			run:  getEvent,
			responses: []cancellationResponse{
				events.message,
				{status: http.StatusOK, body: `{"data":[],"done":false,"iterator":"next"}`},
			},
			wantMessage: "failed to list message destinations",
		},
		testCase{
			name: "resend event/get message",
			run: func(ctx context.Context, provider v4.WebhookProvider) (any, error) {
				return provider.ResendEvent(ctx, events.resendInput())
			},
			wantMessage: "failed to get message for resend",
		},
	)

	causes := []struct {
		name              string
		err               error
		wantProviderError bool
	}{
		{
			name: "canceled",
			err:  context.Canceled,
		},
		{
			name: "deadline exceeded",
			err:  context.DeadlineExceeded,
		},
		{
			name:              "transport failure",
			err:               io.ErrUnexpectedEOF,
			wantProviderError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, cause := range causes {
				t.Run(cause.name, func(t *testing.T) {
					// Given preceding SDK calls succeed before a wrapped transport error.
					transport := &cancellationTransport{
						cause:     cause.err,
						responses: test.responses,
					}

					provider := newCancellationTestProvider(t, transport)

					// When the selected operation is interrupted at the target SDK call.
					result, err := test.run(t.Context(), provider)

					// Then context is retained, no success is returned, and no later step runs.
					require.Nil(t, result)
					require.ErrorIs(t, err, cause.err)
					require.ErrorContains(t, err, test.wantMessage)
					require.ErrorContains(t, err, "svix.application.uid="+namespaceID)
					require.Equal(t, len(test.responses)+1, transport.calls)

					if strings.Contains(test.name, "event/") {
						require.ErrorContains(t, err, "svix.message.event_id="+events.event.ID)
					}

					if test.name == "get event/list destinations on a later page" {
						require.Contains(t, transport.requests[len(transport.requests)-1], "iterator=next")
					}

					if test.wantChannel {
						require.ErrorContains(t, err, "svix.endpoint.uid="+channelID)
					}

					if test.wantEndpointID {
						require.ErrorContains(t, err, "svix.endpoint.id=ep_test")
					}

					var providerErr *v4.ProviderError

					if cause.wantProviderError {
						require.ErrorAs(t, err, &providerErr)
						require.Equal(t, v4.ProviderErrorKindRetryable, providerErr.Kind)
						require.Equal(t, v4.ProviderErrorCodeUnavailable, providerErr.Code)
					} else {
						require.False(t, errors.As(err, &providerErr), "cancellation must not be classified as a provider failure")
					}
				})
			}
		})
	}
}

// newCancellationTestProvider uses the real SDK without retries so each scripted
// request corresponds to one provider operation boundary.
func newCancellationTestProvider(t *testing.T, transport *cancellationTransport) v4.WebhookProvider {
	t.Helper()

	client, err := svix.New("test-token", &svix.SvixOptions{
		HTTPClient:    &http.Client{Transport: transport},
		RetrySchedule: new([]time.Duration{}),
	})
	require.NoError(t, err)

	provider, err := New(Config{
		Client: client,
		Logger: slog.New(slog.NewTextHandler(t.Output(), nil)),
		Tracer: noop.NewTracerProvider().Tracer("svix-cancellation-test"),
	})
	require.NoError(t, err)

	return provider
}

type cancellationResponse struct {
	status int
	body   string
}

// cancellationTransport injects errors at a specific SDK boundary after any
// preceding calls complete, without relying on live-server timing or retries.
type cancellationTransport struct {
	cause       error
	responses   []cancellationResponse
	calls       int
	onInterrupt func()
	requests    []string
}

func (t *cancellationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Body != nil {
		defer request.Body.Close()
	}

	t.requests = append(t.requests, request.Method+" "+request.URL.RequestURI())

	if err := request.Context().Err(); err != nil {
		return nil, fmt.Errorf("request interrupted: %w", err)
	}

	index := t.calls
	t.calls++

	if index >= len(t.responses) {
		if t.onInterrupt != nil {
			t.onInterrupt()
		}

		cause := t.cause

		if err := request.Context().Err(); err != nil {
			cause = err
		}

		return nil, fmt.Errorf("request interrupted: %w", cause)
	}

	response := t.responses[index]

	return &http.Response{
		StatusCode: response.status,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body:    io.NopCloser(strings.NewReader(response.body)),
		Request: request,
	}, nil
}
