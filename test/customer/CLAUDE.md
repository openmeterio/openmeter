# customer

<!-- archie:ai-start -->

> Integration test suite (package `customer`) for the customer domain and its cross-cutting interactions with subjects, subscriptions, entitlements, and billing. Tests run against a real Postgres DB through an interface-based TestEnv rather than testify suites.

## Patterns

**Interface-based TestEnv accessor, not embedded suite** — TestEnv is an interface exposing App(), Customer(), Subscription(), SubscriptionWorkflow(), Entitlement(), Feature(), Subject(), Plan(), Billing(), Meter(), Close(). Test methods read services via s.Env.Customer() etc.; CustomerHandlerTestSuite just holds an Env + namespace. (`service := s.Env.Customer(); subj, err := s.Env.Subject().GetByKey(ctx, models.NamespacedKey{...})`)
**Single Test func drives subtests with explicit ctx** — One TestCustomer(t) creates the env via NewTestEnv(t, ctx), defers env.Close(), then runs named subtests by calling suite methods with the signature (ctx context.Context, t *testing.T). No suite.Run. (`t.Run("TestCreate", func(t *testing.T){ testSuite.TestCreate(ctx, t) })`)
**Fresh ULID namespace per test method** — Each suite method begins with s.setupNamespace(t) which sets s.namespace = ulid.Make().String(); shared test constants (TestKey, TestName, TestAddress, TestSubjectKeys) are package vars. (`func (s *CustomerHandlerTestSuite) TestCreate(ctx, t){ s.setupNamespace(t); ... }`)
**Customer<->subject hooks wired in NewTestEnv** — NewTestEnv registers SubjectCustomerHook and CustomerSubjectHook so creating a customer materializes subjects and vice versa; CustomerOverride is a noopCustomerOverrideService (no billing override side effects here). (`customerService.RegisterHooks(customerSubjectHook); subjectService.RegisterHooks(subjectCustomerHook)`)
**Conflict assertions via typed error checks** — Key/subject conflicts are asserted with customer.IsSubjectKeyConflictError(err) and models.IsGenericConflictError(err) rather than string matching. (`require.True(t, models.IsGenericConflictError(err), "key overlaps with subject")`)
**DB deps from subscriptiontestutils.SetupDBDeps** — Postgres/ent client comes from subscriptiontestutils.SetupDBDeps(t); env.Close() runs dbDeps.Cleanup(t). Real meter adapter (meter/adapter) is used, with MockStreamingConnector for usage. (`dbDeps := subscriptiontestutils.SetupDBDeps(t); closerFunc := func() error { dbDeps.Cleanup(t); return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `testenv.go` | TestEnv interface + testEnv impl + NewTestEnv(t, ctx): wires app/customer/subject/entitlement/feature/plan/billing/subscription/meter from concrete constructors; registers customer/subject hooks and request validators (entitlement, subscription customer) | MultiSubscriptionEnabledFF is false here (ffx.NewTestContextService). CustomerOverride hook uses noopCustomerOverrideService — billing-override side effects are intentionally absent. |
| `customer.go` | CustomerHandlerTestSuite + TestCreate/TestUpdate/TestUpdateWithSubscriptionPresent; package-level Test* fixtures (TestKey, TestAddress, TestSubjectKeys) | UsageAttribution is nil when there are no subject keys (asserted repeatedly). Updating a customer that has a subscription cannot change UsageAttribution. |
| `subject.go` | TestSubjectDeletion (dangling vs attributed subjects) and TestMultiSubjectIntegrationFlow (full meter+feature+plan+subscription+invoice flow) | Deleting a dangling subject is allowed; deleting an attributed subject removes it from usage attribution (leaving UsageAttribution nil), and must not error even with entitlements present. |
| `customer_test.go` | Entry-point TestCustomer wiring env and dispatching Customer + Subject subtests | Single shared env across all subtests; namespaces isolate them, so do not assume a clean DB per subtest. |
| `subject.go` | Cross-domain integration via s.installSandboxApp / s.createDefaultProfile helpers | Uses subscriptionworkflow.CreateFromPlan and plansubscriptionservice.PlanFromPlan; relies on clock.SetTime + t.Cleanup(clock.ResetTime). |

## Anti-Patterns

- Reaching for testify/suite.Run — this package uses a plain TestCustomer with method dispatch and an interface TestEnv.
- Sharing a hardcoded namespace instead of calling setupNamespace (ULID) at the top of each test method.
- Asserting conflicts by error string instead of customer.IsSubjectKeyConflictError / models.IsGenericConflictError.
- Expecting an empty UsageAttribution struct — it is nil when no subject keys exist.
- Forgetting defer env.Close() (dbDeps.Cleanup) — leaks the test database.

## Decisions

- **TestEnv is an interface with one shared instance per Test func** — Customer tests need many collaborating services (subject, entitlement, subscription, billing); a single wired env with namespace isolation is cheaper than per-subtest stacks.
- **CustomerOverride is noop and multi-subscription FF is off** — This suite focuses on customer/subject lifecycle, not billing override or multi-subscription behavior, so those paths are intentionally disabled to keep tests focused.

## Example: Spin up the env and run customer subtests

```
func TestCustomer(t *testing.T) {
  ctx, cancel := context.WithCancel(context.Background()); defer cancel()
  env, err := NewTestEnv(t, ctx)
  require.NoError(t, err)
  defer func(){ _ = env.Close() }()
  testSuite := CustomerHandlerTestSuite{Env: env}
  t.Run("TestCreate", func(t *testing.T){ testSuite.TestCreate(ctx, t) })
}
```

<!-- archie:ai-end -->
