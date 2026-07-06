package customerbalance

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
)

// breakageCreditTransactionLoader loads one public transaction type derived
// from breakage impacts, split by plan source kind.
type breakageCreditTransactionLoader struct {
	service         *service
	txType          CreditTransactionType
	name            string
	planSourceKinds []breakage.SourceKind
}

func newExpiredCreditTransactionLoader(s *service) creditTransactionLoader {
	return &breakageCreditTransactionLoader{
		service: s,
		txType:  CreditTransactionTypeExpired,
		name:    "Expired credits",
		// Plans are only created by issuance and voiding, so this is the exact
		// complement of the voided loader.
		planSourceKinds: []breakage.SourceKind{breakage.SourceKindCreditPurchase},
	}
}

func newVoidedCreditTransactionLoader(s *service) creditTransactionLoader {
	return &breakageCreditTransactionLoader{
		service:         s,
		txType:          CreditTransactionTypeVoided,
		name:            "Voided credits",
		planSourceKinds: []breakage.SourceKind{breakage.SourceKindCreditPurchaseVoid},
	}
}

func (l *breakageCreditTransactionLoader) Load(ctx context.Context, input creditTransactionLoaderInput) (creditTransactionLoaderResult, error) {
	result, err := l.service.Breakage.ListExpiredBreakageImpacts(ctx, breakage.ListExpiredBreakageImpactsInput{
		CustomerID:      input.CustomerID,
		Currency:        input.Currency,
		AsOf:            input.AsOf,
		After:           input.After,
		Before:          input.Before,
		Limit:           input.Limit,
		Route:           featureFilterRoute(input.FeatureFilter),
		PlanSourceKinds: l.planSourceKinds,
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
			Type:        l.txType,
			Currency:    impact.Currency,
			Amount:      impact.Amount,
			Name:        l.name,
			Annotations: impact.Annotations,
			balanceAsOf: &balanceAsOf,
		})
	}

	return creditTransactionLoaderResult{
		Items:   items,
		HasMore: result.HasMore,
	}, nil
}
