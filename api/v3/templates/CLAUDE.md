# templates

<!-- archie:ai-start -->

> A patch file that customises the oapi-codegen Chi middleware template so deepObject 'filter' query parameters route through the custom filters.Parse parser instead of the default runtime binder, ensuring union filter types parse correctly in generated code.

## Patterns

**Patch targets ParamName=filter + Style=deepObject exclusively** — The patch adds a conditional branch: only parameters named 'filter' with style 'deepObject' use filters.Parse; all other parameters keep the standard oapi-codegen runtime binder. (`{{- if and (eq .ParamName "filter") (eq .Style "deepObject") }}
err = filters.Parse(r.URL.Query(), &params.{{.GoName}})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `chi-middleware.tmpl.patch` | Applied during make gen-api to override oapi-codegen's Chi middleware template for filter parameter handling. | Re-apply after any oapi-codegen version upgrade. If filter params regress to standard binding, union filter types silently break. Ensure the filters.Parse import is present in the generated middleware. |

## Anti-Patterns

- Editing the generated chi-middleware output directly instead of maintaining this patch
- Widening the patch condition beyond ParamName=filter — standard deepObject params must use runtime.BindQueryParameterWithOptions

## Decisions

- **Custom parser injected via template patch rather than post-generation sed** — Template patches survive oapi-codegen regeneration as long as the template structure is stable; post-generation edits are wiped on every make gen-api run.

<!-- archie:ai-end -->
