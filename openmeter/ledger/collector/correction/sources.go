package correction

import (
	"cmp"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
)

// collectedSource is one logical “collection” in the group: the FBO→accrued
// forward tx, plus the receivable issue when that slice was advance-backed.
type collectedSource struct {
	// entries is the subset of transaction entries selected from one FBO subaccount (negative amounts only).
	entries                           []ledger.Entry
	transaction                       ledger.Transaction
	group                             ledger.TransactionGroup
	advanceReceivableIssueTransaction ledger.Transaction
}

func compareCollectedFBOCorrectionSourceEntries(left ledger.Entry, right ledger.Entry) int {
	leftOrder, leftHasOrder := left.Annotations().GetInt(ledger.AnnotationCollectionSourceOrder)
	rightOrder, rightHasOrder := right.Annotations().GetInt(ledger.AnnotationCollectionSourceOrder)
	if leftHasOrder && rightHasOrder && leftOrder != rightOrder {
		return cmp.Compare(leftOrder, rightOrder)
	}

	leftPriority := lo.FromPtrOr(left.PostingAddress().Route().Route().CreditPriority, ledger.DefaultCustomerFBOPriority)
	rightPriority := lo.FromPtrOr(right.PostingAddress().Route().Route().CreditPriority, ledger.DefaultCustomerFBOPriority)
	if leftPriority != rightPriority {
		return cmp.Compare(leftPriority, rightPriority)
	}

	if c := cmp.Compare(left.PostingAddress().SubAccountID(), right.PostingAddress().SubAccountID()); c != 0 {
		return c
	}

	return cmp.Compare(left.IdentityKey(), right.IdentityKey())
}

func (c *Corrector) collectedSourceBySortHint(group ledger.TransactionGroup, sortHint int) (collectedSource, error) {
	sources, err := c.collectedSourcesForGroup(group)
	if err != nil {
		return collectedSource{}, fmt.Errorf("map correction sources for group %s: %w", group.ID().ID, err)
	}

	if sortHint < 0 || sortHint >= len(sources) {
		return collectedSource{}, fmt.Errorf("allocation sort hint %d out of range for transaction group %s", sortHint, group.ID().ID)
	}

	return sources[sortHint], nil
}

func (c *Corrector) collectedSourcesForGroup(group ledger.TransactionGroup) ([]collectedSource, error) {
	out := make([]collectedSource, 0)
	for _, transaction := range group.Transactions() {
		templateCode, err := ledger.TransactionTemplateCodeFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s template code: %w", transaction.ID().ID, err)
		}

		direction, err := ledger.TransactionDirectionFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s direction: %w", transaction.ID().ID, err)
		}

		if direction != ledger.TransactionDirectionForward {
			continue
		}
		// Advance-backed collection comes with a receivable issue in the same group.
		var advanceReceivableIssueTransaction ledger.Transaction

		if templateCode == transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}) {
			advanceReceivableIssueTransaction, err = c.forwardTransactionByTemplate(group, transactions.TemplateCode(transactions.IssueCustomerReceivableTemplate{}))
			if err != nil {
				return nil, fmt.Errorf("find issue receivable companion in group %s: %w", group.ID().ID, err)
			}
		}

		sourceIndexBySubAccount := make(map[string]int)
		for _, entry := range transaction.Entries() {
			if entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerFBO && entry.Amount().IsNegative() {
				subAccountID := entry.PostingAddress().SubAccountID()
				if idx, ok := sourceIndexBySubAccount[subAccountID]; ok {
					out[idx].entries = append(out[idx].entries, entry)
					continue
				}
				sourceIndexBySubAccount[subAccountID] = len(out)
				out = append(out, collectedSource{
					entries:                           []ledger.Entry{entry},
					transaction:                       transaction,
					group:                             group,
					advanceReceivableIssueTransaction: advanceReceivableIssueTransaction,
				})
			}
		}
	}

	return out, nil
}

func (c *Corrector) forwardTransactionByTemplate(group ledger.TransactionGroup, templateCode string) (ledger.Transaction, error) {
	for _, transaction := range group.Transactions() {
		currentTemplateCode, err := ledger.TransactionTemplateCodeFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s template code: %w", transaction.ID().ID, err)
		}

		direction, err := ledger.TransactionDirectionFromAnnotations(transaction.Annotations())
		if err != nil {
			return nil, fmt.Errorf("transaction %s direction: %w", transaction.ID().ID, err)
		}

		if currentTemplateCode == templateCode && direction == ledger.TransactionDirectionForward {
			return transaction, nil
		}
	}

	return nil, fmt.Errorf("transaction with template code %s not found", templateCode)
}

func minDecimal(a, b alpacadecimal.Decimal) alpacadecimal.Decimal {
	if a.GreaterThan(b) {
		return b
	}

	return a
}
