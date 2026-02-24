package ledger

import (
	"context"
	"fmt"

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
	// TODO: Implement
	return nil
}

func ValidateEntryInput(ctx context.Context, entry EntryInput) error {
	// Let's validate the address
	if err := ValidateAddress(ctx, entry.PostingAddress()); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	return nil
}

func ValidateAddress(ctx context.Context, address PostingAddress) error {
	return nil
}

func ValidateTransactionInput(ctx context.Context, transaction TransactionInput) error {
	// Let's validate that the entries add up
	if err := ValidateInvariance(ctx, lo.Map(transaction.EntryInputs(), func(e EntryInput, _ int) EntryInput {
		return e
	})); err != nil {
		return err
	}

	// Let's validate routing
	if err := ValidateRouting(ctx, lo.Map(transaction.EntryInputs(), func(e EntryInput, _ int) EntryInput {
		return e
	})); err != nil {
		return err
	}

	// Let's validate the entries themselves
	for _, entry := range transaction.EntryInputs() {
		if err := ValidateEntryInput(ctx, entry); err != nil {
			return fmt.Errorf("invalid entry: %w", err)
		}
	}

	return nil
}
