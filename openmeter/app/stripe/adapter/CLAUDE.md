# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing the appstripe.Adapter interface; owns all DB reads and writes for Stripe app config (AppStripe entity), Stripe-customer mappings (AppStripeCustomer entity), and secret retrieval. All mutations go through entutils.TransactingRepo / TransactingRepoWithNoValue so ctx-bound transactions are honored.

## Patterns

**TransactingRepo on every mutation** — Every method that writes to the DB must be wrapped with entutils.TransactingRepo or TransactingRepoWithNoValue(ctx, a, func(ctx, repo *adapter) ...). Never call repo.db directly outside that wrapper. (`entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (T, error) { ... repo.db.AppStripe.Create()... })`)
**WithTx / Self / Tx triad** — adapter implements TxCreator via Tx(), rebinds to an existing tx via WithTx(), and returns itself via Self(). These three methods are the contract that entutils.TransactingRepo relies on — they must stay consistent. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), ...} }`)
**Input.Validate() before any DB call** — Every exported adapter method calls input.Validate() first and wraps validation failures in models.NewGenericValidationError. (`if err := input.Validate(); err != nil { return ..., models.NewGenericValidationError(fmt.Errorf("error ...: %w", err)) }`)
**entdb.IsNotFound / IsConstraintError error mapping** — Map entdb.IsNotFound to app.NewAppNotFoundError or app.NewAppCustomerPreConditionError; map entdb.IsConstraintError to models.NewGenericConflictError. (`if entdb.IsNotFound(err) { return ..., app.NewAppNotFoundError(input.AppID) }`)
**Config.Validate() in New()** — The constructor calls config.Validate() and returns an error on failure. All required fields (Client, AppService, CustomerService, SecretService, Logger) must be present. (`func New(config Config) (appstripe.Adapter, error) { if err := config.Validate(); err != nil { return nil, ... } }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter struct definition, Config, New constructor, Tx/WithTx/Self transaction helpers. | WithTx must copy all factory fields (stripeClientFactory, stripeAppClientFactory) when rebinding; missing a field silently breaks calls inside the transaction. |
| `customer.go` | GetStripeCustomerData, UpsertStripeCustomerData, DeleteStripeCustomerData, createStripeCustomer — all Stripe-customer mapping operations. | UpsertStripeCustomerData calls app.EnsureCustomer inside the transaction before the actual upsert; skipping EnsureCustomer breaks the app-customer relationship FK. |
| `stripe.go` | CreateStripeApp, UpdateAPIKey, GetStripeAppData, DeleteStripeAppData, GetWebhookSecret, SetCustomerDefaultPaymentMethod, CreateCheckoutSession. | GetWebhookSecret intentionally omits namespace filter (webhook payload is signed, namespace is unknown before validation); do not add a namespace predicate there. |

## Anti-Patterns

- Calling repo.db methods directly without entutils.TransactingRepo — bypasses ctx-bound transaction.
- Returning raw ent errors to callers instead of mapping them to app.NewAppNotFoundError / models.NewGenericConflictError.
- Implementing new business logic here — this is a pure persistence layer; all orchestration belongs in appservice.

## Decisions

- **Adapter holds both StripeClientFactory and StripeAppClientFactory rather than creating clients inline.** — Allows tests to inject a mock factory without hitting the real Stripe API.
- **WithTx creates a new adapter value (not pointer mutation) so the original adapter is not affected by transaction rebinding.** — Prevents accidental cross-request transaction leakage when multiple goroutines hold the same adapter pointer.

## Example: Writing a new adapter method that upserts a row inside the caller's transaction

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
