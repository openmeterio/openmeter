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

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func (h svixHandler) GetOrUpdateEndpointHeaders(ctx context.Context, appID, endpointID string, headers map[string]string) (map[string]string, error) {
	var resp map[string]string

	if len(headers) > 0 {
		input := svix.EndpointHeadersIn{
			Headers: headers,
		}

		err := h.client.Endpoint.UpdateHeaders(ctx, appID, endpointID, input)
		if err != nil {
			err = unwrapSvixError(err)

			return nil, fmt.Errorf("failed to set custom headers for Svix endpoint: %w", err)
		}

		resp = headers
	} else {
		out, err := h.client.Endpoint.GetHeaders(ctx, appID, endpointID)
		if err != nil || out == nil {
			err = unwrapSvixError(err)

			return nil, fmt.Errorf("failed to get custom headers for Svix endpoint: %w", err)
		}

		if len(out.Headers) > 0 {
			resp = out.Headers
		}
	}

	return resp, nil
}

func (h svixHandler) GetOrUpdateEndpointSecret(ctx context.Context, appID, endpointID string, secret *string) (string, error) {
	var resp string

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

		idempotencyKey, err := toIdempotencyKey(input, time.Now())
		if err != nil {
			return resp, fmt.Errorf("failed to generate idempotency key: %w", err)
		}

		err = h.client.Endpoint.RotateSecret(ctx, appID, endpointID, input, &svix.EndpointRotateSecretOptions{
			IdempotencyKey: &idempotencyKey,
		})
		if err != nil {
			err = unwrapSvixError(err)

			return resp, fmt.Errorf("failed to update Svix endpoint secret: %w", err)
		}

		resp = *secret
	}

	return resp, nil
}

func (h svixHandler) CreateWebhook(ctx context.Context, params webhook.CreateWebhookInput) (*webhook.Webhook, error) {
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

	if params.Secret == nil || *params.Secret == "" {
		var secret string

		secret, err = webhook.NewSigningSecretWithDefaultSize()
		if err != nil {
			return nil, fmt.Errorf("failed to generate signing secret: %w", err)
		}

		params.Secret = &secret
	}

	var endpointUID string
	if params.ID != nil {
		endpointUID = *params.ID
	} else {
		uid, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ULID for webhook: %w", err)
		}
		endpointUID = uid.String()
	}

	input := svix.EndpointIn{
		Uid: &endpointUID,
		Description: convert.SafeDeRef(params.Description, func(p string) *string {
			if p != "" {
				return &p
			}

			return nil
		}),
		Url:         params.URL,
		Disabled:    &params.Disabled,
		RateLimit:   params.RateLimit,
		Secret:      params.Secret,
		FilterTypes: params.EventTypes,
		Channels:    params.Channels,
		Metadata: func() *map[string]string {
			if len(params.Metadata) > 0 {
				return &params.Metadata
			}

			return nil
		}(),
	}

	idempotencyKey, err := toIdempotencyKey(input, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to generate idempotency key: %w", err)
	}

	endpoint, err := h.client.Endpoint.Create(ctx, app.Id, input, &svix.EndpointCreateOptions{
		IdempotencyKey: &idempotencyKey,
	})
	if err != nil {
		err = unwrapSvixError(err)

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

func (h svixHandler) UpdateWebhook(ctx context.Context, params webhook.UpdateWebhookInput) (*webhook.Webhook, error) {
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

	if len(params.EventTypes) == 0 && len(params.Channels) == 0 {
		params.Channels = []string{
			NullChannel,
		}
	}

	input := svix.EndpointUpdate{
		Uid: &params.ID,
		Description: convert.SafeDeRef(params.Description, func(p string) *string {
			if p != "" {
				return &p
			}

			return nil
		}),
		Url:         params.URL,
		Disabled:    &params.Disabled,
		RateLimit:   params.RateLimit,
		FilterTypes: params.EventTypes,
		Channels:    params.Channels,
		Metadata: func() *map[string]string {
			if len(params.Metadata) > 0 {
				return &params.Metadata
			}

			return nil
		}(),
	}

	endpoint, err := h.client.Endpoint.Update(ctx, app.Id, params.ID, input)
	if err != nil {
		err = unwrapSvixError(err)

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

func (h svixHandler) UpdateWebhookChannels(ctx context.Context, params webhook.UpdateWebhookChannelsInput) (*webhook.Webhook, error) {
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

func (h svixHandler) DeleteWebhook(ctx context.Context, params webhook.DeleteWebhookInput) error {
	if err := params.Validate(); err != nil {
		return fmt.Errorf("failed to validate DeleteWebhookInputs: %w", err)
	}

	err := h.client.Endpoint.Delete(ctx, params.Namespace, params.ID)
	if err != nil {
		err = unwrapSvixError(err)

		return fmt.Errorf("failed to delete Svix endpoint: %w", err)
	}

	return nil
}

func (h svixHandler) GetWebhook(ctx context.Context, params webhook.GetWebhookInput) (*webhook.Webhook, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate GetWebhookInputs: %w", err)
	}

	endpoint, err := h.client.Endpoint.Get(ctx, params.Namespace, params.ID)
	if err != nil {
		err = unwrapSvixError(err)

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

func (h svixHandler) ListWebhooks(ctx context.Context, params webhook.ListWebhooksInput) ([]webhook.Webhook, error) {
	listOut, err := h.client.Endpoint.List(ctx, params.Namespace, nil)
	if err != nil {
		err = unwrapSvixError(err)

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

	webhooks := make([]webhook.Webhook, 0, len(listOut.Data))
	for _, endpointOut := range listOut.Data {
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

		webhooks = append(webhooks, *wh)
	}

	return webhooks, nil
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
