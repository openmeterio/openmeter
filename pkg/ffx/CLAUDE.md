# ffx

<!-- archie:ai-start -->

> Minimal feature flag service with three implementations (static, context-based, test hybrid). All feature-flag reads go through the Service interface so callers are decoupled from the backing store; the primary constraint is that unknown features return an error, not false.

## Patterns

**Service interface for all reads** — All consumers call Service.IsFeatureEnabled(ctx, Feature) — never inspect AccessConfig directly outside this package. (`svc := ffx.NewStaticService(ffx.AccessConfig{MyFeature: true})
enabled, err := svc.IsFeatureEnabled(ctx, MyFeature)`)
**Named Feature constants, not raw strings** — Feature is type Feature string — always declare named Feature constants and pass them to IsFeatureEnabled; raw string literals bypass static reference checking. (`const FeatureBetaBilling ffx.Feature = "beta-billing"`)
**Context-carried AccessConfig for per-request flags** — SetAccessOnContext injects an AccessConfig into ctx; NewContextService reads it back. Missing key returns ErrContextMissing, not false. (`ctx = ffx.SetAccessOnContext(ctx, ffx.AccessConfig{ffx.Feature("beta"): true})`)
**Test hybrid service falls back to static** — NewTestContextService tries ctx first, then falls back to provided static defaults — lets tests override individual flags without rebuilding context per case. (`svc := ffx.NewTestContextService(ffx.AccessConfig{ffx.Feature("x"): false})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `featureflag.go` | Declares Feature, AccessConfig, and the Service interface. Single source of truth for the package contract. | AccessConfig.Merge mutates the receiver — always pass a fresh copy if merging should not modify the original map. |
| `context.go` | SetAccessOnContext/GetAccessFromContext + contextService + testContextService. ErrContextMissing is the sentinel for missing ctx key. | contextService returns an error if the feature key is absent from the config; callers must handle the error, not just check the bool. |
| `static.go` | staticService backed by an in-process AccessConfig map. Used for server startup wiring and test defaults. | Returns error (not false) for unknown features — do not assume unknown == disabled. |

## Anti-Patterns

- Bypassing Service by reading AccessConfig maps directly outside this package.
- Using raw string literals instead of typed Feature constants.
- Ignoring the error return from IsFeatureEnabled — unknown feature != false.
- Storing AccessConfig in a global variable instead of in context or Wire-injected service.

## Decisions

- **Three separate service implementations (static, context, test hybrid) rather than a single configurable one.** — Static is safe for server startup; context-based serves per-request tenant flags; the test hybrid lets unit tests override without rebuilding context, all without nil checks or type switches in callers.

## Example: Wire a static service at startup and check a feature flag in a handler

```
import "github.com/openmeterio/openmeter/pkg/ffx"

const FeatureBetaBilling ffx.Feature = "beta-billing"

// At wiring time:
svc := ffx.NewStaticService(ffx.AccessConfig{FeatureBetaBilling: cfg.BillingEnabled})

// In handler:
enabled, err := svc.IsFeatureEnabled(ctx, FeatureBetaBilling)
if err != nil {
    return fmt.Errorf("feature flag: %w", err)
}
if enabled {
    // use new billing path
}
```

<!-- archie:ai-end -->
