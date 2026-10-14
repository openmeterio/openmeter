package adapter

import (
	"maps"
	"slices"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"

	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	ledgertransactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type minimumBalanceBucketsQuery struct {
	query ledger.MinimumBalanceBucketQuery
}

// Start with each bucket's balance at AsOf, add its later movements in time
// order, then take the lowest balance. Sum the past entries first so the running
// sum doesn't have to process the whole history.
// Future exclusions only affect later movements, not the starting balance.
func (q minimumBalanceBucketsQuery) SQL() (string, []any, error) {
	if err := q.query.Validate(); err != nil {
		return "", nil, err
	}

	entryScope, err := q.scopedEntries()
	if err != nil {
		return "", nil, err
	}

	movements := q.openingBalances(entryScope).UnionAll(q.futureMovements(entryScope))
	ctes := sql.Dialect(dialect.Postgres).With("movements").As(movements).
		With("running_balances").As(q.runningBalances()).
		With(balanceBucketCTEName).As(q.minimumBalances())
	query, args := selectBalanceBuckets().Prefix(ctes).Query()

	return query, args, nil
}

// Apply the shared filters, but keep entries after AsOf: we need those to see
// how low the balance gets later.
func (q minimumBalanceBucketsQuery) scopedEntries() (*sql.Selector, error) {
	filters := q.query.Filters
	filters.AsOf = nil
	predicates, err := (&sumEntriesQuery{query: ledger.Query{
		Namespace: q.query.Namespace,
		Filters:   filters,
	}}).entryPredicates()
	if err != nil {
		return nil, err
	}

	for _, key := range slices.Sorted(maps.Keys(q.query.ExcludeAnnotationFilters)) {
		predicates = append(predicates, ledgerentrydb.HasTransactionWith(transactionAnnotationNotEqual(key, q.query.ExcludeAnnotationFilters[key])))
	}

	d := sql.Dialect(dialect.Postgres)
	entries := d.Table(ledgerentrydb.Table)
	txs := d.Table(ledgertransactiondb.Table).As("transactions")
	scope := d.Select().From(entries).
		Join(txs).On(entries.C(ledgerentrydb.FieldTransactionID), txs.C(ledgertransactiondb.FieldID))
	for _, predicate := range predicates {
		predicate(scope)
	}

	return scope, nil
}

// Sum everything booked at or before AsOf. This is the starting balance for
// each bucket, so future-only exclusions don't apply here.
func (q minimumBalanceBucketsQuery) openingBalances(scopedEntries *sql.Selector) *sql.Selector {
	d := sql.Dialect(dialect.Postgres)
	entries := d.Table(ledgerentrydb.Table)
	txs := d.Table(ledgertransactiondb.Table).As("transactions")
	groupColumns := q.entryBucketColumns()
	at := *q.query.Filters.AsOf

	return scopedEntries.Clone().Select(groupColumns...).
		AppendSelectExprAs(sql.ExprFunc(func(b *sql.Builder) {
			b.Arg(at).Join(sql.Raw("::timestamptz"))
		}), "effective_at").
		AppendSelect(
			sql.As(sql.Sum(entries.C(ledgerentrydb.FieldAmount)), "amount"),
			sql.As(sql.Min(entries.C(ledgerentrydb.FieldCreatedAt)), balanceBucketFieldOldestMatchingEntryCreatedAt),
		).
		Where(sql.LTE(txs.C(ledgertransactiondb.FieldBookedAt), at)).GroupBy(groupColumns...)
}

// Combine entries with the same booked time before calculating balances.
// Otherwise, processing a debit before a credit could show a dip that never
// actually existed. Excluded entries count as zero so we still return their
// bucket and keep its oldest entry timestamp.
func (q minimumBalanceBucketsQuery) futureMovements(scopedEntries *sql.Selector) *sql.Selector {
	d := sql.Dialect(dialect.Postgres)
	entries := d.Table(ledgerentrydb.Table)
	txs := d.Table(ledgertransactiondb.Table).As("transactions")
	groupColumns := q.entryBucketColumns()
	bookedAt := txs.C(ledgertransactiondb.FieldBookedAt)
	amount := entries.C(ledgerentrydb.FieldAmount)

	futureAmount := sql.ExprFunc(func(b *sql.Builder) {
		if len(q.query.ExcludeFutureAnnotationFilters) == 0 {
			b.Ident(sql.Sum(amount))

			return
		}

		annotations := txs.C(ledgertransactiondb.FieldAnnotations)
		included := make([]*sql.Predicate, 0, len(q.query.ExcludeFutureAnnotationFilters))
		for _, key := range slices.Sorted(maps.Keys(q.query.ExcludeFutureAnnotationFilters)) {
			included = append(included, sql.Or(
				sql.Not(sqljson.HasKey(annotations, sqljson.Path(key))),
				sqljson.ValueNEQ(annotations, q.query.ExcludeFutureAnnotationFilters[key], sqljson.Path(key)),
			))
		}

		b.Join(sql.Raw("SUM(CASE WHEN ")).Join(sql.And(included...)).
			Join(sql.Raw(" THEN ")).Ident(amount).Join(sql.Raw(" ELSE 0 END)"))
	})

	return scopedEntries.Clone().Select(groupColumns...).
		AppendSelect(sql.As(bookedAt, "effective_at")).
		AppendSelectExprAs(futureAmount, "amount").
		AppendSelect(sql.As(sql.Min(entries.C(ledgerentrydb.FieldCreatedAt)), balanceBucketFieldOldestMatchingEntryCreatedAt)).
		Where(sql.GT(bookedAt, *q.query.Filters.AsOf)).GroupBy(groupColumns...).GroupBy(bookedAt)
}

// Add each later change to the starting balance, separately for each bucket.
// Entries at the same time have already been combined.
func (q minimumBalanceBucketsQuery) runningBalances() *sql.Selector {
	d := sql.Dialect(dialect.Postgres)
	groupColumns := q.bucketColumns()

	return d.Select(groupColumns...).AppendSelect("effective_at", balanceBucketFieldOldestMatchingEntryCreatedAt).
		AppendSelectExprAs(sql.Window(func(b *sql.Builder) {
			b.Ident(sql.Sum("amount"))
		}).PartitionBy(groupColumns...).OrderBy("effective_at"), "balance").From(d.Table("movements"))
}

// Take the lowest balance. If a bucket has no entries at AsOf, include zero:
// credit that only arrives later isn't available at AsOf. Negative balances
// still need to come through as negative.
func (q minimumBalanceBucketsQuery) minimumBalances() *sql.Selector {
	d := sql.Dialect(dialect.Postgres)
	groupColumns := q.bucketColumns()

	balances := d.Table("running_balances")
	buckets := d.Select(balances.C(ledgerentrydb.FieldSubAccountID)).From(balances)
	appendBalanceBucketDimensionSelect(buckets, balances, q.query.GroupBy, ledger.BalanceBucketGroupBySourceChargeID, ledgerentrydb.FieldSourceChargeID)
	appendBalanceBucketDimensionSelect(buckets, balances, q.query.GroupBy, ledger.BalanceBucketGroupBySpendChargeID, ledgerentrydb.FieldSpendChargeID)
	appendBalanceBucketDimensionSelect(buckets, balances, q.query.GroupBy, ledger.BalanceBucketGroupByCollectionOriginID, ledgerentrydb.FieldCollectionOriginID)

	buckets.AppendSelectExprAs(sql.ExprFunc(func(b *sql.Builder) {
		b.Join(sql.Raw("CASE WHEN ")).Ident(sql.Min("effective_at")).WriteOp(sql.OpGT).Arg(*q.query.Filters.AsOf).
			Join(sql.Raw(" THEN LEAST(MIN(balance), 0) ELSE MIN(balance) END"))
	}), balanceBucketFieldSumAmount).AppendSelect(
		sql.As(sql.Min(balanceBucketFieldOldestMatchingEntryCreatedAt), balanceBucketFieldOldestMatchingEntryCreatedAt),
	).GroupBy(groupColumns...)

	return buckets
}

func (q minimumBalanceBucketsQuery) bucketColumns() []string {
	return append([]string{ledgerentrydb.FieldSubAccountID}, q.query.GroupBy...)
}

func (q minimumBalanceBucketsQuery) entryBucketColumns() []string {
	entries := sql.Dialect(dialect.Postgres).Table(ledgerentrydb.Table)
	columns := q.bucketColumns()
	for idx, column := range columns {
		columns[idx] = entries.C(column)
	}

	return columns
}
