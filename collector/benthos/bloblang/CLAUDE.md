# bloblang

<!-- archie:ai-start -->

> Side-effect-only plugin package that registers custom Bloblang functions as Benthos/Redpanda Connect plugins via Go init(). Must be blank-imported by the collector binary to activate registrations; exports nothing.

## Patterns

**init-based plugin registration** — Each custom function is registered inside init() via bloblang.RegisterFunctionV2; no exported symbols — activation is solely by blank import. (`func init() { err := bloblang.RegisterFunctionV2("resource_quantity", spec, factory); if err != nil { panic(err) } }`)
**PluginSpec with typed params** — Use bloblang.NewPluginSpec() with explicit Param declarations so callers get typed argument extraction via args.GetString/args.GetInt64. (`spec := bloblang.NewPluginSpec().Description("...").Param(bloblang.NewStringParam("value").Description("..."))`)
**Lazy-closure return** — The factory validates/parses arguments once, then returns a func() (any, error) closure for repeated invocation; heavy parsing happens in the factory, not the closure. (`return func() (any, error) { return quantity.AsDec().String(), nil }, nil`)
**Empty-value guard before parsing** — Check for empty string before delegating to external parsers and return a zero value (not an error) so pipelines tolerate optional fields. (`if value == "" { return func() (any, error) { return 0, nil }, nil }`)
**Panic on registration failure** — RegisterFunctionV2 errors are fatal — panic immediately (a name collision or bad spec must not be silently ignored). (`if err != nil { panic(err) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `parse_resource.go` | Registers the 'resource_quantity' Bloblang function converting Kubernetes resource.Quantity strings (e.g. '100m', '2Gi') to decimal via k8s.io/apimachinery resource.ParseQuantity. | Panics on RegisterFunctionV2 error — function names must be globally unique within the binary. |

## Anti-Patterns

- Exporting functions or types — the package is side-effect-only
- Heavy I/O or slow computation inside the returned closure instead of the factory
- Returning hard errors for empty/zero input instead of a zero value
- Registering the same function name in multiple files (init order is undefined; duplicate name panics)
- Skipping the blank import in cmd/benthos-collector — without it nothing registers

## Decisions

- **Use init() auto-registration instead of an explicit Register() call** — Benthos plugins are designed for blank-import activation; a central Register() would couple the launcher to this package's internals.
- **Depend on k8s.io/apimachinery resource.ParseQuantity** — Kubernetes resource strings have complex SI/binary suffix rules; reusing the authoritative parser guarantees correctness.

<!-- archie:ai-end -->
