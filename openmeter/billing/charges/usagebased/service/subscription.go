package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) UpdateSubscriptionItemID(ctx context.Context, charge usagebased.Charge, newSubscriptionItemID string) (usagebased.Charge, error) {
	var errs []error

	if err := charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if newSubscriptionItemID == "" {
		errs = append(errs, errors.New("subscription item ID is required"))
	}

	if err := models.NewNillableGenericValidationError(errors.Join(errs...)); err != nil {
		return usagebased.Charge{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (usagebased.Charge, error) {
		return s.adapter.UpdateSubscriptionItemID(ctx, charge, newSubscriptionItemID)
	})
}

var _ usagebased.UsageBasedService = (*service)(nil)
