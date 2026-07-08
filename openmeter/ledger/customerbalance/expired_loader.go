package customerbalance

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
)

func newExpiredCreditTransactionLoader(s *service) creditTransactionLoader {
	return &expiredCreditTransactionLoader{
		service: s,
	}
}

type expiredCreditTransactionLoader struct {
	service *service
}

func (l *expiredCreditTransactionLoader) Load(ctx context.Context, input creditTransactionLoaderInput) (creditTransactionLoaderResult, error) {
	result, err := l.service.Breakage.ListExpiredBreakageImpacts(ctx, breakage.ListExpiredBreakageImpactsInput{
		CustomerID: input.CustomerID,
		Currency:   input.Currency,
		AsOf:       input.AsOf,
		After:      input.After,
		Before:     input.Before,
		Limit:      input.Limit,
		Route:      featureFilterRoute(input.FeatureFilter),
	})
	if err != nil {
		return creditTransactionLoaderResult{}, fmt.Errorf("list expired breakage impacts: %w", err)
	}

	items := make([]CreditTransaction, 0, len(result.Items))
	for _, impact := range result.Items {
		balanceAsOf := impact.BookedAt
		items = append(items, CreditTransaction{
			ID:          impact.ID,
			CreatedAt:   impact.CreatedAt,
			BookedAt:    impact.BookedAt,
			Type:        CreditTransactionTypeExpired,
			Currency:    impact.Currency,
			Amount:      impact.Amount,
			Name:        "Expired credits",
			Annotations: impact.Annotations,
			balanceAsOf: &balanceAsOf,
		})
	}

	return creditTransactionLoaderResult{
		Items:   items,
		HasMore: result.HasMore,
	}, nil
}
