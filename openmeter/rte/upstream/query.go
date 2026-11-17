package upstream

import (
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/lib/pq"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	chargedb "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebased"
	overridedb "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedoverride"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	subjectdb "github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	entitlementdb "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	featuredb "github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	accountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgercustomeraccount"
	entrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	subaccountdb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccount"
	routedb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgersubaccountroute"
	transactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	meterdb "github.com/openmeterio/openmeter/openmeter/ent/db/meter"
	demanddb "github.com/openmeterio/openmeter/openmeter/ent/db/rtedemand"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/rte"
)

// Only already-tracked keys are candidates. Routing a fact must not activate keys.
func activeDemandSelector(namespace string, now time.Time) *sql.Selector {
	pg := sql.Dialect(dialect.Postgres)
	d := pg.Table(demanddb.Table).As("d")

	return pg.Select(d.C(demanddb.FieldNamespace), d.C(demanddb.FieldKind), d.C(demanddb.FieldKey)).From(d).
		Where(sql.And(
			sql.EQ(d.C(demanddb.FieldNamespace), namespace),
			sql.IsNull(d.C(demanddb.FieldDeletedAt)),
			sql.Or(sql.IsTrue(d.C(demanddb.FieldPushEnabled)), sql.GT(d.C(demanddb.FieldPullUntil), now)),
		))
}

type ledgerQuery struct {
	namespace, groupID string
	now                time.Time
}

func (q ledgerQuery) SQL() (string, []any) {
	pg := sql.Dialect(dialect.Postgres)
	d := activeDemandSelector(q.namespace, q.now)
	t := pg.Table(transactiondb.Table).As("t")
	e := pg.Table(entrydb.Table).As("e")
	s := pg.Table(subaccountdb.Table).As("s")
	a := pg.Table(accountdb.Table).As("a")
	r := pg.Table(routedb.Table).As("r")

	entries := pg.Select(t.C(transactiondb.FieldID)).From(t).
		Join(e).OnP(sql.And(
		sql.ColumnsEQ(e.C(entrydb.FieldNamespace), t.C(transactiondb.FieldNamespace)),
		sql.ColumnsEQ(e.C(entrydb.FieldTransactionID), t.C(transactiondb.FieldID)),
	)).
		Join(s).OnP(sql.And(
		sql.ColumnsEQ(s.C(subaccountdb.FieldNamespace), e.C(entrydb.FieldNamespace)),
		sql.ColumnsEQ(s.C(subaccountdb.FieldID), e.C(entrydb.FieldSubAccountID)),
	)).
		Join(a).OnP(sql.And(
		sql.ColumnsEQ(a.C(accountdb.FieldNamespace), s.C(subaccountdb.FieldNamespace)),
		sql.ColumnsEQ(a.C(accountdb.FieldAccountID), s.C(subaccountdb.FieldAccountID)),
	)).
		Join(r).OnP(sql.And(
		sql.ColumnsEQ(r.C(routedb.FieldNamespace), s.C(subaccountdb.FieldNamespace)),
		sql.ColumnsEQ(r.C(routedb.FieldID), s.C(subaccountdb.FieldRouteID)),
	)).
		Where(sql.And(
			sql.ColumnsEQ(t.C(transactiondb.FieldNamespace), d.C(demanddb.FieldNamespace)),
			sql.EQ(t.C(transactiondb.FieldGroupID), q.groupID),
			sql.ColumnsEQ(a.C(accountdb.FieldCustomerID), d.C(demanddb.FieldOwnerID)),
			ledgerCurrencyScope(d.C(demanddb.FieldKey), r.C(routedb.FieldCurrency)),
		))

	return d.Where(sql.And(sql.EQ(d.C(demanddb.FieldKind), rte.CalculationKindCreditBalance), sql.Exists(entries))).Query()
}

// Custom ledger routes include the stable currency ID in their fourth component.
// Fiat routes match by code; feature scopes share the same ledger funding changes.
func ledgerCurrencyScope(key, currency string) *sql.Predicate {
	custom := sqljson.Path("currency", "custom_currency_id")
	return sql.Or(
		sql.And(
			sql.Not(sqljson.HasKey(key, custom, sqljson.Unquote(true))),
			sqljson.ValueEQ(key, sql.ExprFunc(func(b *sql.Builder) { b.Ident(currency) }), sqljson.Path("currency", "code")),
		),
		sql.And(
			sqljson.HasKey(key, custom, sqljson.Unquote(true)),
			sqljson.ValueEQ(key, sql.ExprFunc(func(b *sql.Builder) {
				b.WriteString("split_part").Wrap(func(b *sql.Builder) { b.Ident(currency).Comma().Arg("|").Comma().Arg(4) })
			}), custom),
		),
	)
}

type customerQuery struct {
	namespace, customerID string
	now                   time.Time
}

func (q customerQuery) SQL() (string, []any) {
	pg := sql.Dialect(dialect.Postgres)
	d := activeDemandSelector(q.namespace, q.now)
	e := pg.Table(entitlementdb.Table).As("e")

	entitlements := pg.Select(e.C(entitlementdb.FieldID)).From(e).Where(sql.And(
		sql.ColumnsEQ(e.C(entitlementdb.FieldNamespace), d.C(demanddb.FieldNamespace)),
		sql.ColumnsEQ(e.C(entitlementdb.FieldID), d.C(demanddb.FieldOwnerID)),
		sql.EQ(e.C(entitlementdb.FieldCustomerID), q.customerID),
	))

	return d.Where(sql.Or(
		sql.And(sql.EQ(d.C(demanddb.FieldKind), rte.CalculationKindCreditBalance), sql.EQ(d.C(demanddb.FieldOwnerID), q.customerID)),
		sql.And(sql.EQ(d.C(demanddb.FieldKind), rte.CalculationKindMeteredEntitlement), sql.Exists(entitlements)),
	)).Query()
}

type featureQuery struct {
	namespace, featureID string
	now                  time.Time
}

type meterQuery struct {
	namespace, meterID string
	now                time.Time
}

// A meter change fans out through all its features, including ID-less charges
// that still refer to a feature by key.
func (q meterQuery) SQL() (string, []any) {
	pg := sql.Dialect(dialect.Postgres)
	source := pg.Table(featuredb.Table)
	features := pg.Select(source.C(featuredb.FieldID), source.C(featuredb.FieldKey)).From(source).
		Where(sql.And(sql.EQ(source.C(featuredb.FieldNamespace), q.namespace), sql.EQ(source.C(featuredb.FieldMeterID), q.meterID)))

	d := activeDemandSelector(q.namespace, q.now)
	f := pg.Table("affected_features").As("f")
	e := pg.Table(entitlementdb.Table).As("e")
	c := pg.Table(chargedb.Table).As("c")

	entitlements := pg.Select(e.C(entitlementdb.FieldID)).From(e).
		Join(f).On(f.C(featuredb.FieldID), e.C(entitlementdb.FieldFeatureID)).
		Where(sql.And(
			sql.ColumnsEQ(e.C(entitlementdb.FieldNamespace), d.C(demanddb.FieldNamespace)),
			sql.ColumnsEQ(e.C(entitlementdb.FieldID), d.C(demanddb.FieldOwnerID)),
		))

	charges := pg.Select(c.C(chargedb.FieldID)).From(c).Join(f).OnP(chargeFeatureMatch(c, f)).
		Where(sql.And(
			sql.ColumnsEQ(c.C(chargedb.FieldNamespace), d.C(demanddb.FieldNamespace)),
			sql.ColumnsEQ(c.C(chargedb.FieldCustomerID), d.C(demanddb.FieldOwnerID)),
			creditChargeScope(d.C(demanddb.FieldKey), c),
		))

	return d.Prefix(pg.With("affected_features").As(features)).Where(sql.Or(
		sql.And(sql.EQ(d.C(demanddb.FieldKind), rte.CalculationKindMeteredEntitlement), sql.Exists(entitlements)),
		sql.And(sql.EQ(d.C(demanddb.FieldKind), rte.CalculationKindCreditBalance), sql.Exists(charges)),
	)).Query()
}

func (q featureQuery) SQL() (string, []any) {
	pg := sql.Dialect(dialect.Postgres)
	d := activeDemandSelector(q.namespace, q.now)
	e := pg.Table(entitlementdb.Table).As("e")
	c := pg.Table(chargedb.Table).As("c")
	f := pg.Table(featuredb.Table).As("f")

	entitlements := pg.Select(e.C(entitlementdb.FieldID)).From(e).Where(sql.And(
		sql.ColumnsEQ(e.C(entitlementdb.FieldNamespace), d.C(demanddb.FieldNamespace)),
		sql.ColumnsEQ(e.C(entitlementdb.FieldID), d.C(demanddb.FieldOwnerID)),
		sql.EQ(e.C(entitlementdb.FieldFeatureID), q.featureID),
	))

	charges := pg.Select(c.C(chargedb.FieldID)).From(c).Join(f).OnP(sql.And(
		sql.ColumnsEQ(f.C(featuredb.FieldNamespace), c.C(chargedb.FieldNamespace)),
		chargeFeatureMatch(c, f),
	)).Where(sql.And(
		sql.ColumnsEQ(c.C(chargedb.FieldNamespace), d.C(demanddb.FieldNamespace)),
		sql.ColumnsEQ(c.C(chargedb.FieldCustomerID), d.C(demanddb.FieldOwnerID)),
		sql.EQ(f.C(featuredb.FieldID), q.featureID),
		creditChargeScope(d.C(demanddb.FieldKey), c),
	))

	return d.Where(sql.Or(
		sql.And(sql.EQ(d.C(demanddb.FieldKind), rte.CalculationKindMeteredEntitlement), sql.Exists(entitlements)),
		sql.And(sql.EQ(d.C(demanddb.FieldKind), rte.CalculationKindCreditBalance), sql.Exists(charges)),
	)).Query()
}

func chargeFeatureMatch(c, f *sql.SelectTable) *sql.Predicate {
	return sql.Or(
		sql.ColumnsEQ(f.C(featuredb.FieldID), c.C(chargedb.FieldFeatureID)),
		sql.And(sql.IsNull(c.C(chargedb.FieldFeatureID)), sql.ColumnsEQ(f.C(featuredb.FieldKey), c.C(chargedb.FieldFeatureKey))),
	)
}

// Match current customerbalance live calculation identity. A feature-bound
// charge contributes to the portfolio and that feature, not unrestricted views.
func creditChargeScope(key string, c *sql.SelectTable) *sql.Predicate {
	custom := sqljson.Path("currency", "custom_currency_id")
	feature := sqljson.Path("featureFilter")
	return sql.And(
		sql.Or(
			sql.And(
				sql.NotNull(c.C(chargedb.FieldFiatCurrencyCode)),
				sqljson.ValueEQ(key, sql.ExprFunc(func(b *sql.Builder) { b.Ident(c.C(chargedb.FieldFiatCurrencyCode)) }), sqljson.Path("currency", "code")),
				sql.Not(sqljson.HasKey(key, custom, sqljson.Unquote(true))),
			),
			sql.And(
				sql.NotNull(c.C(chargedb.FieldCustomCurrencyID)),
				sqljson.ValueEQ(key, sql.ExprFunc(func(b *sql.Builder) { b.Ident(c.C(chargedb.FieldCustomCurrencyID)) }), custom),
			),
		),
		sql.Or(
			sql.Not(sqljson.HasKey(key, feature)),
			sqljson.ValueIsNull(key, feature),
			sql.P(func(b *sql.Builder) {
				b.Join(sqljson.ValuePath(key, feature)).WriteString(" @> jsonb_build_array").Wrap(func(b *sql.Builder) {
					b.Ident(c.C(chargedb.FieldFeatureKey))
				})
			}),
		),
	)
}

type ingestQuery struct {
	namespace, subject     string
	meterSlugs, eventTypes []string
	timestamps             []time.Time
	now                    time.Time
}

func (q ingestQuery) SQL() (string, []any) {
	pg := sql.Dialect(dialect.Postgres)
	d := activeDemandSelector(q.namespace, q.now)
	ctes := pg.With("customers_by_subject").As(q.customersBySubject()).
		With("features_by_meter").As(q.featuresByMeter())

	return d.Prefix(ctes).Where(sql.Or(
		sql.And(sql.EQ(d.C(demanddb.FieldKind), rte.CalculationKindMeteredEntitlement), sql.Exists(q.meteredEntitlements(d))),
		sql.And(sql.EQ(d.C(demanddb.FieldKind), rte.CalculationKindCreditBalance), sql.Exists(q.creditCharges(d))),
	)).Query()
}

func (q ingestQuery) customersBySubject() *sql.Selector {
	pg := sql.Dialect(dialect.Postgres)
	c := pg.Table(customerdb.Table).As("c")
	cs := pg.Table(subjectdb.Table).As("cs")
	mapping := pg.Select(cs.C(subjectdb.FieldID)).From(cs).Where(sql.And(
		sql.ColumnsEQ(cs.C(subjectdb.FieldNamespace), c.C(customerdb.FieldNamespace)),
		sql.ColumnsEQ(cs.C(subjectdb.FieldCustomerID), c.C(customerdb.FieldID)),
		sql.EQ(cs.C(subjectdb.FieldSubjectKey), q.subject),
		sql.IsNull(cs.C(subjectdb.FieldDeletedAt)),
	))

	return pg.Select(c.C(customerdb.FieldID)).From(c).Where(sql.And(
		sql.EQ(c.C(customerdb.FieldNamespace), q.namespace),
		sql.Or(sql.IsNull(c.C(customerdb.FieldDeletedAt)), sql.GT(c.C(customerdb.FieldDeletedAt), q.now)),
		sql.Or(sql.EQ(c.C(customerdb.FieldKey), q.subject), sql.Exists(mapping)),
	))
}

func (q ingestQuery) featuresByMeter() *sql.Selector {
	pg := sql.Dialect(dialect.Postgres)
	f := pg.Table(featuredb.Table).As("f")
	m := pg.Table(meterdb.Table).As("m")

	return pg.Select(f.C(featuredb.FieldID), f.C(featuredb.FieldKey)).From(f).Join(m).OnP(sql.And(
		sql.ColumnsEQ(m.C(meterdb.FieldNamespace), f.C(featuredb.FieldNamespace)),
		sql.ColumnsEQ(m.C(meterdb.FieldID), f.C(featuredb.FieldMeterID)),
	)).Where(sql.And(
		sql.EQ(f.C(featuredb.FieldNamespace), q.namespace),
		sql.Or(
			sql.P(func(b *sql.Builder) {
				b.Ident(m.C(meterdb.FieldKey)).WriteString(" = ANY(").Arg(pq.Array(q.meterSlugs)).WriteString("::text[])")
			}),
			sql.P(func(b *sql.Builder) {
				b.Ident(m.C(meterdb.FieldEventType)).WriteString(" = ANY(").Arg(pq.Array(q.eventTypes)).WriteString("::text[])")
			}),
		),
	))
}

// Do not compare StoredAt to previous source timestamps. Late events can change
// an earlier balance. The by-ID calculator also includes usage after active_to.
func (q ingestQuery) meteredEntitlements(d *sql.Selector) *sql.Selector {
	pg := sql.Dialect(dialect.Postgres)
	e := pg.Table(entitlementdb.Table).As("e")
	cs := pg.Table("customers_by_subject").As("cs")
	f := pg.Table("features_by_meter").As("f")
	times := q.usageTimes().Where(sql.And(
		sql.P(func(b *sql.Builder) {
			b.Ident(pg.Table("t").C("at")).WriteOp(sql.OpGTE).Join(coalesce(e.C(entitlementdb.FieldMeasureUsageFrom), e.C(entitlementdb.FieldActiveFrom), e.C(entitlementdb.FieldCreatedAt)))
		}),
		sql.Or(sql.IsNull(e.C(entitlementdb.FieldDeletedAt)), sql.ColumnsLT(pg.Table("t").C("at"), e.C(entitlementdb.FieldDeletedAt))),
	))

	return pg.Select(e.C(entitlementdb.FieldID)).From(e).
		Join(cs).On(cs.C(customerdb.FieldID), e.C(entitlementdb.FieldCustomerID)).
		Join(f).On(f.C(featuredb.FieldID), e.C(entitlementdb.FieldFeatureID)).
		Where(sql.And(
			sql.ColumnsEQ(e.C(entitlementdb.FieldNamespace), d.C(demanddb.FieldNamespace)),
			sql.ColumnsEQ(e.C(entitlementdb.FieldID), d.C(demanddb.FieldOwnerID)),
			sql.EQ(e.C(entitlementdb.FieldEntitlementType), entitlement.EntitlementTypeMetered),
			sql.Exists(times),
		))
}

func (q ingestQuery) creditCharges(d *sql.Selector) *sql.Selector {
	pg := sql.Dialect(dialect.Postgres)
	c := pg.Table(chargedb.Table).As("c")
	o := pg.Table(overridedb.Table).As("o")
	cs := pg.Table("customers_by_subject").As("cs")
	f := pg.Table("features_by_meter").As("f")
	from := coalesce(o.C(overridedb.FieldServicePeriodFrom), c.C(chargedb.FieldServicePeriodFrom))
	to := coalesce(o.C(overridedb.FieldServicePeriodTo), c.C(chargedb.FieldServicePeriodTo))
	times := q.usageTimes().Where(sql.And(
		sql.P(func(b *sql.Builder) { b.Ident(pg.Table("t").C("at")).WriteOp(sql.OpGTE).Join(from) }),
		sql.P(func(b *sql.Builder) { b.Ident(pg.Table("t").C("at")).WriteOp(sql.OpLT).Join(to) }),
	))

	return pg.Select(c.C(chargedb.FieldID)).From(c).
		Join(cs).On(cs.C(customerdb.FieldID), c.C(chargedb.FieldCustomerID)).
		Join(f).OnP(chargeFeatureMatch(c, f)).
		LeftJoin(o).OnP(sql.And(
		sql.ColumnsEQ(o.C(overridedb.FieldNamespace), c.C(chargedb.FieldNamespace)),
		sql.ColumnsEQ(o.C(overridedb.FieldChargeID), c.C(chargedb.FieldID)),
	)).Where(sql.And(
		sql.ColumnsEQ(c.C(chargedb.FieldNamespace), d.C(demanddb.FieldNamespace)),
		sql.ColumnsEQ(c.C(chargedb.FieldCustomerID), d.C(demanddb.FieldOwnerID)),
		sql.IsNull(c.C(chargedb.FieldDeletedAt)),
		sql.NotIn(c.C(chargedb.FieldStatus), usagebased.StatusFinal, usagebased.StatusDeleted),
		sql.In(c.C(chargedb.FieldSettlementMode), productcatalog.CreditOnlySettlementMode, productcatalog.CreditThenInvoiceSettlementMode),
		sql.P(func(b *sql.Builder) { b.Join(from).WriteOp(sql.OpLTE).Arg(q.now) }),
		sql.Exists(times),
		creditChargeScope(d.C(demanddb.FieldKey), c),
	))
}

func (q ingestQuery) usageTimes() *sql.Selector {
	timestamps := make([]string, 0, len(q.timestamps))
	for _, timestamp := range q.timestamps {
		timestamps = append(timestamps, timestamp.UTC().Format(time.RFC3339Nano))
	}
	pg := sql.Dialect(dialect.Postgres)

	return pg.Select(pg.Table("t").C("at")).FromExpr(sql.ExprFunc(func(b *sql.Builder) {
		b.WriteString("unnest(").Arg(pq.Array(timestamps)).WriteString("::timestamptz[]) AS ").Ident("t").WriteString("(").Ident("at").WriteString(")")
	}))
}

func coalesce(columns ...string) sql.Querier {
	return sql.ExprFunc(func(b *sql.Builder) {
		b.WriteString("COALESCE").Wrap(func(b *sql.Builder) { b.IdentComma(columns...) })
	})
}
