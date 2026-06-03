# render

<!-- archie:ai-start -->

> Thin generic helpers RenderJSON and RenderYAML to write typed response bodies with consistent Content-Type and status handling for v3 handlers. Primary constraint: all v3 JSON/YAML writes must go through these helpers so Content-Type is set before WriteHeader.

## Patterns

**RenderJSON for all v3 JSON responses** — render.RenderJSON(w, payload, opts...) sets Content-Type to application/json if unset, calls WriteHeader with the configured status, then writes json.Marshal output. (`return render.RenderJSON(w, resp, render.WithStatus(http.StatusCreated))`)
**Option pattern for status and headers** — Use WithStatus(code), WithContentType(ct), WithHeader(k,v) to configure the response before writing; options apply in order and headers must be set before WriteHeader. (`render.RenderJSON(w, errBody, render.WithContentType(apierrors.ContentTypeProblemValue), render.WithStatus(http.StatusBadRequest))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `render.go` | Sole file. Exports RenderJSON[O], RenderYAML[O], WithStatus, WithContentType, WithHeader; uses github.com/invopop/yaml for YAML. | WriteHeader runs after all Option functions — do not call w.WriteHeader before RenderJSON or the status is ignored (duplicate WriteHeader is dropped). |

## Anti-Patterns

- Calling w.WriteHeader then render.RenderJSON — the explicit WriteHeader wins and the status option is ignored
- Using json.NewEncoder(w).Encode() directly in v3 handlers — bypasses the Content-Type default and status option
- Setting Content-Type after calling render.RenderJSON — headers after WriteHeader are ignored

## Decisions

- **Generic RenderJSON[O] / RenderYAML[O] instead of interface{}-based helpers** — Type parameter prevents accidentally passing error values or nil pointers as the body; compile-time enforcement of response shape.

<!-- archie:ai-end -->
