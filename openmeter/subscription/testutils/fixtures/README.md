# Persisted subscription fixtures

A small experiment using `samber/do` to lazily create and cache prerequisite
resources. `Customer()`, `Feature()`, and `Plan()` reuse one resource per scenario;
`CreateCustomer`, `CreatePlan`, and `CreateSubscription` create fresh resources
through the existing services. The injector stays inside the fixture package.

The default graph is a USD customer and a published monthly plan with one
flat-fee rate card and a boolean entitlement backed by a unique feature. Explicit
customer or plan inputs bypass those defaults. A supplied plan is complete:
its features and other prerequisites must already exist.

```go
scenario := fixtures.New(t, deps)
first := scenario.CreateSubscription(fixtures.CreateSubscriptionInput{})
otherCustomer := scenario.CreateCustomer(customer.CustomerMutate{Name: "Other customer"})
second := scenario.CreateSubscription(fixtures.CreateSubscriptionInput{Customer: otherCustomer})
```

Subscriptions are always fresh. Lifecycle actions stay explicit: overlapping
subscriptions for the same customer remain subject to the service's uniqueness
rules. See [the fixture examples](scenario_test.go) for adjacent subscriptions,
explicit prerequisites, and independent default graphs, and
[the multi-customer regression](../../service/listviews_test.go) for shared-plan setup.

Scenarios share the supplied harness's database and example namespace; unique
keys prevent their defaults from colliding. They retain the harness's mock
streaming connector and billing overrides. These fixtures exercise subscription
persistence and entitlement materialization, without billing reconciliation.
Returned values are creation snapshots; reload through the services after mutations.
