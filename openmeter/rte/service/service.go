package service

import (
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Meter               metric.Meter
	CreditBalances      Registration[creditbalance.Key, creditbalance.Balance]
	MeteredEntitlements Registration[meteredentitlement.Key, meteredentitlement.Balance]
	Logger              *slog.Logger
	Publisher           eventbus.Publisher
}

func (c Config) Validate() error {
	var errs []error

	if err := c.CreditBalances.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credit balances: %w", err))
	}

	if err := c.MeteredEntitlements.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("metered entitlements: %w", err))
	}

	if c.Meter == nil {
		errs = append(errs, errors.New("meter is required"))
	}

	if c.Publisher == nil {
		errs = append(errs, errors.New("publisher is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type service struct {
	creditBalances      rte.Calculation[creditbalance.Key, creditbalance.Balance]
	meteredEntitlements rte.Calculation[meteredentitlement.Key, meteredentitlement.Balance]
}

func New(config Config) (rte.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	creditMetrics, err := newCalculationMetrics(config.Meter, rte.CalculationKindCreditBalance)
	if err != nil {
		return nil, err
	}

	meteredMetrics, err := newCalculationMetrics(config.Meter, rte.CalculationKindMeteredEntitlement)
	if err != nil {
		return nil, err
	}

	return &service{
		creditBalances:      newCalculation(config.CreditBalances, creditMetrics, config.Publisher, rte.NewCalculationUpdatedEvent[creditbalance.Key, creditbalance.Balance]),
		meteredEntitlements: newCalculation(config.MeteredEntitlements, meteredMetrics, config.Publisher, rte.NewCalculationUpdatedEvent[meteredentitlement.Key, meteredentitlement.Balance]),
	}, nil
}

func (s *service) CreditBalances() rte.Calculation[creditbalance.Key, creditbalance.Balance] {
	return s.creditBalances
}

func (s *service) MeteredEntitlements() rte.Calculation[meteredentitlement.Key, meteredentitlement.Balance] {
	return s.meteredEntitlements
}
