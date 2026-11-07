package calculator

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Source interface {
	GetEntitlementBalance(context.Context, models.NamespacedID, time.Time) (*meteredentitlement.Balance, error)
}

type Config struct {
	Source Source
	Logger *slog.Logger
}

func (c Config) Validate() error {
	var errs []error

	if c.Source == nil {
		errs = append(errs, errors.New("source is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func New(config Config) (rte.Calculator[meteredentitlement.Key, meteredentitlement.Balance], error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &calculator{source: config.Source, logger: config.Logger}, nil
}

type calculator struct {
	source Source
	logger *slog.Logger
}

func (c *calculator) Calculate(ctx context.Context, input rte.CalculateInput[meteredentitlement.Key]) (meteredentitlement.Balance, error) {
	if err := input.Validate(); err != nil {
		return meteredentitlement.Balance{}, err
	}

	// The existing calculator owns minute rounding and usage-period semantics.
	balance, err := c.source.GetEntitlementBalance(ctx, input.Key.EntitlementID, input.At)
	if err != nil {
		return meteredentitlement.Balance{}, err
	}

	return *balance, nil
}
