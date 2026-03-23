package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s *service) ListCharges(ctx context.Context, input charges.ListChargesInput) (pagination.Result[charges.Charge], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[charges.Charge]{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[charges.Charge], error) {
		chargesWithTypes, err := s.adapter.ListCharges(ctx, input)
		if err != nil {
			return pagination.Result[charges.Charge]{}, err
		}

		expandedCharges, err := s.expandChargesWithTypes(ctx, chargesWithTypes.Items, input.Expands)
		if err != nil {
			return pagination.Result[charges.Charge]{}, err
		}

		return pagination.Result[charges.Charge]{
			Page:       chargesWithTypes.Page,
			TotalCount: chargesWithTypes.TotalCount,
			Items:      expandedCharges,
		}, nil
	})
}
