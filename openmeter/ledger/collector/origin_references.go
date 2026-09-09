package collector

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// originPair retains immutable routes and exact reversal capacity for the writer.
// Economic source selection uses account positions, not these posting amounts.
type originPair struct {
	transaction ledger.Transaction
	debit       ledger.Entry
	credit      ledger.Entry
	remaining   alpacadecimal.Decimal
	role        originRole
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

func originPairRole(debit, credit ledger.Entry) (originRole, error) {
	d, c := debit.PostingAddress().AccountType(), credit.PostingAddress().AccountType()
	switch {
	case d == ledger.AccountTypeCustomerFBO && c == ledger.AccountTypeCustomerAccrued:
		return originRoleCollection, nil
	case d == ledger.AccountTypeCustomerReceivable && c == ledger.AccountTypeCustomerFBO:
		return originRoleAdvanceIssue, nil
	case d == ledger.AccountTypeCustomerFBO && c == ledger.AccountTypeCustomerReceivable:
		return originRoleCoverage, nil
	case d == ledger.AccountTypeCustomerAccrued && c == ledger.AccountTypeCustomerAccrued:
		return originRoleBacking, nil
	case d == ledger.AccountTypeCustomerReceivable && c == ledger.AccountTypeCustomerReceivable:
		return originRoleAttribution, nil
	case d == ledger.AccountTypeCustomerAccrued && c == ledger.AccountTypeEarnings:
		return originRoleRecognition, nil
	default:
		return 0, fmt.Errorf("unsupported collection movement %s -> %s", d, c)
	}
}

type originReferences struct {
	transactions []ledger.Transaction
	pairs        []*originPair
}

func (c *accrualCorrector) loadOriginReferences(ctx context.Context, namespace, collectionOriginID string) (originReferences, error) {
	history := originReferences{}
	query := ledger.ListTransactionsInput{Namespace: namespace, CollectionOriginID: &collectionOriginID, Limit: 100}
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
			if lo.FromPtr(entry.CollectionOriginID()) != collectionOriginID {
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
			if lo.FromPtr(entry.CollectionOriginID()) != collectionOriginID {
				continue
			}
			key := lo.FromPtr(entry.SourceChargeID())
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
				if pair.debit != nil {
					return originReferences{}, fmt.Errorf("ambiguous debit in origin transaction %s", tx.ID().ID)
				}
				pair.debit = entry
			} else {
				if pair.credit != nil {
					return originReferences{}, fmt.Errorf("ambiguous credit in origin transaction %s", tx.ID().ID)
				}
				pair.credit = entry
			}
		}
		for _, key := range order {
			pair := pairs[key]
			if pair.debit == nil || pair.credit == nil || !pair.debit.Amount().Neg().Equal(pair.credit.Amount()) {
				return originReferences{}, fmt.Errorf("unbalanced origin pair in transaction %s", tx.ID().ID)
			}
			debitRemaining := pair.debit.Amount().Add(corrections[pair.debit.ID().ID]).Neg()
			creditRemaining := pair.credit.Amount().Add(corrections[pair.credit.ID().ID])
			if debitRemaining.IsNegative() || !debitRemaining.Equal(creditRemaining) {
				return originReferences{}, fmt.Errorf("invalid prior reversals in origin transaction %s", tx.ID().ID)
			}
			role, err := originPairRole(pair.debit, pair.credit)
			if err != nil {
				return originReferences{}, err
			}
			pair.role = role
			pair.remaining = debitRemaining
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
