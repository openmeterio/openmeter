# externalid

<!-- archie:ai-start -->

> Generic, reusable Ent mixins and value-objects for storing external invoicing/payment/tax app IDs (e.g. Stripe invoice/line IDs) on invoice and line entities. Pairs schema field definitions with typed generic Create/Update/Map helpers.

## Patterns

**Ent mixin + generic typed accessor pattern** — Each entity gets a Mixin (LineMixin/InvoiceMixin) declaring Optional().Nillable() String fields, plus generic Creator/Updater/Getter interfaces and Create/Update/Map functions parameterized over the Ent builder type T. (`func CreateLineExternalID[T LineExternalIDCreator[T]](creator LineExternalIDCreator[T], ids LineExternalIDs) T { ... }`)
**EmptyableToPtr for optional persistence** — Create/Update helpers use lo.EmptyableToPtr to turn empty strings into nil before SetNillable.../SetOrClear..., and lo.FromPtr when mapping back from DB. (`creator.SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(ids.Invoicing))`)
**Create vs Update builder contracts differ** — Creators use SetNillable* (insert), updaters use SetOrClear* (update can null out an existing value); pick the matching interface for the Ent operation. (`SetOrClearInvoicingAppExternalID vs SetNillableInvoicingAppExternalID`)
**Value-object Equal / nil-safe getter** — LineExternalIDs.Equal compares by Invoicing; InvoiceExternalIDs.GetInvoicingOrEmpty is nil-safe on a pointer receiver. (`func (i *InvoiceExternalIDs) GetInvoicingOrEmpty() string { if i == nil { return "" } ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mixin.go` | Ent mixins (LineMixin, InvoiceMixin) and the generic Create/Update/Map helpers + Creator/Updater/Getter interfaces. | Invoice has 3 app IDs (invoicing/payment/tax) but the InvoiceExternalIDCreator/Updater only wire invoicing+payment; tax_app_external_id is schema-only here. Keep Create/Update helper field sets in sync with the struct in model.go. |
| `model.go` | InvoiceExternalIDs and LineExternalIDs value structs with omitempty JSON tags. | GetInvoicingOrEmpty is the only nil-safe accessor; InvoiceExternalIDs has no Equal (only Line does). |

## Anti-Patterns

- Persisting empty strings instead of nil — always route optional IDs through lo.EmptyableToPtr.
- Using a Creator interface for an update path (loses SetOrClear null-out semantics) or vice versa.
- Adding a new external ID field to the struct without updating both the mixin Fields() and the generic Create/Update helpers.

## Decisions

- **External IDs live in generic mixins reused across both invoice and line schemas.** — Multiple Ent entities (invoices, detailed lines) need identical external-app-ID columns; generics over the builder type avoid per-entity boilerplate.

<!-- archie:ai-end -->
