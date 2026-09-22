package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

func (s *service) PlanBackfill(ctx context.Context, input advance.BackfillInput) (advance.BackfillPlan, error) {
	if err := input.Validate(); err != nil {
		return advance.BackfillPlan{}, err
	}

	selected, err := s.selectBackfill(ctx, input)
	if err != nil {
		return advance.BackfillPlan{}, err
	}

	plan := advance.BackfillPlan{LegacyAllocations: selected.allocations}

	for _, attribution := range mergeAdvanceAttributions(selected.attributions) {
		plan.Amount = plan.Amount.Add(attribution.advanceAmount)
		plan.Backfills = append(plan.Backfills, advance.Backfill{
			Amount:             attribution.advanceAmount,
			SpendChargeID:      attribution.spendChargeID,
			CollectionOriginID: attribution.collectionOriginID,
		})
		plan.Templates = append(plan.Templates, transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{
			At:                 input.At,
			Amount:             attribution.advanceAmount,
			Currency:           input.Currency.Reference(),
			CostBasisCurrency:  input.CostBasisCurrency,
			CostBasis:          &input.CostBasis,
			AdvanceFilters:     attribution.advanceFilters,
			AttributedFilters:  input.Filters,
			SourceChargeID:     &input.SourceChargeID,
			SpendChargeID:      attribution.spendChargeID,
			CollectionOriginID: attribution.collectionOriginID,
		})

		if attribution.accruedAmount.IsPositive() {
			plan.Templates = append(plan.Templates, transactions.TranslateCustomerAccruedCostBasisTemplate{
				At:                 input.At,
				Amount:             attribution.accruedAmount,
				Currency:           input.Currency.Reference(),
				TaxCode:            attribution.taxCode,
				TaxBehavior:        attribution.taxBehavior,
				FromCostBasis:      nil,
				ToCostBasis:        &input.CostBasis,
				CostBasisCurrency:  input.CostBasisCurrency,
				SourceChargeID:     &input.SourceChargeID,
				SpendChargeID:      attribution.spendChargeID,
				CollectionOriginID: attribution.collectionOriginID,
			})
		}
	}

	return plan, nil
}
