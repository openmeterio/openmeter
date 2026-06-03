# ent1

<!-- archie:ai-start -->

> Ent codegen bootstrap for the ent1 test-fixture database. Drives entc.Generate for the schema/ child and produces a generated client under db/ used exclusively by entutils transaction and cursor tests. Not a production package.

## Patterns

**Ignored-main codegen driver** — entc.go carries //go:build ignore and is invoked only via go generate through generate.go's //go:generate directive — never imported at runtime. (`//go:build ignore
func main() { entc.Generate("./schema", &gen.Config{...}) }`)
**Full feature set matching production** — entc.Generate must request FeatureVersionedMigration, FeatureLock, FeatureUpsert, FeatureExecQuery — identical to the production schema generator — so test-generated code exercises the same capabilities. (`Features: []gen.Feature{gen.FeatureVersionedMigration, gen.FeatureLock, gen.FeatureUpsert, gen.FeatureExecQuery}`)
**Full extension stack (cursor + expose + paginate)** — ent1 registers entcursor, entexpose and entpaginate so the generated client includes every helper used in production adapters. ent2 deliberately omits cursor/paginate — do not collapse ent1 to match it. (`entc.Extensions(entcursor.New(), entexpose.New(), entpaginate.New())`)
**Package path scoped under testutils/ent1/db** — Target and Package must stay under pkg/framework/entutils/testutils/ent1/db to avoid overwriting production entdb or ent2 generated code. (`Package: "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entc.go` | Single-use codegen driver; invoked by go generate, never imported at runtime. | Diverging the Feature list or Extensions from production codegen — test helpers must match production capabilities. |
| `generate.go` | Holds the //go:generate go run -mod=mod entc.go directive for this fixture. | Changing the package name away from ent1 breaks go generate resolution. |

## Anti-Patterns

- Removing FeatureVersionedMigration/FeatureLock/FeatureUpsert/FeatureExecQuery — diverges the test client from production capabilities.
- Dropping entcursor or entpaginate extensions — leaves the test client missing helpers transaction/cursor tests depend on.
- Pointing Target or Package at production db paths or ent2 paths — overwrites generated code.
- Adding production business logic or domain imports (openmeter/billing, openmeter/customer) to entc.go.
- Adding edges or complex fields to the schema/ child — this is a minimal test fixture.

## Decisions

- **ent1 includes all three framework extensions (cursor, expose, paginate) while ent2 includes only entexpose.** — ent1 is the primary transaction/cursor test fixture and must exercise the full extension stack; ent2 is a minimal second-database fixture only needing expose for multi-DB isolation tests.
- **entc.go uses //go:build ignore to exclude it from normal compilation.** — Keeps the codegen driver out of the test binary while allowing go run entc.go via go generate without a separate tool binary.

## Example: Full entc.Generate call for ent1 with correct features and extensions

```
//go:build ignore

package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entcursor"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entexpose"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entpaginate"
)

func main() {
// ...
```

<!-- archie:ai-end -->
