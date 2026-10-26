package svix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"
	svixv2 "github.com/svix/svix-webhooks/v2/go"
	svixmodels "github.com/svix/svix-webhooks/v2/go/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	tracexv2 "github.com/openmeterio/openmeter/pkg/framework/tracex/v2"
)

const (
	providerDataMessageID                 = "svix.message_id"
	providerDataAttemptID                 = "svix.attempt_id"
	providerDataDispatchGeneration        = "svix.dispatch_generation"
	providerDataObservedAttempts          = "svix.observed_attempts"
	providerDataResendBaseline            = "svix.resend_baseline"
	eventPageSize                  uint64 = 100
)

func (p *Provider) SendEvent(ctx context.Context, input v4.SendEventInput) (*v4.SendEventResult, error) {
	fn := func(ctx context.Context) (*v4.SendEventResult, error) {
		if err := input.Validate(); err != nil {
			return nil, fmt.Errorf(
				"invalid send message input [svix.message.event_id=%s svix.application.uid=%s]: %w",
				input.Event.ID,
				input.Event.Namespace,
				err,
			)
		}

		// One message fans out according to endpoint configuration. Destinations
		// describe Notification's tracking set, not Svix message channels.
		message, err := p.client.Message().Create(ctx, input.Event.Namespace, toAPIMessageIn(input.Event),
			&svixv2.MessageCreateOptions{
				WithContent:    new(false),
				IdempotencyKey: input.IdempotencyKey,
			},
		)

		if svixErr, ok := errors.AsType[*svixv2.Error](err); ok && svixErr.Status() == http.StatusConflict {
			// Recover a message accepted before its response or local state was lost.
			message, err = p.getMessage(ctx, input.Event)
			if err != nil {
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					err = normalizeError(err)
				}

				return nil, fmt.Errorf(
					"failed to recover existing message [svix.message.event_id=%s svix.application.uid=%s]: %w",
					input.Event.ID,
					input.Event.Namespace,
					err,
				)
			}
		}

		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				err = normalizeError(err)
			}

			return nil, fmt.Errorf(
				"failed to create message [svix.message.event_id=%s svix.application.uid=%s]: %w",
				input.Event.ID,
				input.Event.Namespace,
				err,
			)
		}

		if lo.FromPtr(message).Id == "" {
			return nil, fmt.Errorf(
				"invalid send message response [svix.message.id=%s svix.message.event_id=%s svix.application.uid=%s]: %w",
				lo.FromPtr(message).Id,
				input.Event.ID,
				input.Event.Namespace,
				normalizeError(errors.New("missing message ID")),
			)
		}

		if lo.FromPtr(message.EventId) != input.Event.ID {
			return nil, fmt.Errorf(
				"invalid send message response [svix.message.id=%s svix.message.event_id=%s svix.application.uid=%s]: %w",
				message.Id,
				input.Event.ID,
				input.Event.Namespace,
				normalizeError(fmt.Errorf("unexpected event ID: %s", lo.FromPtr(message.EventId))),
			)
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("svix.message.id", message.Id),
		)

		result := &v4.SendEventResult{
			EventProviderData: v4.ProviderData{providerDataMessageID: message.Id},
			Destinations:      make([]v4.SendEventDestinationResult, 0, len(input.Destinations)),
		}

		for _, destination := range input.Destinations {
			result.Destinations = append(result.Destinations, v4.SendEventDestinationResult{
				ChannelID: destination.ChannelID,
				EventDeliveryStatusProviderData: v4.ProviderData{
					providerDataDispatchGeneration: "1",
				},
			})
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.Int("notification.destination.error_count", 0),
		)

		return result, nil
	}

	return tracexv2.NewSpan(
		p.tracer,
		"notification.provider.svix.send_event",
		trace.WithAttributes(
			attribute.String("svix.application.uid", input.Event.Namespace),
			attribute.String("svix.message.event_id", input.Event.ID),
			attribute.Int("notification.destination.count", len(input.Destinations)),
		),
	).Do2(ctx, fn)
}

func (p *Provider) ResendEvent(ctx context.Context, input v4.ResendEventInput) (*v4.ResendEventResult, error) {
	fn := func(ctx context.Context) (*v4.ResendEventResult, error) {
		if err := input.Validate(); err != nil {
			return nil, fmt.Errorf(
				"invalid resend message input [svix.message.event_id=%s svix.application.uid=%s]: %w",
				input.Event.ID,
				input.Event.Namespace,
				err,
			)
		}

		message, err := p.getMessage(ctx, input.Event)
		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				err = normalizeError(err)
			}

			return nil, fmt.Errorf(
				"failed to get message for resend [svix.message.event_id=%s svix.application.uid=%s]: %w",
				input.Event.ID,
				input.Event.Namespace,
				err,
			)
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("svix.message.id", message.Id),
		)

		result := &v4.ResendEventResult{
			EventProviderData: v4.ProviderData{providerDataMessageID: message.Id},
			Destinations:      make([]v4.ResendEventDestinationResult, 0, len(input.Destinations)),
		}

		destinationErrors := 0

		for _, destination := range input.Destinations {
			fn := func(ctx context.Context) (v4.ResendEventDestinationResult, error) {
				outcome := v4.ResendEventDestinationResult{ChannelID: destination.ChannelID}
				endpointID := destination.ChannelProviderData[providerDataEndpointID]

				if endpointID == "" {
					channel, err := p.GetChannel(ctx, v4.GetChannelInput{
						Namespace: input.Event.Namespace,
						ChannelID: destination.ChannelID,
					})
					if err != nil {
						outcome.Error = fmt.Errorf(
							"failed to get endpoint for resend [svix.message.id=%s svix.message.event_id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
							message.Id,
							input.Event.ID,
							destination.ChannelID,
							input.Event.Namespace,
							err,
						)

						return outcome, outcome.Error
					}

					endpointID = channel.ProviderData[providerDataEndpointID]
				}

				trace.SpanFromContext(ctx).SetAttributes(
					attribute.String("svix.endpoint.id", endpointID),
				)

				baseline, err := resolveResendBaseline(destination)
				if err != nil {
					outcome.Error = fmt.Errorf(
						"invalid resend observation data [svix.message.id=%s svix.message.event_id=%s svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
						message.Id,
						input.Event.ID,
						endpointID,
						destination.ChannelID,
						input.Event.Namespace,
						&v4.ProviderError{
							Kind: v4.ProviderErrorKindConfiguration,
							Code: v4.ProviderErrorCodeConfigurationInvalid,
							Err:  err,
						},
					)

					return outcome, outcome.Error
				}

				baselineJSON, err := json.Marshal(baseline)
				if err != nil {
					outcome.Error = fmt.Errorf(
						"failed to encode resend baseline [svix.message.id=%s svix.message.event_id=%s svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
						message.Id,
						input.Event.ID,
						endpointID,
						destination.ChannelID,
						input.Event.Namespace,
						normalizeError(err),
					)

					return outcome, outcome.Error
				}

				_, err = p.client.MessageAttempt().Resend(ctx, input.Event.Namespace, message.Id, endpointID,
					&svixv2.MessageAttemptResendOptions{IdempotencyKey: input.IdempotencyKey},
				)
				if err != nil {
					if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
						err = normalizeError(err)
					}

					outcome.Error = fmt.Errorf(
						"failed to resend message [svix.message.id=%s svix.message.event_id=%s svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
						message.Id,
						input.Event.ID,
						endpointID,
						destination.ChannelID,
						input.Event.Namespace,
						err,
					)
				} else {
					outcome.EventDeliveryStatusProviderData = v4.ProviderData{
						providerDataDispatchGeneration: strconv.FormatUint(destination.DispatchGeneration, 10),
						providerDataResendBaseline:     string(baselineJSON),
					}
				}

				return outcome, outcome.Error
			}

			outcome, err := tracexv2.NewSpan(
				p.tracer,
				"notification.provider.svix.resend_event_destination",
				trace.WithAttributes(
					attribute.String("svix.application.uid", input.Event.Namespace),
					attribute.String("svix.message.event_id", input.Event.ID),
					attribute.String("svix.message.id", message.Id),
					attribute.String("svix.endpoint.uid", destination.ChannelID),
					attribute.String("notification.dispatch_generation", strconv.FormatUint(destination.DispatchGeneration, 10)),
				),
			).Do2(ctx, fn)

			outcome.Error = err
			if err != nil {
				destinationErrors++
			}

			result.Destinations = append(result.Destinations, outcome)
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.Int("notification.destination.error_count", destinationErrors),
		)

		return result, nil
	}

	return tracexv2.NewSpan(
		p.tracer,
		"notification.provider.svix.resend_event",
		trace.WithAttributes(
			attribute.String("svix.application.uid", input.Event.Namespace),
			attribute.String("svix.message.event_id", input.Event.ID),
			attribute.Int("notification.destination.count", len(input.Destinations)),
		),
	).Do2(ctx, fn)
}

func (p *Provider) GetEvent(ctx context.Context, input v4.GetEventInput) (*v4.GetEventResult, error) {
	fn := func(ctx context.Context) (*v4.GetEventResult, error) {
		if err := input.Validate(); err != nil {
			return nil, fmt.Errorf(
				"invalid get message input [svix.message.event_id=%s svix.application.uid=%s]: %w",
				input.Event.ID,
				input.Event.Namespace,
				err,
			)
		}

		message, err := p.getMessage(ctx, input.Event)
		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				providerErr := normalizeError(err)
				if providerErr.Code == v4.ProviderErrorCodeResourceNotFound {
					// A previously accepted message may not be visible yet. Observation
					// retries must not turn this into a new dispatch.
					providerErr.Kind = v4.ProviderErrorKindRetryable
				}

				err = providerErr
			}

			return nil, fmt.Errorf(
				"failed to get message [svix.message.event_id=%s svix.application.uid=%s]: %w",
				input.Event.ID,
				input.Event.Namespace,
				err,
			)
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("svix.message.id", message.Id),
		)

		endpoints := make(map[string]svixmodels.MessageEndpointOut)
		options := &svixv2.MessageAttemptListAttemptedDestinationsOptions{
			Limit: new(eventPageSize),
		}
		seen := make(map[string]struct{})

		for {
			page, err := p.client.MessageAttempt().ListAttemptedDestinations(ctx, input.Event.Namespace, message.Id, options)
			if err != nil {
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					err = normalizeError(err)
				}

				return nil, fmt.Errorf(
					"failed to list message destinations [svix.message.id=%s svix.message.event_id=%s svix.application.uid=%s]: %w",
					message.Id,
					input.Event.ID,
					input.Event.Namespace,
					err,
				)
			}

			if page == nil {
				return nil, fmt.Errorf(
					"invalid message destinations response [svix.message.id=%s svix.message.event_id=%s svix.application.uid=%s]: %w",
					message.Id,
					input.Event.ID,
					input.Event.Namespace,
					normalizeError(errors.New("nil response")),
				)
			}

			for _, endpoint := range page.Data {
				endpoints[endpoint.Id] = endpoint
			}

			if page.Done {
				break
			}

			cursor := lo.FromPtr(page.Iterator)
			if _, exists := seen[cursor]; cursor == "" || exists {
				return nil, fmt.Errorf(
					"invalid message destinations pagination [svix.message.id=%s svix.message.event_id=%s svix.application.uid=%s]: %w",
					message.Id,
					input.Event.ID,
					input.Event.Namespace,
					normalizeError(errors.New("pagination did not advance")),
				)
			}

			seen[cursor] = struct{}{}
			options.Iterator = page.Iterator
		}

		result := &v4.GetEventResult{
			EventProviderData: v4.ProviderData{providerDataMessageID: message.Id},
			Destinations:      make([]v4.GetEventDestinationResult, 0, len(input.Destinations)),
		}

		destinationErrors := 0

		for _, destination := range input.Destinations {
			fn := func(ctx context.Context) (v4.GetEventDestinationResult, error) {
				outcome := v4.GetEventDestinationResult{
					ChannelID:                       destination.ChannelID,
					State:                           v4.EventDeliveryStatusStateSending,
					EventDeliveryStatusProviderData: svixProviderData(destination.EventDeliveryStatusProviderData),
					Attempts:                        []v4.EventDeliveryAttempt{},
				}

				endpoint, found := endpoints[destination.ChannelProviderData[providerDataEndpointID]]
				if !found {
					for _, candidate := range endpoints {
						if lo.FromPtr(candidate.Uid) == destination.ChannelID {
							endpoint, found = candidate, true
							break
						}
					}
				}

				// Acceptance precedes visibility of destinations and attempts. Absence
				// is not failure and must never initiate another send.
				if !found {
					outcome.Error = &v4.ProviderError{
						Kind: v4.ProviderErrorKindRetryable,
						Code: v4.ProviderErrorCodeUnavailable,
						Err: fmt.Errorf(
							"failed to observe message destination [svix.message.id=%s svix.message.event_id=%s svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: destination is not visible yet",
							message.Id,
							input.Event.ID,
							destination.ChannelProviderData[providerDataEndpointID],
							destination.ChannelID,
							input.Event.Namespace,
						),
					}

					return outcome, outcome.Error
				}

				trace.SpanFromContext(ctx).SetAttributes(
					attribute.String("svix.endpoint.id", endpoint.Id),
				)

				attempts, err := p.listEventAttempts(ctx, listEventAttemptsInput{
					Namespace:  input.Event.Namespace,
					MessageID:  message.Id,
					EndpointID: endpoint.Id,
				})
				if err != nil {
					if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
						providerErr := normalizeError(err)
						if providerErr.Code == v4.ProviderErrorCodeResourceNotFound {
							providerErr.Kind = v4.ProviderErrorKindRetryable
						}

						err = providerErr
					}

					outcome.Error = fmt.Errorf(
						"failed to list message attempts [svix.message.id=%s svix.message.event_id=%s svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
						message.Id,
						input.Event.ID,
						endpoint.Id,
						destination.ChannelID,
						input.Event.Namespace,
						err,
					)

					return outcome, outcome.Error
				}

				var baseline []string

				if destination.DispatchGeneration > 1 {
					var metadataErr error

					baseline, metadataErr = resolveResendBaseline(destination)
					if metadataErr != nil {
						outcome.Error = fmt.Errorf(
							"invalid resend observation data [svix.message.id=%s svix.message.event_id=%s svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
							message.Id,
							input.Event.ID,
							endpoint.Id,
							destination.ChannelID,
							input.Event.Namespace,
							&v4.ProviderError{
								Kind: v4.ProviderErrorKindRetryable,
								Code: v4.ProviderErrorCodeUnavailable,
								Err:  metadataErr,
							},
						)

						return outcome, outcome.Error
					}
				}

				var latest *svixmodels.MessageAttemptOut

				for _, attempt := range attempts {
					mapped, err := fromAPIMessageAttemptOut(input.Event.Namespace, attempt)
					if err != nil {
						outcome.Error = fmt.Errorf(
							"failed to map message attempt [svix.attempt.id=%s svix.message.id=%s svix.message.event_id=%s svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
							attempt.Id,
							message.Id,
							input.Event.ID,
							endpoint.Id,
							destination.ChannelID,
							input.Event.Namespace,
							normalizeError(err),
						)

						break
					}

					outcome.Attempts = append(outcome.Attempts, *mapped)

					if !slices.Contains(baseline, attempt.Id) && (latest == nil || attempt.Timestamp.After(latest.Timestamp)) {
						latest = &attempt
					}
				}

				if outcome.Error == nil && (destination.DispatchGeneration == 1 || latest != nil) {
					status := endpoint.Status
					if latest != nil {
						// Attempts expose canceled explicitly; the aggregate API may report
						// canceled attempts as success for backwards compatibility.
						status = latest.Status
					}

					outcome.State, err = fromAPIMessageStatus(input.Event.Namespace, status)
					if err != nil {
						outcome.Error = fmt.Errorf(
							"invalid message status [svix.message.id=%s svix.message.event_id=%s svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
							message.Id,
							input.Event.ID,
							endpoint.Id,
							destination.ChannelID,
							input.Event.Namespace,
							normalizeError(err),
						)
					}

					if status == svixmodels.MESSAGESTATUS_FAIL && endpoint.NextAttempt != nil {
						outcome.State = v4.EventDeliveryStatusStateSending
					}
				}

				if outcome.State == v4.EventDeliveryStatusStateSending {
					outcome.NextObservationAt = endpoint.NextAttempt
				}

				if outcome.Error == nil {
					observedAttemptIDs := make([]string, 0, len(outcome.Attempts))

					for _, attempt := range outcome.Attempts {
						observedAttemptIDs = append(observedAttemptIDs, attempt.ProviderData[providerDataAttemptID])
					}

					slices.Sort(observedAttemptIDs)

					observedJSON, err := json.Marshal(observedAttemptIDs)
					if err != nil {
						outcome.Error = fmt.Errorf(
							"failed to encode observed attempts [svix.message.id=%s svix.message.event_id=%s svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
							message.Id,
							input.Event.ID,
							endpoint.Id,
							destination.ChannelID,
							input.Event.Namespace,
							normalizeError(err),
						)

						return outcome, outcome.Error
					}

					data := svixProviderData(destination.EventDeliveryStatusProviderData)
					data[providerDataDispatchGeneration] = strconv.FormatUint(destination.DispatchGeneration, 10)
					data[providerDataObservedAttempts] = string(observedJSON)

					if destination.DispatchGeneration > 1 {
						baselineJSON, err := json.Marshal(baseline)
						if err != nil {
							outcome.Error = fmt.Errorf(
								"failed to encode resend baseline [svix.message.id=%s svix.message.event_id=%s svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
								message.Id,
								input.Event.ID,
								endpoint.Id,
								destination.ChannelID,
								input.Event.Namespace,
								normalizeError(err),
							)

							return outcome, outcome.Error
						}

						data[providerDataResendBaseline] = string(baselineJSON)
					}

					outcome.EventDeliveryStatusProviderData = data

					trace.SpanFromContext(ctx).SetAttributes(
						attribute.String("notification.delivery.state", string(outcome.State)),
					)
				}

				return outcome, outcome.Error
			}

			outcome, err := tracexv2.NewSpan(
				p.tracer,
				"notification.provider.svix.get_event_destination",
				trace.WithAttributes(
					attribute.String("svix.application.uid", input.Event.Namespace),
					attribute.String("svix.message.event_id", input.Event.ID),
					attribute.String("svix.message.id", message.Id),
					attribute.String("svix.endpoint.uid", destination.ChannelID),
					attribute.String("notification.dispatch_generation", strconv.FormatUint(destination.DispatchGeneration, 10)),
				),
			).Do2(ctx, fn)

			outcome.Error = err
			if err != nil {
				destinationErrors++
			}

			result.Destinations = append(result.Destinations, outcome)
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.Int("notification.destination.error_count", destinationErrors),
		)

		return result, nil
	}

	return tracexv2.NewSpan(
		p.tracer,
		"notification.provider.svix.get_event",
		trace.WithAttributes(
			attribute.String("svix.application.uid", input.Event.Namespace),
			attribute.String("svix.message.event_id", input.Event.ID),
			attribute.Int("notification.destination.count", len(input.Destinations)),
		),
	).Do2(ctx, fn)
}

// resolveResendBaseline uses the previous generation's persisted observation for
// a new resend, or preserves the current generation's established baseline.
// Reading live attempts here would absorb an accepted resend when retrying it.
func resolveResendBaseline(destination v4.GetEventDestination) ([]string, error) {
	generation := destination.DispatchGeneration
	if generation < 2 {
		return nil, errors.New("invalid resend generation: must be at least 2")
	}

	previousGeneration := generation - 1
	data := destination.EventDeliveryStatusProviderData

	var encoded string

	switch data[providerDataDispatchGeneration] {
	case strconv.FormatUint(generation, 10):
		encoded = data[providerDataResendBaseline]

	case strconv.FormatUint(previousGeneration, 10):
		encoded = data[providerDataObservedAttempts]

	default:
		return nil, fmt.Errorf("invalid observation generation: expected %d or %d, got %q",
			previousGeneration, generation, data[providerDataDispatchGeneration])
	}

	var baseline []string

	if err := json.Unmarshal([]byte(encoded), &baseline); err != nil {
		return nil, fmt.Errorf("invalid resend baseline: %w", err)
	}

	if baseline == nil {
		return nil, errors.New("invalid resend baseline: expected a JSON array")
	}

	seen := make(map[string]struct{}, len(baseline))

	for _, attemptID := range baseline {
		if strings.TrimSpace(attemptID) == "" {
			return nil, errors.New("invalid resend baseline: empty attempt ID")
		}

		if _, exists := seen[attemptID]; exists {
			return nil, fmt.Errorf("invalid resend baseline: duplicate attempt ID %q", attemptID)
		}

		seen[attemptID] = struct{}{}
	}

	slices.Sort(baseline)

	return baseline, nil
}

func (p *Provider) getMessage(ctx context.Context, event v4.Event) (*svixv2.MessageOut, error) {
	message, err := p.client.Message().Get(ctx, event.Namespace, event.ID, &svixv2.MessageGetOptions{
		WithContent: new(false),
	})
	if err != nil {
		return nil, err
	}

	var responseErr error

	switch {
	case lo.FromPtr(message).Id == "":
		responseErr = errors.New("missing message ID")

	case lo.FromPtr(message.EventId) != event.ID:
		responseErr = fmt.Errorf(
			"unexpected event ID: %s",
			lo.FromPtr(message.EventId),
		)

	default:
		return message, nil
	}

	return nil, fmt.Errorf(
		"invalid get message response [svix.message.id=%s svix.message.event_id=%s svix.application.uid=%s]: %w",
		lo.FromPtr(message).Id,
		event.ID,
		event.Namespace,
		responseErr,
	)
}
