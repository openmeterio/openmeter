# ent2

<!-- archie:ai-start -->

> Ent codegen bootstrap for the ent2 test fixture database. Mirrors ent1's structure (entc.go + generate.go) but registers only the entexpose extension, making it a lightweight second-database fixture for multi-DB transaction tests.

## Patterns

**Ignored-main codegen driver** — entc.go carries //go:build ignore and is invoked only via go generate. (`//go:build ignore
func main() { entc.Generate(...) }`)
**Feature set mirrors production** — FeatureVersionedMigration, FeatureLock, FeatureUpsert, FeatureExecQuery must be present to stay consistent with production. (`Features: []gen.Feature{gen.FeatureVersionedMigration, gen.FeatureLock, gen.FeatureUpsert, gen.FeatureExecQuery}`)
**Extension: entexpose only** — ent2 intentionally omits entcursor and entpaginate — it is a minimal fixture and does not need cursor or pagination helpers. (`entc.Extensions(entexpose.New())`)
**Package path under testutils/ent2** — Target and Package must stay under pkg/framework/entutils/testutils/ent2/db to avoid colliding with ent1 or production packages. (`Package: "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent2/db"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entc.go` | Codegen driver for the ent2 fixture; excluded from normal builds via build tag. | Adding entcursor or entpaginate here would make ent2 generate unnecessary code and blur the distinction from ent1. |
| `generate.go` | //go:generate hook that runs entc.go for the ent2 fixture. | Package name must remain ent2. |

## Anti-Patterns

- Adding entcursor or entpaginate extensions — ent2 is intentionally minimal
- Pointing Target/Package at production or ent1 db paths
- Importing domain packages (openmeter/billing, openmeter/customer, etc.) into this fixture
- Removing the standard four Feature flags — diverges from production codegen

## Decisions

- **ent2 uses only entexpose, not the full extension set used by ent1.** — ent2 exists solely to provide a second independent Ent client for multi-DB transaction tests; it does not need cursor or pagination support.

<!-- archie:ai-end -->
