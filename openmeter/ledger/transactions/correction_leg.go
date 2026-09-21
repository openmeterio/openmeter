package transactions

import (
	"cmp"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// routePairingKey pairs source and counterpart sub-accounts during collection,
// receivable coverage, and earnings correction.
type routePairingKey struct {
	currency           string
	costBasisCurrency  mo.Option[currencyx.Code]
	taxCode            mo.Option[string]
	taxBehavior        mo.Option[ledger.TaxBehavior]
	features           string
	costBasis          mo.Option[string]
	sourceChargeID     mo.Option[string]
	spendChargeID      mo.Option[string]
	collectionOriginID mo.Option[string]
}

func (k routePairingKey) String() string {
	return fmt.Sprintf(
		"currency=%s,cost_basis_currency=%s,tax_code=%s,tax_behavior=%s,features=%s,cost_basis=%s,source_charge_id=%s,spend_charge_id=%s,collection_origin_id=%s",
		k.currency,
		k.costBasisCurrency.OrElse("null"),
		k.taxCode.OrElse("null"),
		k.taxBehavior.OrElse("null"),
		k.features,
		k.costBasis.OrElse("null"),
		k.sourceChargeID.OrElse("null"),
		k.spendChargeID.OrElse("null"),
		k.collectionOriginID.OrElse("null"),
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
				Provenance: entry.Provenance(),
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
				Provenance:       leg.identity.Provenance,
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
