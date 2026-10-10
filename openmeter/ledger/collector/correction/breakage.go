package correction

import (
	"context"
	"fmt"
	"sort"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

type correctedFBOEntry struct {
	entry  ledger.Entry
	amount alpacadecimal.Decimal
}

func (c *Corrector) resolveBreakageReopenInputs(ctx context.Context, input Input, correction transactions.CorrectionInput) ([]ledger.TransactionInput, []breakage.PendingRecord, error) {
	templateCode, err := ledger.TransactionTemplateCodeFromAnnotations(correction.OriginalTransaction.Annotations())
	if err != nil {
		return nil, nil, fmt.Errorf("transaction %s template code: %w", correction.OriginalTransaction.ID().ID, err)
	}

	if templateCode != transactions.TemplateCode(transactions.TransferCustomerFBOToAccruedTemplate{}) &&
		templateCode != transactions.TemplateCode(transactions.CoverCustomerReceivableTemplate{}) {
		return nil, nil, nil
	}

	correctedEntries := correctedFBOEntriesForAmount(correction.OriginalTransaction, correction.Amount)
	if correction.SourceEntryAmounts != nil {
		correctedEntries = nil
		for _, entry := range correction.OriginalTransaction.Entries() {
			if amount := correction.SourceEntryAmounts[entry.ID().ID]; amount.IsPositive() {
				correctedEntries = append(correctedEntries, correctedFBOEntry{entry: entry, amount: amount})
			}
		}
	}

	if len(correctedEntries) == 0 {
		return nil, nil, nil
	}

	sourceEntryIDs := make([]string, 0, len(correctedEntries))
	for _, correctedEntry := range correctedEntries {
		sourceEntryIDs = append(sourceEntryIDs, correctedEntry.entry.ID().ID)
	}

	releases, err := c.breakage.ListReleases(ctx, breakage.ListReleasesInput{
		CustomerID:    customer.CustomerID{Namespace: input.Namespace, ID: input.CustomerID},
		SourceEntryID: sourceEntryIDs,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("list breakage releases: %w", err)
	}

	releasesBySourceEntryID := make(map[string][]breakage.Release, len(releases))
	for _, release := range releases {
		if release.SourceEntryID == nil || *release.SourceEntryID == "" {
			continue
		}

		releasesBySourceEntryID[*release.SourceEntryID] = append(releasesBySourceEntryID[*release.SourceEntryID], release)
	}

	inputs := make([]ledger.TransactionInput, 0, len(releases))
	pending := make([]breakage.PendingRecord, 0, len(releases))
	for _, correctedEntry := range correctedEntries {
		remaining := correctedEntry.amount
		for _, release := range releasesBySourceEntryID[correctedEntry.entry.ID().ID] {
			if !remaining.IsPositive() {
				break
			}

			amount := minDecimal(release.OpenAmount, remaining)
			if !amount.IsPositive() {
				continue
			}

			reopenInput, reopenRecord, err := c.breakage.ReopenRelease(ctx, breakage.ReopenReleaseInput{
				Release:            release,
				Amount:             amount,
				SourceKind:         breakage.SourceKindUsageCorrection,
				SourceChargeID:     correctedEntry.entry.Provenance().SourceChargeID,
				SpendChargeID:      correctedEntry.entry.Provenance().SpendChargeID,
				CollectionOriginID: correctedEntry.entry.Provenance().CollectionOriginID,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("resolve breakage reopen: %w", err)
			}

			inputs = append(inputs, reopenInput)
			pending = append(pending, reopenRecord)
			remaining = remaining.Sub(amount)
		}
	}

	return inputs, pending, nil
}

func correctedFBOEntriesForAmount(transaction ledger.Transaction, amount alpacadecimal.Decimal) []correctedFBOEntry {
	sourceEntries := make([]ledger.Entry, 0)
	for _, entry := range transaction.Entries() {
		if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsNegative() {
			sourceEntries = append(sourceEntries, entry)
		}
	}

	sort.SliceStable(sourceEntries, func(i, j int) bool {
		return compareCollectedFBOCorrectionSourceEntries(sourceEntries[i], sourceEntries[j]) < 0
	})

	remaining := amount
	out := make([]correctedFBOEntry, 0, len(sourceEntries))
	for idx := len(sourceEntries) - 1; idx >= 0 && remaining.IsPositive(); idx-- {
		entry := sourceEntries[idx]
		entryAmount := entry.Amount().Abs()
		if entryAmount.GreaterThan(remaining) {
			entryAmount = remaining
		}

		out = append(out, correctedFBOEntry{
			entry:  entry,
			amount: entryAmount,
		})
		remaining = remaining.Sub(entryAmount)
	}

	return out
}
