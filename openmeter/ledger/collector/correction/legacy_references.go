package correction

import (
	"context"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

// correctedSourceAmounts reads immutable correction links so a later partial
// correction resumes the original source suffix instead of restoring it twice.
func (c *Corrector) correctedSourceAmounts(ctx context.Context, input Input) (map[string]alpacadecimal.Decimal, error) {
	out := make(map[string]alpacadecimal.Decimal)
	var cursor *ledger.TransactionCursor

	for {
		page, err := c.ledger.ListTransactions(ctx, ledger.ListTransactionsInput{
			Namespace: input.Namespace, Limit: 100, Cursor: cursor,
			AnnotationFilters: map[string]string{
				ledger.AnnotationChargeID:             input.ChargeID,
				ledger.AnnotationTransactionDirection: string(ledger.TransactionDirectionCorrection),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("list prior charge corrections: %w", err)
		}

		for _, tx := range page.Items {
			for _, entry := range tx.Entries() {
				if !entry.Amount().IsPositive() {
					continue
				}
				_, identity, err := ledger.EntryIdentityKeyText(entry.IdentityKey()).Parse()
				if err != nil {
					return nil, fmt.Errorf("parse correction entry identity: %w", err)
				}

				if identity.CorrectionSource != nil {
					out[*identity.CorrectionSource] = out[*identity.CorrectionSource].Add(entry.Amount())
				}
			}
		}

		if page.NextCursor == nil {
			return out, nil
		}

		cursor = page.NextCursor
	}
}

// reserveCorrectionSources resumes the reverse original collection order after
// subtracting both persisted corrections and reservations earlier in this batch.
func reserveCorrectionSources(entries []ledger.Entry, amount alpacadecimal.Decimal, used map[string]alpacadecimal.Decimal) (map[string]alpacadecimal.Decimal, error) {
	entries = slices.Clone(entries)
	slices.SortStableFunc(entries, compareCollectedFBOCorrectionSourceEntries)
	out := make(map[string]alpacadecimal.Decimal)
	remaining := amount
	for i := len(entries) - 1; i >= 0 && remaining.IsPositive(); i-- {
		entry := entries[i]
		available := entry.Amount().Abs().Sub(used[entry.ID().ID])
		if !available.IsPositive() {
			continue
		}

		selected := minDecimal(available, remaining)
		out[entry.ID().ID] = selected
		used[entry.ID().ID] = used[entry.ID().ID].Add(selected)
		remaining = remaining.Sub(selected)
	}

	if remaining.IsPositive() {
		return nil, fmt.Errorf("correction exceeds remaining source capacity by %s", remaining)
	}

	return out, nil
}

// Recognition groups can include multiple spends and cost bases. Reverse only
// the accrued buckets that the source unwind will debit, including a backfill's
// translation from unknown to known cost basis.
type recognizedSourceAmountsInput struct {
	recognition ledger.Transaction
	plan        legacyCorrectionPlan
	amount      alpacadecimal.Decimal
	used        map[string]alpacadecimal.Decimal
}

func (c *Corrector) recognizedSourceAmounts(ctx context.Context, input recognizedSourceAmountsInput) (map[string]alpacadecimal.Decimal, error) {
	accrued := make(map[correctionPostingKey]alpacadecimal.Decimal)
	inputs := slices.Clone(input.plan.inputs)

	for _, correction := range input.plan.legacyCorrections {
		resolved, err := transactions.CorrectTransaction(ctx, c.deps, correction)
		if err != nil {
			return nil, fmt.Errorf("resolve recognition source unwind: %w", err)
		}

		inputs = append(inputs, resolved...)
	}

	for _, tx := range inputs {
		for _, entry := range tx.EntryInputs() {
			if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerAccrued {
				key := correctionEntryKey(entry)
				accrued[key] = accrued[key].Sub(entry.Amount())
			}
		}
	}

	out := make(map[string]alpacadecimal.Decimal)
	selected := alpacadecimal.Zero

	for _, entry := range input.recognition.Entries() {
		if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerAccrued || !entry.Amount().IsNegative() {
			continue
		}

		key := correctionEntryKey(entry)
		required := accrued[key]
		available := entry.Amount().Abs().Sub(input.used[entry.ID().ID])
		if !required.IsPositive() || !available.IsPositive() {
			continue
		}

		take := minDecimal(required, available)
		out[entry.ID().ID] = take
		input.used[entry.ID().ID] = input.used[entry.ID().ID].Add(take)
		accrued[key] = required.Sub(take)
		selected = selected.Add(take)
	}

	if !selected.Equal(input.amount) {
		return nil, fmt.Errorf("recognition source coverage %s does not match correction amount %s", selected, input.amount)
	}

	for _, remaining := range accrued {
		if !remaining.IsZero() {
			return nil, fmt.Errorf("recognition correction leaves unmatched accrued amount %s", remaining)
		}
	}

	return out, nil
}

type correctionPostingKey struct{ subAccountID, sourceChargeID, spendChargeID string }

func correctionEntryKey(entry ledger.EntryInput) correctionPostingKey {
	return correctionPostingKey{
		subAccountID:   entry.PostingAddress().SubAccountID(),
		sourceChargeID: lo.FromPtrOr(entry.Provenance().SourceChargeID, ""),
		spendChargeID:  lo.FromPtrOr(entry.Provenance().SpendChargeID, ""),
	}
}
