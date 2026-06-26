# httphandler

<!-- archie:ai-start -->

> HTTP transport layer (package httpdriver) for the subject API. Builds httptransport handlers for Get/List/Upsert/Delete subjects, resolves the namespace from context, calls subject.Service, and maps domain subjects to api.Subject.

## Patterns

**httptransport handler triple** — Each operation is a method returning a typed Handler built from NewHandler / NewHandlerWithArgs with (decode, business, encode) functions plus WithOperationName option. (`httptransport.NewHandlerWithArgs(decodeFn, bizFn, commonhttp.JSONResponseEncoderWithStatus[GetSubjectResponse](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("getSubject"))...)`)
**Handler interface aggregation** — handler implements Handler (= SubjectHandler) exposing GetSubject/ListSubjects/UpsertSubject/DeleteSubject; New(...) injects namespaceDecoder, logger, subjectService, entitlementConnector, options. (`type SubjectHandler interface { GetSubject() GetSubjectHandler; ... }`)
**Namespace resolved via decoder** — Every decode func calls h.resolveNamespace(ctx) which reads namespaceDecoder.GetNamespace and returns a 500 HTTPError when absent. (`ns, err := h.resolveNamespace(ctx)`)
**Typed request/response aliases per op** — Each op declares ResponseT, HandlerT, and a RequestT struct (with unexported namespace field); params-bearing ops use HandlerWithArgs with a string ...Params alias. (`type ( GetSubjectParams = string; GetSubjectResponse = api.Subject; GetSubjectHandler httptransport.HandlerWithArgs[GetSubjectRequest, GetSubjectResponse, GetSubjectParams] )`)
**FromSubject is the single domain→api mapper** — mapping.go's FromSubject copies fields and rebuilds Metadata into a *map; all encode paths route domain subjects through it. (`return FromSubject(sub), nil`)
**Double-parse body for null-vs-undefined** — UpsertSubject reads the body twice (struct + raw map) so it can detect field presence and set subject.OptionalNullable{IsSet:true} only for keys actually present in JSON. (`if _, ok := rawPayload["displayName"]; ok { updateInput.DisplayName = subject.OptionalNullable[string]{IsSet: true, Value: payload.DisplayName} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | handler struct, Handler/SubjectHandler interfaces, New constructor, resolveNamespace. | entitlementConnector is injected but unused by the shown ops; resolveNamespace returns 500, not 400, when namespace missing. |
| `subject.go` | All four operation builders + their request/response types. | UpsertSubject manually parses the body twice and validates stripeCustomerId has a cus_ prefix (400 on failure); DeleteSubject first GetByIdOrKey then deletes by resolved Id; both Upsert and Delete return list/nil bodies. |
| `mapping.go` | FromSubject domain→api.Subject converter. | Metadata is rebuilt into a new *map only when non-nil; do not skip FromSubject and hand-build api.Subject. |

## Anti-Patterns

- Calling subjectService directly without resolveNamespace (subject ops are namespace-scoped).
- Hand-constructing api.Subject instead of using FromSubject.
- Decoding the upsert body only once — loses the null-vs-undefined distinction the OptionalNullable update relies on.
- Returning raw service errors without fmt.Errorf wrapping / commonhttp.NewHTTPError for client-facing validation.

## Decisions

- **Body double-parse (struct + raw map) in UpsertSubject** — oapi-codegen cannot model optional body fields (deepmap/oapi-codegen#1039); presence in the raw map is the only way to distinguish erase (null) from leave-as-is (undefined).
- **Upsert performs list-then-create-or-update per key** — There is no native batch upsert; the handler lists existing keys, keys them with lo.KeyBy, then creates or partially updates each (marked TODO to optimize).

## Example: A read handler with namespace resolution and httptransport wiring

```
func (h *handler) GetSubject() GetSubjectHandler {
  return httptransport.NewHandlerWithArgs(
    func(ctx context.Context, r *http.Request, idOrKey GetSubjectParams) (GetSubjectRequest, error) {
      ns, err := h.resolveNamespace(ctx)
      if err != nil { return GetSubjectRequest{}, err }
      return GetSubjectRequest{namespace: ns, subjectIdOrKey: idOrKey}, nil
    },
    func(ctx context.Context, req GetSubjectRequest) (GetSubjectResponse, error) {
      sub, err := h.subjectService.GetByIdOrKey(ctx, req.namespace, req.subjectIdOrKey)
      if err != nil { return GetSubjectResponse{}, fmt.Errorf("failed to get subject: %w", err) }
      return FromSubject(sub), nil
    },
    commonhttp.JSONResponseEncoderWithStatus[GetSubjectResponse](http.StatusOK),
    httptransport.AppendOptions(h.options, httptransport.WithOperationName("getSubject"))...,
  )
// ...
```

<!-- archie:ai-end -->
