package advance

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CorrectionSource retains original entries and the amount still reversible.
// The caller reads remaining capacity under the posting lock, including prior corrections.
type CorrectionSource struct {
	Transaction     ledger.Transaction
	NegativeEntry   ledger.Entry
	PositiveEntry   ledger.Entry
	RemainingAmount alpacadecimal.Decimal
}

func (s CorrectionSource) Reverse(at time.Time, amount alpacadecimal.Decimal) (ledger.TransactionInput, error) {
	if amount.GreaterThan(s.RemainingAmount) {
		return nil, fmt.Errorf("reversal exceeds remaining original entry amount")
	}

	return transactions.ReverseOriginEntryPair(transactions.ReverseOriginEntryPairInput{
		At:            at,
		Amount:        amount,
		Transaction:   s.Transaction,
		NegativeEntry: s.NegativeEntry,
		PositiveEntry: s.PositiveEntry,
	})
}

type BackfillCorrection struct {
	Amount     alpacadecimal.Decimal
	Accrued    CorrectionSource
	Receivable CorrectionSource
}

type CorrectionInput struct {
	CustomerID customer.CustomerID
	At         time.Time
	Amount     alpacadecimal.Decimal
	Collection CorrectionSource
	Issue      CorrectionSource
	Backfills  []BackfillCorrection
}

func (i CorrectionInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if i.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}

	if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
		errs = append(errs, fmt.Errorf("amount: %w", err))
	}

	// Reversal validation checks pair identity, direction, signs and capacity
	// before the service reads backfill groups or plans breakage changes.
	if _, err := i.Collection.Reverse(i.At, i.Amount); err != nil {
		errs = append(errs, fmt.Errorf("collection: %w", err))
	}

	if _, err := i.Issue.Reverse(i.At, i.Amount); err != nil {
		errs = append(errs, fmt.Errorf("issue: %w", err))
	}

	total := alpacadecimal.Zero

	for idx, backfill := range i.Backfills {
		if _, err := backfill.Accrued.Reverse(i.At, backfill.Amount); err != nil {
			errs = append(errs, fmt.Errorf("backfills[%d].accrued: %w", idx, err))
		}

		if _, err := backfill.Receivable.Reverse(i.At, backfill.Amount); err != nil {
			errs = append(errs, fmt.Errorf("backfills[%d].receivable: %w", idx, err))
		}

		total = total.Add(backfill.Amount)
	}

	if total.GreaterThan(i.Amount) {
		errs = append(errs, errors.New("backfill corrections exceed advance correction amount"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CorrectionPlan struct {
	Inputs          []ledger.TransactionInput
	BreakagePending []breakage.PendingRecord
	// LegacyCorrections remain unresolved so the caller can merge selections
	// from the same original transaction before invoking template correction.
	LegacyCorrections []transactions.CorrectionInput
}

type LegacyCorrectionInput struct {
	CustomerID     customer.CustomerID
	ChargeID       string
	At             time.Time
	Amount         alpacadecimal.Decimal
	OriginalGroup  ledger.TransactionGroup
	Collection     ledger.Transaction
	Issue          ledger.Transaction
	BackingGroupID *string
}

func (i LegacyCorrectionInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge id is required"))
	}

	if i.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}

	if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
		errs = append(errs, fmt.Errorf("amount: %w", err))
	}

	if i.OriginalGroup == nil || i.Collection == nil || i.Issue == nil {
		errs = append(errs, errors.New("original advance group, collection and issue are required"))
	}

	if i.BackingGroupID != nil && *i.BackingGroupID == "" {
		errs = append(errs, errors.New("backing group id cannot be empty"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
