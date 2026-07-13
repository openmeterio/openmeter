package adapter

import (
	"slices"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	ledgeraccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgeraccount"
	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	ledgersubaccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	ledgersubaccountroutedb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

const (
	balanceBucketCTEName = "balance_buckets"

	balanceBucketFieldSubAccountID   = "sub_account_id"
	balanceBucketFieldSourceChargeID = "source_charge_id"
	balanceBucketFieldSpendChargeID  = "spend_charge_id"
	balanceBucketFieldSumAmount      = "sum_amount"
)

type balanceBucketsQuery struct {
	query ledger.BalanceBucketQuery
}

func (q balanceBucketsQuery) SQL() (string, []any, error) {
	bucketSelector, err := q.bucketSelector()
	if err != nil {
		return "", nil, err
	}

	buckets := sql.With(balanceBucketCTEName).As(bucketSelector)
	buckets.SetDialect(dialect.Postgres)

	subAccounts := sql.Table(ledgersubaccountdb.Table).As("sub_accounts")
	subAccounts.SetDialect(dialect.Postgres)

	accounts := sql.Table(ledgeraccountdb.Table).As("accounts")
	accounts.SetDialect(dialect.Postgres)

	routes := sql.Table(ledgersubaccountroutedb.Table).As("routes")
	routes.SetDialect(dialect.Postgres)

	selector := sql.Select(
		buckets.C(balanceBucketFieldSubAccountID),
		buckets.C(balanceBucketFieldSourceChargeID),
		buckets.C(balanceBucketFieldSpendChargeID),
		buckets.C(balanceBucketFieldSumAmount),
		subAccounts.C(ledgersubaccountdb.FieldRouteID),
		accounts.C(ledgeraccountdb.FieldAccountType),
		routes.C(ledgersubaccountroutedb.FieldRoutingKeyVersion),
		routes.C(ledgersubaccountroutedb.FieldRoutingKey),
		routes.C(ledgersubaccountroutedb.FieldCurrency),
		routes.C(ledgersubaccountroutedb.FieldSource),
		routes.C(ledgersubaccountroutedb.FieldTaxCode),
		routes.C(ledgersubaccountroutedb.FieldTaxBehavior),
		routes.C(ledgersubaccountroutedb.FieldFeatures),
		routes.C(ledgersubaccountroutedb.FieldCostBasis),
		routes.C(ledgersubaccountroutedb.FieldCreditPriority),
		routes.C(ledgersubaccountroutedb.FieldTransactionAuthorizationStatus),
	).
		From(buckets).
		Join(subAccounts).
		On(buckets.C(balanceBucketFieldSubAccountID), subAccounts.C(ledgersubaccountdb.FieldID)).
		Join(accounts).
		On(subAccounts.C(ledgersubaccountdb.FieldAccountID), accounts.C(ledgeraccountdb.FieldID)).
		Join(routes).
		On(subAccounts.C(ledgersubaccountdb.FieldRouteID), routes.C(ledgersubaccountroutedb.FieldID)).
		Prefix(buckets).
		OrderBy(
			buckets.C(balanceBucketFieldSubAccountID),
			buckets.C(balanceBucketFieldSourceChargeID),
			buckets.C(balanceBucketFieldSpendChargeID),
		)
	selector.SetDialect(dialect.Postgres)

	sqlQuery, args := selector.Query()
	return sqlQuery, args, nil
}

func (q balanceBucketsQuery) bucketSelector() (*sql.Selector, error) {
	entryPredicates, err := (&sumEntriesQuery{
		query: ledger.Query{
			Namespace: q.query.Namespace,
			Filters:   q.query.Filters,
		},
	}).entryPredicates()
	if err != nil {
		return nil, err
	}

	entries := sql.Table(ledgerentrydb.Table)
	entries.SetDialect(dialect.Postgres)

	selector := sql.Select(entries.C(ledgerentrydb.FieldSubAccountID)).From(entries)
	appendBalanceBucketDimensionSelect(selector, entries, q.query.GroupBy, ledger.BalanceBucketGroupBySourceChargeID, ledgerentrydb.FieldSourceChargeID)
	appendBalanceBucketDimensionSelect(selector, entries, q.query.GroupBy, ledger.BalanceBucketGroupBySpendChargeID, ledgerentrydb.FieldSpendChargeID)
	selector.AppendSelect(sql.As(sql.Sum(entries.C(ledgerentrydb.FieldAmount)), balanceBucketFieldSumAmount))
	selector.SetDialect(dialect.Postgres)
	for _, predicate := range entryPredicates {
		predicate(selector)
	}

	groupColumns := []string{entries.C(ledgerentrydb.FieldSubAccountID)}
	if slices.Contains(q.query.GroupBy, ledger.BalanceBucketGroupBySourceChargeID) {
		groupColumns = append(groupColumns, entries.C(ledgerentrydb.FieldSourceChargeID))
	}
	if slices.Contains(q.query.GroupBy, ledger.BalanceBucketGroupBySpendChargeID) {
		groupColumns = append(groupColumns, entries.C(ledgerentrydb.FieldSpendChargeID))
	}
	selector.GroupBy(groupColumns...)

	return selector, nil
}

func appendBalanceBucketDimensionSelect(selector *sql.Selector, entries *sql.SelectTable, groupBy []string, dimension string, field string) {
	if slices.Contains(groupBy, dimension) {
		selector.AppendSelect(entries.C(field))
		return
	}

	selector.AppendSelectExpr(sql.ExprFunc(func(b *sql.Builder) {
		b.WriteString("NULL AS ")
		b.Ident(field)
	}))
}
