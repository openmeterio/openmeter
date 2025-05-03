package svix

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/samber/lo"
	svix "github.com/svix/svix-webhooks/go"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
)

const (
	// NullChannel is an internal channel type which should receive no messages at any time.
	// Channels and EventTypes are used as message filters in Svix
	// which means that a webhook without any filtering will receive all messages
	// sent to the application the webhook belongs to. In order to prevent this we
	// use the NullChannel as a dummy filter, so it is possible to set up webhook endpoint
	// prior knowing what type of messages are going to be routed to it.
	NullChannel = "__null_channel"
)

type SvixConfig struct {
	// Svix server config
	APIKey    string
	ServerURL string
	Debug     bool
}

func (c SvixConfig) Validate() error {
	var errs []error

	if c.ServerURL != "" {
		if _, err := url.Parse(c.ServerURL); err != nil {
			errs = append(errs, fmt.Errorf("invalid server URL: %w", err))
		}

		if c.APIKey == "" {
			errs = append(errs, errors.New("API key is required"))
		}
	}

	return errors.Join(errs...)
}

func (c SvixConfig) IsEnabled() bool {
	return c.ServerURL != "" || c.APIKey != ""
}

type Config struct {
	SvixConfig

	RegisterEventTypes      []webhook.EventType
	RegistrationTimeout     time.Duration
	SkipRegistrationOnError bool

	Logger *slog.Logger
}

func (c Config) Validate() error {
	var errs []error

	if err := c.SvixConfig.Validate(); err != nil {
		errs = append(errs, err)
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (webhook.Handler, error) {
	var errs []error

	if config.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if err := config.SvixConfig.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	if config.RegisterEventTypes == nil {
		config.RegisterEventTypes = webhook.NotificationEventTypes
	}

	if config.RegistrationTimeout == 0 {
		config.RegistrationTimeout = webhook.DefaultRegistrationTimeout
	}

	handler, err := NewHandler(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Svix webhook handler: %w", err)
	}

	if len(config.RegisterEventTypes) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), config.RegistrationTimeout)
		defer cancel()

		err = handler.RegisterEventTypes(ctx, webhook.RegisterEventTypesInputs{
			EventTypes: config.RegisterEventTypes,
		})
		if err != nil {
			if config.SkipRegistrationOnError {
				config.Logger.WarnContext(ctx, "failed to register event types", "error", err)
			} else {
				return nil, fmt.Errorf("failed to register event types: %w", err)
			}
		}
	}

	return handler, nil
}

var _ webhook.Handler = (*svixHandler)(nil)

type svixHandler struct {
	client *svix.Svix
}

func NewHandler(config Config) (webhook.Handler, error) {
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

	client, err := svix.New(config.APIKey, &opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create svix client: %w", err)
	}

	return &svixHandler{
		client: client,
	}, nil
}

func (h svixHandler) CreateApplication(ctx context.Context, id string) (*svix.ApplicationOut, error) {
	input := svix.ApplicationIn{
		Name: id,
		Uid:  &id,
	}

	idempotencyKey, err := toIdempotencyKey(input, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to generate idempotency key: %w", err)
	}

	app, err := h.client.Application.GetOrCreate(ctx, input, &svix.ApplicationCreateOptions{
		IdempotencyKey: &idempotencyKey,
	})
	if err != nil {
		err = unwrapSvixError(err)

		return nil, fmt.Errorf("failed to get or create Svix application: %w", err)
	}

	return app, nil
}

type idempotencyKeyTypes interface {
	svix.ApplicationIn | svix.EndpointIn | svix.EndpointSecretRotateIn | svix.MessageIn
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

type svixErrorBody struct {
	Detail []struct {
		Message string `json:"msg"`
	} `json:"detail"`
}

func unwrapSvixError(err error) error {
	if err == nil {
		return nil
	}

	svixErr, ok := lo.ErrorsAs[*svix.Error](err)
	if !ok {
		return err
	}

	body := svixErrorBody{}

	// ignore error
	_ = json.Unmarshal(svixErr.Body(), &body)

	if len(body.Detail) > 0 {
		var errs []error

		for _, detail := range body.Detail {
			errs = append(errs, errors.New(detail.Message))
		}

		err = errors.Join(errs...)
	}

	if svixErr.Status() >= 400 && svixErr.Status() < 500 {
		err = webhook.ValidationError{
			Err: err,
		}
	}

	return err
}
