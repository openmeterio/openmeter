package svix

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	svix "github.com/svix/svix-webhooks/go"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook/svix/internal"
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

type Error = internal.SvixError

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
	RegisterEventTypes      []webhook.EventType
	RegistrationTimeout     time.Duration
	SkipRegistrationOnError bool

	SvixAPIClient *svix.Svix
	Logger        *slog.Logger
	Tracer        trace.Tracer
}

func (c Config) Validate() error {
	var errs []error

	if c.SvixAPIClient == nil {
		errs = append(errs, errors.New("svix client is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.Tracer == nil {
		errs = append(errs, errors.New("tracer is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (webhook.Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid svix webhook handler config: %w", err)
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
	logger *slog.Logger
	tracer trace.Tracer
}

func NewHandler(config Config) (webhook.Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid svix webhook handler config: %w", err)
	}

	return &svixHandler{
		client: config.SvixAPIClient,
		logger: config.Logger,
		tracer: config.Tracer,
	}, nil
}
