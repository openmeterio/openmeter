package ledger

import "github.com/alpacahq/alpacadecimal"

func TransactionImpact(tx Transaction, filter ImpactFilter) alpacadecimal.Decimal {
	total := alpacadecimal.Zero
	for _, entry := range tx.Entries() {
		if EntryMatchesImpactFilter(entry, filter) {
			total = total.Add(entry.Amount())
		}
	}

	return total
}

func EntryMatchesImpactFilter(entry Entry, filter ImpactFilter) bool {
	address := entry.PostingAddress()
	if filter.AccountType != "" && address.AccountType() != filter.AccountType {
		return false
	}

	if !address.Route().Route().Matches(filter.Route) {
		return false
	}

	return true
}
