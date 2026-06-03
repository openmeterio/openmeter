# taxcodes

<!-- archie:ai-start -->

> Full CRUD v3 HTTP handler package for billing tax codes (list/get/create/upsert/delete) plus organization-default tax-code management. Uses Goverter-generated conversions driven by annotated convert.go and a backwards-compatible AppType mapping.

## Patterns

**Goverter variables pattern for conversions** — convert.go declares package-level var function fields with goverter directives; convert.gen.go is generated (DO NOT EDIT) and assigns them in init(). The //go:generate directive at the top of convert.go regenerates via 'make generate'. (`// goverter:variables
var FromAPICreateTaxCodeRequest func(namespace string, req api.CreateTaxCodeRequest) (taxcode.CreateTaxCodeInput, error)`)
**Backwards-compatible AppType mapping** — FromAPIBillingAppType maps API 'external_invoicing' to app.AppTypeCustomInvoicing; ToAPIBillingAppType maps back to 'external_invoicing'. The pair must stay in sync. (`func FromAPIBillingAppType(s api.BillingAppType) app.AppType { if s == "external_invoicing" { return app.AppTypeCustomInvoicing }; return app.AppType(s) }`)
**Nil-to-empty-slice in app mappings** — ToAPIBillingTaxCodeAppMappings returns []api.BillingTaxCodeAppMapping{} (not nil) when the source is nil, preventing a JSON null on the AppMappings field. (`func ToAPIBillingTaxCodeAppMappings(s taxcode.TaxCodeAppMappings) []api.BillingTaxCodeAppMapping { if s == nil { return []api.BillingTaxCodeAppMapping{} }; ... }`)
**Goverter context for namespace/namespacedID injection** — goverter:context + goverter:map ... | NamespaceFromContext / ResolveNamespacedIDFromContext inject the namespace/NamespacedID into generated converters from a context argument. (`// goverter:context namespace
// goverter:map Namespace | NamespaceFromContext
var FromAPICreateTaxCodeRequest func(namespace string, ...) (...)`)
**Upsert semantics for the update endpoint** — update.go accepts api.UpsertTaxCodeRequest (not UpdateTaxCodeRequest) and uses operation name 'upsert-tax-code' — the endpoint is an upsert, not a partial update. (`httptransport.WithOperationName("upsert-tax-code")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `convert.go` | Source-of-truth for Goverter declarations: //go:generate directive, goverter:variables/extend block, var function fields, and hand-written helpers (AppType mapping, nil-safe app mappings, tax-code reference <-> ID). | Editing annotations requires 'make generate'. A new goverter:extend helper must also be registered in the top goverter:extend block. |
| `convert.gen.go` | Goverter-generated implementations assigning the var fields in init(); carries the DO NOT EDIT header. | Never edit directly — regenerate with 'make generate'. Out-of-sync conversions produce silently incorrect mappings. |
| `handler.go` | Handler interface (7 methods incl. organization-default tax codes) and handler struct with resolveNamespace + taxcode.Service only. | No secondary service — taxcode.Service handles all CRUD and organization defaults. |
| `update.go` | Upsert via api.UpsertTaxCodeRequest; passes a models.NamespacedID to FromAPIUpsertTaxCodeRequest. | Operation name must be 'upsert-tax-code'. Using api.UpdateTaxCodeRequest breaks JSON deserialization. |
| `upsert_organization_default_tax_codes.go / get_organization_default_tax_codes.go` | Org-default endpoints using bare httptransport.NewHandler (no path arg); reference IDs convert via the *ReferenceToIDString helpers. | Reference helpers (InvoicingTaxCodeReferenceToIDString etc.) return NewGenericValidationError when id is missing — required-field enforcement. |
| `list.go / create.go / get.go / delete.go` | Standard list/create/get/delete; conversions via ToAPIBillingTaxCode and FromAPICreateTaxCodeRequest. | delete.go defines DeleteTaxCodeRequest as a non-alias struct (taxcode.DeleteTaxCodeInput, no =) and rebuilds the input before the service call. |

## Anti-Patterns

- Editing convert.gen.go directly — regenerate via 'make generate' after changing goverter annotations.
- Breaking the AppType round-trip — 'external_invoicing' must map both ways through FromAPIBillingAppType and ToAPIBillingAppType.
- Returning nil for AppMappings instead of an empty slice — causes a JSON null that breaks the array contract.
- Adding a goverter:extend helper without registering it in the top goverter:extend block.
- Using api.UpdateTaxCodeRequest in update.go instead of api.UpsertTaxCodeRequest.

## Decisions

- **Goverter for tax-code conversions instead of hand-written mappers.** — Tax codes are straightforward field mapping with no discriminated unions; Goverter cuts boilerplate while context injects namespace cleanly.
- **'external_invoicing' <-> 'custom_invoicing' mapping in both directions.** — The domain type was renamed from external_invoicing to custom_invoicing; the API preserves the old name for backwards compatibility.

## Example: Adding a new Goverter conversion variable following the existing pattern

```
// convert.go
// goverter:context namespace
// goverter:map Namespace | NamespaceFromContext
// goverter:map Labels Metadata
// goverter:ignore Annotations
var FromAPINewTaxCodeRequest func(namespace string, req api.NewTaxCodeRequest) (taxcode.NewTaxCodeInput, error)
// run: make generate  (regenerates convert.gen.go)
// in operation file: req, err := FromAPINewTaxCodeRequest(ns, body)
```

<!-- archie:ai-end -->
