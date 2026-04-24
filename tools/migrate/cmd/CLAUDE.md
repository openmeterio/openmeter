# cmd

<!-- archie:ai-start -->

> Organisational parent for CLI entrypoints that operate on the migration toolchain. Currently contains only the viewgen sub-command; acts as the conventional cmd/ layer separating thin CLI wiring from library logic in tools/migrate/.

## Patterns

**Thin entrypoint delegation** — main.go does nothing except parse flags and call into the sibling library package (tools/migrate/viewgen). All logic lives in the library, never in main. (`viewgen.GenerateFile(*schemaPath, *outPath) — one call, no embedded logic`)
**stdlib flag, not Cobra/Viper** — Single-purpose tools under this cmd/ use the stdlib flag package. Cobra/Viper is reserved for multi-command binaries (cmd/server, cmd/jobs). (`flag.String("schema", "./openmeter/ent/schema", ...) / flag.Parse()`)
**exitf helper for error reporting** — Errors are written to stderr via a local exitf helper and os.Exit(1) — no log package, no panic. (`func exitf(format string, args ...any) { fmt.Fprintf(os.Stderr, ...) ; os.Exit(1) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `tools/migrate/cmd/viewgen/main.go` | CLI entrypoint: parses --schema and --out flags, delegates to viewgen.GenerateFile, exits on error. | Do not add SQL generation or Ent schema inspection logic here; keep it in tools/migrate/viewgen. |

## Anti-Patterns

- Adding SQL generation or schema-inspection logic directly in main.go instead of tools/migrate/viewgen
- Importing app/common or any openmeter/ domain package — this binary only needs the ent schema path
- Using Cobra or Viper for a single-purpose tool that only needs two flags
- Adding a second command to this package instead of creating a sibling cmd/<name>/ directory

## Decisions

- **Library logic lives in tools/migrate/viewgen, not in main.go** — Keeps the library testable and importable without invoking os.Exit; the cmd layer is purely a thin CLI shim.
- **stdlib flag instead of Cobra** — A tool with two flags needs no sub-command routing; Cobra adds unnecessary ceremony and import weight.

## Example: Adding a new single-purpose migration tool under tools/migrate/cmd/

```
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openmeterio/openmeter/tools/migrate/mytool"
)

func main() {
	inputPath := flag.String("input", "./tools/migrate/migrations", "migrations dir")
	flag.Parse()
	if err := mytool.Run(*inputPath); err != nil {
		exitf("%v", err)
// ...
```

<!-- archie:ai-end -->
