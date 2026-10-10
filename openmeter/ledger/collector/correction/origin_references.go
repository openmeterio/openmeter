package correction

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

// originPair retains immutable routes and exact reversal capacity for the writer.
// Economic source selection uses account positions, not these posting amounts.
type originPair struct {
	transaction   ledger.Transaction
	negativeEntry ledger.Entry
	positiveEntry ledger.Entry
	remaining     alpacadecimal.Decimal
	role          originRole
}

func (p originPair) correctionSource() advance.CorrectionSource {
	return advance.CorrectionSource{
		Transaction:     p.transaction,
		NegativeEntry:   p.negativeEntry,
		PositiveEntry:   p.positiveEntry,
		RemainingAmount: p.remaining,
	}
}

func (pair originPair) reverse(at time.Time, amount alpacadecimal.Decimal) (ledger.TransactionInput, error) {
	if amount.GreaterThan(pair.remaining) {
		return nil, fmt.Errorf("reversal exceeds remaining original entry amount")
	}

	return transactions.ReverseOriginEntryPair(transactions.ReverseOriginEntryPairInput{
		At:            at,
		Amount:        amount,
		Transaction:   pair.transaction,
		NegativeEntry: pair.negativeEntry,
		PositiveEntry: pair.positiveEntry,
	})
}

// Roles describe account movements, independent of the template implementation.
type originRole int

const (
	originRoleCollection originRole = iota
	originRoleAdvanceIssue
	originRoleCoverage
	originRoleBacking
	originRoleAttribution
	originRoleRecognition
)

func originPairRole(negativeEntry, positiveEntry ledger.Entry) (originRole, error) {
	source, destination := negativeEntry.PostingAddress().AccountType(), positiveEntry.PostingAddress().AccountType()

	switch {
	case source == ledger.AccountTypeCustomerFBO && destination == ledger.AccountTypeCustomerAccrued:
		return originRoleCollection, nil
	case source == ledger.AccountTypeCustomerReceivable && destination == ledger.AccountTypeCustomerFBO:
		return originRoleAdvanceIssue, nil
	case source == ledger.AccountTypeCustomerFBO && destination == ledger.AccountTypeCustomerReceivable:
		return originRoleCoverage, nil
	case source == ledger.AccountTypeCustomerAccrued && destination == ledger.AccountTypeCustomerAccrued:
		return originRoleBacking, nil
	case source == ledger.AccountTypeCustomerReceivable && destination == ledger.AccountTypeCustomerReceivable:
		return originRoleAttribution, nil
	case source == ledger.AccountTypeCustomerAccrued && destination == ledger.AccountTypeEarnings:
		return originRoleRecognition, nil
	default:
		return 0, fmt.Errorf("unsupported collection movement %s -> %s", source, destination)
	}
}

type originReferences struct {
	transactions []ledger.Transaction
	pairs        []*originPair
}

func (h originReferences) attributionForBackfill(backfill *originPair) (*originPair, error) {
	var attribution *originPair

	for _, pair := range h.pairs {
		if pair.role != originRoleAttribution || pair.transaction.GroupID() != backfill.transaction.GroupID() ||
			lo.FromPtr(pair.negativeEntry.Provenance().SourceChargeID) != lo.FromPtr(backfill.positiveEntry.Provenance().SourceChargeID) {
			continue
		}

		if attribution != nil {
			return nil, fmt.Errorf("ambiguous advance attribution for origin")
		}

		attribution = pair
	}

	if attribution == nil {
		return nil, fmt.Errorf("advance backfill has no matching receivable attribution")
	}

	return attribution, nil
}

func (c *Corrector) loadOriginReferences(ctx context.Context, namespace, collectionOriginID string) (originReferences, error) {
	history := originReferences{}
	query := ledger.ListTransactionsInput{
		Namespace: namespace,
		EntryFilter: ledger.TransactionEntryFilter{
			Provenance: ledger.ProvenanceFilter{CollectionOriginID: mo.Some(&collectionOriginID)},
		},
		ReturnOnlyMatchingEntries: true,
		Limit:                     100,
	}

	for {
		page, err := c.ledger.ListTransactions(ctx, query)
		if err != nil {
			return originReferences{}, fmt.Errorf("load origin history: %w", err)
		}

		history.transactions = append(history.transactions, page.Items...)

		if page.NextCursor == nil {
			break
		}

		query.Cursor = page.NextCursor
	}

	// Recording order expresses dependencies even when a later purchase is
	// booked before the usage it backfills.
	slices.SortStableFunc(history.transactions, func(a, b ledger.Transaction) int {
		if c := a.Cursor().CreatedAt.Compare(b.Cursor().CreatedAt); c != 0 {
			return -c
		}

		return -cmp.Compare(a.ID().ID, b.ID().ID)
	})

	corrections := make(map[string]alpacadecimal.Decimal)

	for _, tx := range history.transactions {
		direction, err := ledger.TransactionDirectionFromAnnotations(tx.Annotations())
		if err != nil {
			return originReferences{}, err
		}

		for _, entry := range tx.Entries() {
			if lo.FromPtr(entry.Provenance().CollectionOriginID) != collectionOriginID {
				continue
			}

			if err := ledger.ValidateEntryIdentityKey(entry); err != nil {
				return originReferences{}, err
			}

			_, identity, err := ledger.EntryIdentityKeyText(entry.IdentityKey()).Parse()
			if err != nil {
				return originReferences{}, err
			}

			if direction == ledger.TransactionDirectionCorrection && identity.CorrectionSource != nil {
				id := *identity.CorrectionSource
				corrections[id] = corrections[id].Add(entry.Amount())
			}
		}
	}

	for _, tx := range history.transactions {
		direction, _ := ledger.TransactionDirectionFromAnnotations(tx.Annotations())
		if direction != ledger.TransactionDirectionForward {
			continue
		}

		// Expiry remains owned by breakage's existing release/reopen records.
		breakageMovement, sameAccount := false, true

		for _, entry := range tx.Entries() {
			if entry.PostingAddress().AccountType() == ledger.AccountTypeBreakage {
				breakageMovement = true
			}

			if entry.PostingAddress().AccountType() != tx.Entries()[0].PostingAddress().AccountType() {
				sameAccount = false
			}
		}

		if breakageMovement {
			continue
		}

		pairs := make(map[string]*originPair)
		order := make([]string, 0)

		for _, entry := range tx.Entries() {
			if lo.FromPtr(entry.Provenance().CollectionOriginID) != collectionOriginID {
				continue
			}

			key := lo.FromPtr(entry.Provenance().SourceChargeID)
			if sameAccount {
				key = "" // These two-legged translations deliberately change source.
			}

			pair, ok := pairs[key]
			if !ok {
				pair = &originPair{transaction: tx}
				pairs[key] = pair

				order = append(order, key)
			}

			if entry.Amount().IsNegative() {
				if pair.negativeEntry != nil {
					return originReferences{}, fmt.Errorf("ambiguous negative entry in origin transaction %s", tx.ID().ID)
				}

				pair.negativeEntry = entry
			} else {
				if pair.positiveEntry != nil {
					return originReferences{}, fmt.Errorf("ambiguous positive entry in origin transaction %s", tx.ID().ID)
				}

				pair.positiveEntry = entry
			}
		}

		for _, key := range order {
			pair := pairs[key]
			if pair.negativeEntry == nil || pair.positiveEntry == nil || !pair.negativeEntry.Amount().Neg().Equal(pair.positiveEntry.Amount()) {
				return originReferences{}, fmt.Errorf("unbalanced origin pair in transaction %s", tx.ID().ID)
			}

			negativeRemaining := pair.negativeEntry.Amount().Add(corrections[pair.negativeEntry.ID().ID]).Neg()
			positiveRemaining := pair.positiveEntry.Amount().Add(corrections[pair.positiveEntry.ID().ID])

			if negativeRemaining.IsNegative() || !negativeRemaining.Equal(positiveRemaining) {
				return originReferences{}, fmt.Errorf("invalid prior reversals in origin transaction %s", tx.ID().ID)
			}

			role, err := originPairRole(pair.negativeEntry, pair.positiveEntry)
			if err != nil {
				return originReferences{}, err
			}

			pair.role = role
			pair.remaining = negativeRemaining
			history.pairs = append(history.pairs, pair)
		}
	}

	return history, nil
}

func (h originReferences) pairForTransaction(id string) (*originPair, error) {
	var found *originPair

	for _, pair := range h.pairs {
		if pair.transaction.ID().ID != id {
			continue
		}

		if found != nil {
			return nil, fmt.Errorf("ambiguous original origin transaction %s", id)
		}

		found = pair
	}

	if found == nil {
		return nil, fmt.Errorf("original transaction %s missing from origin history", id)
	}

	return found, nil
}
