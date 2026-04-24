# templates

<!-- archie:ai-start -->

> A patch file that customises the oapi-codegen Chi middleware template to route deepObject 'filter' query parameters through a custom filters.Parse parser instead of the default runtime.BindQueryParameterWithOptions. This ensures union filter types are correctly parsed during code generation.

## Patterns

**Patch targets ParamName=filter + Style=deepObject exclusively** — The template patch adds a conditional branch: only parameters named 'filter' with style 'deepObject' use the custom path. All other parameters continue using the standard oapi-codegen runtime binder. (`{{- if and (eq .ParamName "filter") (eq .Style "deepObject") }}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `chi-middleware.tmpl.patch` | Applied during make gen-api to override oapi-codegen's Chi middleware template for filter parameter handling. | This patch must be re-applied after any oapi-codegen version upgrade. If filter params regress to standard binding, union filter types will silently break. The custom parser is filters.Parse from an imported package — ensure that import is present in the generated middleware. |

## Anti-Patterns

- Editing the generated chi-middleware output directly instead of maintaining this patch
- Widening the patch condition beyond ParamName=filter — standard deepObject params must use runtime.BindQueryParameterWithOptions

## Decisions

- **Custom parser injected via template patch rather than post-generation sed** — Template patches survive oapi-codegen regeneration as long as the template structure is stable; post-generation edits are wiped on every make gen-api run.

<!-- archie:ai-end -->
