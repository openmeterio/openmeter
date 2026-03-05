package adapter

import (
	"strconv"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgerdimensiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerdimension"
	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	ledgersubaccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	ledgersubaccountroutedb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	ledgertransactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type sumEntriesQuery struct {
	query ledger.Query
}

func (b *sumEntriesQuery) Build(client *db.Client) *db.LedgerEntryQuery {
	entryPredicates := b.entryPredicates()
	return client.LedgerEntry.Query().Where(entryPredicates...)
}

// SQL returns the final SQL shape and args used for sum aggregation.
func (b *sumEntriesQuery) SQL() (string, []any) {
	e := sql.Table(ledgerentrydb.Table)
	selector := sql.Select(sql.As(sql.Sum(e.C(ledgerentrydb.FieldAmount)), "sum_amount")).From(e)
	selector.SetDialect(dialect.Postgres)

	for _, predicate := range b.entryPredicates() {
		predicate(selector)
	}

	return selector.Query()
}

func (b *sumEntriesQuery) entryPredicates() []predicate.LedgerEntry {
	entryPredicates := make([]predicate.LedgerEntry, 0, 4)
	entryPredicates = append(entryPredicates, ledgerentrydb.Namespace(b.query.Namespace))

	if b.query.Filters.TransactionID != nil {
		entryPredicates = append(entryPredicates, ledgerentrydb.TransactionID(*b.query.Filters.TransactionID))
	}

	if b.query.Filters.BookedAtPeriod != nil {
		transactionPredicates := make([]predicate.LedgerTransaction, 0, 2)
		if b.query.Filters.BookedAtPeriod.From != nil {
			transactionPredicates = append(transactionPredicates, ledgertransactiondb.BookedAtGTE(*b.query.Filters.BookedAtPeriod.From))
		}
		if b.query.Filters.BookedAtPeriod.To != nil {
			transactionPredicates = append(transactionPredicates, ledgertransactiondb.BookedAtLT(*b.query.Filters.BookedAtPeriod.To))
		}
		if len(transactionPredicates) > 0 {
			entryPredicates = append(entryPredicates, ledgerentrydb.HasTransactionWith(transactionPredicates...))
		}
	}

	subAccountPredicates := b.subAccountPredicates()
	if len(subAccountPredicates) > 0 {
		entryPredicates = append(entryPredicates, ledgerentrydb.HasSubAccountWith(subAccountPredicates...))
	}

	return entryPredicates
}

func (b *sumEntriesQuery) subAccountPredicates() []predicate.LedgerSubAccount {
	subAccountPredicates := make([]predicate.LedgerSubAccount, 0, 1)
	routePredicates := make([]predicate.LedgerSubAccountRoute, 0, 2)
	if b.query.Filters.Dimensions.CurrencyID != "" {
		routePredicates = append(routePredicates, ledgersubaccountroutedb.CurrencyDimensionID(b.query.Filters.Dimensions.CurrencyID))
	}
	if b.query.Filters.Dimensions.CreditPriority != nil {
		routePredicates = append(routePredicates,
			ledgersubaccountroutedb.HasCreditPriorityDimensionWith(
				ledgerdimensiondb.DimensionKey(string(ledger.DimensionKeyCreditPriority)),
				ledgerdimensiondb.DimensionValue(strconv.Itoa(*b.query.Filters.Dimensions.CreditPriority)),
			),
		)
	}
	// DEFERRED: tax/feature filters are not active yet.

	if len(routePredicates) > 0 {
		subAccountPredicates = append(subAccountPredicates, ledgersubaccountdb.HasRouteWith(routePredicates...))
	}

	return subAccountPredicates
}
