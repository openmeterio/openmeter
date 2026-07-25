package customerbalance

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
)

type voidedCreditTransactionLoader struct {
	service *service
}

func newVoidedCreditTransactionLoader(s *service) creditTransactionLoader {
	return &voidedCreditTransactionLoader{
		service: s,
	}
}

func (l *voidedCreditTransactionLoader) Load(ctx context.Context, input creditTransactionLoaderInput) (creditTransactionLoaderResult, error) {
	result, err := l.service.CreditVoid.ListVoidedCreditImpacts(ctx, creditvoid.ListVoidedCreditImpactsInput{
		CustomerID: input.CustomerID,
		Currency:   input.Currency,
		AsOf:       input.AsOf,
		After:      input.After,
		Before:     input.Before,
		Limit:      input.Limit,
		Route:      featureFilterRoute(input.FeatureFilter),
	})
	if err != nil {
		return creditTransactionLoaderResult{}, fmt.Errorf("list voided credit impacts: %w", err)
	}

	items := make([]CreditTransaction, 0, len(result.Items))
	for _, impact := range result.Items {
		balanceAsOf := impact.VoidedAt
		items = append(items, CreditTransaction{
			ID:          impact.ID,
			CreatedAt:   impact.CreatedAt,
			BookedAt:    impact.VoidedAt,
			Type:        CreditTransactionTypeVoided,
			Currency:    impact.Currency,
			Amount:      impact.Amount,
			Name:        "Voided credits",
			Annotations: impact.Annotations,
			balanceAsOf: &balanceAsOf,
		})
	}

	return creditTransactionLoaderResult{
		Items:   items,
		HasMore: result.HasMore,
	}, nil
}
