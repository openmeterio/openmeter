package service

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type FeatureFlags struct {
	EnableFlatFeeInAdvanceProrating bool
	EnableFlatFeeInArrearsProrating bool
}

type Config struct {
	BillingService          billing.Service
	SubscriptionService     subscription.Service
	SubscriptionSyncAdapter subscriptionsync.Adapter
	FeatureFlags            FeatureFlags
	Logger                  *slog.Logger
	Tracer                  trace.Tracer
}

func (c Config) Validate() error {
	if c.BillingService == nil {
		return fmt.Errorf("billing service is required")
	}

	if c.SubscriptionService == nil {
		return fmt.Errorf("subscription service is required")
	}

	if c.SubscriptionSyncAdapter == nil {
		return fmt.Errorf("subscription sync adapter is required")
	}

	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	if c.Tracer == nil {
		return fmt.Errorf("tracer is required")
	}

	return nil
}

var _ subscriptionsync.Service = (*Service)(nil)

type Service struct {
	billingService          billing.Service
	subscriptionService     subscription.Service
	subscriptionSyncAdapter subscriptionsync.Adapter
	featureFlags            FeatureFlags
	logger                  *slog.Logger
	tracer                  trace.Tracer
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Service{
		billingService:          config.BillingService,
		subscriptionSyncAdapter: config.SubscriptionSyncAdapter,
		featureFlags:            config.FeatureFlags,
		subscriptionService:     config.SubscriptionService,
		logger:                  config.Logger,
		tracer:                  config.Tracer,
	}, nil
}

func (s *Service) GetSyncStates(ctx context.Context, input subscriptionsync.GetSyncStatesInput) ([]subscriptionsync.SyncState, error) {
	return transaction.Run(ctx, s.subscriptionSyncAdapter, func(ctx context.Context) ([]subscriptionsync.SyncState, error) {
		return s.subscriptionSyncAdapter.GetSyncStates(ctx, input)
	})
}
