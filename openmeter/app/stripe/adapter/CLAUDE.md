# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing the appstripe.Adapter interface; owns all DB reads and writes for Stripe app config (AppStripe entity), Stripe-customer mappings (AppStripeCustomer entity), and secret retrieval. Pure persistence layer — no business logic belongs here.

## Patterns

**TransactingRepo on every mutation** — Every method that writes to the DB must wrap its body with entutils.TransactingRepo or TransactingRepoWithNoValue. Never call repo.db directly outside that wrapper. (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error { return repo.db.AppStripe.Update().Where(...).Exec(ctx) })`)
**Tx / WithTx / Self triad** — adapter implements TxCreator via Tx() (HijackTx + NewTxDriver), rebinds via WithTx() (NewTxClientFromRawConfig), and returns itself via Self(). WithTx must copy ALL struct fields including stripeClientFactory and stripeAppClientFactory or calls inside the transaction will use stale factories. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger, appService: a.appService, ...} }`)
**input.Validate() before any DB call** — Every exported adapter method calls input.Validate() first and wraps failures in models.NewGenericValidationError. Validation errors must NOT reach the DB layer. (`if err := input.Validate(); err != nil { return appstripe.AppData{}, models.NewGenericValidationError(fmt.Errorf("error getting stripe app data: %w", err)) }`)
**entdb error mapping** — Map entdb.IsNotFound to app.NewAppNotFoundError or app.NewAppCustomerPreConditionError; map entdb.IsConstraintError to models.NewGenericConflictError or app.NewAppCustomerPreConditionError. Never leak raw ent errors to callers. (`if entdb.IsNotFound(err) { return appstripe.AppData{}, app.NewAppNotFoundError(input.AppID) }`)
**EnsureCustomer before app-customer upsert** — UpsertStripeCustomerData must call repo.appService.EnsureCustomer inside the transaction before the actual DB upsert. Skipping EnsureCustomer breaks the app-customer relationship FK and leaves orphaned rows. (`err := repo.appService.EnsureCustomer(ctx, app.EnsureCustomerInput{AppID: input.AppID, CustomerID: input.CustomerID})`)
**GetWebhookSecret omits namespace filter** — GetWebhookSecret deliberately does NOT filter by namespace when querying AppStripe — the webhook payload is Stripe-signed but the namespace is unknown before validation. Do not add a namespace predicate to this query. (`repo.db.AppStripe.Query().Where(appstripedb.ID(input.AppID)).Only(ctx) // no namespace predicate`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter struct definition, Config, New() constructor, and the Tx/WithTx/Self transaction helpers. | WithTx must copy all factory fields (stripeClientFactory, stripeAppClientFactory); missing any field silently breaks calls inside the transaction. |
| `customer.go` | GetStripeCustomerData, UpsertStripeCustomerData, DeleteStripeCustomerData, createStripeCustomer — all Stripe-customer mapping operations. | UpsertStripeCustomerData validates the Stripe customer exists in the Stripe account before the DB upsert; skipping this check allows orphaned Stripe IDs. |
| `stripe.go` | CreateStripeApp, UpdateAPIKey, GetStripeAppData, DeleteStripeAppData, GetWebhookSecret, SetCustomerDefaultPaymentMethod, CreateCheckoutSession. | GetWebhookSecret intentionally omits the namespace filter — do not add one. UpdateAPIKey calls secretService.UpdateAppSecret first, then optionally updates the DB only if the secret ID changed. |

## Anti-Patterns

- Calling repo.db methods directly without entutils.TransactingRepo — bypasses the ctx-bound Ent transaction.
- Returning raw ent errors to callers instead of mapping to app.NewAppNotFoundError / models.NewGenericConflictError.
- Implementing business logic here — this is a pure persistence layer; all orchestration belongs in appservice.
- Omitting EnsureCustomer before AppStripeCustomer upsert — leaves the app_customer FK relationship missing.
- Adding a namespace filter to GetWebhookSecret — the namespace is unknown at webhook validation time.

## Decisions

- **Adapter holds StripeClientFactory and StripeAppClientFactory rather than creating clients inline.** — Allows tests to inject a mock factory without hitting the real Stripe API.
- **WithTx creates a new adapter value rather than pointer mutation.** — Prevents accidental cross-request transaction leakage when multiple goroutines hold the same adapter pointer.

## Example: New adapter method that upserts inside the caller's transaction

```
func (a *adapter) MyUpsert(ctx context.Context, input appstripe.MyInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(fmt.Errorf("my upsert: %w", err))
	}
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error {
		err := repo.db.AppStripe.Update().
			Where(appstripedb.Namespace(input.AppID.Namespace)).
			Where(appstripedb.ID(input.AppID.ID)).
			SetField(input.Value).
			Exec(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return app.NewAppNotFoundError(input.AppID)
			}
			return fmt.Errorf("failed to upsert: %w", err)
// ...
```

<!-- archie:ai-end -->
