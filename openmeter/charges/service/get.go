package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) GetChargeByID(ctx context.Context, input models.NamespacedID) (charges.Charge, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charge, error) {
		return s.adapter.GetChargeByID(ctx, input)
	})
}
