# cmd

<!-- archie:ai-start -->

> Organisational parent for thin CLI entrypoints that operate on the migration toolchain. Separates CLI wiring from library logic — all business logic lives in sibling library packages under tools/migrate/; cmd/ sub-directories contain only flag parsing and delegation.

## Patterns

**Thin entrypoint delegation** — main.go in each sub-directory only parses flags and calls the corresponding library function. No SQL generation, schema inspection, or file I/O beyond the final write. (`// main.go: parse -schema and -out flags, call viewgen.GenerateFile(schema, out)`)
**stdlib flag for single-purpose tools** — Uses stdlib flag package, not Cobra or Viper. A single-purpose tool with one or two flags does not need a command framework. (`schema := flag.String("schema", "openmeter/ent/schema", "path to ent schema"); flag.Parse()`)
**exitf helper for error reporting** — Errors are reported via a local exitf helper (fmt.Fprintf(os.Stderr, ...) + os.Exit(1)) rather than log.Fatal or panic, to produce clean error messages for CI. (`func exitf(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...); os.Exit(1) }`)
**One sub-command per directory** — Each cmd/<name>/ directory contains exactly one main.go producing one binary. Never add a second command to an existing cmd/<name>/ directory; create a sibling instead. (`// tools/migrate/cmd/viewgen/main.go -> binary: viewgen`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `tools/migrate/cmd/viewgen/main.go` | Thin CLI wrapper: parses -schema and -out flags, delegates to viewgen.GenerateFile. No business logic. | Must not import app/common or any openmeter/ domain package. Only needs the schema path and the viewgen library. |

## Anti-Patterns

- Adding SQL generation or schema-inspection logic directly in main.go instead of the sibling library package
- Importing app/common or any openmeter/ domain package — this binary only needs the schema path and the library
- Using Cobra or Viper for a single-purpose tool that only needs one or two flags
- Adding a second command to an existing cmd/<name>/ package instead of creating a sibling cmd/<othername>/ directory
- Using log.Fatal or panic instead of a local exitf helper for error exits

## Decisions

- **Library logic lives in tools/migrate/viewgen, not in main.go** — Separating library from CLI entrypoint allows view_parity_test.go to call viewgen.GenerateSQL directly without shelling out, making tests faster and eliminating binary build dependencies in the test path.

<!-- archie:ai-end -->
