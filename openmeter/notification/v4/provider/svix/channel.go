package svix

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/samber/lo"
	svix "github.com/svix/svix-webhooks/v2/go"
	svixutils "github.com/svix/svix-webhooks/v2/go/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	notificationprovider "github.com/openmeterio/openmeter/openmeter/notification/v4/provider"
	webhooksecret "github.com/openmeterio/openmeter/openmeter/notification/webhook/secret"
	tracexv2 "github.com/openmeterio/openmeter/pkg/framework/tracex/v2"
)

const providerDataEndpointID = "svix.endpoint_id"

func (p *Provider) GetChannel(ctx context.Context, input notificationprovider.GetChannelInput) (*notificationprovider.GetChannelResult, error) {
	fn := func(ctx context.Context) (*notificationprovider.GetChannelResult, error) {
		if err := input.Validate(); err != nil {
			return nil, fmt.Errorf(
				"invalid get endpoint input [svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		endpoint, err := p.client.Endpoint().Get(ctx, input.Namespace, input.ChannelID)
		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				err = normalizeError(err)
			}

			return nil, fmt.Errorf(
				"failed to get endpoint [svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		result, err := fromAPIEndpointOut(endpoint)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid get endpoint response [svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				lo.FromPtr(endpoint).Id,
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("svix.endpoint.id", endpoint.Id),
		)

		return result, nil
	}

	return tracexv2.NewSpan(
		p.tracer,
		"notification.provider.svix.get_channel",
		trace.WithAttributes(
			attribute.String("svix.application.uid", input.Namespace),
			attribute.String("svix.endpoint.uid", input.ChannelID),
		),
	).Do2(ctx, fn)
}

func (p *Provider) CreateChannel(ctx context.Context, input notificationprovider.CreateChannelInput[v4.WebhookConfig]) (*notificationprovider.CreateChannelResult, error) {
	fn := func(ctx context.Context) (*notificationprovider.CreateChannelResult, error) {
		if err := input.Validate(); err != nil {
			return nil, fmt.Errorf(
				"invalid create endpoint input [svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		config := input.Config
		eventTypes := make([]string, 0, len(input.EventTypes))

		for _, eventType := range input.EventTypes {
			eventTypes = append(eventTypes, eventType.String())
		}

		headers := config.CustomHeaders
		if headers == nil {
			headers = map[string]string{}
		}

		endpoint, err := p.client.Endpoint().Create(ctx, input.Namespace, svix.EndpointIn{
			Uid:         &input.ChannelID,
			Description: &input.Name,
			Url:         config.URL,
			Disabled:    &input.Disabled,
			EventTypes:  eventTypes,
			Headers:     &headers,
			Secret:      lo.EmptyableToPtr(config.SigningSecret),
		}, &svix.EndpointCreateOptions{IdempotencyKey: input.IdempotencyKey})

		var svixErr *svix.Error

		if errors.As(err, &svixErr) && svixErr.Status() == http.StatusConflict {
			// UID uniqueness survives lost responses and the provider's idempotency
			// window. Finish reconciliation when an earlier create already succeeded.
			result, err := p.UpdateChannel(ctx, notificationprovider.UpdateChannelInput[v4.WebhookConfigUpdate]{
				Namespace: input.Namespace,
				ChannelID: input.ChannelID,
				Name:      &input.Name,
				Config: v4.WebhookConfigUpdate{
					URL:           &config.URL,
					CustomHeaders: &config.CustomHeaders,
					SigningSecret: lo.EmptyableToPtr(config.SigningSecret),
				},
				Disabled:     &input.Disabled,
				EventTypes:   &input.EventTypes,
				ProviderData: input.ProviderData,
			})
			if err != nil {
				return nil, err
			}

			trace.SpanFromContext(ctx).SetAttributes(
				attribute.String("svix.endpoint.id", result.ProviderData[providerDataEndpointID]),
			)

			return result, nil
		}

		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				err = normalizeError(err)
			}

			return nil, fmt.Errorf(
				"failed to create endpoint [svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		result, err := fromAPIEndpointOut(endpoint)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid create endpoint response [svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				lo.FromPtr(endpoint).Id,
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("svix.endpoint.id", endpoint.Id),
		)

		// Some Svix servers ignore headers in the create request. Apply them
		// explicitly before reporting provisioning complete, including on retries.
		if err := p.client.Endpoint().SetHeaders(ctx, input.Namespace, input.ChannelID, svix.EndpointHeadersIn{Headers: headers}); err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				err = normalizeError(err)
			}

			return nil, fmt.Errorf(
				"failed to set endpoint headers [svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				endpoint.Id,
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		return result, nil
	}

	return tracexv2.NewSpan(
		p.tracer,
		"notification.provider.svix.create_channel",
		trace.WithAttributes(
			attribute.String("svix.application.uid", input.Namespace),
			attribute.String("svix.endpoint.uid", input.ChannelID),
		),
	).Do2(ctx, fn)
}

func (p *Provider) UpdateChannel(ctx context.Context, input notificationprovider.UpdateChannelInput[v4.WebhookConfigUpdate]) (*notificationprovider.UpdateChannelResult, error) {
	fn := func(ctx context.Context) (*notificationprovider.UpdateChannelResult, error) {
		if err := input.Validate(); err != nil {
			return nil, fmt.Errorf(
				"invalid update endpoint input [svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		config := input.Config
		patch := svix.EndpointPatch{
			Description: input.Name,
			Url:         config.URL,
			Disabled:    input.Disabled,
		}

		if input.EventTypes != nil {
			eventTypes := make([]string, 0, len(*input.EventTypes))
			for _, eventType := range *input.EventTypes {
				eventTypes = append(eventTypes, eventType.String())
			}

			// Svix rejects [] on PATCH; explicit null selects catch-all. Legacy
			// Rule-ID routing is cleared only when routing is explicitly updated.
			patch.EventTypes = svixutils.NewNullableFromPtr[[]string](nil)
			if len(eventTypes) > 0 {
				patch.EventTypes = svixutils.NewNullable(eventTypes)
			}

			patch.Channels = svixutils.NewNullableFromPtr[[]string](nil)
		}

		appID := input.Namespace
		endpointID := input.ChannelID
		endpoints := p.client.Endpoint()

		var endpoint *svix.EndpointOut
		var err error

		// Header/secret-only and empty updates still require an existing endpoint.
		switch {
		case input.Name == nil && config.URL == nil && input.Disabled == nil && input.EventTypes == nil:
			endpoint, err = endpoints.Get(ctx, appID, endpointID)
		default:
			endpoint, err = endpoints.Patch(ctx, appID, endpointID, patch)
		}

		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				err = normalizeError(err)
			}

			return nil, fmt.Errorf(
				"failed to update endpoint [svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		result, err := fromAPIEndpointOut(endpoint)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid update endpoint response [svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				lo.FromPtr(endpoint).Id,
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		trace.SpanFromContext(ctx).SetAttributes(
			attribute.String("svix.endpoint.id", endpoint.Id),
		)

		if config.CustomHeaders != nil {
			headers := *config.CustomHeaders
			if headers == nil {
				headers = map[string]string{}
			}

			if err := endpoints.SetHeaders(ctx, appID, endpointID, svix.EndpointHeadersIn{Headers: headers}); err != nil {
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					err = normalizeError(err)
				}

				return nil, fmt.Errorf(
					"failed to set endpoint headers [svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
					endpoint.Id,
					input.ChannelID,
					input.Namespace,
					err,
				)
			}
		}

		// An omitted secret preserves the current key. Compare before rotation so
		// retries do not rotate an already-applied key or restart its grace period.
		if config.SigningSecret != nil {
			secret, err := endpoints.GetSecret(ctx, appID, endpointID)
			if err != nil {
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					err = normalizeError(err)
				}

				return nil, fmt.Errorf(
					"failed to get endpoint secret [svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
					endpoint.Id,
					input.ChannelID,
					input.Namespace,
					err,
				)
			}

			if secret == nil || secret.Key == "" {
				return nil, &notificationprovider.ProviderError{
					Kind: notificationprovider.ProviderErrorKindTerminal,
					Code: notificationprovider.ProviderErrorCodeRequestFailed,
					Err: fmt.Errorf(
						"invalid endpoint secret response [svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: missing key",
						endpoint.Id,
						input.ChannelID,
						input.Namespace,
					),
				}
			}

			if strings.TrimPrefix(secret.Key, webhooksecret.SigningSecretPrefix) != strings.TrimPrefix(*config.SigningSecret, webhooksecret.SigningSecretPrefix) {
				_, err := endpoints.RotateSecret(ctx, appID, endpointID, svix.EndpointSecretRotateIn{
					Key: config.SigningSecret,
				}, nil)
				if err != nil {
					if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
						err = normalizeError(err)
					}

					return nil, fmt.Errorf(
						"failed to rotate endpoint secret [svix.endpoint.id=%s svix.endpoint.uid=%s svix.application.uid=%s]: %w",
						endpoint.Id,
						input.ChannelID,
						input.Namespace,
						err,
					)
				}
			}
		}

		return result, nil
	}

	return tracexv2.NewSpan(
		p.tracer,
		"notification.provider.svix.update_channel",
		trace.WithAttributes(
			attribute.String("svix.application.uid", input.Namespace),
			attribute.String("svix.endpoint.uid", input.ChannelID),
		),
	).Do2(ctx, fn)
}

func (p *Provider) DeleteChannel(ctx context.Context, input notificationprovider.DeleteChannelInput) error {
	fn := func(ctx context.Context) error {
		if err := input.Validate(); err != nil {
			return fmt.Errorf(
				"invalid delete endpoint input [svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		err := p.client.Endpoint().Delete(ctx, input.Namespace, input.ChannelID)

		var svixErr *svix.Error

		if errors.As(err, &svixErr) && svixErr.Status() == http.StatusNotFound {
			return nil
		}

		if err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				err = normalizeError(err)
			}

			return fmt.Errorf(
				"failed to delete endpoint [svix.endpoint.uid=%s svix.application.uid=%s]: %w",
				input.ChannelID,
				input.Namespace,
				err,
			)
		}

		return nil
	}

	return tracexv2.NewSpan(
		p.tracer,
		"notification.provider.svix.delete_channel",
		trace.WithAttributes(
			attribute.String("svix.application.uid", input.Namespace),
			attribute.String("svix.endpoint.uid", input.ChannelID),
		),
	).Do(ctx, fn)
}

func fromAPIEndpointOut(endpoint *svix.EndpointOut) (*notificationprovider.ChannelOperationResult, error) {
	if endpoint == nil || endpoint.Id == "" || endpoint.Uid == nil || *endpoint.Uid == "" {
		return nil, &notificationprovider.ProviderError{
			Kind: notificationprovider.ProviderErrorKindTerminal,
			Code: notificationprovider.ProviderErrorCodeRequestFailed,
			Err:  errors.New("invalid endpoint response: missing ID or UID"),
		}
	}

	return &notificationprovider.ChannelOperationResult{
		State: notificationprovider.ChannelOperationStateInSync,
		ProviderData: map[string]string{
			providerDataEndpointID: endpoint.Id,
		},
	}, nil
}
