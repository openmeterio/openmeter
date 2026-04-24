# httphandler

<!-- archie:ai-start -->

> HTTP handler layer for the subject domain (v1 API), translating Chi HTTP requests into subject.Service calls and encoding domain types to api.Subject responses. Package name is httpdriver despite the directory name being httphandler.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs** — Every endpoint is a separate method returning a typed handler (e.g. GetSubjectHandler). Use httptransport.NewHandler for path-param-free endpoints and NewHandlerWithArgs when a path param (e.g. subjectIdOrKey string) is needed. Never implement http.Handler directly. (`func (h *handler) GetSubject() GetSubjectHandler { return httptransport.NewHandlerWithArgs(decoder, operation, encoder, opts...) }`)
**Handler interface with per-operation return types** — Handler interface declares one method per endpoint returning a typed handler alias (e.g. GetSubjectHandler = httptransport.HandlerWithArgs[...]). SubjectHandler sub-interface groups all subject endpoints; Handler embeds it. Compile-time check: var _ Handler = (*handler)(nil). (`type GetSubjectHandler httptransport.HandlerWithArgs[GetSubjectRequest, GetSubjectResponse, GetSubjectParams]`)
**Namespace resolution via namespaceDecoder** — All handlers call h.resolveNamespace(ctx) as the first step. On failure it returns an HTTP 500 internal error. Never hardcode or skip namespace resolution. (`ns, err := h.resolveNamespace(ctx); if err != nil { return GetSubjectRequest{}, err }`)
**Request/Response type aliases per handler** — Each endpoint defines its own *Request struct plus type aliases for *Params (path param type), *Response (api type), and *Handler. Group them in a type() block immediately above the method. (`type (\n\tGetSubjectParams   = string\n\tGetSubjectResponse = api.Subject\n\tGetSubjectHandler  httptransport.HandlerWithArgs[GetSubjectRequest, GetSubjectResponse, GetSubjectParams]\n)`)
**httptransport.AppendOptions with WithOperationName** — Every handler passes httptransport.AppendOptions(h.options, httptransport.WithOperationName("camelCaseName")) as options. This enables OTel span naming and middleware chaining. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("getSubject"))...`)
**Domain-to-API mapping in mapping.go** — All subject.Subject → api.Subject conversions go through FromSubject in mapping.go. Do not inline field mappings inside operation closures. (`return FromSubject(sub), nil`)
**Double-body-read for optional nullable fields in upsert** — UpsertSubject reads the body twice: once into []api.SubjectUpsert and once into []map[string]interface{} to detect which fields were present (undefined vs. null). This pattern is required when oapi-codegen cannot distinguish absent from null. (`bodyBytes, _ := io.ReadAll(r.Body); r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)); json.Unmarshal(bodyBytes, &rawPayloads)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface, handler struct with all dependencies (namespaceDecoder, subjectService, entitlementConnector, logger, options), New constructor, and resolveNamespace helper. | New dependencies must be added to both the handler struct and the New constructor signature. entitlementConnector is injected but used only in delete flows — check before removing. |
| `subject.go` | All four endpoint implementations: GetSubject, ListSubjects, UpsertSubject, DeleteSubject. Upsert logic (create-or-update) lives here. | Upsert does N+1 individual Create/Update calls inside a loop — not batched at DB level. The entitlementConnector field in handler is declared but not visibly used in this file; verify before any refactor. |
| `mapping.go` | Single FromSubject function mapping domain type to API type. Keep all field mapping logic here. | Metadata is always copied into a new map to avoid sharing a reference with the domain model. |

## Anti-Patterns

- Implementing http.Handler directly instead of using httptransport.NewHandler/NewHandlerWithArgs
- Calling subject.Service methods outside the operation closure (second argument to NewHandler)
- Inlining domain-to-API field mapping instead of using FromSubject in mapping.go
- Skipping httptransport.WithOperationName — breaks OTel tracing and middleware
- Adding business logic (create-or-update decisions) outside the operation closure

## Decisions

- **Package named httpdriver (not httphandler) despite directory name httphandler** — Follows the openmeter convention where HTTP handler packages are named httpdriver; the directory name is a legacy inconsistency.
- **entitlementConnector injected into handler** — DeleteSubject may need to validate or clean up entitlements before removing a subject; injecting it at construction avoids circular imports and keeps the handler self-contained.

## Example: Adding a new endpoint (e.g. BulkDeleteSubjects) following the existing pattern

```
// In subject.go
type (
	BulkDeleteSubjectsResponse = interface{}
	BulkDeleteSubjectsHandler  httptransport.Handler[BulkDeleteSubjectsRequest, BulkDeleteSubjectsResponse]
)

type BulkDeleteSubjectsRequest struct {
	namespace string
	ids       []string
}

func (h *handler) BulkDeleteSubjects() BulkDeleteSubjectsHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (BulkDeleteSubjectsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
// ...
```

<!-- archie:ai-end -->
