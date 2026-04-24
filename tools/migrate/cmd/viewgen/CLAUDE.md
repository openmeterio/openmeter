# viewgen

<!-- archie:ai-start -->

> Minimal CLI entrypoint that invokes viewgen.GenerateFile to produce ClickHouse view SQL from the Ent schema. It is a thin main package with no business logic — all logic lives in tools/migrate/viewgen.

## Patterns

**Thin entrypoint delegation** — main.go only parses flags and delegates to viewgen.GenerateFile; no SQL generation or schema inspection logic belongs here. (`viewgen.GenerateFile(*schemaPath, *outPath)`)
**flag-based CLI** — Uses stdlib flag package (not Cobra/Viper) for the two flags: --schema (path to ent schema) and --out (output SQL file path). Any new flags must follow the same flag.String pattern. (`schemaPath = flag.String("schema", "./openmeter/ent/schema", "...")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.go` | CLI entry point: parse --schema and --out flags, call viewgen.GenerateFile, exit non-zero on error via exitf. | Do not add logic here; keep it as a one-call wrapper. exitf writes to stderr and calls os.Exit(1) — do not use log.Fatal or panic. |

## Anti-Patterns

- Adding SQL generation logic directly in main.go instead of tools/migrate/viewgen
- Using Cobra/Viper instead of stdlib flag for this single-purpose tool
- Importing app/common or domain packages — this tool only needs the ent schema path and viewgen

## Decisions

- **Separate cmd/viewgen from the viewgen library package** — Keeps the generator logic testable and reusable independently of the CLI invocation surface.

<!-- archie:ai-end -->
