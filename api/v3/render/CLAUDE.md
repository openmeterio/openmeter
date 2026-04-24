# render

<!-- archie:ai-start -->

> Provides thin generic helpers RenderJSON and RenderYAML to write typed response bodies with consistent Content-Type and status code handling for v3 handlers. Primary constraint: all v3 JSON/YAML response writes must go through these helpers to ensure header ordering (Content-Type set before WriteHeader).

## Patterns

**RenderJSON for all v3 JSON responses** — Call render.RenderJSON(w, payload, opts...) to write a JSON response. The helper sets Content-Type to application/json if not already set, calls WriteHeader with the configured status, then writes json.Marshal output. (`return render.RenderJSON(w, resp, render.WithStatus(http.StatusCreated))`)
**Option pattern for status and headers** — Use render.WithStatus(code), render.WithContentType(ct), and render.WithHeader(k,v) to configure the response before writing. Options are applied in order and headers must be set before WriteHeader is called. (`render.RenderJSON(w, errBody, render.WithContentType(apierrors.ContentTypeProblemValue), render.WithStatus(http.StatusBadRequest))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `render.go` | Sole file. Exports RenderJSON[O], RenderYAML[O], WithStatus, WithContentType, WithHeader. Uses github.com/invopop/yaml for YAML serialisation. | WriteHeader is called after all Option functions run — do not call w.WriteHeader before RenderJSON, or the status code will be ignored (http.ResponseWriter ignores duplicate WriteHeader calls). |

## Anti-Patterns

- Calling w.WriteHeader then render.RenderJSON — the explicit WriteHeader wins and the status option is silently ignored
- Using json.NewEncoder(w).Encode() directly in v3 handlers — bypasses the Content-Type default and status option
- Setting Content-Type after calling render.RenderJSON — headers sent after WriteHeader are ignored

## Decisions

- **Generic RenderJSON[O] / RenderYAML[O] instead of interface{}-based helpers** — Type parameter prevents accidental passing of error values or nil pointers as the response body; compile-time enforcement of the response shape.

<!-- archie:ai-end -->
