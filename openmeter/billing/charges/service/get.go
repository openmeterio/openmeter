package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) GetChargeByID(ctx context.Context, input models.NamespacedID) (charges.Charge, error) {
	if err := input.Validate(); err != nil {
		return charges.Charge{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charge, error) {
		return s.adapter.GetChargeByID(ctx, input)
	})
}

func (s *service) GetChargesByIDs(ctx context.Context, namespace string, ids []string) (charges.Charges, error) {
	if namespace == "" {
		return nil, charges.ErrChargeNamespaceEmpty
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		return s.adapter.GetChargesByIDs(ctx, namespace, ids)
	})
}
