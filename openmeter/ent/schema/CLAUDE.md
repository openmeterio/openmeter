# schema

<!-- archie:ai-start -->

> Hand-written Ent entity definitions that are the single source of truth for the PostgreSQL schema. ~35 files (customer, billing, charges*, ledger_*, subscription, entitlement, grant, feature, meter, notification, productcatalog, taxcode, app*) compile via `make generate` into the read-only `openmeter/ent/db` and feed Atlas migration diffs.

## Patterns

**Standard ent.Schema quartet** — Each entity is a struct embedding `ent.Schema` with `Mixin()`, `Fields()`, `Edges()`, `Indexes()` methods. Compose shared columns via entutils mixins rather than redeclaring them. (`func (Customer) Mixin() []ent.Mixin { return []ent.Mixin{entutils.ResourceMixin{}, entutils.CustomerAddressMixin{FieldPrefix: "billing"}, entutils.AnnotationsMixin{}} }`)
**entutils mixins for identity/namespace/time** — Use entutils.IDMixin / NamespaceMixin / TimeMixin / MetadataMixin / AnnotationsMixin / ResourceMixin (or UniqueResourceMixin). Never hand-roll id/namespace/created_at columns. (`Mixin: entutils.IDMixin{}, entutils.NamespaceMixin{}, entutils.TimeMixin{}`)
**ULID FK columns as char(26)** — Every foreign-key string field that references another entity's ULID id sets SchemaType char(26) for Postgres. (`field.String("customer_id").SchemaType(map[string]string{dialect.Postgres: "char(26)"})`)
**Decimals via alpacadecimal numeric** — Monetary/quantity values use field.Other(name, alpacadecimal.Decimal{}) with Postgres SchemaType "numeric"; never field.Float for money. (`field.Other("per_unit_amount", alpacadecimal.Decimal{}).SchemaType(map[string]string{dialect.Postgres: "numeric"})`)
**JSON columns via ValueScanner + jsonb** — Complex domain types are persisted as jsonb either with field.JSON or field.String(...).GoType(...).ValueScanner(...). Package-level scanners are built with entutils.JSONStringValueScanner[T]() or a manual field.ValueScannerFunc (see notification ChannelConfig/RuleConfig, AnnotationsValueScanner). (`field.String("price").GoType(&productcatalog.Price{}).ValueScanner(PriceValueScanner).SchemaType(map[string]string{dialect.Postgres: "jsonb"})`)
**Soft-delete-aware partial unique indexes** — Uniqueness on namespaced business keys is enforced with index.Fields(...).Annotations(entsql.IndexWhere("deleted_at IS NULL")).Unique() so deleted rows don't collide. GIN indexes are declared for jsonb annotation columns. (`index.Fields("namespace", "key", "version").Annotations(entsql.IndexWhere("deleted_at IS NULL")).Unique()`)
**Edge ownership via From/Ref + Field** — FK ownership is declared with edge.From(name, T.Type).Ref("...").Field("fk_col").Unique().Required(); cascade behavior set with entsql.OnDelete(entsql.Cascade/SetNull). Same-ID 1:1 app subtables use StorageKey(edge.Column("id")) instead of Field to avoid ent generation breaking. (`edge.From("customer", Customer.Type).Ref("billing_customer_override").Field("customer_id").Unique().Required().Immutable()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `billing.go` | BillingProfile, BillingWorkflowConfig, BillingCustomerOverride, BillingInvoiceLine (+ flat-fee/usage-based line configs) — the largest schema file; defines package-level BillingDiscountsValueScanner etc. | BillingInvoiceLine carries deprecated fields (line_ids) and a wide edge set to charges; do not remove deprecated fields without a migration. Quantity is optional/nillable because UBP quantity is only known at issue time. |
| `charges.go` | Charge (typed parent) plus ChargesSearchV1 ent.View that UNION ALLs the three charge subtype tables. | ent.View (ChargesSearchV1) does NOT appear in generated migrate.Tables — Atlas diff reports no changes for view DDL; the view needs an explicit SQL migration. The view's Mixin() must stay empty (no edges/indexes) or ent generation panics. |
| `chargesflatfee.go / chargesusagebased.go / chargescreditpurchase.go` | Charge subtype tables and their realization-run/payment/credit-allocation children, composed from chargemeta/payment/creditrealization/invoicedusage/stddetailedline mixins. | ChargeUsageBased pins its table name via Annotations entsql.Annotation{Table: "charge_usage_based"}; renaming structs without the annotation changes the physical table. |
| `customer.go` | Customer + CustomerSubjects join table; central hub with edges to apps, subscriptions, entitlements, billing override, and all three charge subtypes. | Large in-code comment warns that v3 ILIKE filters need pg_trgm GIN indexes (a custom SQL migration) before the customers list handler is exposed — the btree indexes here cannot serve leading-wildcard search. |
| `feature.go` | Feature entity with LLM unit-cost columns and CHECK constraints. | Annotations() declares Postgres CHECK constraints (unit_cost_llm_*_mutual_exclusive); changing the mutually-exclusive column pairs requires updating these checks. |
| `ledger_account.go / ledger_entry.go / ledger_transaction*.go` | Double-entry ledger tables (account, sub-account, route, entry, transaction, transaction_group). | LedgerSubAccountRoute stores denormalized routing values (currency, tax_code as TaxCode.Key string, features text[]) that are NOT FKs. LedgerEntry/Transaction fields are Immutable() — append-only accounting source of truth. |
| `notification.go` | Notification channels/rules with manual field.ValueScannerFunc scanners (ChannelConfigValueScanner, RuleConfigValueScanner). | Uses the verbose field.ValueScannerFunc[T, *sql.NullString]{V:..., S:...} form (also AnnotationsValueScanner) rather than entutils.JSONStringValueScanner — keep nil/!Valid handling in the S func. |
| `ratecard.go / productcatalog.go / planaddon.go / addon.go` | Plan/addon/ratecard product-catalog schema; defines shared productcatalog scanners (PriceValueScanner, DiscountsValueScanner, EntitlementTemplateValueScanner, TaxConfigValueScanner, ProRatingConfigValueScanner). | AddonRateCard.Fields() borrows RateCard{}.Fields() via direct call (commented 'ent/runtime.go bug') and appends — keep that pattern when extending. |

## Anti-Patterns

- Editing generated code under openmeter/ent/db/ — it is regenerated from these schemas (DO NOT EDIT header).
- Adding/changing a field, index, or edge without running `make generate` then `atlas migrate --env local diff <name>` to produce up/down SQL + atlas.sum.
- Using field.Float or default varchar for money/quantity instead of alpacadecimal.Decimal with numeric SchemaType.
- Declaring a plain unique index on a business key without entsql.IndexWhere("deleted_at IS NULL") — soft-deleted rows will collide.
- Adding mixins, edges, or indexes to an ent.View (e.g. ChargesSearchV1) — ent generation panics; views also need hand-written SQL migrations.

## Decisions

- **Ent schema is the DB source of truth, Atlas diffs migrations from it** — Single Go definition drives generated query code (ent/db), migration metadata, and type-safe edges; migrations are reproducible diffs rather than hand-written DDL (except views).
- **Soft delete (deleted_at) + partial indexes everywhere** — Resources are rarely hard-deleted; partial unique indexes let a key be reused after delete while preserving history and audit trails.
- **Same-ID 1:1 subtype tables (App↔AppStripe, Charge↔ChargeFlatFee) joined by StorageKey(id)** — Avoids ent generating conflicting SetAppID/SetXAppID setters; the subtype shares the parent's ULID so the join is trivial.

## Example: A namespaced, soft-deletable entity with a jsonb annotations column, ULID FK, decimal amount, and a partial-unique index

```
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Widget struct{ ent.Schema }
// ...
```

<!-- archie:ai-end -->
