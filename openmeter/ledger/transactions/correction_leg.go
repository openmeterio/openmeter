package transactions

import (
	"cmp"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type routePairingKey struct {
	currency  currencyx.Code
	costBasis string
}

func (k routePairingKey) String() string {
	return fmt.Sprintf("currency=%s,cost_basis=%s", k.currency, k.costBasis)
}

type correctionLeg struct {
	sourceAddress      ledger.PostingAddress
	counterpartAddress ledger.PostingAddress
	amount             alpacadecimal.Decimal
}

type correctionPosting struct {
	address ledger.PostingAddress
	amount  alpacadecimal.Decimal
}

func allocateCorrectionLegs(
	sourceEntries []ledger.Entry,
	counterpartEntries []ledger.Entry,
	keyForAddress func(ledger.PostingAddress) routePairingKey,
	sourceAmount func(ledger.Entry) alpacadecimal.Decimal,
	amount alpacadecimal.Decimal,
) ([]correctionPosting, error) {
	counterpartAddressesByKey := make(map[routePairingKey]ledger.PostingAddress, len(counterpartEntries))
	for _, entry := range counterpartEntries {
		key := keyForAddress(entry.PostingAddress())
		address, ok := counterpartAddressesByKey[key]
		if ok && !address.Equal(entry.PostingAddress()) {
			return nil, fmt.Errorf("multiple counterpart addresses for correction key %s", key)
		}

		counterpartAddressesByKey[key] = entry.PostingAddress()
	}

	// Pair each source entry to exactly one counterpart route. The caller owns
	// source ordering because different templates need different reversal order.
	legs := make([]correctionLeg, 0, len(sourceEntries))
	available := alpacadecimal.Zero
	for _, entry := range sourceEntries {
		entryAmount := sourceAmount(entry)
		if !entryAmount.IsPositive() {
			continue
		}

		key := keyForAddress(entry.PostingAddress())
		counterpartAddress, ok := counterpartAddressesByKey[key]
		if !ok {
			return nil, fmt.Errorf("missing counterpart entry for correction key %s", key)
		}

		legs = append(legs, correctionLeg{
			sourceAddress:      entry.PostingAddress(),
			counterpartAddress: counterpartAddress,
			amount:             entryAmount,
		})
		available = available.Add(entryAmount)
	}

	if amount.GreaterThan(available) {
		return nil, fmt.Errorf("correction amount %s exceeds available amount %s", amount.String(), available.String())
	}

	postings := make([]correctionPosting, 0, len(legs)*2)
	postingsBySubAccountID := make(map[string]int, len(legs)*2)
	// Coalesce here so correction mapping cannot emit duplicate sub-account
	// postings, which routing rules reject.
	addPosting := func(address ledger.PostingAddress, amount alpacadecimal.Decimal) {
		subAccountID := address.SubAccountID()
		if idx, ok := postingsBySubAccountID[subAccountID]; ok {
			postings[idx].amount = postings[idx].amount.Add(amount)
			return
		}

		postingsBySubAccountID[subAccountID] = len(postings)
		postings = append(postings, correctionPosting{
			address: address,
			amount:  amount,
		})
	}

	remaining := amount
	for idx := len(legs) - 1; idx >= 0 && remaining.IsPositive(); idx-- {
		leg := legs[idx]
		if leg.amount.GreaterThan(remaining) {
			leg.amount = remaining
		}

		addPosting(leg.sourceAddress, leg.amount)
		addPosting(leg.counterpartAddress, leg.amount.Neg())
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
			address: posting.address,
			amount:  posting.amount,
		})
	}

	return entryInputs
}
