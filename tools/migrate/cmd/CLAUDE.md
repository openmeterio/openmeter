# cmd

<!-- archie:ai-start -->

> Organisational parent for thin CLI entrypoints that operate on the migration toolchain. Its sole child, viewgen/, is a binary that generates ClickHouse view SQL from the Ent schema; all real logic lives in sibling library packages under tools/migrate/, never here.

## Patterns

**Thin entrypoint delegation** — Each cmd/<name>/main.go only parses flags and calls the corresponding library function — no SQL generation, schema inspection, or file I/O beyond the final delegated call. (`// viewgen/main.go: parse -schema and -out flags, then viewgen.GenerateFile(schema, out)`)
**stdlib flag over Cobra/Viper** — Single-purpose tools with one or two flags use the stdlib flag package, not a command framework. (`schema := flag.String("schema", "openmeter/ent/schema", "path to ent schema"); flag.Parse()`)
**exitf for error reporting** — Errors exit via a local exitf helper (fmt.Fprintf(os.Stderr, ...) + os.Exit(1)), not log.Fatal or panic, for clean CI output. (`func exitf(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...); os.Exit(1) }`)
**One sub-command per directory** — Each cmd/<name>/ directory holds exactly one main.go producing one binary. Add a sibling directory rather than a second command in an existing one. (`// tools/migrate/cmd/viewgen/main.go -> binary: viewgen`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `viewgen/main.go` | Thin CLI wrapper: parses -schema and -out flags and delegates to viewgen.GenerateFile. | Must not import app/common or any openmeter/ domain package — only the schema path and the viewgen library are needed. |

## Anti-Patterns

- Adding SQL generation or schema-inspection logic in main.go instead of the sibling tools/migrate/viewgen library.
- Importing app/common or any openmeter/ domain package from a cmd entrypoint.
- Using Cobra or Viper for a tool that only needs one or two flags.
- Adding a second command to an existing cmd/<name>/ package instead of creating a sibling directory.
- Using log.Fatal or panic instead of the local exitf helper for error exits.

## Decisions

- **Library logic lives in tools/migrate/viewgen, not in main.go.** — Lets view_parity_test.go call viewgen.GenerateSQL directly without shelling out, keeping tests fast and free of binary build dependencies.

<!-- archie:ai-end -->
