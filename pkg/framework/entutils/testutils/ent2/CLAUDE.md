# ent2

<!-- archie:ai-start -->

> Ent codegen bootstrap for the ent2 test fixture database — a minimal second Ent client used exclusively to test multi-database transaction isolation in entutils. Intentionally stripped-down: only the entexpose extension, no cursor or pagination helpers.

## Patterns

**Ignored-main codegen driver** — entc.go carries //go:build ignore and is invoked only via `go generate` through generate.go, never imported at runtime. (`//go:build ignore
func main() { entc.Generate("./schema", ...) }`)
**Full feature set matching production** — FeatureVersionedMigration, FeatureLock, FeatureUpsert, FeatureExecQuery must be present — identical to production and ent1 — to keep test-generated code consistent. (`Features: []gen.Feature{gen.FeatureVersionedMigration, gen.FeatureLock, gen.FeatureUpsert, gen.FeatureExecQuery}`)
**entexpose only — no cursor/paginate** — ent2 intentionally registers only entexpose. Adding entcursor or entpaginate blurs ent2's role as a lightweight fixture and generates unused code. (`entc.Extensions(entexpose.New())`)
**Package path scoped under testutils/ent2/db** — Target and Package must remain under pkg/framework/entutils/testutils/ent2/db to avoid overwriting ent1 or production generated code. (`Package: "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent2/db"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entc.go` | Codegen driver for the ent2 fixture; excluded from normal builds via build tag. | Adding entcursor or entpaginate here would make ent2 generate unnecessary code and blur the distinction from ent1. |
| `generate.go` | //go:generate hook that runs entc.go for the ent2 fixture. | Package name must remain ent2; changing it breaks go generate resolution. |

## Anti-Patterns

- Adding entcursor or entpaginate extensions — ent2 is intentionally minimal compared to ent1
- Pointing Target or Package at production db paths or ent1 paths — overwrites generated code
- Importing domain packages (openmeter/billing, openmeter/customer, etc.) into this fixture
- Removing the standard four Feature flags — diverges from production codegen and ent1
- Adding relations, edges, or domain fields to the schema/ child — this is a minimal test-only fixture

## Decisions

- **ent2 registers only entexpose, not the full extension set used by ent1.** — ent2 exists solely to provide a second independent Ent client for multi-DB transaction isolation tests; cursor and pagination helpers are unnecessary for this role.
- **entc.go uses //go:build ignore to exclude it from normal compilation.** — Keeps the codegen driver out of the test binary while allowing `go run entc.go` via go generate without a separate tool binary.

## Example: Full entc.Generate call for ent2 with correct feature flags and minimal extension (entexpose only)

```
//go:build ignore

package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entexpose"
)

func main() {
	err := entc.Generate("./schema",
		&gen.Config{
// ...
```

<!-- archie:ai-end -->
