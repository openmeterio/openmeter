package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) ListFundedCreditActivities(ctx context.Context, input charges.ListFundedCreditActivitiesInput) (charges.ListFundedCreditActivitiesResult, error) {
	if err := input.Validate(); err != nil {
		return charges.ListFundedCreditActivitiesResult{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.ListFundedCreditActivitiesResult, error) {
		return s.adapter.ListFundedCreditActivities(ctx, input)
	})
}
