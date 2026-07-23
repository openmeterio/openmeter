package billingservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *Service) GetGatheringLinesForSubscription(ctx context.Context, input billing.GetLinesForSubscriptionInput) (billing.GatheringLines, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.GatheringLines, error) {
		return s.adapter.GetGatheringLinesForSubscription(ctx, input)
	})
}
