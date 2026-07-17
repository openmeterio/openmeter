package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/ffx"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type WorkflowServiceConfig struct {
	Service      subscription.Service
	AddonService subscriptionaddon.Service
	// connectors
	CustomerService  customer.Service
	CurrencyResolver productcatalog.CurrencyResolver
	// framework
	TransactionManager transaction.Creator
	Logger             *slog.Logger
	Lockr              *lockr.Locker
	FeatureFlags       ffx.Service
}

func (c WorkflowServiceConfig) Validate() error {
	var errs []error

	if c.Service == nil {
		errs = append(errs, errors.New("subscription service is required"))
	}

	if c.AddonService == nil {
		errs = append(errs, errors.New("subscription add-on service is required"))
	}

	if c.CustomerService == nil {
		errs = append(errs, errors.New("customer service is required"))
	}

	if c.CurrencyResolver == nil {
		errs = append(errs, errors.New("currency resolver is required"))
	}

	if c.TransactionManager == nil {
		errs = append(errs, errors.New("transaction manager is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.Lockr == nil {
		errs = append(errs, errors.New("locker is required"))
	}

	if c.FeatureFlags == nil {
		errs = append(errs, errors.New("feature flags service is required"))
	}

	return errors.Join(errs...)
}

type service struct {
	WorkflowServiceConfig
}

func NewWorkflowService(cfg WorkflowServiceConfig) (subscriptionworkflow.Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid workflow service config: %w", err)
	}

	return &service{
		WorkflowServiceConfig: cfg,
	}, nil
}

var _ subscriptionworkflow.Service = &service{}

func (s *service) lockCustomer(ctx context.Context, customerId string) error {
	key, err := subscription.GetCustomerLock(customerId)
	if err != nil {
		return fmt.Errorf("failed to get customer lock: %w", err)
	}

	if err := s.Lockr.LockForTX(ctx, key); err != nil {
		return fmt.Errorf("failed to lock customer: %w", err)
	}

	return nil
}
