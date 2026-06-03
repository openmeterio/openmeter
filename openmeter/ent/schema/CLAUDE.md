# schema

<!-- archie:ai-start -->

> Single source of truth for every PostgreSQL table and view in OpenMeter — 35+ hand-written Ent schema structs that Atlas diffs against migration history to produce timestamped SQL. Never edit generated output in openmeter/ent/db/; edit schemas here and run make generate.

## Patterns

**Mixin composition for identity and timestamps** — Every entity embeds the right entutils mixins: IDMixin (ULID PK), NamespaceMixin (multi-tenancy), TimeMixin (created/updated/deleted_at); use UniqueResourceMixin/ResourceMixin for key+metadata. Never hand-roll these columns. (`func (Addon) Mixin() []ent.Mixin { return []ent.Mixin{ entutils.UniqueResourceMixin{} } }`)
**Postgres-typed fields with SchemaType** — Money uses field.Other(name, alpacadecimal.Decimal{}) SchemaType numeric; ULID FKs char(26); currency varchar(3); JSON blobs jsonb. Omitting SchemaType silently falls back to a less-precise default. (`field.Other("amount", alpacadecimal.Decimal{}).SchemaType(map[string]string{dialect.Postgres: "numeric"})`)
**GoType + ValueScanner for complex jsonb types** — Non-scalar domain types stored in jsonb must set .GoType(T{}).ValueScanner(...). Declare the scanner as a package-level var. Missing ValueScanner causes silent scan errors at runtime. (`var PriceValueScanner = entutils.JSONStringValueScanner[*productcatalog.Price]()
field.String("price").GoType(&productcatalog.Price{}).ValueScanner(PriceValueScanner).SchemaType(map[string]string{dialect.Postgres: "jsonb"})`)
**Partial unique indexes for soft-delete safety** — Uniqueness that must ignore deleted rows uses .Annotations(entsql.IndexWhere("deleted_at IS NULL")).Unique(). A plain .Unique() on (namespace, key) rejects restoring a deleted record with the same key. (`index.Fields("namespace", "key", "version").Annotations(entsql.IndexWhere("deleted_at IS NULL")).Unique()`)
**Cascade deletes via entsql.Annotation on edges** — Owned child entities set entsql.Annotation{OnDelete: entsql.Cascade} on the parent's edge.To(...). Without it, deleting the parent leaves orphan rows that break FK constraints. (`edge.To("phases", PlanPhase.Type).Annotations(entsql.Annotation{OnDelete: entsql.Cascade})`)
**ent.View for read-only union views** — Cross-table search views (ChargesSearchV1) implement ent.View and declare SQL via entsql.ViewFor; never add mixins with index/edge definitions to a View (ent generation panics). Views may not appear in Atlas migrate.Tables — add an explicit SQL migration if atlas reports no diff. (`func (v ChargesSearchV1) Annotations() []schema.Annotation { return []schema.Annotation{ entsql.ViewFor(dialect.Postgres, func(s *sql.Selector){ ... }) } }`)
**Shared mixin structs for repeated field groups** — Recurring field sets (TaxMixin, RateCard, stddetailedline.Mixin, payment.InvoicedMixin, chargemeta.Mixin) are extracted into mixin structs; embed them rather than copying fields by hand. (`func (PlanRateCard) Fields() []ent.Field { fields := RateCard{}.Fields(); fields = append(fields, ...); return fields }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `billing.go` | Largest schema: BillingProfile, BillingWorkflowConfig, BillingCustomerOverride, BillingInvoice, BillingInvoiceLine (+flat-fee/usage-based configs), SplitLineGroup, detailed lines, discounts. | BillingInvoiceLine has >30 fields plus edges to all charge sub-types; new fields need a migration and adapter mapping update. BillingDiscountsValueScanner must stay in sync with its domain type. |
| `charges.go` | Charge pivot entity (one row per charge, FK to the type-specific table) and the ChargesSearchV1 union view. | buildChargesSearchV1TableSelector must list chargesSearchV1Columns exactly as they appear in each charge sub-table; adding a chargemeta.Mixin column without updating that list breaks the union view. |
| `chargesflatfee.go / chargesusagebased.go / chargescreditpurchase.go` | Per-type charge schemas with run, payment, invoiced-usage, credit-allocation, detailed-line sub-entities. | Each charge type needs exactly one edge.To("charge", Charge.Type).Unique().Immutable(); add fields to payment.InvoicedMixin / invoicedusage.Mixin rather than duplicating. |
| `subscription.go` | Subscription, SubscriptionPhase, SubscriptionItem three-level hierarchy linking plan rate cards to entitlements and billing lines. | SubscriptionItem duplicates RateCard fields (snapshot immutability by design); productcatalog RateCard field changes must be mirrored here and in the adapter. |
| `taxcode.go` | TaxCode entity plus TaxMixin (tax_code_id + tax_behavior). | Adding TaxMixin to a new entity also requires the back-edge in TaxCode.Edges() or ent generation fails with missing ref. |
| `notification.go` | NotificationChannel/Rule/Event/EventDeliveryStatus plus ChannelConfigValueScanner/RuleConfigValueScanner (type-switching jsonb serializers). | A new ChannelType/EventType needs both V (marshal) and S (unmarshal) switch cases; a missing case errors at runtime, not compile time. |
| `ledger_*.go` | Double-entry ledger schema: accounts, sub-accounts, routes, transaction groups, transactions, entries. | LedgerCustomerAccount intentionally has no edge to LedgerAccount (avoids import cycle); these tables are only written when credits.enabled=true. |

## Anti-Patterns

- Editing files under openmeter/ent/db/ — fully generated; changes are overwritten by make generate.
- Adding a unique index without entsql.IndexWhere("deleted_at IS NULL") on soft-deletable entities — breaks recreating a deleted record.
- Declaring a mixin with index or edge definitions inside an ent.View schema — ent generation panics.
- Storing money/decimal as float64 or text instead of field.Other(alpacadecimal.Decimal{}) SchemaType numeric — loses billing precision.
- Manually editing tools/migrate/migrations/ SQL — Atlas owns it and validates the hash chain.

## Decisions

- **One Go file per domain area rather than per entity** — Grouping tightly-coupled entities (billing.go, charges*.go, subscription.go) colocates cross-entity edge definitions, reducing missing back-edges.
- **Shared mixin structs (TaxMixin, RateCard, chargemeta.Mixin) for repeated field groups** — Gives a single edit point so a field addition propagates to all tables via make generate without touching each schema.
- **Pivot Charge entity with optional per-sub-type FK fields** — Divergent charge schemas would make a single union table sparse; the pivot enables one FK from BillingInvoiceLine to any charge type, with ChargesSearchV1 as a unified search surface.

## Example: New entity with soft-deletable unique key, decimal amount, and jsonb config

```
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var MyConfigScanner = entutils.JSONStringValueScanner[mydomain.Config]()

type MyEntity struct{ ent.Schema }
// ...
```

<!-- archie:ai-end -->
