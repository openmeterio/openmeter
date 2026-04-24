# bloblang

<!-- archie:ai-start -->

> Registers custom Bloblang functions as Benthos/Redpanda Connect plugins via Go init(). Each file in this package adds one or more named functions that become available in Benthos pipeline configs; the package must be blank-imported to activate the registrations.

## Patterns

**init-based plugin registration** — Every custom function is registered inside an init() function using bloblang.RegisterFunctionV2. The package has no exported symbols; activation is achieved by blank-importing the package. (`func init() { err := bloblang.RegisterFunctionV2("resource_quantity", spec, factory); if err != nil { panic(err) } }`)
**PluginSpec + typed params** — Use bloblang.NewPluginSpec() with explicit Param declarations (bloblang.NewStringParam, etc.) so callers get typed argument extraction via args.GetString / args.GetInt64. (`spec := bloblang.NewPluginSpec().Description("...").Param(bloblang.NewStringParam("value").Description("..."))`)
**Lazy-closure return** — The factory passed to RegisterFunctionV2 evaluates arguments once and returns a func() (any, error) closure for repeated invocation — heavy parsing (e.g. resource.ParseQuantity) happens in the factory, not in the closure. (`return func(args *bloblang.ParsedParams) (bloblang.Function, error) { qty := parse(args); return func() (any, error) { return qty.String(), nil }, nil }`)
**Empty-value guard before parsing** — Check for empty string before delegating to external parsers; return a zero value (0, not an error) on empty input to let pipelines handle optional fields gracefully. (`if value == "" { return func() (any, error) { return 0, nil }, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `parse_resource.go` | Registers the 'resource_quantity' Bloblang function that converts Kubernetes resource.Quantity strings to decimal strings. | Panics on RegisterFunctionV2 error — any registration name collision or bad spec causes a startup panic. Ensure function names are globally unique within the binary. |

## Anti-Patterns

- Exporting functions or types from this package — it is a side-effect-only plugin package and should have no exported API.
- Performing heavy I/O or network calls inside the returned closure — parse and validate in the factory, not on every invocation.
- Returning hard errors for empty/zero input — prefer returning a zero value to keep pipelines tolerant of missing fields.
- Registering the same function name in multiple files — init() order is undefined and the second registration will panic.

## Decisions

- **Use init() for auto-registration instead of an explicit Register() call** — Benthos plugin packages are designed for blank-import activation; a central Register() call would require the collector binary to enumerate every plugin, coupling it to this package's internals.
- **Depend on k8s.io/apimachinery resource.ParseQuantity for Kubernetes resource parsing** — Kubernetes resource strings (e.g. '100m', '2Gi') have complex SI/binary suffix rules; reusing the authoritative parser avoids reimplementing that logic.

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
		Param(bloblang.NewStringParam("value").Description("The duration string to parse, e.g. '1h30m'."))

	err := bloblang.RegisterFunctionV2("parse_duration", spec, func(args *bloblang.ParsedParams) (bloblang.Function, error) {
		value, err := args.GetString("value")
// ...
```

<!-- archie:ai-end -->
