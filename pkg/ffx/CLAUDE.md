# ffx

<!-- archie:ai-start -->

> Minimal feature-flag service with three implementations (static, context-based, test hybrid). All flag reads go through the Service interface so callers are decoupled from the backing store; the primary constraint is that unknown features return an error, not false.

## Patterns

**Service interface for all reads** — All consumers call Service.IsFeatureEnabled(ctx, Feature) — never inspect AccessConfig directly outside this package. (`svc := ffx.NewStaticService(ffx.AccessConfig{MyFeature: true})
enabled, err := svc.IsFeatureEnabled(ctx, MyFeature)`)
**Named Feature constants, not raw strings** — Feature is a string type; declare named constants and pass them to IsFeatureEnabled. Raw literals bypass static reference checking. (`const FeatureBetaBilling ffx.Feature = "beta-billing"`)
**Context-carried AccessConfig for per-request flags** — SetAccessOnContext injects an AccessConfig into ctx; NewContextService reads it back. Missing key returns ErrContextMissing, not false. (`ctx = ffx.SetAccessOnContext(ctx, ffx.AccessConfig{ffx.Feature("beta"): true})`)
**Test hybrid service falls back to static** — NewTestContextService tries ctx first then provided static defaults, letting tests override individual flags without rebuilding context. (`svc := ffx.NewTestContextService(ffx.AccessConfig{ffx.Feature("x"): false})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `featureflag.go` | Declares Feature, AccessConfig, and the Service interface — single source of truth for the contract. | AccessConfig.Merge mutates the receiver — pass a fresh copy if the original must stay unchanged. |
| `context.go` | SetAccessOnContext/GetAccessFromContext + contextService + testContextService; ErrContextMissing sentinel. | contextService errors if the feature key is absent; callers must handle the error, not just the bool. |
| `static.go` | staticService backed by an in-process AccessConfig map; used for startup wiring and test defaults. | Returns error (not false) for unknown features — do not assume unknown == disabled. |

## Anti-Patterns

- Bypassing Service by reading AccessConfig maps directly outside this package.
- Using raw string literals instead of typed Feature constants.
- Ignoring the error return from IsFeatureEnabled — unknown feature != false.
- Storing AccessConfig in a global variable instead of context or a Wire-injected service.

## Decisions

- **Three separate service implementations (static, context, test hybrid).** — Static is safe at startup; context serves per-request tenant flags; the test hybrid overrides without rebuilding context — all without nil checks or type switches in callers.

## Example: Wire a static service and check a flag in a handler

```
import "github.com/openmeterio/openmeter/pkg/ffx"

const FeatureBetaBilling ffx.Feature = "beta-billing"
// At wiring time:
svc := ffx.NewStaticService(ffx.AccessConfig{FeatureBetaBilling: cfg.BillingEnabled})
// In handler:
enabled, err := svc.IsFeatureEnabled(ctx, FeatureBetaBilling)
if err != nil { return fmt.Errorf("feature flag: %w", err) }
if enabled { /* new billing path */ }
```

<!-- archie:ai-end -->
