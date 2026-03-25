package routingrules

import (
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type EntryView struct {
	entry        ledger.EntryInput
	accountType  ledger.AccountType
	decodedRoute ledger.Route
}

func newEntryView(entry ledger.EntryInput) (EntryView, error) {
	return EntryView{
		entry:        entry,
		accountType:  entry.PostingAddress().AccountType(),
		decodedRoute: entry.PostingAddress().Route().Route(),
	}, nil
}

func (e EntryView) Amount() alpacadecimal.Decimal {
	return e.entry.Amount()
}

func (e EntryView) AccountType() ledger.AccountType {
	return e.accountType
}

func (e EntryView) Route() ledger.Route {
	return e.decodedRoute
}

func (e EntryView) Entry() ledger.EntryInput {
	return e.entry
}

type TxView struct {
	entries []EntryView
}

func NewTxView(entries []ledger.EntryInput) (TxView, error) {
	items := make([]EntryView, 0, len(entries))
	for _, entry := range entries {
		item, err := newEntryView(entry)
		if err != nil {
			return TxView{}, err
		}

		items = append(items, item)
	}

	return TxView{
		entries: items,
	}, nil
}

func (t TxView) Entries() []EntryView {
	return slices.Clone(t.entries)
}

func (t TxView) EntriesOf(accountType ledger.AccountType) []EntryView {
	out := make([]EntryView, 0, len(t.entries))
	for _, entry := range t.entries {
		if entry.AccountType() == accountType {
			out = append(out, entry)
		}
	}

	return out
}

func (t TxView) HasAccountType(accountType ledger.AccountType) bool {
	for _, entry := range t.entries {
		if entry.AccountType() == accountType {
			return true
		}
	}

	return false
}

func (t TxView) HasAccountTypes(accountTypes ...ledger.AccountType) bool {
	for _, accountType := range accountTypes {
		if !t.HasAccountType(accountType) {
			return false
		}
	}

	return true
}

func (t TxView) AccountTypes() []ledger.AccountType {
	seen := map[ledger.AccountType]struct{}{}
	out := make([]ledger.AccountType, 0, len(t.entries))

	for _, entry := range t.entries {
		accountType := entry.AccountType()
		if _, ok := seen[accountType]; ok {
			continue
		}

		seen[accountType] = struct{}{}
		out = append(out, accountType)
	}

	slices.Sort(out)

	return out
}

func optionalStringEqual(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func optionalIntEqual(left *int, right *int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func optionalTransactionAuthorizationStatusEqual(left *ledger.TransactionAuthorizationStatus, right *ledger.TransactionAuthorizationStatus) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func optionalDecimalEqual(left *alpacadecimal.Decimal, right *alpacadecimal.Decimal) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return left.Equal(*right)
}

func stringSliceEqual(left []string, right []string) bool {
	return slices.Equal(left, right)
}
