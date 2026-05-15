# viewgen

<!-- archie:ai-start -->

> Thin CLI entrypoint that parses two flags (--schema, --out) and delegates entirely to viewgen.GenerateFile to produce ClickHouse view SQL from the Ent schema. Contains zero business logic — all generation resides in the sibling tools/migrate/viewgen package.

## Patterns

**Thin entrypoint delegation** — main.go calls exactly one library function (viewgen.GenerateFile) after flag parsing. No SQL generation, schema inspection, or domain logic belongs here. (`viewgen.GenerateFile(*schemaPath, *outPath)`)
**stdlib flag for single-purpose tools** — Uses stdlib flag package (not Cobra/Viper) for the two flags --schema and --out. New flags must use flag.String with the same pattern. (`schemaPath = flag.String("schema", "./openmeter/ent/schema", "...")`)
**exitf error reporting** — Errors are reported via the local exitf helper which writes to stderr and calls os.Exit(1). Never use log.Fatal, panic, or fmt.Println for error exit paths. (`exitf("%v", err)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | CLI entry: parse --schema and --out flags, call viewgen.GenerateFile, exit non-zero on error via exitf. | Do not add any SQL or schema logic here. exitf is the only valid error exit — never substitute log.Fatal or panic. |

## Anti-Patterns

- Adding SQL generation or schema inspection logic directly in main.go instead of tools/migrate/viewgen
- Using Cobra or Viper instead of stdlib flag for this single-purpose tool
- Importing app/common or any openmeter domain package — this tool only needs the ent schema path and viewgen
- Using log.Fatal or panic instead of exitf for error exits

## Decisions

- **Separate cmd/viewgen entrypoint from the viewgen library package** — Keeps SQL generation logic testable and reusable independently of the CLI surface; the library can be imported by other tools without pulling in flag/os.Exit side-effects.

## Example: Adding a new flag and passing it to GenerateFile

```
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/openmeterio/openmeter/tools/migrate/viewgen"
)

func main() {
	var (
		schemaPath = flag.String("schema", "./openmeter/ent/schema", "path to the ent schema package")
		outPath    = flag.String("out", viewgen.DefaultOutputPath, "output SQL file path")
	)
// ...
```

<!-- archie:ai-end -->
