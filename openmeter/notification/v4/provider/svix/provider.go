package svix

import (
	"errors"
	"fmt"
	"log/slog"

	svixv2 "github.com/svix/svix-webhooks/v2/go"
	"go.opentelemetry.io/otel/trace"

	notificationprovider "github.com/openmeterio/openmeter/openmeter/notification/v4/provider"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Client *svixv2.Svix
	Logger *slog.Logger
	Tracer trace.Tracer
}

var _ models.Validator = Config{}

func (c Config) Validate() error {
	var errs []error

	if c.Client == nil {
		errs = append(errs, errors.New("invalid svix client: required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("invalid logger: required"))
	}

	if c.Tracer == nil {
		errs = append(errs, errors.New("invalid tracer: required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Provider translates Notification resources to Svix resources. It does not
// persist provider data or schedule reconciliation.
type Provider struct {
	client *svixv2.Svix
	logger *slog.Logger
	tracer trace.Tracer
}

var _ notificationprovider.WebhookProvider = (*Provider)(nil)

// RoutingMode reports endpoint event-type filtering for initial sends.
// Resends target individual endpoints independently of this routing mode.
func (*Provider) RoutingMode() notificationprovider.ProviderRoutingMode {
	return notificationprovider.ProviderRoutingModeEventType
}

func New(config Config) (notificationprovider.WebhookProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid svix provider configuration: %w", err)
	}

	return &Provider{
		client: config.Client,
		logger: config.Logger,
		tracer: config.Tracer,
	}, nil
}
