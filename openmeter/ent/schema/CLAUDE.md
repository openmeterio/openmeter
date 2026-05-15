# schema

<!-- archie:ai-start -->

> Single source of truth for every PostgreSQL table and view in OpenMeter — 35+ Ent schema structs that Atlas diffs against migration history to produce timestamped SQL migrations. Never edit generated output in openmeter/ent/db/; edit schemas here and run make generate.

## Patterns

**Mixin composition for identity and timestamps** — Every entity must embed the appropriate entutils mixins: IDMixin (ULID PK), NamespaceMixin (multi-tenancy), TimeMixin (created_at/updated_at/deleted_at). Use UniqueResourceMixin or ResourceMixin for entities that also need a human-readable key and metadata. Never hand-roll these columns. (`func (Addon) Mixin() []ent.Mixin { return []ent.Mixin{ entutils.UniqueResourceMixin{} } }`)
**Postgres-typed fields with SchemaType** — Money/decimal columns use field.Other(name, alpacadecimal.Decimal{}) with SchemaType{dialect.Postgres: "numeric"}. ULID FK columns use SchemaType{dialect.Postgres: "char(26)"}. Currency codes use SchemaType{dialect.Postgres: "varchar(3)"}. JSON blobs use SchemaType{dialect.Postgres: "jsonb"}. Omitting SchemaType silently falls back to a less-precise default. (`field.Other("amount", alpacadecimal.Decimal{}).SchemaType(map[string]string{dialect.Postgres: "numeric"})`)
**GoType + ValueScanner for complex domain types stored as jsonb** — Non-scalar domain types stored in jsonb columns must set .GoType(domainType{}) and .ValueScanner(entutils.JSONStringValueScanner[domainType]()) (or a custom ValueScannerFunc). Declare the scanner as a package-level var. Missing ValueScanner causes silent scan errors at runtime. (`var PriceValueScanner = entutils.JSONStringValueScanner[*productcatalog.Price]()
field.String("price").GoType(&productcatalog.Price{}).ValueScanner(PriceValueScanner).SchemaType(map[string]string{dialect.Postgres: "jsonb"})`)
**Partial unique indexes with entsql.IndexWhere for soft-delete safety** — Uniqueness constraints that must ignore deleted rows must use .Annotations(entsql.IndexWhere("deleted_at IS NULL")).Unique(). Do NOT use a plain .Unique() on (namespace, key) columns — it will reject restoring a previously deleted record with the same key. (`index.Fields("namespace", "key", "version").Annotations(entsql.IndexWhere("deleted_at IS NULL")).Unique()`)
**Cascade deletes via entsql.Annotation on edges** — Owned child entities (e.g. phases cascaded from plan, lines from invoice) must set entsql.Annotation{OnDelete: entsql.Cascade} on the parent's edge.To(...). Without it, deleting the parent leaves orphan rows that break FK constraints. (`edge.To("phases", PlanPhase.Type).Annotations(entsql.Annotation{OnDelete: entsql.Cascade})`)
**ent.View for read-only union views; no mixins with indexes or edges** — Cross-table search views (e.g. ChargesSearchV1) implement ent.View and declare their SQL via entsql.ViewFor in Annotations(). Never add mixins with index or edge definitions to a View struct — ent generation panics. Views may not appear in Atlas migrate.Tables; add an explicit SQL migration if atlas reports no diff. (`type ChargesSearchV1 struct { ent.View }
func (v ChargesSearchV1) Annotations() []schema.Annotation { return []schema.Annotation{ entsql.ViewFor(dialect.Postgres, func(s *sql.Selector) { ... }) } }`)
**Shared mixin structs for repeated field groups** — Field groups that recur across multiple entities (tax fields, rate card fields, payment fields, detailed line fields) are extracted into mixin.Schema structs (TaxMixin, RateCard, stddetailedline.Mixin, payment.InvoicedMixin, chargemeta.Mixin). Embed them in Mixin() to avoid duplication. Do NOT copy fields by hand. (`func (PlanRateCard) Mixin() []ent.Mixin { return []ent.Mixin{ entutils.UniqueResourceMixin{}, TaxMixin{} } }
func (PlanRateCard) Fields() []ent.Field { fields := RateCard{}.Fields(); fields = append(fields, ...); return fields }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `billing.go` | Largest schema file: BillingProfile, BillingWorkflowConfig, BillingCustomerOverride, BillingInvoice, BillingInvoiceLine, BillingInvoiceFlatFeeLineConfig, BillingInvoiceUsageBasedLineConfig, BillingInvoiceSplitLineGroup, BillingStandardInvoiceDetailedLine, and discount sub-entities. | BillingInvoiceLine has >30 fields plus edges to all charge sub-types; adding a new field requires a migration and updating the adapter mapping. BillingDiscountsValueScanner and BillingCreditsAppliedValueScanner are declared here and must be kept in sync with their domain types. |
| `charges.go` | Defines the Charge pivot entity (one row per charge regardless of type, holding FK to the type-specific table) and the ChargesSearchV1 union view. | The view's buildChargesSearchV1TableSelector must list chargesSearchV1Columns exactly as they appear in each charge sub-table. Adding a column to chargemeta.Mixin without updating chargesSearchV1Columns breaks the union view. |
| `chargesflatfee.go / chargesusagebased.go / chargescreditpurchase.go` | Per-type charge schemas including their run, payment, invoiced-usage, credit-allocation, and detailed-line sub-entities. | Each charge type must have exactly one edge back to Charge via edge.To("charge", Charge.Type).Unique().Immutable(). Payment and invoiced-usage sub-entities use shared mixins (payment.InvoicedMixin, invoicedusage.Mixin) — add fields to those mixins rather than duplicating. |
| `subscription.go` | Subscription, SubscriptionPhase, SubscriptionItem — the three-level hierarchy that links plan rate cards to entitlements and billing lines. | SubscriptionItem duplicates all RateCard fields instead of using an edge (by design for snapshot immutability). Changes to productcatalog RateCard fields must be mirrored here manually and in the adapter mapping. |
| `taxcode.go` | TaxCode entity plus TaxMixin (adds tax_code_id + tax_behavior to any entity). TaxCode.Edges lists every entity that carries a tax code FK. | When adding TaxMixin to a new entity, also add the corresponding back-edge in TaxCode.Edges() or ent generation will fail with missing ref. |
| `notification.go` | NotificationChannel, NotificationRule, NotificationEvent, NotificationEventDeliveryStatus. Also defines ChannelConfigValueScanner and RuleConfigValueScanner — custom type-switching serializers for polymorphic jsonb config. | Adding a new ChannelType or EventType requires updating both the ValueScanner V (marshal) and S (unmarshal) switch statements; missing case causes an error at runtime, not compile time. |
| `ledger_account.go / ledger_customer_account.go / ledger_entry.go / ledger_transaction.go / ledger_transaction_group.go` | Double-entry ledger schema: accounts, sub-accounts, routing rules, transaction groups, transactions, and entries. | LedgerCustomerAccount intentionally has no edges (no FK to LedgerAccount) to avoid import cycles. These tables must only be written when credits.enabled=true; see wiring guards in app/common. |

## Anti-Patterns

- Editing files under openmeter/ent/db/ — that directory is fully generated; changes are overwritten by make generate.
- Adding a unique index without entsql.IndexWhere("deleted_at IS NULL") on entities that support soft delete — causes constraint violations when re-creating a deleted record.
- Declaring mixin with index or edge definitions inside an ent.View schema — ent generation panics.
- Storing money/decimal as float64 or text instead of field.Other with alpacadecimal.Decimal{} and SchemaType numeric — loses precision for billing arithmetic.
- Manually editing tools/migrate/migrations/ SQL files — Atlas owns that directory and validates the hash chain; manual edits break migrate-check-validate.

## Decisions

- **One Go file per domain area rather than one file per entity** — billing.go, charges*.go, subscription.go group tightly coupled entities so that cross-entity edge definitions (e.g. BillingInvoiceLine <-> Charge) are colocated, reducing the chance of missing back-edges.
- **Shared mixin structs (TaxMixin, RateCard, chargemeta.Mixin) for repeated field groups** — Multiple charge and catalog entities share identical field sets. Mixins give a single edit point so a field addition propagates correctly to all tables via make generate without manually touching each schema.
- **Pivot Charge entity with optional FK fields per sub-type** — The three charge types (flat-fee, usage-based, credit-purchase) have divergent schemas; a single union table would be too sparse. The Charge pivot enables a single FK from BillingInvoiceLine to any charge type, while the ChargesSearchV1 view provides a unified search surface.

## Example: Adding a new entity with soft-deletable unique key, decimal amount, and jsonb config blob

```
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/mydomain"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var MyConfigScanner = entutils.JSONStringValueScanner[mydomain.Config]()
// ...
```

<!-- archie:ai-end -->
