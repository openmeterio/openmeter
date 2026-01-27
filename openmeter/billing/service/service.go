package billingservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ billing.Service = (*Service)(nil)

type Service struct {
	adapter            billing.Adapter
	customerService    customer.Service
	appService         app.Service
	logger             *slog.Logger
	invoiceCalculator  invoicecalc.Calculator
	featureService     feature.FeatureConnector
	meterService       meter.Service
	streamingConnector streaming.Connector

	publisher eventbus.Publisher

	advancementStrategy          billing.AdvancementStrategy
	fsNamespaceLockdown          []string
	maxParallelQuantitySnapshots int
}

type Config struct {
	Adapter                      billing.Adapter
	CustomerService              customer.Service
	AppService                   app.Service
	Logger                       *slog.Logger
	FeatureService               feature.FeatureConnector
	MeterService                 meter.Service
	StreamingConnector           streaming.Connector
	Publisher                    eventbus.Publisher
	AdvancementStrategy          billing.AdvancementStrategy
	FSNamespaceLockdown          []string
	MaxParallelQuantitySnapshots int
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.CustomerService == nil {
		return errors.New("customer service cannot be null")
	}

	if c.AppService == nil {
		return errors.New("app service cannot be null")
	}

	if c.Logger == nil {
		return errors.New("logger cannot be null")
	}

	if c.FeatureService == nil {
		return errors.New("feature connector cannot be null")
	}

	if c.MeterService == nil {
		return errors.New("meter repo cannot be null")
	}

	if c.StreamingConnector == nil {
		return errors.New("streaming connector cannot be null")
	}

	if c.Publisher == nil {
		return errors.New("publisher cannot be null")
	}

	if err := c.AdvancementStrategy.Validate(); err != nil {
		return fmt.Errorf("validating advancement strategy: %w", err)
	}

	if c.MaxParallelQuantitySnapshots < 1 {
		return errors.New("max parallel snapshots must be greater than 0")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	svc := &Service{
		adapter:                      config.Adapter,
		customerService:              config.CustomerService,
		appService:                   config.AppService,
		logger:                       config.Logger,
		featureService:               config.FeatureService,
		meterService:                 config.MeterService,
		streamingConnector:           config.StreamingConnector,
		publisher:                    config.Publisher,
		advancementStrategy:          config.AdvancementStrategy,
		fsNamespaceLockdown:          config.FSNamespaceLockdown,
		maxParallelQuantitySnapshots: config.MaxParallelQuantitySnapshots,
		invoiceCalculator:            invoicecalc.New(),
	}

	return svc, nil
}

func (s Service) WithInvoiceCalculator(calc invoicecalc.Calculator) *Service {
	s.invoiceCalculator = calc

	return &s
}

func (s Service) InvoiceCalculator() invoicecalc.Calculator {
	return s.invoiceCalculator
}

// transcationForInvoiceManipulation is a helper function that wraps the given function in a transaction and ensures that
// an update lock is held on the customer record.
func transcationForInvoiceManipulation[T any](ctx context.Context, svc *Service, customerID customer.CustomerID, fn func(ctx context.Context) (T, error)) (T, error) {
	var empty T

	if err := customerID.Validate(); err != nil {
		return empty, fmt.Errorf("validating customer: %w", err)
	}

	// NOTE: This should not be in transaction, or we can get a conflict for parallel writes
	err := svc.adapter.UpsertCustomerLock(ctx, customerID)
	if err != nil {
		var empty T
		return empty, fmt.Errorf("upserting customer lock: %w", err)
	}

	return transaction.Run(ctx, svc.adapter, func(ctx context.Context) (T, error) {
		if err := svc.adapter.LockCustomerForUpdate(ctx, customerID); err != nil {
			var empty T
			return empty, fmt.Errorf("locking customer for update: %w", err)
		}

		return fn(ctx)
	})
}

func (s Service) GetAdvancementStrategy() billing.AdvancementStrategy {
	return s.advancementStrategy
}

func (s Service) WithAdvancementStrategy(strategy billing.AdvancementStrategy) billing.Service {
	s.advancementStrategy = strategy

	return &s
}

func (s *Service) WithLockedNamespaces(namespaces []string) billing.Service {
	s.fsNamespaceLockdown = namespaces

	return s
}
