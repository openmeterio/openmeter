# ent2

<!-- archie:ai-start -->

> Second standalone Ent codegen package (Example2 schema → `db/` client) used alongside ent1 so entutils transaction helpers can be tested across two independent generated clients; deliberately leaner than ent1 (only entexpose, no entcursor/entpaginate).

## Patterns

**Ignored-build codegen driver** — entc.go is a `//go:build ignore` main invoked by `go generate`, mirroring ent1 but with a reduced extension set. (`entc.Generate("./schema", &gen.Config{Target: "./db", Package: ".../ent2/db"}, entc.Extensions(entexpose.New()))`)
**Generate directive in package file** — generate.go holds the sole `//go:generate go run -mod=mod entc.go` directive under `package ent2`. (`package ent2
//go:generate go run -mod=mod entc.go`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entc.go` | Ent codegen entrypoint for the ent2 client: targets `./db`, enables FeatureVersionedMigration/Lock/Upsert/ExecQuery, registers only entexpose.New(). | Intentionally omits entcursor/entpaginate (unlike ent1). Keep it minimal — this client exists to prove entutils works against a second, differently-configured generated client. |
| `generate.go` | `package ent2` declaration plus the `//go:generate` directive that runs entc.go. | Regenerate db/ via codegen; never hand-edit generated output. |

## Anti-Patterns

- Hand-editing the generated ent2/db package instead of editing schema/ and regenerating.
- Adding entcursor/entpaginate here to match ent1 — the divergence is intentional; ent2 is the lean fixture.
- Removing the `//go:build ignore` tag from entc.go and pulling it into the package build.

## Decisions

- **Maintain ent2 as a deliberately smaller second client.** — Two distinct generated clients let entutils verify transaction/mixin behavior is not coupled to a single client's feature set or package path.

<!-- archie:ai-end -->
