# ffx

<!-- archie:ai-start -->

> Lightweight feature-flag service (`ffx` = feature-flag-x) keyed by `Feature` string, with three `Service` implementations: context-carried access, static config, and a test composite. Distinct from pkg/featuregate — this resolves per-feature booleans from an `AccessConfig` map carried on the request context.

## Patterns

**Service interface + multiple impls** — All implementations satisfy `Service.IsFeatureEnabled(ctx, Feature) (bool, error)`; pick via constructor: `NewContextService`, `NewStaticService`, `NewTestContextService` (`var _ Service = &staticService{}`)
**Context-carried access config** — `SetAccessOnContext`/`GetAccessFromContext` stash an `AccessConfig` under unexported `accessContextKey`; missing or nil access yields `ErrContextMissing` (`ctx = ffx.SetAccessOnContext(ctx, ffx.AccessConfig{feat: true})`)
**Unknown feature is an error, not false** — Both contextService and staticService return an explicit `feature %s not found` error when the feature key is absent — callers must distinguish disabled from unknown (`if !ok { return false, fmt.Errorf("feature %s not found", feature) }`)
**Test composite falls back static-after-context** — `testContextService` tries the context service first and only consults the static default when the context lookup errored (`v, err := s.contextService.IsFeatureEnabled(...); if err == nil { return v, nil }; return s.staticService....`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `featureflag.go` | Core types: `Feature string`, `Service` interface, `AccessConfig map[Feature]bool` with `Merge` | `AccessConfig.Merge` mutates the receiver in place (writes other's keys into c) and also returns it — aliasing surprises |
| `context.go` | `contextService` (reads AccessConfig from ctx) + `testContextService` composite + ctx get/set helpers | GetAccessFromContext treats both a missing key and a nil AccessConfig as ErrContextMissing |
| `static.go` | `staticService` backed by a fixed AccessConfig, via `NewStaticService` | No fallback — an absent feature key always errors |

## Anti-Patterns

- Treating a not-found feature as disabled instead of handling the returned error
- Relying on AccessConfig.Merge being non-mutating — it overwrites the receiver
- Using NewTestContextService outside tests as the production resolver

## Decisions

- **Separate static vs context-carried services behind one Service interface** — Lets request-scoped access (multi-tenant) override a static default while keeping consumers (subscription, billing) depending only on Service

<!-- archie:ai-end -->
