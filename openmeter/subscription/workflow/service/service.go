package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/customer"
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
	CustomerService customer.Service
	// framework
	TransactionManager transaction.Creator
	Logger             *slog.Logger
	Lockr              *lockr.Locker
	FeatureFlags       ffx.Service
}

type service struct {
	WorkflowServiceConfig
}

func NewWorkflowService(cfg WorkflowServiceConfig) subscriptionworkflow.Service {
	return &service{
		WorkflowServiceConfig: cfg,
	}
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
