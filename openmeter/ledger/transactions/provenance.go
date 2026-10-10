package transactions

import (
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/equal"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ReverseOriginEntryPairInput identifies concrete original legs. Amount bounds
// from previously committed reversals are enforced by the caller under its
// posting lock; this operation validates the immutable pair itself.
type ReverseOriginEntryPairInput struct {
	At            time.Time
	Amount        alpacadecimal.Decimal
	Transaction   ledger.Transaction
	NegativeEntry ledger.Entry
	PositiveEntry ledger.Entry
}

func (i ReverseOriginEntryPairInput) Validate() error {
	var errs []error

	if i.At.IsZero() {
		errs = append(errs, errors.New("at is required"))
	}

	if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
		errs = append(errs, fmt.Errorf("amount: %w", err))
	}

	if i.Transaction == nil || i.NegativeEntry == nil || i.PositiveEntry == nil {
		return models.NewGenericValidationError(errors.New("original transaction and both entries are required"))
	}

	if i.NegativeEntry.TransactionID() != i.Transaction.ID() || i.PositiveEntry.TransactionID() != i.Transaction.ID() {
		errs = append(errs, errors.New("entries must belong to the original transaction"))
	}

	if i.NegativeEntry.Provenance().CollectionOriginID == nil || !equal.ComparablePtrEqual(i.NegativeEntry.Provenance().CollectionOriginID, i.PositiveEntry.Provenance().CollectionOriginID) {
		errs = append(errs, errors.New("entries must share a collection origin"))
	}

	if !i.NegativeEntry.Amount().IsNegative() || !i.PositiveEntry.Amount().IsPositive() || !i.NegativeEntry.Amount().Neg().Equal(i.PositiveEntry.Amount()) {
		errs = append(errs, errors.New("original entries must form an equal and opposite pair"))
	}

	if i.Amount.GreaterThan(i.PositiveEntry.Amount()) {
		errs = append(errs, errors.New("amount exceeds original pair"))
	}

	if !i.NegativeEntry.PostingAddress().Route().Route().Currency.Equal(i.PositiveEntry.PostingAddress().Route().Route().Currency) {
		errs = append(errs, errors.New("entry-pair reversal cannot convert currency"))
	}

	direction, err := ledger.TransactionDirectionFromAnnotations(i.Transaction.Annotations())
	if err != nil {
		errs = append(errs, err)
	} else if direction != ledger.TransactionDirectionForward {
		errs = append(errs, errors.New("only forward transactions can be reversed"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ReverseOriginEntryPair preserves each leg's original route and provenance.
// Both reversal entries reference the exact entry they offset, making repeated
// partial reversals reconstructable without mutable amount snapshots.
func ReverseOriginEntryPair(input ReverseOriginEntryPairInput) (ledger.TransactionInput, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	code, err := ledger.TransactionTemplateCodeFromAnnotations(input.Transaction.Annotations())
	if err != nil {
		return nil, err
	}

	entries := make([]*EntryInput, 0, 2)

	for _, original := range []ledger.Entry{input.NegativeEntry, input.PositiveEntry} {
		amount := input.Amount
		if original.Amount().IsPositive() {
			amount = amount.Neg()
		}

		id := original.ID().ID
		entries = append(entries, &EntryInput{
			address: original.PostingAddress(),
			amount:  amount,
			identity: ledger.EntryIdentityParts{
				Provenance:       original.Provenance(),
				CorrectionSource: &id,
			},
		})
	}

	return &TransactionInput{
		bookedAt:    input.At,
		entryInputs: entries,
		annotations: ledger.TransactionAnnotations(code, ledger.TransactionDirectionCorrection),
	}, nil
}
