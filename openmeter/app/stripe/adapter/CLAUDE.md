# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing appstripe.Adapter — owns all DB reads/writes for Stripe app config (AppStripe entity) and Stripe-customer mappings (AppStripeCustomer entity), plus secret retrieval. Pure persistence layer; no business logic.

## Patterns

**TransactingRepo on every mutation** — Wrap every DB-writing method body in entutils.TransactingRepo or TransactingRepoWithNoValue; never call a.db directly outside that wrapper. (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error { return repo.db.AppStripe.Update().Where(...).Exec(ctx) })`)
**Tx / WithTx / Self triad** — Implement TxCreator: Tx() via HijackTx+NewTxDriver, WithTx() via NewTxClientFromRawConfig, Self() returns itself. WithTx MUST copy all struct fields including stripeClientFactory and stripeAppClientFactory. (`func (a *adapter) WithTx(ctx, tx) *adapter { return &adapter{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client(), stripeClientFactory: a.stripeClientFactory, ...} }`)
**input.Validate() before any DB call** — Every exported method calls input.Validate() first and wraps failures in models.NewGenericValidationError; validation errors must not reach the DB layer. (`if err := input.Validate(); err != nil { return appstripe.AppData{}, models.NewGenericValidationError(fmt.Errorf("...: %w", err)) }`)
**entdb error mapping** — Map entdb.IsNotFound to app.NewAppNotFoundError/app.NewAppCustomerPreConditionError; entdb.IsConstraintError to models.NewGenericConflictError/app.NewAppCustomerPreConditionError. Never leak raw ent errors. (`if entdb.IsNotFound(err) { return appstripe.AppData{}, app.NewAppNotFoundError(input.AppID) }`)
**EnsureCustomer before app-customer upsert** — UpsertStripeCustomerData calls repo.appService.EnsureCustomer inside the transaction before the AppStripeCustomer upsert, or the FK relationship is left orphaned. (`err := repo.appService.EnsureCustomer(ctx, app.EnsureCustomerInput{AppID: input.AppID, CustomerID: input.CustomerID})`)
**GetWebhookSecret omits namespace filter** — GetWebhookSecret deliberately queries AppStripe by ID only — the webhook payload is Stripe-signed but namespace is unknown before validation. Do not add a namespace predicate. (`repo.db.AppStripe.Query().Where(appstripedb.ID(input.AppID)).Only(ctx) // no namespace predicate`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | adapter struct, Config, New() constructor, Tx/WithTx/Self transaction helpers; defaults nil factories to NewStripeClient/NewStripeAppClient. | WithTx must copy all factory fields; missing any silently breaks calls inside the transaction. |
| `customer.go` | GetStripeCustomerData, UpsertStripeCustomerData, DeleteStripeCustomerData, createStripeCustomer. | UpsertStripeCustomerData verifies the Stripe customer (and payment method ownership) exists in the Stripe account before the DB upsert. |
| `stripe.go` | CreateStripeApp, UpdateAPIKey, GetStripeAppData, DeleteStripeAppData, GetWebhookSecret, SetCustomerDefaultPaymentMethod, CreateCheckoutSession. | UpdateAPIKey calls secretService.UpdateAppSecret first, then updates the DB only if the secret ID changed; GetWebhookSecret omits the namespace filter. |

## Anti-Patterns

- Calling repo.db methods directly without entutils.TransactingRepo — bypasses the ctx-bound Ent transaction.
- Returning raw ent errors instead of mapping to app.NewAppNotFoundError / models.NewGenericConflictError.
- Implementing business logic here — orchestration belongs in appservice.
- Omitting EnsureCustomer before AppStripeCustomer upsert — leaves the app_customer FK missing.
- Adding a namespace filter to GetWebhookSecret — namespace is unknown at webhook validation time.

## Decisions

- **Adapter holds StripeClientFactory/StripeAppClientFactory rather than creating clients inline.** — Lets tests inject a mock factory without hitting the real Stripe API.
- **WithTx creates a new adapter value rather than pointer mutation.** — Prevents cross-request transaction leakage when goroutines share the same adapter pointer.

## Example: New adapter method upserting inside the caller's transaction

```
func (a *adapter) MyUpsert(ctx context.Context, input appstripe.MyInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(fmt.Errorf("my upsert: %w", err))
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error {
		err := repo.db.AppStripe.Update().Where(appstripedb.Namespace(input.AppID.Namespace)).Where(appstripedb.ID(input.AppID.ID)).Exec(ctx)
		if entdb.IsNotFound(err) { return app.NewAppNotFoundError(input.AppID) }
		return err
	})
}
```

<!-- archie:ai-end -->
