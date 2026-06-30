package transactions

import (
	"cmp"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// routePairingKey pairs source and counterpart sub-accounts during accrual and
// earnings correction.
type routePairingKey struct {
	currency       currencyx.Code
	taxCode        string
	taxBehavior    string
	costBasis      string
	sourceChargeID string
	spendChargeID  string
}

func (k routePairingKey) String() string {
	return fmt.Sprintf(
		"currency=%s,tax_code=%s,tax_behavior=%s,cost_basis=%s,source_charge_id=%s,spend_charge_id=%s",
		k.currency,
		k.taxCode,
		k.taxBehavior,
		k.costBasis,
		k.sourceChargeID,
		k.spendChargeID,
	)
}

type correctionLeg struct {
	sourceAddress      ledger.PostingAddress
	sourceEntryID      string
	counterpartAddress ledger.PostingAddress
	amount             alpacadecimal.Decimal
	identity           ledger.EntryIdentityParts
}

type correctionPosting struct {
	address  ledger.PostingAddress
	amount   alpacadecimal.Decimal
	identity ledger.EntryIdentityParts
}

func allocateCorrectionLegs(
	sourceEntries []ledger.Entry,
	counterpartEntries []ledger.Entry,
	keyForEntry func(ledger.Entry) routePairingKey,
	sourceAmount func(ledger.Entry) alpacadecimal.Decimal,
	amount alpacadecimal.Decimal,
) ([]correctionPosting, error) {
	counterpartAddressesByKey := make(map[routePairingKey]ledger.PostingAddress, len(counterpartEntries))
	for _, entry := range counterpartEntries {
		key := keyForEntry(entry)
		address, ok := counterpartAddressesByKey[key]
		if ok && !address.Equal(entry.PostingAddress()) {
			return nil, fmt.Errorf("multiple counterpart addresses for correction key %s", key)
		}

		counterpartAddressesByKey[key] = entry.PostingAddress()
	}

	// Pair each source entry to exactly one counterpart entry fact. The caller owns
	// source ordering because different templates need different reversal order.
	legs := make([]correctionLeg, 0, len(sourceEntries))
	available := alpacadecimal.Zero
	for _, entry := range sourceEntries {
		entryAmount := sourceAmount(entry)
		if !entryAmount.IsPositive() {
			continue
		}

		key := keyForEntry(entry)
		counterpartAddress, ok := counterpartAddressesByKey[key]
		if !ok {
			return nil, fmt.Errorf("missing counterpart entry for correction key %s", key)
		}

		legs = append(legs, correctionLeg{
			sourceAddress:      entry.PostingAddress(),
			sourceEntryID:      entry.ID().ID,
			counterpartAddress: counterpartAddress,
			amount:             entryAmount,
			identity: ledger.EntryIdentityParts{
				SourceChargeID: entry.SourceChargeID(),
				SpendChargeID:  entry.SpendChargeID(),
			},
		})
		available = available.Add(entryAmount)
	}

	if amount.GreaterThan(available) {
		return nil, fmt.Errorf("correction amount %s exceeds available amount %s", amount.String(), available.String())
	}

	postings := make([]correctionPosting, 0, len(legs)*2)
	postingsByIdentity := make(map[string]int, len(legs)*2)
	// Source postings keep one entry per corrected source entry so correction
	// ordering remains visible. Counterpart postings can coalesce only when they
	// share the same address and source/spend facts.
	addPosting := func(address ledger.PostingAddress, amount alpacadecimal.Decimal, identity ledger.EntryIdentityParts, coalesce bool) {
		identityText, _ := identity.Text()
		coalesceKey := address.SubAccountID() + ":" + string(identityText)
		if idx, ok := postingsByIdentity[coalesceKey]; ok && coalesce {
			postings[idx].amount = postings[idx].amount.Add(amount)
			return
		}

		if coalesce {
			postingsByIdentity[coalesceKey] = len(postings)
		}
		postings = append(postings, correctionPosting{
			address:  address,
			amount:   amount,
			identity: identity,
		})
	}

	remaining := amount
	for idx := len(legs) - 1; idx >= 0 && remaining.IsPositive(); idx-- {
		leg := legs[idx]
		if leg.amount.GreaterThan(remaining) {
			leg.amount = remaining
		}

		addPosting(
			leg.sourceAddress,
			leg.amount,
			ledger.EntryIdentityParts{
				CorrectionSource: &leg.sourceEntryID,
				SourceChargeID:   leg.identity.SourceChargeID,
				SpendChargeID:    leg.identity.SpendChargeID,
			},
			false,
		)
		addPosting(leg.counterpartAddress, leg.amount.Neg(), leg.identity, true)
		remaining = remaining.Sub(leg.amount)
	}

	return postings, nil
}

func compareSubAccountID(left ledger.Entry, right ledger.Entry) int {
	return cmp.Compare(left.PostingAddress().SubAccountID(), right.PostingAddress().SubAccountID())
}

func mapCorrectionPostingsToEntryInputs(postings []correctionPosting) []*EntryInput {
	entryInputs := make([]*EntryInput, 0, len(postings))
	for _, posting := range postings {
		entryInputs = append(entryInputs, &EntryInput{
			address:  posting.address,
			amount:   posting.amount,
			identity: posting.identity,
		})
	}

	return entryInputs
}
