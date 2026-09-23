package advance

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/crediteligibility"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

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
	Filters           crediteligibility.Filters
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

	if err := i.Filters.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("filters: %w", err))
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type BackfillPlan struct {
	Amount    alpacadecimal.Decimal
	Templates []transactions.TransactionTemplate
	Backfills []Backfill
	// LegacyAllocations must be persisted with the group that books Templates.
	LegacyAllocations []legacylineage.AdvanceBackfillAllocation
}

// Backfill can cover outstanding advance receivable without matching accrued
// value. In that case, only receivable is attributed; no accrued posting or
// legacy lineage backfill is created.
type Backfill struct {
	Amount             alpacadecimal.Decimal
	SpendChargeID      *string
	CollectionOriginID *string
}
