package ledger

import (
	"slices"

	"github.com/alpacahq/alpacadecimal"
)

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

	route := address.Route().Route()
	routeFilter := filter.Route
	if routeFilter.Currency != "" && route.Currency != routeFilter.Currency {
		return false
	}
	if routeFilter.TaxCode.IsPresent() {
		taxCode, _ := routeFilter.TaxCode.Get()
		switch {
		case taxCode == nil && route.TaxCode != nil:
			return false
		case taxCode != nil && route.TaxCode == nil:
			return false
		case taxCode != nil && route.TaxCode != nil && *taxCode != *route.TaxCode:
			return false
		}
	}
	if routeFilter.TaxBehavior.IsPresent() {
		taxBehavior, _ := routeFilter.TaxBehavior.Get()
		switch {
		case taxBehavior == nil && route.TaxBehavior != nil:
			return false
		case taxBehavior != nil && route.TaxBehavior == nil:
			return false
		case taxBehavior != nil && route.TaxBehavior != nil && *taxBehavior != *route.TaxBehavior:
			return false
		}
	}
	if len(routeFilter.Features) > 0 && !slices.Equal(route.Features, SortedFeatures(routeFilter.Features)) {
		return false
	}
	if routeFilter.CostBasis.IsPresent() {
		costBasis, _ := routeFilter.CostBasis.Get()
		switch {
		case costBasis == nil && route.CostBasis != nil:
			return false
		case costBasis != nil && route.CostBasis == nil:
			return false
		case costBasis != nil && route.CostBasis != nil && !costBasis.Equal(*route.CostBasis):
			return false
		}
	}
	if routeFilter.CreditPriority != nil && (route.CreditPriority == nil || *route.CreditPriority != *routeFilter.CreditPriority) {
		return false
	}
	if routeFilter.TransactionAuthorizationStatus != nil && (route.TransactionAuthorizationStatus == nil || *route.TransactionAuthorizationStatus != *routeFilter.TransactionAuthorizationStatus) {
		return false
	}

	return true
}
