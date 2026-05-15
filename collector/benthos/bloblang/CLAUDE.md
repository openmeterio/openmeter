# bloblang

<!-- archie:ai-start -->

> Side-effect-only plugin package that registers custom Bloblang functions as Benthos/Redpanda Connect plugins via Go init(). Must be blank-imported by the collector binary to activate registrations; exports nothing.

## Patterns

**init-based plugin registration** — Every custom function is registered inside an init() function using bloblang.RegisterFunctionV2. No exported symbols — activation is achieved solely by blank-importing the package. (`func init() { err := bloblang.RegisterFunctionV2("resource_quantity", spec, factory); if err != nil { panic(err) } }`)
**PluginSpec with typed params** — Use bloblang.NewPluginSpec() with explicit Param declarations (bloblang.NewStringParam, etc.) so callers get typed argument extraction via args.GetString / args.GetInt64. (`spec := bloblang.NewPluginSpec().Description("...").Param(bloblang.NewStringParam("value").Description("..."))`)
**Lazy-closure return** — The factory passed to RegisterFunctionV2 evaluates and validates arguments once, then returns a func() (any, error) closure for repeated invocation. Heavy parsing happens in the factory, not in the closure. (`return func(args *bloblang.ParsedParams) (bloblang.Function, error) { qty := parse(args); return func() (any, error) { return qty.String(), nil }, nil }`)
**Empty-value guard before parsing** — Check for empty string before delegating to external parsers; return a zero value (not an error) on empty input to let pipelines handle optional fields gracefully. (`if value == "" { return func() (any, error) { return 0, nil }, nil }`)
**Panic on registration failure** — RegisterFunctionV2 errors are treated as fatal — panic immediately. Registration failures indicate name collisions or bad specs and must not be silently ignored. (`if err != nil { panic(err) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `parse_resource.go` | Registers the 'resource_quantity' Bloblang function that converts Kubernetes resource.Quantity strings (e.g. '100m', '2Gi') to decimal strings using k8s.io/apimachinery/pkg/api/resource.ParseQuantity. | Panics on RegisterFunctionV2 error — any registration name collision or bad PluginSpec causes a startup panic. Ensure function names are globally unique within the binary. |

## Anti-Patterns

- Exporting functions or types from this package — it is side-effect-only and must have no exported API.
- Performing heavy I/O, network calls, or slow computation inside the returned func() closure — validate and parse in the factory, not on every invocation.
- Returning hard errors for empty or zero input — prefer returning a zero value to keep pipelines tolerant of missing fields.
- Registering the same function name in multiple files — init() order is undefined and a duplicate name causes a startup panic.
- Skipping the blank import in cmd/benthos-collector — the package registers nothing unless blank-imported; forgetting the import silently omits all custom functions.

## Decisions

- **Use init() for auto-registration instead of an explicit Register() call.** — Benthos plugin packages are designed for blank-import activation; a central Register() call would require the collector binary to enumerate every plugin, coupling it to this package's internals.
- **Depend on k8s.io/apimachinery resource.ParseQuantity for Kubernetes resource string parsing.** — Kubernetes resource strings have complex SI/binary suffix rules; reusing the authoritative parser avoids reimplementing that logic and guarantees correctness for all valid quantity formats.

## Example: Add a new Bloblang function 'parse_duration' that converts a Go duration string to milliseconds

```
package bloblang

import (
	"time"

	"github.com/redpanda-data/benthos/v4/public/bloblang"
)

func init() {
	spec := bloblang.NewPluginSpec().
		Description("Parse a Go duration string and return its value in milliseconds.").
		Param(bloblang.NewStringParam("value").Description("Duration string, e.g. '1h30m'."))

	err := bloblang.RegisterFunctionV2("parse_duration", spec, func(args *bloblang.ParsedParams) (bloblang.Function, error) {
		value, err := args.GetString("value")
// ...
```

<!-- archie:ai-end -->
