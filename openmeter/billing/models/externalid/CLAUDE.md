# externalid

<!-- archie:ai-start -->

> Defines external-ID domain structs (InvoiceExternalIDs, LineExternalIDs) and generic Ent mixin/helper functions for reading, writing, and mapping external app IDs (invoicing, payment) on invoice and line Ent entities. Bridges the domain model and the Ent schema without coupling either side directly.

## Patterns

**Generic Creator/Updater/Getter interfaces per entity type** — Each Ent CRUD operation has a typed generic interface (LineExternalIDCreator[T], LineExternalIDUpdater[T], LineExternalIDGetter). Adapters must call the package helpers (CreateLineExternalID, UpdateLineExternalID, MapLineExternalIDFromDB) rather than calling Set* methods directly. (`func CreateLineExternalID[T LineExternalIDCreator[T]](creator LineExternalIDCreator[T], ids LineExternalIDs) T { return creator.SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(ids.Invoicing)) }`)
**lo.EmptyableToPtr for optional string fields** — Empty strings are converted to nil pointers using lo.EmptyableToPtr when writing to Ent nullable columns. New external-ID fields must follow this pattern to avoid storing empty strings in nullable DB columns. (`creator.SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(ids.Invoicing))`)
**Ent mixin for schema field declarations** — LineMixin and InvoiceMixin embed mixin.Schema and declare optional/nillable string fields. New external-ID fields must be added to the correct mixin, then corresponding getter/setter must be added to the typed interfaces. (`field.String("invoicing_app_external_id").Optional().Nillable()`)
**Lockstep field addition across four locations** — Adding a new external-ID field requires updating: (1) the Mixin Fields(), (2) the Creator/Updater/Getter interfaces, (3) the Create/Update/Map helpers, and (4) the model.go struct (including Equal if present). (`// See mixin.go InvoiceMixin + InvoiceExternalIDCreator[T] + CreateInvoiceExternalID + InvoiceExternalIDs struct`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mixin.go` | Ent mixin schema definitions and all generic helper functions for CRUD operations on external IDs for both invoice and line entities. | Adding a new external-ID field requires updating all four locations: Mixin Fields(), Creator/Updater/Getter interfaces, Create/Update/Map helpers, and model.go struct. |
| `model.go` | Pure domain structs InvoiceExternalIDs and LineExternalIDs with nil-safe accessor methods and equality check (LineExternalIDs.Equal). | LineExternalIDs.Equal must be updated when new fields are added. GetInvoicingOrEmpty uses nil-receiver guard — replicate this pattern for new pointer-receiver accessors. |

## Anti-Patterns

- Storing empty strings in nullable external-ID columns — always use lo.EmptyableToPtr.
- Calling Set* Ent methods directly in adapters instead of using the typed helpers — breaks consistency when fields are added.
- Adding business logic or validation to this package — it is purely a model/mixin utility layer.
- Forgetting to update LineExternalIDs.Equal when adding a new string field to LineExternalIDs.

## Decisions

- **Generic interfaces (Creator[T], Updater[T], Getter) rather than concrete Ent types.** — Both invoice and line Ent entities share the same external-ID fields; generics allow a single helper to work across both entity types without duplication.

<!-- archie:ai-end -->
