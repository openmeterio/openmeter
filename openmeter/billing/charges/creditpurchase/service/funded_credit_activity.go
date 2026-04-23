package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) ListFundedCreditActivities(ctx context.Context, input creditpurchase.ListFundedCreditActivitiesInput) (creditpurchase.ListFundedCreditActivitiesResult, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.ListFundedCreditActivitiesResult{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.ListFundedCreditActivitiesResult, error) {
		return s.adapter.ListFundedCreditActivities(ctx, input)
	})
}
