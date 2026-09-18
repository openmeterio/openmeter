package advance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type BackfillDependencies struct {
	Ledger          ledger.Ledger
	BalanceQuerier  ledger.BalanceQuerier
	AccountResolver ledger.AccountResolver
}

type BackfillInput struct {
	CustomerID customer.CustomerID
	Currency   currencies.Currency
	// Amount is the purchased credit available to fund outstanding advances.
	Amount alpacadecimal.Decimal
	// At is the effective time for attribution, which may precede credit issuance.
	At                time.Time
	SourceChargeID    string
	CostBasis         alpacadecimal.Decimal
	CostBasisCurrency *currencyx.Code
	Features          []string
	LegacyLineages    []legacylineage.Lineage
}

func (i BackfillInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
		errs = append(errs, fmt.Errorf("amount: %w", err))
	}

	if i.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}

	if i.SourceChargeID == "" {
		errs = append(errs, errors.New("source charge id is required"))
	}

	if err := ledger.ValidateCostBasis(i.CostBasis); err != nil {
		errs = append(errs, fmt.Errorf("cost basis: %w", err))
	}

	if err := ledger.ValidateCostBasisCurrency(i.Currency.Reference().Code, i.CostBasisCurrency, &i.CostBasis); err != nil {
		errs = append(errs, fmt.Errorf("cost basis currency: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type BackfillResult struct {
	// Amount includes receivable-only attribution as well as accrued backfill.
	Amount       alpacadecimal.Decimal
	Templates    []transactions.TransactionTemplate
	Attributions []BackfillAttribution
	// LegacyAllocations must be persisted with the group that books Templates.
	LegacyAllocations []legacylineage.AdvanceBackfillAllocation
}

// BackfillAttribution identifies purchased credit consumed by an advance, including
// receivable-only attribution. It lets issuance release the corresponding breakage.
type BackfillAttribution struct {
	Amount             alpacadecimal.Decimal
	SpendChargeID      *string
	CollectionOriginID *string
}

// PlanBackfill selects advances in collection order and builds their attribution
// templates. The caller must hold the customer's posting locks in the database
// transaction that will commit these templates and persist legacy allocations.
func PlanBackfill(ctx context.Context, deps BackfillDependencies, input BackfillInput) (BackfillResult, error) {
	if err := input.Validate(); err != nil {
		return BackfillResult{}, err
	}

	planner := backfillPlanner{BackfillDependencies: deps}
	selected, err := planner.selectBackfill(ctx, input)
	if err != nil {
		return BackfillResult{}, err
	}

	result := BackfillResult{LegacyAllocations: selected.allocations}

	for _, attribution := range mergeAdvanceAttributions(selected.attributions) {
		result.Amount = result.Amount.Add(attribution.advanceAmount)
		result.Attributions = append(result.Attributions, BackfillAttribution{
			Amount:             attribution.advanceAmount,
			SpendChargeID:      attribution.spendChargeID,
			CollectionOriginID: attribution.collectionOriginID,
		})
		result.Templates = append(result.Templates, transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{
			At:                 input.At,
			Amount:             attribution.advanceAmount,
			Currency:           input.Currency.Reference(),
			CostBasisCurrency:  input.CostBasisCurrency,
			CostBasis:          &input.CostBasis,
			AdvanceFeatures:    attribution.advanceFeatures,
			AttributedFeatures: input.Features,
			SourceChargeID:     &input.SourceChargeID,
			SpendChargeID:      attribution.spendChargeID,
			CollectionOriginID: attribution.collectionOriginID,
		})

		if attribution.accruedAmount.IsPositive() {
			result.Templates = append(result.Templates, transactions.TranslateCustomerAccruedCostBasisTemplate{
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

	return result, nil
}
