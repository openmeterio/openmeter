package svix

import (
	"context"
	"crypto/rand"
	"fmt"
	"slices"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	svix "github.com/svix/svix-webhooks/go"
	svixmodels "github.com/svix/svix-webhooks/go/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	webhooksecret "github.com/openmeterio/openmeter/openmeter/notification/webhook/secret"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook/svix/internal"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/idempotency"
)

func (h svixHandler) GetOrUpdateEndpointHeaders(ctx context.Context, appID, endpointID string, headers map[string]string) (map[string]string, error) {
	fn := func(ctx context.Context) (map[string]string, error) {
		if appID == "" {
			return nil, fmt.Errorf("appID is required")
		}

		if endpointID == "" {
			return nil, fmt.Errorf("endpointID is required")
		}

		var resp map[string]string

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("svix.app_id", appID),
			attribute.String("svix.endpoint_id", endpointID),
		}

		if len(headers) > 0 {
			span.AddEvent("updating endpoint headers", trace.WithAttributes(spanAttrs...))

			input := svix.EndpointHeadersIn{
				Headers: headers,
			}

			err := h.client.Endpoint.UpdateHeaders(ctx, appID, endpointID, input)
			if err = internal.WrapSvixError(err); err != nil {
				return nil, fmt.Errorf("failed to set custom headers for Svix endpoint: %w", err)
			}

			resp = headers
		} else {
			span.AddEvent("fetching endpoint headers", trace.WithAttributes(spanAttrs...))

			out, err := h.client.Endpoint.GetHeaders(ctx, appID, endpointID)
			if err = internal.WrapSvixError(err); err != nil || out == nil {
				return nil, fmt.Errorf("failed to get custom headers for Svix endpoint: %w", err)
			}

			if len(out.Headers) > 0 {
				resp = out.Headers
			}
		}

		return resp, nil
	}

	return tracex.Start[map[string]string](ctx, h.tracer, "svix.get_or_update_endpoint_headers").Wrap(fn)
}

func (h svixHandler) GetOrUpdateEndpointSecret(ctx context.Context, appID, endpointID string, secret *string) (string, error) {
	fn := func(ctx context.Context) (string, error) {
		var resp string

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String("svix.app_id", appID),
			attribute.String("svix.endpoint_id", endpointID),
		}

		span.AddEvent("getting endpoint secret", trace.WithAttributes(spanAttrs...))

		secretOut, err := h.client.Endpoint.GetSecret(ctx, appID, endpointID)
		if err != nil {
			return resp, fmt.Errorf("failed to get Svix endpoint secret: %w", err)
		}
		if secretOut == nil {
			return resp, fmt.Errorf("failed to get Svix endpoint secret: %w", err)
		}

		resp = secretOut.Key

		if secret != nil && *secret != secretOut.Key {
			input := svix.EndpointSecretRotateIn{
				Key: secret,
			}

			idempotencyKey, err := idempotency.Key()
			if err != nil {
				return resp, fmt.Errorf("failed to generate idempotency key: %w", err)
			}

			span.AddEvent("rotating endpoint secret", trace.WithAttributes(spanAttrs...))

			err = h.client.Endpoint.RotateSecret(ctx, appID, endpointID, input, &svix.EndpointRotateSecretOptions{
				IdempotencyKey: &idempotencyKey,
			})
			if err = internal.WrapSvixError(err); err != nil {
				return resp, fmt.Errorf("failed to update Svix endpoint secret: %w", err)
			}

			resp = *secret
		}

		return resp, nil
	}

	return tracex.Start[string](ctx, h.tracer, "svix.get_or_update_endpoint_secret").Wrap(fn)
}

func (h svixHandler) CreateWebhook(ctx context.Context, params webhook.CreateWebhookInput) (*webhook.Webhook, error) {
	fn := func(ctx context.Context) (*webhook.Webhook, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate CreateWebhookInput: %w", err)
		}
		// Ensure that application is created for namespace
		app, err := h.CreateApplication(ctx, params.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to ensure Svix application is created for namespace: %w", err)
		}

		// Create webhook endpoint in application.
		//
		// Channels and EventTypes are used as message filters in Svix,
		// which means that a webhook without any filtering will receive all messages
		// sent to the application the webhook belongs to. To prevent this from happening, we
		// use the NullFilterChannel as a dummy filter, so it is possible to set up a webhook endpoint before
		// knowing what type of messages are going to be routed to it.

		if len(params.EventTypes) == 0 && len(params.Channels) == 0 {
			params.Channels = []string{
				NullChannel,
			}
		}

		if lo.FromPtr(params.Secret) == "" {
			var secret string

			secret, err = webhooksecret.NewSigningSecretWithDefaultSize()
			if err != nil {
				return nil, fmt.Errorf("failed to generate signing secret: %w", err)
			}

			params.Secret = &secret
		}

		endpointUID := lo.FromPtr(params.ID)
		if endpointUID == "" {
			uid, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
			if err != nil {
				return nil, fmt.Errorf("failed to generate ULID for webhook: %w", err)
			}
			endpointUID = uid.String()
		}

		input := svix.EndpointIn{
			Uid:         &endpointUID,
			Description: lo.EmptyableToPtr(lo.FromPtr(params.Description)),
			Url:         params.URL,
			Disabled:    &params.Disabled,
			RateLimit:   params.RateLimit,
			Secret:      params.Secret,
			FilterTypes: params.EventTypes,
			Channels:    params.Channels,
			Metadata:    lo.EmptyableToPtr(params.Metadata),
		}

		idempotencyKey, err := idempotency.Key()
		if err != nil {
			return nil, fmt.Errorf("failed to generate idempotency key: %w", err)
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String(AnnotationApplicationUID, params.Namespace),
			attribute.String(AnnotationEndpointUID, endpointUID),
			attribute.String(AnnotationEndpointURL, input.Url),
			attribute.String("idempotency_key", idempotencyKey),
		}

		span.AddEvent("creating endpoint", trace.WithAttributes(spanAttrs...))

		endpoint, err := h.client.Endpoint.Create(ctx, app.Id, input, &svix.EndpointCreateOptions{
			IdempotencyKey: &idempotencyKey,
		})
		if err = internal.WrapSvixError(err); err != nil {
			return nil, fmt.Errorf("failed to create Svix endpoint: %w", err)
		}

		wh := WebhookFromSvixEndpointOut(endpoint)
		wh.Namespace = params.Namespace

		// Get signing secret for webhook endpoint.
		// Skip fetching the secret if it was provided in the request.

		wh.Secret, err = h.GetOrUpdateEndpointSecret(ctx, app.Id, endpoint.Id, nil)
		if err != nil {
			return nil, err
		}

		// Set custom HTTP headers for webhook endpoint if provided

		if len(params.CustomHeaders) > 0 {
			wh.CustomHeaders, err = h.GetOrUpdateEndpointHeaders(ctx, app.Id, endpoint.Id, params.CustomHeaders)
			if err != nil {
				return nil, err
			}
		}

		return wh, nil
	}

	return tracex.Start[*webhook.Webhook](ctx, h.tracer, "svix.create_webhook").Wrap(fn)
}

func (h svixHandler) UpdateWebhook(ctx context.Context, params webhook.UpdateWebhookInput) (*webhook.Webhook, error) {
	fn := func(ctx context.Context) (*webhook.Webhook, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate UpdateWebhookInputs: %w", err)
		}

		// Ensure that an application is created for namespace

		app, err := h.CreateApplication(ctx, params.Namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to ensure Svix application is created for namespace: %w", err)
		}

		// Update webhook endpoint in application
		//
		// Channels and EventTypes are used as message filters in Svix,
		// which means that a webhook without any filtering will receive all messages
		// sent to the application the webhook belongs to. To prevent this from happening, we
		// use the NullFilterChannel as a dummy filter, so it is possible to set up a webhook endpoint before
		// knowing what type of messages are going to be routed to it.

		if len(params.Channels) == 0 {
			params.Channels = []string{
				NullChannel,
			}
		}

		input := svix.EndpointUpdate{
			Uid:         &params.ID,
			Description: lo.EmptyableToPtr(lo.FromPtr(params.Description)),
			Url:         params.URL,
			Disabled:    &params.Disabled,
			RateLimit:   params.RateLimit,
			FilterTypes: params.EventTypes,
			Channels:    params.Channels,
			Metadata:    lo.EmptyableToPtr(params.Metadata),
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String(AnnotationApplicationUID, params.Namespace),
			attribute.String(AnnotationEndpointUID, params.ID),
			attribute.String(AnnotationEndpointURL, params.URL),
		}

		span.AddEvent("updating endpoint", trace.WithAttributes(spanAttrs...))

		endpoint, err := h.client.Endpoint.Update(ctx, app.Id, params.ID, input)
		if err = internal.WrapSvixError(err); err != nil {
			return nil, fmt.Errorf("failed to update Svix endpoint: %w", err)
		}

		wh := WebhookFromSvixEndpointOut(endpoint)
		wh.Namespace = params.Namespace

		// Update signing secret for webhook endpoint if provided

		// Get signing secret for webhook endpoint.
		// Skip fetching the secret if it was provided in the request.

		wh.Secret, err = h.GetOrUpdateEndpointSecret(ctx, app.Id, endpoint.Id, params.Secret)
		if err != nil {
			return nil, err
		}

		// Set custom HTTP headers for webhook endpoint if provided

		if len(params.CustomHeaders) > 0 {
			wh.CustomHeaders, err = h.GetOrUpdateEndpointHeaders(ctx, app.Id, endpoint.Id, params.CustomHeaders)
			if err != nil {
				return nil, err
			}
		}

		return wh, nil
	}

	return tracex.Start[*webhook.Webhook](ctx, h.tracer, "svix.update_webhook").Wrap(fn)
}

func (h svixHandler) UpdateWebhookChannels(ctx context.Context, params webhook.UpdateWebhookChannelsInput) (*webhook.Webhook, error) {
	fn := func(ctx context.Context) (*webhook.Webhook, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate UpdateWebhookChannelsInput: %w", err)
		}

		wh, err := h.GetWebhook(ctx, webhook.GetWebhookInput{
			Namespace: params.Namespace,
			ID:        params.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get webhook: %w", err)
		}

		channels := func(channels, add, remove []string) []string {
			channelKeys := make(map[string]struct{})

			for _, channel := range channels {
				channelKeys[channel] = struct{}{}
			}

			for _, channel := range add {
				channelKeys[channel] = struct{}{}
			}

			for _, channel := range remove {
				delete(channelKeys, channel)
			}

			result := make([]string, 0, len(channels))
			for channel := range channelKeys {
				result = append(result, channel)
			}

			return result
		}(wh.Channels, params.AddChannels, params.RemoveChannels)

		if len(channels) == 0 {
			channels = []string{
				NullChannel,
			}
		}

		if len(channels) > webhook.MaxChannelsPerWebhook {
			return nil, webhook.NewValidationError(webhook.ErrMaxChannelsPerWebhookExceeded)
		}

		wh, err = h.UpdateWebhook(ctx, webhook.UpdateWebhookInput{
			Namespace:     wh.Namespace,
			ID:            wh.ID,
			URL:           wh.URL,
			CustomHeaders: wh.CustomHeaders,
			Disabled:      wh.Disabled,
			Secret:        &wh.Secret,
			RateLimit:     wh.RateLimit,
			Description:   &wh.Description,
			EventTypes:    wh.EventTypes,
			Channels:      channels,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update webhook channels: %w", err)
		}

		return wh, nil
	}

	return tracex.Start[*webhook.Webhook](ctx, h.tracer, "svix.update_webhook_channels").Wrap(fn)
}

func (h svixHandler) DeleteWebhook(ctx context.Context, params webhook.DeleteWebhookInput) error {
	fn := func(ctx context.Context) error {
		if err := params.Validate(); err != nil {
			return fmt.Errorf("failed to validate DeleteWebhookInputs: %w", err)
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String(AnnotationApplicationUID, params.Namespace),
			attribute.String(AnnotationEndpointUID, params.ID),
		}

		span.AddEvent("deleting endpoint", trace.WithAttributes(spanAttrs...))

		err := h.client.Endpoint.Delete(ctx, params.Namespace, params.ID)
		if err = internal.WrapSvixError(err); err != nil {
			if webhook.IsNotFoundError(err) {
				return nil
			}

			return fmt.Errorf("failed to delete Svix endpoint: %w", err)
		}

		return nil
	}

	return tracex.StartWithNoValue(ctx, h.tracer, "svix.delete_webhook").Wrap(fn)
}

func (h svixHandler) GetWebhook(ctx context.Context, params webhook.GetWebhookInput) (*webhook.Webhook, error) {
	fn := func(ctx context.Context) (*webhook.Webhook, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate GetWebhookInputs: %w", err)
		}

		span := trace.SpanFromContext(ctx)

		spanAttrs := []attribute.KeyValue{
			attribute.String(AnnotationApplicationUID, params.Namespace),
			attribute.String(AnnotationEndpointUID, params.ID),
		}

		span.AddEvent("fetching endpoint", trace.WithAttributes(spanAttrs...))

		endpoint, err := h.client.Endpoint.Get(ctx, params.Namespace, params.ID)
		if err = internal.WrapSvixError(err); err != nil {
			return nil, fmt.Errorf("failed to get Svix endpoint: %w", err)
		}

		wh := WebhookFromSvixEndpointOut(endpoint)
		wh.Namespace = params.Namespace

		// Get signing secret for webhook endpoint.

		wh.Secret, err = h.GetOrUpdateEndpointSecret(ctx, params.Namespace, endpoint.Id, nil)
		if err != nil {
			return nil, err
		}

		// Get custom HTTP headers for webhook endpoint if provided

		wh.CustomHeaders, err = h.GetOrUpdateEndpointHeaders(ctx, params.Namespace, endpoint.Id, nil)
		if err != nil {
			return nil, err
		}

		return wh, nil
	}

	return tracex.Start[*webhook.Webhook](ctx, h.tracer, "svix.get_webhook").Wrap(fn)
}

func (h svixHandler) ListWebhooks(ctx context.Context, params webhook.ListWebhooksInput) ([]webhook.Webhook, error) {
	fn := func(ctx context.Context) ([]webhook.Webhook, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate ListWebhooksInputs: %w", err)
		}

		var (
			result []webhook.Webhook
			stop   bool
		)

		opts := &svix.EndpointListOptions{
			Limit:    lo.ToPtr[uint64](100),
			Iterator: nil,
			Order:    lo.ToPtr(svixmodels.ORDERING_ASCENDING),
		}

		span := trace.SpanFromContext(ctx)

		for !stop {
			spanAttrs := []attribute.KeyValue{
				attribute.String(AnnotationApplicationUID, params.Namespace),
				attribute.String("iterator", lo.FromPtr(opts.Iterator)),
			}

			span.AddEvent("fetching endpoints in batch", trace.WithAttributes(spanAttrs...))

			out, err := h.client.Endpoint.List(ctx, params.Namespace, opts)
			if err = internal.WrapSvixError(err); err != nil {
				return nil, fmt.Errorf("failed to list Svix endpoints: %w", err)
			}

			filterFn := func(o svix.EndpointOut) bool {
				if len(params.IDs) == 0 && len(params.EventTypes) == 0 && len(params.Channels) == 0 {
					return true
				}

				if slices.Contains(params.IDs, o.Id) {
					return true
				}

				if o.Uid != nil && slices.Contains(params.IDs, *o.Uid) {
					return true
				}

				if o.FilterTypes != nil {
					for _, eventType := range params.EventTypes {
						if slices.Contains(o.FilterTypes, eventType) {
							return true
						}
					}
				}

				if o.Channels != nil {
					for _, channel := range params.Channels {
						if slices.Contains(o.Channels, channel) {
							return true
						}
					}
				}

				return false
			}

			for _, endpointOut := range out.Data {
				if !filterFn(endpointOut) {
					continue
				}

				wh := WebhookFromSvixEndpointOut(&endpointOut)
				wh.Namespace = params.Namespace

				// Get signing secret for webhook endpoint.
				// Skip fetching the secret if it was provided in the request.

				wh.Secret, err = h.GetOrUpdateEndpointSecret(ctx, params.Namespace, wh.ID, nil)
				if err != nil {
					return nil, err
				}

				wh.CustomHeaders, err = h.GetOrUpdateEndpointHeaders(ctx, params.Namespace, wh.ID, nil)
				if err != nil {
					return nil, err
				}

				result = append(result, *wh)
			}

			if out.Done {
				stop = true
			}

			opts.Iterator = out.Iterator
		}

		return result, nil
	}

	return tracex.Start[[]webhook.Webhook](ctx, h.tracer, "svix.list_webhooks").Wrap(fn)
}

func WebhookFromSvixEndpointOut(e *svix.EndpointOut) *webhook.Webhook {
	return &webhook.Webhook{
		ID:          lo.FromPtr(e.Uid),
		URL:         e.Url,
		Disabled:    lo.FromPtrOr(e.Disabled, false),
		RateLimit:   e.RateLimit,
		Description: e.Description,
		EventTypes:  e.FilterTypes,
		Channels: lo.Filter(e.Channels, func(s string, _ int) bool {
			return s != NullChannel
		}),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
