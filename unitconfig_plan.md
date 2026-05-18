# UnitConfig roadmap

Step 0 — v3 read-side translation of v1 dynamic and package prices into
`UnitPrice + UnitConfig` — has shipped. Each phase below is independently
shippable and assumes the previous ones are landed.

## Phase 1 — Domain + persistence (no behavior change)

**Goal:** UnitConfig becomes a first-class thing in the codebase without
changing what users see.

- `openmeter/productcatalog/unitconfig.go`: `UnitConfig`,
  `UnitConfigOperation`, `UnitConfigRoundingMode`, `Validate()`, `Clone()`,
  `Equal()`.
- Add `UnitConfig *UnitConfig` to `RateCardMeta`.
- Ent schema: store as a JSON column on the rate-card row (1:1, simple,
  matches API shape).
- Migration via `atlas migrate --env local diff add_unit_config_to_ratecards`.
- Keep v3 read translation from step 0 intact — it still synthesizes
  UnitConfig from v1 Dynamic/Package on the fly. Empty `UnitConfig` column
  for those rows.

**Gate:** domain type round-trips through DB; nothing else changes.

**Status — shipped on `feat/unitconfig-poc`.** Domain type added with
`Validate`/`Clone`/`Equal`; `UnitConfig *UnitConfig` threaded through
`RateCardMeta` (Clone/Equal/Validate updated); `unit_config` JSON column
on the shared `RateCard` Ent mixin → present on both `plan_rate_cards`
and `addon_rate_cards`; adapter mappings updated for read + write in
`plan/adapter/mapping.go` and `addon/adapter/mapping.go`; bulk-create
builders extended in `plan/adapter/phase.go` and `addon/adapter/addon.go`.
Migration `20260514145018_add_unit_config_to_ratecards`. Round-trip gated
by `TestPostgresAdapter/Plan/Create` (fixture now includes a non-nil
UnitConfig). v3 read translation untouched; reject-on-write check stays.

**Gotcha for future phases.** The bulk-create builder is a `q.Set*` chain
that's separate from the entity-struct construction. Adding a new
RateCard-mixin field requires updating BOTH places — the entity struct
in `mapping.go` AND the `Set*` chain in the bulk-create builder.
Otherwise the field silently drops on write. The duplicative pattern is
load-bearing, not vestigial.

## Phase 2 — v3 authoring

**Goal:** v3 clients can create/update plans with `UnitConfig`.

- Flip visibility on `unit_config` from `Lifecycle.Read` to full
  read/create/update in `ratecard.tsp`.
- `FromAPIBillingRateCard` parses it.
- Validation: `UnitConfig` is only valid when the price is unit / graduated
  / volume. Reject on flat/free, and reject on dynamic/package (the v1-only
  types). Decide whether `conversion_factor > 0` is enforced server-side.
- Equivalence rule: a v3-authored `UnitPrice + UnitConfig{multiply, m}` is
  *not* equal to a v1-authored `DynamicPrice{m}` for editing/diff purposes
  — the read translation produces lookalike output but they're different
  rows in storage. Decision needed: do we collapse them at write time or
  keep them distinct?

**Gate:** v3 round-trip works; v1 plans untouched; charges/billing still
rate on the underlying price (UnitConfig still inert).

**Status — shipped on `feat/unitconfig-poc`.** TypeSpec `unit_config`
flipped to full read/create/update (`ratecard.tsp:66`); regenerated
`api/v3/api.gen.go` + `api/v3/openapi.yaml`. `FromAPIBillingUnitConfig`
/ `ToAPIBillingUnitConfig` helpers added in
`api/v3/handlers/plans/convert.go`; `FromAPIBillingRateCard` parses
UnitConfig into `meta.UnitConfig` instead of rejecting. Read path
prefers persisted `meta.UnitConfig` over v1 synthesis;
`ToAPIBillingRateCardUnitConfig` remains as the v1 Dynamic/Package
synthesis fallback. Semantic validation added in `RateCardMeta.Validate()`
(`ratecard.go:261`): UnitConfig is only accepted with `UnitPriceType` /
`TieredPriceType`; rejected on flat / free / dynamic / package via new
sentinel `ErrRateCardUnitConfigRequiresUsageBasedPrice` (`errors.go`).
Tests cover parse (multiply, divide+rounding+display, invalid decimal),
domain validate (unit+UC and tiered+UC accept, flat+UC reject), and
read-path verbatim (persisted wins over synthesis).

**Deferred.** Step 5 (v1 list refusal for v3-only shapes — tiered +
UnitConfig) is not implemented. Once a v3-authored tiered+UnitConfig
plan is persisted, v1 list still serializes the underlying tiered price
without the UnitConfig context. Decision was to ship Phase 2 without it;
revisit before any production use that mixes v1 list consumers with v3
authoring. See Decision #1.

**Equivalence policy as enforced.** Decision #1 ("replace, not collapse")
is enforced by construction: the domain validator forbids
`Dynamic + UnitConfig` and `Package + UnitConfig`, so the only way to
persist a UnitConfig is alongside a Unit or Tiered price. There is no
write-time collapse logic and no defensive read fallback — trust
validation. Gotcha for Phase 3: when the rating engine starts consuming
UnitConfig, the v1 Dynamic/Package synthesis path in
`ToAPIBillingRateCardUnitConfig` is read-only display — the *stored*
price is still v1 Dynamic/Package, not Unit+UnitConfig, so rating must
keep handling those types natively or back-project them at the rating
layer (see Phase 3 bullet 4).

## Phase 3 — Charges + rating engine

**Goal:** UnitConfig actually transforms metered quantities at billing time.

- Charges: when realizing usage, apply `operation × conversion_factor`,
  then rounding/precision, before producing the billable quantity.
- Rating: line amount uses converted quantity × unit price.
- Invoice line: emit `InvoiceUsageQuantityDetail` snapshot (raw /
  converted / invoiced / display_unit / applied_unit_config) per
  `unitconfig.tsp:105`.
- Backfill the v1 read-path equivalence at the rating layer too: a stored
  `DynamicPrice` should rate identically whether read as v1 or as the
  synthesized v3 form.

**Gate:** invoices for plans with UnitConfig produce the same totals as
the equivalent v1 dynamic/package plan.

**Status — partially shipped on `feat/unitconfig-poc`.** Sub-steps 3a–3c
landed; 3d (API surface) and 3e (end-to-end equivalence gate test) are
remaining.

**Step 3a — UnitConfig carried on the charge intent (shipped).**
`usagebased.Intent` (`openmeter/billing/charges/usagebased/charge.go`)
carries `UnitConfig *productcatalog.UnitConfig`; `Intent.Validate()`
runs `UnitConfig.Validate()` inline. The subscription-sync charge
constructor `newUsageBasedChargeIntent`
(`openmeter/billing/worker/subscriptionsync/service/reconciler/patchchargeusagebased.go`)
sets it from `rateCardMeta.UnitConfig`. New JSONB column `unit_config`
on the `charge_usage_based` Ent schema plus migration
`20260518163515_add_unit_config_to_charge_usage_based.{up,down}.sql`;
adapter `Create`/`UpdateOne` paths and the DB→domain mapper updated.
No behavior change yet — UnitConfig is carried but nothing reads it.

**Step 3b — UnitConfig applied at the rating-input path (shipped).**
The two callsites that build `usagebased.RateableIntent` from a snapshot
quantity now apply `intent.UnitConfig.Apply(rawCumulative)` and pass the
*invoiced* cumulative quantity as `MeterValue`:
`openmeter/billing/charges/usagebased/service/rating/totals.go`
(`GetTotalsForUsage`) and `.../delta/engine.go` (`Engine.Rate`). The
domain helper `UnitConfig.Apply(raw) (converted, invoiced Decimal)` is
the single conversion entry point. Tests under
`.../delta/unitconfig_test.go` cover multiply equivalence, divide+ceiling
equivalence, and the load-bearing no-double-billing property when raw
usage moves within a package boundary.

**Step 3c — InvoiceUsageQuantityDetail snapshot persisted (shipped).**
`billing.UsageBasedLine` (`openmeter/billing/stdinvoiceline.go`) gained
`ConvertedQuantity *Decimal` (precise line-period after conversion) and
`AppliedUnitConfig *UnitConfig` (snapshot in effect at billing time).
Linemapper `populateUsageBasedStandardLineFromRun`
(`openmeter/billing/charges/usagebased/service/linemapper.go`) now
accepts the charge intent and writes both fields when UnitConfig is set;
both callers in `lineengine.go` pass `charge.Intent`. The line-period
converted quantity uses **cumulative-then-diff** (`Apply(cumulative_current).converted -
Apply(cumulative_prior).converted`) so ceiling/floor rounding stays
correct across runs. Adapter read (`stdinvoicelinemapper.go`) and write
(`stdinvoicelines.go:upsertUsageBasedConfig` via
`SetNillableConvertedQuantity` / `SetAppliedUnitConfig`) updated.
Existing `Quantity` / `PreLinePeriodQuantity` semantics on the
UsageBasedLine were intentionally left unchanged — they remain
discount-aware, raw-units values. The new fields are the audit-trail
half only.

**Decisions made during Phase 3 implementation:**

1. **Conversion lives at the charge layer, not the rating engine.**
   Rating stays UnitConfig-unaware; it sees whatever billable quantity
   the charge layer hands it. This is what made the v1↔v3 equivalence
   tests fall out cleanly and makes Phase 4 entitlement (precise
   converted, no rounding) easy to fork off the same Apply.
2. **Cumulative-then-diff is load-bearing for ceiling/floor rounding.**
   Applying UnitConfig to per-run diffs would double-bill customers who
   added raw usage inside an existing package boundary. The delta engine
   test `TestUnitConfigDivideCeilingCumulativeNoDoubleBilling` locks
   this in. Any refactor that tries to apply UnitConfig to a delta
   instead of cumulative will break it.
3. **UnitConfig persists on both the charge and the invoice line.** On
   the charge so re-rates read the right config after persistence; on
   the line as a snapshot for invoice display history. The two are
   redundant in normal flow but the line snapshot is the immutable
   audit record.
4. **`Quantity`/`PreLinePeriodQuantity` semantics unchanged in 3c.**
   These remain the discount-aware raw-unit values. If invoice display
   needs them in converted units, that's a follow-on; the
   `InvoiceUsageQuantityDetail` surface (3d) carries the explicit
   raw/converted/invoiced trio for that purpose.
5. **Ent "silent drop" pattern hit twice in Phase 3.** Both
   `BillingInvoiceUsageBasedLineConfig` and `ChargeUsageBased` have
   hand-written `Create()/Update*().Set*()` chains that needed explicit
   `Set<Field>` calls for the new columns. Memory note
   `project-ratecard-ent-builder-pattern` was generalized to cover both
   sites.

**Remaining work — Step 3d (API surface).** Map the persisted line
fields to the v3 `BillingInvoiceUsageQuantityDetail` model on invoice
line responses. TypeSpec already defines the model
(`api/spec/packages/aip/src/productcatalog/unitconfig.tsp:120`); the v3
invoice-line response shape needs a field referencing it, and the
`To...` handler needs to assemble `{raw, converted, invoiced,
display_unit, applied_unit_config}` from
`UsageBasedLine.{MeteredQuantity, ConvertedQuantity, Quantity,
AppliedUnitConfig}`. Confirm where this mapping should live in the v3
invoice handler before editing.

**Remaining work — Step 3e (end-to-end gate test).** The Phase 3 gate
in the plan says: "invoices for plans with UnitConfig produce the same
totals as the equivalent v1 dynamic/package plan." The delta tests
cover this at the rating layer. Step 3e is the integration-level
version: drive two full plan flows (v1 `DynamicPrice` and v3
`UnitPrice + UnitConfig{multiply}`) through subscription sync → charge
realization → invoice line and assert identical money totals. Same for
package vs divide+ceiling. Likely belongs in
`test/billing/` using `SubscriptionMixin`.

**Skipped for Phase 3 by explicit decision.** The plan's fourth Phase 3
bullet ("Backfill the v1 read-path equivalence at the rating layer too:
a stored DynamicPrice should rate identically whether read as v1 or as
the synthesized v3 form") was deliberately NOT unified into a single
code path — v1 Dynamic and Package keep their existing rate functions,
and equivalence is enforced by tests rather than by code unification.
This matches Phase 2 Decision #1 ("replace, not collapse"). See the
delta `unitconfig_test.go` tests.

## Phase 4 — Entitlements

**Goal:** entitlement balance checks see *precise* (unrounded) converted
quantities.

- Apply `operation × conversion_factor` only — skip rounding. Per
  `unitconfig.tsp:31`.
- Touch points: balance worker, entitlement reset/recurrence, and any
  place that compares raw meter output against entitlement quotas.

**Gate:** a customer on a `divide` UnitConfig hits their entitlement cap
on the converted axis, not the raw meter axis.

## Phase 5 — Tiered price semantics

**Open design question, not deferrable past this point.** The TypeSpec
docs (`price.tsp:137,165,191`) state that `up_to_amount` on tier
boundaries is expressed in *converted* (billing) units when UnitConfig is
present. That's a meaningful semantic shift.

- Decide: do we apply UnitConfig *before* tier matching (tiers are in
  billing units) or *after* (tiers are in raw units)? The doc says before.
  But it changes how plan authors think about tiers.
- Document and enforce uniformly across rating, entitlement balance
  display, and invoice rendering.

**Gate:** consistent tier behavior under UnitConfig, with a clear contract.

## Phase 6 — Subscription propagation

**Goal:** active subscriptions carry UnitConfig forward.

- Subscription view / sync: when a subscription is attached to a plan
  version, the subscription's per-rate-card snapshot includes the
  UnitConfig.
- Plan version bump: pinning vs migration policy. Existing behavior is to
  pin to the version attached at; same applies here unless we explicitly
  migrate.
- Subscription editing API: accept UnitConfig on rate-card overrides.

**Gate:** a subscription on a UnitConfig-bearing plan rates correctly
through plan version changes and edits.

## Phase 7 — Migration & deprecation policy

**Decision, not implementation.** Two options:

- **Coexist forever.** Keep v1 dynamic/package authoring; v3 read
  translation is the bridge. Pro: zero risk to existing customers. Con:
  two ways to do the same thing in storage forever.
- **Backfill + deprecate.** One-off migration converts stored
  `Dynamic` → `UnitConfig{multiply}`, `Package` → `UnitConfig{divide, ceiling}`.
  Stop accepting new Dynamic/Package via v1. Pro: single source of truth.
  Con: real migration risk and a v1 SDK breaking change.

Recommend deferring this decision until Phase 3 is shipped — by then we'll
know whether the equivalence is truly lossless in production data.

---

## Cross-cutting open questions to resolve before Phase 2

1. **Equivalence policy** between v1 Dynamic/Package and v3
   UnitConfig+UnitPrice forms (Phase 2 can't ship without an answer).
2. **Where validation lives** — TypeSpec constraints, domain `Validate()`,
   or service layer? Suggested split: shape constraints in TypeSpec,
   semantic constraints (which prices accept UnitConfig) in domain.
3. **`conversion_factor` precision** — what's the max practical scale for
   decimal? Multiplier-style configs likely want more than money-style 2dp.
4. **Display semantics** — does the `display_unit` show on customer-portal
   invoices today? Where else does it surface (PDF, hosted page, webhooks)?

## Decisions (to revisit when implementing)

These are working answers to the four questions above. None are committed
code yet; revisit when actually implementing Phase 2+.

1. **Equivalence: replace, not collapse.** When v3 authoring lands, a
   write with `UnitPrice + UnitConfig` replaces a stored
   `DynamicPrice`/`PackagePrice` row. The v1 list endpoint
   back-projects the simple cases (`UnitPrice + UnitConfig{multiply}` →
   `DynamicPrice`, etc.) and refuses to list v3-only shapes (e.g. tiered
   + UnitConfig). Reason: equivalence detection on every edit is fragile;
   replace gives one canonical storage shape; the v1-read-side surprise
   ("plan no longer listable") is the honest failure mode for plans v1
   can't express.
2. **Validation layering:** TypeSpec for shape (enum membership,
   nullability, decimal range if supported by validator); domain
   `Validate()` for cross-field semantics ("UnitConfig only valid with
   Unit / Tiered"); handler-layer checks only for transitional rules
   ("not yet accepted") that will be deleted in Phase 2. The current
   reject-on-write in `convert.go` is in the transitional bucket.
3. **`conversion_factor` precision:** single field-level cap of ~18 dp
   storage, applied uniformly to multiply and divide. The interesting
   precision/rounding decisions belong in the rating pipeline (Phase 3),
   not on the storage field.
4. **Display:** persist `InvoiceUsageQuantityDetail` (raw / converted /
   invoiced / applied UnitConfig) on the invoice line in Phase 3. Defer
   per-surface rendering (portal, PDF, webhook, email) — each surface
   makes its own call once the underlying data is available.

---

## Resume prompt — pick up at Step 3d

Copy this into a fresh session on `feat/unitconfig-poc` to continue
where the last session left off:

> Continuing UnitConfig POC on branch `feat/unitconfig-poc` in this
> repo. Phases 1, 2, and Phase 3 sub-steps 3a/3b/3c have shipped. Pick
> up at Step 3d (API surface) and then 3e (end-to-end equivalence gate
> test).
>
> Read these to orient — they have everything you need:
>
> 1. `unitconfig_plan.md` — phased roadmap. The Phase 3 "Status —
>    partially shipped" block has shipped sub-steps with file pointers,
>    the load-bearing design decisions (cumulative-then-diff,
>    charge-layer conversion, etc.), and the scope of the remaining 3d
>    + 3e work.
> 2. `unitconfig-eli5.md`, `tiered-pricing-eli5.md`,
>    `prorating-vs-progressive-billing-eli5.md` — conceptual primers.
>    Skim only if needed.
> 3. Internal docs vault at `/Users/roland.spekker/repos/indernal-docs`
>    (the typo "indernal" is intentional — that's the actual path). Key
>    notes: `Primitives/UnitConfig.md`, `level-4-rating-engine.md`. The
>    `obsidian` CLI is on PATH; run commands from inside that directory
>    so the vault is detected.
> 4. Auto-memory entry `project-ratecard-ent-builder-pattern` — the Ent
>    "silent drop" pattern. Now generalized to cover RateCard mixin,
>    `BillingInvoiceUsageBasedLineConfig`, and `ChargeUsageBased` sites.
>    If 3d/3e adds any new Ent fields, this applies again.
> 5. Auto-memory entry `feedback-noisy-commands-subagent` — delegate
>    `make generate` / `gen-api` / `test` / `build` to a subagent or
>    hand me a copyable command; don't run them inline in the main
>    session.
>
> **Step 3d scope (from the plan):** map the persisted
> `UsageBasedLine.{MeteredQuantity, ConvertedQuantity, Quantity,
> AppliedUnitConfig}` to the v3 `BillingInvoiceUsageQuantityDetail`
> model on invoice-line responses. TypeSpec model already exists at
> `api/spec/packages/aip/src/productcatalog/unitconfig.tsp:120`. Surface
> only emits when `AppliedUnitConfig != nil`. Identify the v3 invoice
> handler / convert function and confirm where the mapping lives before
> editing.
>
> **Step 3e scope:** end-to-end gate test under `test/billing/` using
> `SubscriptionMixin`. Drive two plan flows (v1 `DynamicPrice{m}` and
> v3 `UnitPrice(1) + UnitConfig{multiply, m}`) through subscription
> sync → charge realization → invoice line; assert identical money
> totals. Same for `PackagePrice{amount, qty}` vs `UnitPrice(amount) +
> UnitConfig{divide, qty, ceiling}`. This is the plan's stated Phase 3
> gate.
>
> Before any code, confirm 3d scope with me. Start with recon, propose
> a sequencing plan before writing code.
