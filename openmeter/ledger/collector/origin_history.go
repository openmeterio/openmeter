package collector

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

// originPair is an immutable forward posting and its still-reversible amount,
// reconstructed from exact correction references in the journal.
type originPair struct {
	transaction ledger.Transaction
	debit       ledger.Entry
	credit      ledger.Entry
	remaining   alpacadecimal.Decimal
	code        string
}

type originHistory struct {
	transactions []ledger.Transaction
	pairs        []*originPair
}

func (c *accrualCorrector) loadOriginHistory(ctx context.Context, namespace, originID string) (originHistory, error) {
	history := originHistory{}
	query := ledger.ListTransactionsInput{Namespace: namespace, OriginID: &originID, Limit: 100}
	for {
		page, err := c.ledger.ListTransactions(ctx, query)
		if err != nil {
			return originHistory{}, fmt.Errorf("load origin history: %w", err)
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
			return originHistory{}, err
		}
		for _, entry := range tx.Entries() {
			if lo.FromPtr(entry.OriginID()) != originID {
				continue
			}
			if err := ledger.ValidateEntryIdentityKey(entry); err != nil {
				return originHistory{}, err
			}
			_, identity, err := ledger.EntryIdentityKeyText(entry.IdentityKey()).Parse()
			if err != nil {
				return originHistory{}, err
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
		code, err := ledger.TransactionTemplateCodeFromAnnotations(tx.Annotations())
		if err != nil {
			return originHistory{}, err
		}
		switch code {
		case transactions.TemplateCode(transactions.TransferCustomerFBOToAccruedTemplate{}),
			transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}),
			transactions.TemplateCode(transactions.CoverCustomerReceivableTemplate{}),
			transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}),
			transactions.TemplateCode(transactions.TranslateCustomerAccruedCostBasisTemplate{}),
			transactions.TemplateCode(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{}),
			transactions.TemplateCode(transactions.RecognizeEarningsFromAttributableAccruedTemplate{}):
		case transactions.TemplateCode(transactions.ReleaseCustomerFBOBreakageTemplate{}), transactions.TemplateCode(transactions.ReopenCustomerFBOBreakageTemplate{}):
			continue // Breakage owns its release/reopen records.
		default:
			return originHistory{}, fmt.Errorf("unsupported forward template %s in collection origin", code)
		}
		pairs := make(map[string]*originPair)
		order := make([]string, 0)
		for _, entry := range tx.Entries() {
			if lo.FromPtr(entry.OriginID()) != originID {
				continue
			}
			key := lo.FromPtr(entry.SourceChargeID())
			if code == transactions.TemplateCode(transactions.TranslateCustomerAccruedCostBasisTemplate{}) ||
				code == transactions.TemplateCode(transactions.AttributeCustomerAdvanceReceivableCostBasisTemplate{}) {
				key = "" // These two-legged translations deliberately change source.
			}
			pair, ok := pairs[key]
			if !ok {
				pair = &originPair{transaction: tx, code: code}
				pairs[key] = pair
				order = append(order, key)
			}
			if entry.Amount().IsNegative() {
				if pair.debit != nil {
					return originHistory{}, fmt.Errorf("ambiguous debit in origin transaction %s", tx.ID().ID)
				}
				pair.debit = entry
			} else {
				if pair.credit != nil {
					return originHistory{}, fmt.Errorf("ambiguous credit in origin transaction %s", tx.ID().ID)
				}
				pair.credit = entry
			}
		}
		for _, key := range order {
			pair := pairs[key]
			if pair.debit == nil || pair.credit == nil || !pair.debit.Amount().Neg().Equal(pair.credit.Amount()) {
				return originHistory{}, fmt.Errorf("unbalanced origin pair in transaction %s", tx.ID().ID)
			}
			debitRemaining := pair.debit.Amount().Add(corrections[pair.debit.ID().ID]).Neg()
			creditRemaining := pair.credit.Amount().Add(corrections[pair.credit.ID().ID])
			if debitRemaining.IsNegative() || !debitRemaining.Equal(creditRemaining) {
				return originHistory{}, fmt.Errorf("invalid prior reversals in origin transaction %s", tx.ID().ID)
			}
			pair.remaining = debitRemaining
			history.pairs = append(history.pairs, pair)
		}
	}
	return history, nil
}

func (h originHistory) pairForTransaction(id string) (*originPair, error) {
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
