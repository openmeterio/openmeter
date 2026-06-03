# externalid

<!-- archie:ai-start -->

> Defines external-ID domain structs (InvoiceExternalIDs, LineExternalIDs) and generic Ent mixin/helper functions for reading, writing, and mapping external app IDs (invoicing, payment) on invoice and line entities — bridging domain model and Ent schema without coupling either side.

## Patterns

**Generic Creator/Updater/Getter interfaces per entity** — Each CRUD op has a typed generic interface; adapters call package helpers (CreateLineExternalID, UpdateLineExternalID, MapLineExternalIDFromDB) rather than Set* methods directly. (`func CreateLineExternalID[T LineExternalIDCreator[T]](creator LineExternalIDCreator[T], ids LineExternalIDs) T { return creator.SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(ids.Invoicing)) }`)
**lo.EmptyableToPtr for optional strings** — Empty strings become nil pointers when writing nullable Ent columns; new external-ID fields must follow this. (`creator.SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(ids.Invoicing))`)
**Ent mixin for schema field declarations** — LineMixin and InvoiceMixin declare optional/nillable string fields; new fields go in the correct mixin plus the typed interfaces. (`field.String("invoicing_app_external_id").Optional().Nillable()`)
**Lockstep field addition across four locations** — A new external-ID field requires updating Mixin Fields(), Creator/Updater/Getter interfaces, Create/Update/Map helpers, and the model.go struct (incl. Equal). (`// mixin.go InvoiceMixin + InvoiceExternalIDCreator[T] + CreateInvoiceExternalID + InvoiceExternalIDs struct`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mixin.go` | Ent mixin schema definitions and generic CRUD helpers for invoice and line external IDs. | New field requires updating all four locations: Mixin Fields(), Creator/Updater/Getter interfaces, Create/Update/Map helpers, model.go struct. |
| `model.go` | Pure domain structs InvoiceExternalIDs and LineExternalIDs with nil-safe accessors and LineExternalIDs.Equal. | Update LineExternalIDs.Equal when new fields are added. GetInvoicingOrEmpty uses a nil-receiver guard — replicate for new accessors. |

## Anti-Patterns

- Storing empty strings in nullable external-ID columns — use lo.EmptyableToPtr.
- Calling Set* Ent methods directly in adapters instead of the typed helpers.
- Adding business logic or validation here — it is a model/mixin utility layer.
- Forgetting to update LineExternalIDs.Equal when adding a new string field.

## Decisions

- **Generic interfaces (Creator[T], Updater[T], Getter) rather than concrete Ent types.** — Both invoice and line entities share the same external-ID fields; generics let one helper work across both without duplication.

<!-- archie:ai-end -->
