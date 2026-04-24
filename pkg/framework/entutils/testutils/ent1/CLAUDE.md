# ent1

<!-- archie:ai-start -->

> Ent codegen bootstrap for the ent1 test fixture database. Contains entc.go (the ignored main that drives entgo.io/ent/entc.Generate) and generate.go (the //go:generate directive). Actual schema lives in the schema/ child; generated DB client lands in db/.

## Patterns

**Ignored-main codegen driver** — entc.go carries //go:build ignore so it is excluded from normal builds. It is invoked only via go generate through generate.go. (`//go:build ignore
func main() { entc.Generate(...) }`)
**Feature set mirrors production** — entc.Generate must request FeatureVersionedMigration, FeatureLock, FeatureUpsert, FeatureExecQuery — same as the production schema generator — so test-generated code exercises the same capabilities. (`Features: []gen.Feature{gen.FeatureVersionedMigration, gen.FeatureLock, gen.FeatureUpsert, gen.FeatureExecQuery}`)
**Extensions: entcursor + entexpose + entpaginate** — ent1 registers all three framework extensions (entcursor, entexpose, entpaginate) so the generated client includes cursor helpers, expose helpers, and pagination helpers used in production adapters. (`entc.Extensions(entcursor.New(), entexpose.New(), entpaginate.New())`)
**Package path under testutils** — Target and Package must stay under pkg/framework/entutils/testutils/ent1/db to avoid colliding with production entdb or ent2 packages. (`Package: "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entc.go` | Single-use codegen driver; invoked by go generate, never imported. | Diverging the Feature list or Extensions from what production schemas use — test helpers must match production codegen capabilities. |
| `generate.go` | Contains the //go:generate directive that runs entc.go. | Changing the package name away from ent1 breaks go generate resolution. |

## Anti-Patterns

- Removing FeatureVersionedMigration, FeatureLock, FeatureUpsert, or FeatureExecQuery — diverges test-generated client from production capabilities
- Dropping entcursor or entpaginate extensions — leaves test client missing helpers that transaction tests depend on
- Pointing Target/Package at production db paths — would overwrite generated production code
- Adding production business logic or domain imports to entc.go

## Decisions

- **All three framework extensions (cursor, expose, paginate) are included for ent1 but only entexpose for ent2.** — ent1 is the primary transaction/cursor test fixture and must exercise the full extension stack; ent2 is a minimal second-database fixture only needing expose.

<!-- archie:ai-end -->
