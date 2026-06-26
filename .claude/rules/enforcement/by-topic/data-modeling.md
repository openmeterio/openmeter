# Enforcement: data-modeling (8 rules)

Topic file. Loaded on demand when an agent works on something in the `data-modeling` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Decision Violations (block)

### `dec-mixins-001` — Compose entutils mixins on new Ent entities instead of repeating id/namespace/audit columns

*source: `deep_scan`*

**Why:** ~70 tables share the same multi-tenant, soft-delete, ULID-id, audit, metadata-jsonb shape; mixins enforce it once instead of per-table. ResourceMixin pulls in IDMixin (ULID char(26) PK, unique), NamespaceMixin, MetadataMixin (jsonb), TimeMixin (created/updated/deleted_at) and a unique (namespace,id) index. Repeating id/namespace/timestamp columns per schema, using UUID v4 PKs, or hard-deleting instead of deleted_at breaks the uniform multi-tenant contract.

**Example:**

```
func (Customer) Mixin() []ent.Mixin { return []ent.Mixin{ entutils.ResourceMixin{}, CustomerAddressMixin{} } }
```

### `data-soft-delete-001` — Control-plane rows must be soft-deleted via deleted_at, never hard-deleted

*source: `deep_scan`*

**Why:** Entities soft-delete via deleted_at so historical billing/usage references survive; hard deletes are explicitly out of scope. Almost every Postgres table is namespace-scoped, ULID-id'd, and deleted_at soft-deleted. Customer DeleteCustomer is a SetDeletedAt soft-delete that cascade-deletes only edge-linked children; a hard purge would leave dangling FK-less rows (e.g. LedgerCustomerAccount) undetected.

### `data-namespace-001` — Every Ent query against a namespace-scoped table must filter by namespace

*source: `deep_scan`*

**Why:** OpenMeter is multi-tenant: almost every Postgres table is namespace-scoped with ULID char(26) ids and deleted_at soft-deletes. Queries that omit the namespace predicate leak or mutate data across tenants. Adapter queries pair the namespace and id predicates (e.g. customerdb.Namespace(id.Namespace), customerdb.ID(id.ID)).

## Tradeoff Signals (warn)

### `tr-mixins-002` — Do not rely on UniqueResourceMixin for strict partial uniqueness; ship a custom IndexWhere SQL migration

*source: `deep_scan`*

**Why:** The (namespace,key,deleted_at) UniqueResourceMixin index only approximates partial uniqueness because Ent cannot emit WHERE deleted_at IS NULL without a manual migration, so same-microsecond create/delete/create can collide. Entities needing true WHERE deleted_at IS NULL uniqueness must ship a hand-written IndexWhere SQL migration (as Customer does at openmeter/ent/schema/customer.go:58-62). Deleting that custom migration or trusting the mixin's deleted_at-in-key index for correctness re-opens the resurrect-collision bug.

## Pattern Divergence (inform)

### `data-ledger-fk-001` — Ledger cross-aggregate references are FK-less by design; enforce referential integrity in application code

*source: `deep_scan`*

**Why:** Cross-aggregate references in the Postgres ledger (LedgerCustomerAccount.account_id/customer_id, LedgerSubAccountRoute routing dimensions, LedgerBreakageRecord) are deliberately FK-less and stored as plain char(26)/text columns to avoid import cycles between ledger and customer/account. tax_code stores TaxCode.Key (not a FK). Referential integrity and column alignment are enforced only by application code with no database-level guard, so a hard-deleted account or a renamed TaxCode.Key drifts undetected until a runtime read fails.

### `place-ent-schema-001` — Ent schemas (the DB source of truth) live in openmeter/ent/schema/ as structs embedding ent.Schema with Mixin/Fields/Edges

*source: `deep_scan`*

**Why:** Hand-written Ent schemas are the DB source of truth (<entity>.go embedding ent.Schema with Mixin()/Fields()/Edges()); generated client goes to openmeter/ent/db (DO NOT EDIT). Schemas reuse mixins from pkg/framework/entutils. Confirmed by customer.go.

### `data-customer-fk-001` — CustomerSubjects.subject_key has no FK to Subject; enforce subject existence in application code

*source: `deep_scan`*

**Why:** CustomerSubjects.subject_key has no foreign key to Subject.key because Ent cannot FK non-ID fields (openmeter/ent/schema/customer.go:147), so a customer can be linked to a subject key that does not exist as a Subject row — referential integrity is unenforced at the DB level. Validate the subject's existence in the customer service before linking.

### `data-customer-ilike-001` — Add pg_trgm GIN indexes before exposing case-insensitive contains filters on customer name/email/key

*source: `deep_scan`*

**Why:** The v3 customers list API exposes case-insensitive contains/ocontains filters on name, primary_email and key that compile to leading-wildcard ILIKE and cannot use the plain btree indexes on the customers table, forcing full sequential scans (plus a second seq-scan from Paginate's COUNT(*)). The committed TODO in customer.go warns to add pg_trgm GIN indexes via a custom SQL migration before exposing these ILIKE contains filters.
