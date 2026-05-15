# httphandler

<!-- archie:ai-start -->

> v1 HTTP handler layer for the subject domain, translating Chi HTTP requests into subject.Service calls and encoding domain types to api.Subject responses. Package name is httpdriver despite the directory being named httphandler.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs per endpoint** — Every endpoint is a separate method returning a typed handler alias. Use NewHandler for path-param-free endpoints and NewHandlerWithArgs when a path param is needed. Never implement http.Handler directly. (`func (h *handler) GetSubject() GetSubjectHandler { return httptransport.NewHandlerWithArgs(decoder, operation, encoder, opts...) }`)
**Per-endpoint type block with Request/Params/Response/Handler aliases** — Each endpoint defines its own *Request struct and a type() block declaring *Params, *Response, and *Handler type aliases immediately above the method. (`type (
	GetSubjectParams   = string
	GetSubjectResponse = api.Subject
	GetSubjectHandler  httptransport.HandlerWithArgs[GetSubjectRequest, GetSubjectResponse, GetSubjectParams]
)`)
**Namespace resolution as first step in every decoder** — All decoders call h.resolveNamespace(ctx) as the first step. On failure it returns HTTP 500. Never skip or hardcode namespace. (`ns, err := h.resolveNamespace(ctx); if err != nil { return GetSubjectRequest{}, err }`)
**httptransport.AppendOptions with WithOperationName** — Every handler passes httptransport.AppendOptions(h.options, httptransport.WithOperationName("camelCaseName")) as options to enable OTel span naming and middleware chaining. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("getSubject"))...`)
**Domain-to-API mapping via FromSubject in mapping.go** — All subject.Subject → api.Subject conversions go through FromSubject in mapping.go. Never inline field mappings inside operation closures. (`return FromSubject(sub), nil`)
**Double-body-read for optional nullable fields in UpsertSubject** — UpsertSubject reads the body twice: once into []api.SubjectUpsert and once into []map[string]interface{} to detect absent vs. null fields, because oapi-codegen cannot distinguish them. (`bodyBytes, _ := io.ReadAll(r.Body); r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)); json.Unmarshal(bodyBytes, &rawPayloads)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler and SubjectHandler interfaces, handler struct with all dependencies (namespaceDecoder, subjectService, entitlementConnector, logger, options), New constructor, and resolveNamespace helper. | New dependencies must be added to both the handler struct and the New constructor signature. entitlementConnector is injected but used only in delete flows — verify before removing. |
| `subject.go` | All four endpoint implementations: GetSubject, ListSubjects, UpsertSubject, DeleteSubject. Upsert create-or-update logic lives here. | Upsert does N individual Create/Update calls in a loop — not batched at DB level. DeleteSubject resolves the subject by key first, then deletes by ID. |
| `mapping.go` | Single FromSubject function mapping domain type to API type. All field mapping logic must stay here. | Metadata is always copied into a new map to avoid sharing a reference with the domain model. |

## Anti-Patterns

- Implementing http.Handler directly instead of using httptransport.NewHandler/NewHandlerWithArgs
- Calling subject.Service methods outside the operation closure (second argument to NewHandler)
- Inlining domain-to-API field mapping instead of using FromSubject in mapping.go
- Skipping httptransport.WithOperationName — breaks OTel tracing and middleware
- Adding business logic (create-or-update decisions) outside the operation closure

## Decisions

- **Package named httpdriver despite directory being named httphandler** — Follows the openmeter convention that HTTP handler packages are named httpdriver; the directory name is a legacy inconsistency.
- **entitlementConnector injected into handler at construction** — DeleteSubject may need to validate or clean up entitlements; injecting at construction avoids circular imports and keeps the handler self-contained.

## Example: Adding a new endpoint (e.g. BulkDeleteSubjects) following the existing pattern

```
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
			if err != nil {
// ...
```

<!-- archie:ai-end -->
