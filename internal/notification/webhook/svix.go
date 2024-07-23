package webhook

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/oklog/ulid/v2"
	svix "github.com/svix/svix-webhooks/go"
	"k8s.io/utils/strings/slices"

	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
)

const (
	// NullChannel is an internal channel type which should receive no messages at any time.
	// Channels and EventTypes are used as message filters in Svix
	// which means that a webhook without any filtering will receive all messages
	// sent to the application the webhook belongs to. In order to prevent this we
	// use the NoMessageChannel as a dummy filter, so it is possible to set up webhook endpoint
	// prior knowing what type of messages are going to be routed to it.
	NullChannel = "__null_channel"
)

type svixConfig struct {
	ServerURL string
	AuthToken string
	Debug     bool

	RegisterEvenTypes []EventType
}

var _ Handler = (*svixWebhookHandler)(nil)

type svixWebhookHandler struct {
	client *svix.Svix
}

func newSvixWebhookHandler(config svixConfig) (Handler, error) {
	opts := svix.SvixOptions{
		Debug: config.Debug,
	}

	var err error
	if config.ServerURL != "" {
		opts.ServerUrl, err = url.Parse(config.ServerURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse server URL: %w", err)
		}
	}

	handler := &svixWebhookHandler{
		client: svix.New(config.AuthToken, &opts),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = handler.RegisterEventTypes(ctx, RegisterEventTypesInputs{
		EvenTypes: config.RegisterEvenTypes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register event types: %w", err)
	}

	return handler, nil
}

func (h svixWebhookHandler) RegisterEventTypes(ctx context.Context, params RegisterEventTypesInputs) error {
	for _, evenType := range params.EvenTypes {
		input := &svix.EventTypeUpdate{
			Description: evenType.Description,
			FeatureFlag: *svix.NullableString(nil),
			GroupName:   *svix.NullableString(&evenType.GroupName),
			Schemas:     evenType.Schemas,
		}

		_, err := h.client.EventType.Update(ctx, evenType.Name, input)
		if err != nil {
			return fmt.Errorf("failed to create event type: %w", err)
		}
	}

	return nil
}

func (h svixWebhookHandler) CreateApplication(ctx context.Context, id string) (*svix.ApplicationOut, error) {
	input := &svix.ApplicationIn{
		Name: id,
		Uid:  *svix.NullableString(&id),
	}

	idempotencyKey, err := toIdempotencyKey(input, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to generate idempotency key: %w", err)
	}

	app, err := h.client.Application.GetOrCreateWithOptions(ctx, input, &svix.PostOptions{
		IdempotencyKey: &idempotencyKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get or create Svix application: %w", err)
	}

	return app, nil
}

func (h svixWebhookHandler) GetOrUpdateEndpointHeaders(ctx context.Context, appID, endpointID string, headers map[string]string) (map[string]string, error) {
	var resp map[string]string

	if len(headers) > 0 {
		input := &svix.EndpointHeadersIn{
			Headers: headers,
		}

		err := h.client.Endpoint.UpdateHeaders(ctx, appID, endpointID, input)
		if err != nil {
			return nil, fmt.Errorf("failed to set custom headers for Svix endpoint: %w", err)
		}

		resp = headers
	} else {
		out, err := h.client.Endpoint.GetHeaders(ctx, appID, endpointID)
		if err != nil || out == nil {
			return nil, fmt.Errorf("failed to get custom headers for Svix endpoint: %w", err)
		}

		if len(out.Headers) > 0 {
			resp = out.Headers
		}
	}

	return resp, nil
}

func (h svixWebhookHandler) GetOrUpdateEndpointSecret(ctx context.Context, appID, endpointID string, secret *string) (string, error) {
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
		input := &svix.EndpointSecretRotateIn{
			Key: *svix.NullableString(secret),
		}

		idempotencyKey, err := toIdempotencyKey(input, time.Now())
		if err != nil {
			return resp, fmt.Errorf("failed to generate idempotency key: %w", err)
		}

		err = h.client.Endpoint.RotateSecretWithOptions(ctx, appID, endpointID, input, &svix.PostOptions{
			IdempotencyKey: &idempotencyKey,
		})
		if err != nil {
			return resp, fmt.Errorf("failed to update Svix endpoint secret: %w", err)
		}

		resp = *secret
	}

	return resp, nil
}

func (h svixWebhookHandler) CreateWebhook(ctx context.Context, params CreateWebhookInputs) (*Webhook, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate UpdateWebhookInputs: %w", err)
	}

	// Ensure that application is created for namespace

	app, err := h.CreateApplication(ctx, params.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create Svix webhook: %w", err)
	}

	// Create webhook endpoint in application.
	//
	// Channels and EventTypes are used as message filters in Svix
	// which means that a webhook without any filtering will receive all messages
	// sent to the application the webhook belongs to. In order to prevent this we
	// use the NullFilterChannel as a dummy filter, so it is possible to set up webhook endpoint
	// prior knowing what type of messages are going to be routed to it.

	if len(params.EventTypes) == 0 && len(params.Channels) == 0 {
		params.Channels = []string{
			NullChannel,
		}
	}

	if params.Secret == nil || *params.Secret == "" {
		var secret string

		secret, err = NewSigningSecretWithDefaultSize()
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

	input := &svix.EndpointIn{
		Uid: *svix.NullableString(&endpointUID),
		Description: convert.SafeDeRef(params.Description, func(p string) *string {
			if p != "" {
				return &p
			}

			return nil
		}),
		Url:         params.URL,
		Disabled:    &params.Disabled,
		RateLimit:   *svix.NullableInt32(params.RateLimit),
		Secret:      *svix.NullableString(params.Secret),
		FilterTypes: params.EventTypes,
		Channels:    params.Channels,
	}

	idempotencyKey, err := toIdempotencyKey(input, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to generate idempotency key: %w", err)
	}

	endpoint, err := h.client.Endpoint.CreateWithOptions(ctx, app.Id, input, &svix.PostOptions{
		IdempotencyKey: &idempotencyKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Svix endpoint: %w", err)
	}

	webhook := WebhookFromSvixEndpointOut(endpoint)
	webhook.Namespace = params.Namespace

	// Get signing secret for webhook endpoint.
	// Skip fetching secret if it was provided in the request.

	webhook.Secret, err = h.GetOrUpdateEndpointSecret(ctx, app.Id, endpoint.Id, nil)
	if err != nil {
		return nil, err
	}

	// Set custom HTTP headers for webhook endpoint if provided

	if len(params.CustomHeaders) > 0 {
		webhook.CustomHeaders, err = h.GetOrUpdateEndpointHeaders(ctx, app.Id, endpoint.Id, nil)
		if err != nil {
			return nil, err
		}
	}

	return webhook, nil
}

func (h svixWebhookHandler) UpdateWebhook(ctx context.Context, params UpdateWebhookInputs) (*Webhook, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate UpdateWebhookInputs: %w", err)
	}

	// Ensure that application is created for namespace

	app, err := h.CreateApplication(ctx, params.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create Svix webhook: %w", err)
	}

	// Update webhook endpoint in application
	//
	// Channels and EventTypes are used as message filters in Svix
	// which means that a webhook without any filtering will receive all messages
	// sent to the application the webhook belongs to. In order to prevent this we
	// use the NoMessageChannel as a dummy filter, so it is possible to set up webhook endpoint
	// prior knowing what type of messages are going to be routed to it.

	if len(params.EventTypes) == 0 && len(params.Channels) == 0 {
		params.Channels = []string{
			NullChannel,
		}
	}

	input := &svix.EndpointUpdate{
		Uid: *svix.NullableString(&params.ID),
		Description: convert.SafeDeRef(params.Description, func(p string) *string {
			if p != "" {
				return &p
			}

			return nil
		}),
		Url:         params.URL,
		Disabled:    &params.Disabled,
		RateLimit:   *svix.NullableInt32(params.RateLimit),
		FilterTypes: params.EventTypes,
		Channels:    params.Channels,
	}

	endpoint, err := h.client.Endpoint.Update(ctx, app.Id, params.ID, input)
	if err != nil {
		return nil, fmt.Errorf("failed to update Svix endpoint: %w", err)
	}

	webhook := WebhookFromSvixEndpointOut(endpoint)
	webhook.Namespace = params.Namespace

	// Update signing secret for webhook endpoint if provided

	// Get signing secret for webhook endpoint.
	// Skip fetching secret if it was provided in the request.

	webhook.Secret, err = h.GetOrUpdateEndpointSecret(ctx, app.Id, endpoint.Id, params.Secret)
	if err != nil {
		return nil, err
	}

	// Set custom HTTP headers for webhook endpoint if provided

	if len(params.CustomHeaders) > 0 {
		webhook.CustomHeaders, err = h.GetOrUpdateEndpointHeaders(ctx, app.Id, endpoint.Id, params.CustomHeaders)
		if err != nil {
			return nil, err
		}
	}

	return webhook, nil
}

func (h svixWebhookHandler) UpdateWebhookChannels(ctx context.Context, params UpdateWebhookChannelsInputs) (*Webhook, error) {
	webhook, err := h.GetWebhook(ctx, GetWebhookInputs{
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
	}(webhook.Channels, params.AddChannels, params.RemoveChannels)

	if len(channels) == 0 {
		channels = []string{
			NullChannel,
		}
	}

	webhook, err = h.UpdateWebhook(ctx, UpdateWebhookInputs{
		Namespace:     webhook.Namespace,
		ID:            webhook.ID,
		URL:           webhook.URL,
		CustomHeaders: webhook.CustomHeaders,
		Disabled:      webhook.Disabled,
		Secret:        &webhook.Secret,
		RateLimit:     webhook.RateLimit,
		Description:   &webhook.Description,
		EventTypes:    webhook.EventTypes,
		Channels:      channels,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update webhook channels: %w", err)
	}

	return webhook, nil
}

func (h svixWebhookHandler) DeleteWebhook(ctx context.Context, params DeleteWebhookInputs) error {
	if err := params.Validate(); err != nil {
		return fmt.Errorf("failed to validate DeleteWebhookInputs: %w", err)
	}

	err := h.client.Endpoint.Delete(ctx, params.Namespace, params.ID)
	if err != nil {
		return fmt.Errorf("failed to delete Svix endpoint: %w", err)
	}

	return nil
}

func (h svixWebhookHandler) GetWebhook(ctx context.Context, params GetWebhookInputs) (*Webhook, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate GetWebhookInputs: %w", err)
	}

	endpoint, err := h.client.Endpoint.Get(ctx, params.Namespace, params.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Svix endpoint: %w", err)
	}

	webhook := WebhookFromSvixEndpointOut(endpoint)
	webhook.Namespace = params.Namespace

	// Get signing secret for webhook endpoint.

	webhook.Secret, err = h.GetOrUpdateEndpointSecret(ctx, params.Namespace, endpoint.Id, nil)
	if err != nil {
		return nil, err
	}

	// Get custom HTTP headers for webhook endpoint if provided

	webhook.CustomHeaders, err = h.GetOrUpdateEndpointHeaders(ctx, params.Namespace, endpoint.Id, nil)
	if err != nil {
		return nil, err
	}

	return webhook, nil
}

func (h svixWebhookHandler) ListWebhooks(ctx context.Context, params ListWebhooksInputs) ([]Webhook, error) {
	listOut, err := h.client.Endpoint.List(ctx, params.Namespace, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list Svix endpoints: %w", err)
	}

	filterFn := func(o svix.EndpointOut) bool {
		if slices.Contains(params.IDs, o.Id) {
			return true
		}

		if o.Uid.IsSet() && slices.Contains(params.IDs, *o.Uid.Get()) {
			return true
		}

		for _, eventType := range params.EventTypes {
			if slices.Contains(o.FilterTypes, eventType) {
				return true
			}
		}

		for _, channel := range params.Channels {
			if slices.Contains(o.Channels, channel) {
				return true
			}
		}

		return false
	}

	webhooks := make([]Webhook, 0, len(listOut.Data))
	for _, endpointOut := range listOut.Data {
		if !filterFn(endpointOut) {
			continue
		}

		webhook := WebhookFromSvixEndpointOut(&endpointOut)
		webhook.Namespace = params.Namespace

		// Get signing secret for webhook endpoint.
		// Skip fetching secret if it was provided in the request.

		webhook.Secret, err = h.GetOrUpdateEndpointSecret(ctx, params.Namespace, webhook.ID, nil)
		if err != nil {
			return nil, err
		}

		webhook.CustomHeaders, err = h.GetOrUpdateEndpointHeaders(ctx, params.Namespace, webhook.ID, nil)
		if err != nil {
			return nil, err
		}
	}

	return webhooks, nil
}

func (h svixWebhookHandler) SendMessage(ctx context.Context, params SendMessageInputs) (*Message, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate SendMessageInputs: %w", err)
	}

	var eventID *string
	if params.EventID != "" {
		eventID = &params.EventID
	}

	input := &svix.MessageIn{
		Channels:  params.Channels,
		EventId:   *svix.NullableString(eventID),
		EventType: params.EventType,
		Payload:   params.Payload,
	}

	idempotencyKey, err := toIdempotencyKey(input, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to generate idempotency key: %w", err)
	}

	o, err := h.client.Message.CreateWithOptions(ctx, params.Namespace, input, &svix.PostOptions{
		IdempotencyKey: &idempotencyKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to delete Svix endpoint: %w", err)
	}

	return &Message{
		Namespace: params.Namespace,
		ID:        o.Id,
		EventID: func() string {
			if o.EventId.IsSet() {
				return *o.EventId.Get()
			}

			return ""
		}(),
		EventType: o.EventType,
		Channels:  o.Channels,
		Payload:   o.Payload,
	}, nil
}

func WebhookFromSvixEndpointOut(e *svix.EndpointOut) *Webhook {
	return &Webhook{
		ID: func() string {
			if e.Uid.IsSet() {
				return *e.Uid.Get()
			}

			return e.Id
		}(),
		URL:         e.Url,
		Disabled:    defaultx.WithDefault(e.Disabled, false),
		RateLimit:   e.RateLimit.Get(),
		Description: e.Description,
		EventTypes:  e.FilterTypes,
		Channels: slices.Filter(nil, e.Channels, func(s string) bool {
			return s != NullChannel
		}),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

type idempotencyKeyTypes interface {
	*svix.ApplicationIn | *svix.EndpointIn | *svix.EndpointSecretRotateIn | *svix.MessageIn
}

func toIdempotencyKey[T idempotencyKeyTypes](v T, t time.Time) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	if t.IsZero() {
		t = time.Now().UTC()
	}
	t = t.UTC()

	h := sha256.New()
	h.Write(b)
	h.Write([]byte(t.String()))

	return hex.EncodeToString(h.Sum(nil)), nil
}
