package ledger

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

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
	// Let's validate address dimensions
	// dimKeys := make(map[DimensionKey]bool)
	// for _, dim := range address.Dimensions() {
	// 	if err := ValidateDimensionByKey(dim); err != nil {
	// 		return fmt.Errorf("invalid dimension: %w", err)
	// 	}

	// 	if _, ok := dimKeys[dim.Key()]; ok {
	// 		return fmt.Errorf("dimension key %s is present multiple times", dim.Key())
	// 	}
	// 	dimKeys[dim.Key()] = true
	// }

	return nil
}

// TODO: We need to validate somehow that credit priority is correct...
// // ValidateCreditAccountBalance validates that
// // - customer accounts representing issued credits (either FIAT or CR)
// // don't go below 0. This is to enforce on the ledger level that priority calculations are correct.
// func ValidateCreditAccountBalance(ctx context.Context, acc Account) error {
// 	if acc.Address().Type() == AccountTypeCustomerFBO {
// 		bal, err := acc.GetBalance(ctx)
// 		if err != nil {
// 			return fmt.Errorf("failed to get balance for credit account %s: %w", acc.Address(), err)
// 		}

// 		// If the balance is negative, we need to return an error
// 		if bal.Settled().IsNegative() {
// 			return ErrCreditAccountBalanceIsNegative.WithAttrs(models.Attributes{
// 				"settled": bal.Settled(),
// 				"pending": bal.Pending(),
// 				"account": acc.Address(),
// 			})
// 		}
// 	}

// 	return nil
// }
