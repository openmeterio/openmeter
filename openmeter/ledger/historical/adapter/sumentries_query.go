package adapter

import (
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
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
	routePredicates := make([]predicate.LedgerSubAccountRoute, 0, 4)
	if b.query.Filters.Route.Currency != "" {
		routePredicates = append(routePredicates, ledgersubaccountroutedb.Currency(b.query.Filters.Route.Currency))
	}
	if b.query.Filters.Route.CreditPriority != nil {
		routePredicates = append(routePredicates,
			ledgersubaccountroutedb.CreditPriority(*b.query.Filters.Route.CreditPriority),
		)
	}
	// DEFERRED: tax/feature route filters are not active yet but plumbing is in place.
	if b.query.Filters.Route.TaxCode != nil {
		routePredicates = append(routePredicates, ledgersubaccountroutedb.TaxCode(*b.query.Filters.Route.TaxCode))
	}
	if len(b.query.Filters.Route.Features) > 0 {
		// DB stores features as a sorted jsonb array; filter value is also sorted for canonical comparison.
		routePredicates = append(routePredicates, func(s *sql.Selector) {
			s.Where(sqljson.ValueEQ(ledgersubaccountroutedb.FieldFeatures, ledger.SortedFeatures(b.query.Filters.Route.Features)))
		})
	}

	if len(routePredicates) > 0 {
		subAccountPredicates = append(subAccountPredicates, ledgersubaccountdb.HasRouteWith(routePredicates...))
	}

	return subAccountPredicates
}
