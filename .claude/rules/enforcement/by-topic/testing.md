# Enforcement: testing (1 rule)

Topic file. Loaded on demand when an agent works on something in the `testing` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pattern Divergence (inform)

### `place-testutils-no-appcommon` — Per-domain test fixtures live in openmeter/<domain>/testutils/ and must build dependencies from package constructors directly; _test.go files live next to their source; end-to-end tests live in e2e/.

*source: `deep_scan`*

**Why:** The file placement rule for Test files states: 'Go _test.go files live next to their source; per-domain testutils/ packages provide fixtures and must not import app/common; e2e/ holds end-to-end tests.' Importing app/common from a testutils package creates a test-only import cycle because app/common imports all domain packages.

**Example:**

```
// openmeter/customer/testutils/env.go
func NewTestService(t *testing.T, db *entdb.Client) customer.Service {
    return customer.New(adapter.New(db))
}
```
