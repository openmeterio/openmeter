package svix

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	svixmodels "github.com/svix/svix-webhooks/v2/go/models"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	notificationprovider "github.com/openmeterio/openmeter/openmeter/notification/v4/provider"
	"github.com/openmeterio/openmeter/pkg/models"
)

var eventCancellationCauses = []struct {
	name              string
	err               error
	cancelContext     bool
	wantProviderError bool
}{
	{
		name: "wrapped cancellation",
		err:  context.Canceled,
	},
	{
		name: "wrapped deadline exceeded",
		err:  context.DeadlineExceeded,
	},
	{
		name:          "real context cancellation",
		err:           context.Canceled,
		cancelContext: true,
	},
	{
		name:              "transport failure",
		err:               io.ErrUnexpectedEOF,
		wantProviderError: true,
	},
}

func TestGetEventCancellationPartialResults(t *testing.T) {
	tests := []struct {
		name      string
		laterPage bool
	}{
		{
			name: "first attempt page",
		},
		{
			name:      "later attempt page",
			laterPage: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, cause := range eventCancellationCauses {
				t.Run(cause.name, func(t *testing.T) {
					// Given A can be fully observed before B's attempt listing is interrupted.
					fixture := newEventCancellationFixture(t, ulid.Make().String())
					input := notificationprovider.GetEventInput{Event: fixture.event, Destinations: fixture.destinations}
					persisted := make(map[string]map[string]string, len(input.Destinations))

					for _, destination := range input.Destinations {
						persisted[destination.ChannelID] = maps.Clone(destination.EventDeliveryStatusProviderData)
					}

					responses := []cancellationResponse{fixture.message, fixture.endpoints, fixture.attempts[0]}
					basePath := "/api/v1/app/" + fixture.event.Namespace
					wantRequests := []string{
						"GET " + basePath + "/msg/" + fixture.event.ID + "?with_content=false",
						"GET " + basePath + "/msg/msg_test/endpoint?limit=100",
						"GET " + basePath + "/attempt/msg/msg_test?endpoint_id=ep_a&expanded_statuses=true&limit=100&with_content=true",
						"GET " + basePath + "/attempt/msg/msg_test?endpoint_id=ep_b&expanded_statuses=true&limit=100&with_content=true",
					}

					if test.laterPage {
						responses = append(responses, cancellationResponse{
							status: http.StatusOK,
							body: `{"data":[{"id":"atm_partial","msgId":"msg_test","endpointId":"ep_b",` +
								`"status":0,"timestamp":"2026-10-22T00:00:00Z"}],"done":false,"iterator":"next"}`,
						})

						wantRequests = append(wantRequests,
							"GET "+basePath+"/attempt/msg/msg_test?endpoint_id=ep_b&expanded_statuses=true&iterator=next&limit=100&with_content=true",
						)
					}

					wantRequests = append(wantRequests,
						"GET "+basePath+"/attempt/msg/msg_test?endpoint_id=ep_c&expanded_statuses=true&limit=100&with_content=true",
					)

					ctx, cancel := context.WithCancel(t.Context())
					defer cancel()

					transport := &cancellationTransport{cause: cause.err, responses: responses}

					if cause.cancelContext {
						transport.onInterrupt = cancel
						transport.cause = errors.New("caller context cancellation was not propagated")
					}

					provider := newCancellationTestProvider(t, transport)

					// When B is interrupted, C must still have an identified error result.
					result, err := provider.GetEvent(ctx, input)
					require.NoError(t, err)
					require.NotNil(t, result)
					require.Equal(t, map[string]string{"svix.message_id": "msg_test"}, result.EventProviderData)
					require.Len(t, result.Destinations, 3)
					require.Equal(t, wantRequests, transport.requests)

					byChannel := make(map[string]notificationprovider.GetEventDestinationResult)

					for _, destination := range result.Destinations {
						require.NotContains(t, byChannel, destination.ChannelID)
						byChannel[destination.ChannelID] = destination
					}

					for _, destination := range input.Destinations {
						require.Contains(t, byChannel, destination.ChannelID)
						require.Equal(t, persisted[destination.ChannelID], destination.EventDeliveryStatusProviderData)
					}

					// Then A's completed observation and snapshot survive the later errors.
					completed := byChannel[input.Destinations[0].ChannelID]
					require.NoError(t, completed.Error)
					require.Equal(t, v4.EventDeliveryStatusStateSuccess, completed.State)
					require.Nil(t, completed.NextObservationAt)
					require.Equal(t, map[string]string{
						"svix.dispatch_generation": "1",
						"svix.observed_attempts":   `["atm_a"]`,
					}, completed.EventDeliveryStatusProviderData)
					require.Len(t, completed.Attempts, 1)
					require.Equal(t, v4.EventDeliveryStatusStateSuccess, completed.Attempts[0].State)
					require.Equal(t, fixture.event.CreatedAt, completed.Attempts[0].Timestamp)
					require.Equal(t, map[string]string{"svix.attempt_id": "atm_a"}, completed.Attempts[0].ProviderData)
					require.JSONEq(t,
						`{"status_code":204,"body":"","duration_ms":1,"url":"https://example.com/a","response_truncated":false}`,
						string(completed.Attempts[0].Response),
					)

					for _, destination := range input.Destinations[1:] {
						interrupted := byChannel[destination.ChannelID]

						// Never replace persisted history with a partially fetched snapshot.
						require.ErrorIs(t, interrupted.Error, cause.err)
						require.ErrorContains(t, interrupted.Error, "failed to list message attempts")
						require.ErrorContains(t, interrupted.Error, "svix.application.uid="+fixture.event.Namespace)
						require.ErrorContains(t, interrupted.Error, "svix.message.event_id="+fixture.event.ID)
						require.ErrorContains(t, interrupted.Error, "svix.endpoint.uid="+destination.ChannelID)
						require.Equal(t, v4.EventDeliveryStatusStateSending, interrupted.State)
						require.Empty(t, interrupted.Attempts)
						require.Nil(t, interrupted.NextObservationAt)
						require.Equal(t, persisted[destination.ChannelID], interrupted.EventDeliveryStatusProviderData)

						var providerErr *notificationprovider.ProviderError

						if cause.wantProviderError {
							require.ErrorAs(t, interrupted.Error, &providerErr)
							require.Equal(t, notificationprovider.ProviderErrorKindRetryable, providerErr.Kind)
							require.Equal(t, notificationprovider.ProviderErrorCodeUnavailable, providerErr.Code)
						} else {
							require.False(t, errors.As(interrupted.Error, &providerErr))
						}
					}

					if cause.cancelContext {
						require.ErrorIs(t, ctx.Err(), context.Canceled)
						require.Equal(t, len(responses)+1, transport.calls)
					} else {
						require.NoError(t, ctx.Err())
						require.Equal(t, len(responses)+2, transport.calls)
					}
				})
			}
		})
	}
}

func TestResendEventCancellationPartialResults(t *testing.T) {
	tests := []struct {
		name           string
		lookupEndpoint bool
	}{
		{
			name: "resend request",
		},
		{
			name:           "endpoint lookup",
			lookupEndpoint: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, cause := range eventCancellationCauses {
				t.Run(cause.name, func(t *testing.T) {
					// Given A's resend is accepted before B's provider call is interrupted.
					fixture := newEventCancellationFixture(t, ulid.Make().String())
					input := fixture.resendInput()
					responses := []cancellationResponse{
						fixture.message,
						{status: http.StatusNoContent},
					}

					basePath := "/api/v1/app/" + fixture.event.Namespace
					interruptedRequest := "POST " + basePath + "/msg/msg_test/endpoint/ep_b/resend"

					if test.lookupEndpoint {
						input.Destinations[1].ChannelProviderData = nil
						interruptedRequest = "GET " + basePath + "/endpoint/" + input.Destinations[1].ChannelID
					}

					ctx, cancel := context.WithCancel(t.Context())
					defer cancel()

					transport := &cancellationTransport{cause: cause.err, responses: responses}

					if cause.cancelContext {
						transport.onInterrupt = cancel
						transport.cause = errors.New("caller context cancellation was not propagated")
					}

					provider := newCancellationTestProvider(t, transport)

					// When B is interrupted, A must not be submitted again or discarded.
					result, err := provider.ResendEvent(ctx, input)
					require.NoError(t, err)
					require.NotNil(t, result)
					require.Equal(t, map[string]string{"svix.message_id": "msg_test"}, result.EventProviderData)
					require.Len(t, result.Destinations, 3)
					require.Equal(t, []string{
						"GET " + basePath + "/msg/" + fixture.event.ID + "?with_content=false",
						"POST " + basePath + "/msg/msg_test/endpoint/ep_a/resend",
						interruptedRequest,
						"POST " + basePath + "/msg/msg_test/endpoint/ep_c/resend",
					}, transport.requests)

					byChannel := make(map[string]notificationprovider.ResendEventDestinationResult)

					for _, destination := range result.Destinations {
						require.NotContains(t, byChannel, destination.ChannelID)
						byChannel[destination.ChannelID] = destination
					}

					for _, destination := range input.Destinations {
						require.Contains(t, byChannel, destination.ChannelID)
						require.Equal(t, map[string]string{
							"svix.dispatch_generation": "1",
							"svix.observed_attempts":   `["atm_prior"]`,
						}, destination.EventDeliveryStatusProviderData)
					}

					// Then A retains its new generation and the persisted pre-resend baseline.
					completed := byChannel[input.Destinations[0].ChannelID]
					require.NoError(t, completed.Error)
					require.Equal(t, map[string]string{
						"svix.dispatch_generation": "2",
						"svix.resend_baseline":     `["atm_prior"]`,
					}, completed.EventDeliveryStatusProviderData)

					for _, destination := range input.Destinations[1:] {
						interrupted := byChannel[destination.ChannelID]
						require.ErrorIs(t, interrupted.Error, cause.err)
						require.ErrorContains(t, interrupted.Error, "svix.application.uid="+fixture.event.Namespace)
						require.ErrorContains(t, interrupted.Error, "svix.message.event_id="+fixture.event.ID)
						require.ErrorContains(t, interrupted.Error, "svix.endpoint.uid="+destination.ChannelID)

						if test.lookupEndpoint && destination.ChannelID == input.Destinations[1].ChannelID {
							require.ErrorContains(t, interrupted.Error, "failed to get endpoint for resend")
						} else {
							require.ErrorContains(t, interrupted.Error, "failed to resend message")
						}

						// An interrupted request must not claim an accepted generation.
						require.Empty(t, interrupted.EventDeliveryStatusProviderData)
						require.Nil(t, interrupted.NextObservationAt)

						var providerErr *notificationprovider.ProviderError

						if cause.wantProviderError {
							require.ErrorAs(t, interrupted.Error, &providerErr)
							require.Equal(t, notificationprovider.ProviderErrorKindRetryable, providerErr.Kind)
							require.Equal(t, notificationprovider.ProviderErrorCodeUnavailable, providerErr.Code)
						} else {
							require.False(t, errors.As(interrupted.Error, &providerErr))
						}
					}

					if cause.cancelContext {
						require.ErrorIs(t, ctx.Err(), context.Canceled)
						require.Equal(t, len(responses)+1, transport.calls)
					} else {
						require.NoError(t, ctx.Err())
						require.Equal(t, len(responses)+2, transport.calls)
					}
				})
			}
		})
	}
}

func TestEventOperationCanceledContext(t *testing.T) {
	fixture := newEventCancellationFixture(t, ulid.Make().String())

	operations := []struct {
		name string
		run  func(context.Context, notificationprovider.WebhookProvider) (any, error)
	}{
		{
			name: "send",
			run: func(ctx context.Context, provider notificationprovider.WebhookProvider) (any, error) {
				return provider.SendEvent(ctx, notificationprovider.SendEventInput{
					Event: fixture.event,
					Destinations: []notificationprovider.EventDestination{
						{ChannelID: fixture.destinations[0].ChannelID},
					},
				})
			},
		},
		{
			name: "get",
			run: func(ctx context.Context, provider notificationprovider.WebhookProvider) (any, error) {
				return provider.GetEvent(ctx, notificationprovider.GetEventInput{Event: fixture.event, Destinations: fixture.destinations})
			},
		},
		{
			name: "resend",
			run: func(ctx context.Context, provider notificationprovider.WebhookProvider) (any, error) {
				return provider.ResendEvent(ctx, fixture.resendInput())
			},
		},
	}

	contexts := []struct {
		name     string
		deadline bool
		wantErr  error
	}{
		{
			name:    "already canceled",
			wantErr: context.Canceled,
		},
		{
			name:     "already expired deadline",
			deadline: true,
			wantErr:  context.DeadlineExceeded,
		},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			for _, test := range contexts {
				t.Run(test.name, func(t *testing.T) {
					// Given an interrupted caller context and an otherwise successful SDK script.
					var (
						ctx    context.Context
						cancel context.CancelFunc
					)

					if test.deadline {
						ctx, cancel = context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
					} else {
						ctx, cancel = context.WithCancel(t.Context())
						cancel()
					}

					defer cancel()
					require.ErrorIs(t, ctx.Err(), test.wantErr)

					transport := &cancellationTransport{
						cause: errors.New("caller context was not propagated"),
						responses: []cancellationResponse{
							fixture.message,
							fixture.endpoints,
							fixture.attempts[0],
							fixture.attempts[1],
							fixture.attempts[2],
						},
					}

					provider := newCancellationTestProvider(t, transport)

					// When called, the provider must propagate that context to the SDK.
					result, err := operation.run(ctx, provider)

					// Then no scripted response is consumed and no acceptance is reported.
					require.Nil(t, result)
					require.ErrorIs(t, err, test.wantErr)
					require.ErrorContains(t, err, "svix.application.uid="+fixture.event.Namespace)
					require.ErrorContains(t, err, "svix.message.event_id="+fixture.event.ID)
					require.Zero(t, transport.calls)

					var providerErr *notificationprovider.ProviderError

					require.False(t, errors.As(err, &providerErr))
				})
			}
		})
	}
}

type eventCancellationFixture struct {
	event        v4.Event
	destinations []notificationprovider.GetEventDestination
	message      cancellationResponse
	endpoints    cancellationResponse
	attempts     []cancellationResponse
}

// newEventCancellationFixture supplies matching message, endpoint, and attempt
// identities so injected interruptions are not confused with malformed responses.
func newEventCancellationFixture(t *testing.T, namespace string) eventCancellationFixture {
	t.Helper()

	fixture := eventCancellationFixture{
		event: v4.Event{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: ulid.Make().String()},
			EventType:    v4.EventType{Name: "invoice.created", Version: 1},
			CreatedAt:    time.Date(2026, time.October, 22, 0, 0, 0, 0, time.UTC),
			Payload:      json.RawMessage(`{"invoice_id":"invoice-test"}`),
		},
	}

	messageJSON, err := json.Marshal(svixmodels.MessageOut{
		Id:        "msg_test",
		EventId:   &fixture.event.ID,
		EventType: fixture.event.EventType.String(),
		Timestamp: fixture.event.CreatedAt,
	})
	require.NoError(t, err)

	fixture.message = cancellationResponse{status: http.StatusOK, body: string(messageJSON)}
	endpoints := svixmodels.ListResponseMessageEndpointOut{Done: true}

	for _, suffix := range []string{"a", "b", "c"} {
		channelID := ulid.Make().String()
		endpointID := "ep_" + suffix

		fixture.destinations = append(fixture.destinations, notificationprovider.GetEventDestination{
			ChannelID:           channelID,
			ChannelProviderData: map[string]string{"svix.endpoint_id": endpointID},
			EventDeliveryStatusProviderData: map[string]string{
				"svix.dispatch_generation": "1",
				"svix.observed_attempts":   `["atm_prior"]`,
			},
			DispatchGeneration: 1,
		})

		endpoints.Data = append(endpoints.Data, svixmodels.MessageEndpointOut{
			Id:         endpointID,
			Uid:        &channelID,
			Status:     svixmodels.MESSAGESTATUS_SUCCESS,
			StatusText: svixmodels.MESSAGESTATUSTEXT_SUCCESS,
		})

		attemptsJSON, err := json.Marshal(svixmodels.ListResponseMessageAttemptOut{
			Done: true,
			Data: []svixmodels.MessageAttemptOut{
				{
					Id:                 "atm_" + suffix,
					MsgId:              "msg_test",
					EndpointId:         endpointID,
					Status:             svixmodels.MESSAGESTATUS_SUCCESS,
					StatusText:         svixmodels.MESSAGESTATUSTEXT_SUCCESS,
					Timestamp:          fixture.event.CreatedAt,
					ResponseStatusCode: http.StatusNoContent,
					ResponseDurationMs: 1,
					Url:                "https://example.com/" + suffix,
				},
			},
		})
		require.NoError(t, err)

		fixture.attempts = append(fixture.attempts, cancellationResponse{status: http.StatusOK, body: string(attemptsJSON)})
	}

	endpointsJSON, err := json.Marshal(endpoints)
	require.NoError(t, err)

	fixture.endpoints = cancellationResponse{status: http.StatusOK, body: string(endpointsJSON)}

	return fixture
}

// resendInput advances the requested generation without changing the persisted
// observation, matching the reconciler's retryable resend contract.
func (f eventCancellationFixture) resendInput() notificationprovider.ResendEventInput {
	destinations := slices.Clone(f.destinations)

	for i := range destinations {
		destinations[i].DispatchGeneration = 2
	}

	return notificationprovider.ResendEventInput{
		Event:          f.event,
		Destinations:   destinations,
		IdempotencyKey: new("resend-cancellation-test"),
	}
}
