# externalid

<!-- archie:ai-start -->

> Defines external-ID model types (InvoiceExternalIDs, LineExternalIDs) and generic Ent mixin/helper functions for reading, writing, and mapping external app IDs (invoicing, payment) on invoice and line Ent entities.

## Patterns

**Typed Creator/Updater/Getter generic interfaces** — Each Ent operation (create, update, map-from-DB) has a corresponding generic interface (LineExternalIDCreator[T], LineExternalIDUpdater[T], LineExternalIDGetter). Adapters must use the provided helpers (CreateLineExternalID, UpdateLineExternalID, MapLineExternalIDFromDB) rather than calling Set* methods directly. (`func CreateLineExternalID[T LineExternalIDCreator[T]](creator LineExternalIDCreator[T], ids LineExternalIDs) T`)
**lo.EmptyableToPtr for optional string fields** — Empty strings are converted to nil pointers using lo.EmptyableToPtr when writing to Ent. New external-ID fields must follow this pattern to avoid storing empty strings in nullable DB columns. (`creator.SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(ids.Invoicing))`)
**Ent mixin for schema fields** — LineMixin and InvoiceMixin embed mixin.Schema and declare optional/nillable string fields. Add new external-ID fields to the correct mixin (LineMixin vs InvoiceMixin), then add corresponding getter/setter to the typed interfaces. (`field.String("invoicing_app_external_id").Optional().Nillable()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mixin.go` | Ent mixin schema definitions and generic helper functions for CRUD operations on external IDs. | Adding a new external-ID field requires updating: (1) the Mixin Fields(), (2) the Creator/Updater/Getter interfaces, (3) the Create/Update/Map helpers, and (4) the model.go struct. |
| `model.go` | Pure domain structs InvoiceExternalIDs and LineExternalIDs with nil-safe accessor methods and equality check. | LineExternalIDs.Equal must be updated when new fields are added. |

## Anti-Patterns

- Storing empty strings in nullable external-ID columns — always use lo.EmptyableToPtr.
- Calling Set* Ent methods directly in adapters instead of using the typed helpers — breaks consistency when fields are added.
- Adding business logic to this package — it is purely a model/mixin utility layer.

## Decisions

- **Generic interfaces (Creator[T], Updater[T], Getter) rather than concrete Ent types.** — Both invoice and line Ent entities share the same external-ID fields; generics allow a single helper to work across both entity types without duplication.

<!-- archie:ai-end -->
