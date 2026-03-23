package ledger

import (
	"context"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ValidateInvariance validates that Debit - Credit = 0 for the given entries.
func ValidateInvariance(ctx context.Context, entries []EntryInput) error {
	total := alpacadecimal.NewFromInt(0)
	for _, entry := range entries {
		total = total.Add(entry.Amount())
	}

	if total.IsZero() {
		return nil
	}

	return ErrInvalidTransactionTotal.WithAttrs(models.Attributes{
		"total":   total,
		"entries": entries,
	})
}

func ValidateRouting(ctx context.Context, entries []EntryInput) error {
	// Routing validation is implementation-specific and can be injected by the concrete ledger.
	return nil
}

func ValidateEntryInput(ctx context.Context, entry EntryInput) error {
	if entry == nil {
		return ErrEntryInvalid.WithAttrs(models.Attributes{
			"reason": "entry_required",
		})
	}

	// Let's validate the address
	if err := ValidateAddress(ctx, entry.PostingAddress()); err != nil {
		return ErrEntryInvalid.WithAttrs(models.Attributes{
			"reason": "invalid_address",
			"error":  err,
		})
	}

	return nil
}

func ValidateAddress(ctx context.Context, address PostingAddress) error {
	if address == nil {
		return ErrAddressInvalid.WithAttrs(models.Attributes{
			"reason": "address_required",
		})
	}

	return nil
}

func ValidateTransactionInput(ctx context.Context, transaction TransactionInput) error {
	return ValidateTransactionInputWith(ctx, transaction, nil)
}

func ValidateTransactionInputWith(ctx context.Context, transaction TransactionInput, routingValidator RoutingValidator) error {
	if transaction == nil {
		return ErrTransactionInputRequired
	}

	// Let's validate that the entries add up
	if err := ValidateInvariance(ctx, lo.Map(transaction.EntryInputs(), func(e EntryInput, _ int) EntryInput {
		return e
	})); err != nil {
		return err
	}

	// Let's validate the entries themselves
	for _, entry := range transaction.EntryInputs() {
		if err := ValidateEntryInput(ctx, entry); err != nil {
			return err
		}
	}

	if routingValidator != nil {
		if err := routingValidator.ValidateEntries(transaction.EntryInputs()); err != nil {
			return err
		}
	}

	return nil
}
