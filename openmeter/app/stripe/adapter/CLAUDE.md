# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer for the Stripe app integration. Owns CRUD over the appstripe and appstripecustomer DB tables (app data, customer linkage, webhook secrets, default payment method) and orchestrates the live Stripe API via injected client factories. Implements appstripe.Adapter / appstripe.AppStripeAdapter.

## Patterns

**Transaction-aware repo via entutils.TransactingRepo** — Every write/read that must be atomic runs inside entutils.TransactingRepo(ctx, a, func(ctx, repo *adapter)...) (or TransactingRepoWithNoValue); the closure uses repo.db, not a.db, so it rebinds to the tx carried in ctx. (`entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (appstripe.AppData, error) { ... repo.db.AppStripe.Query()... })`)
**Tx/WithTx/Self triad** — adapter implements entutils.TxCreator: Tx() hijacks an ent tx, WithTx() returns a new adapter bound to the tx client via entdb.NewTxClientFromRawConfig, Self() returns the receiver. (`func (a *adapter) WithTx(ctx, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), ...} }`)
**Validate input then wrap with models error constructors** — Each public method calls input.Validate() first and wraps the result in models.NewGenericValidationError; ent errors are mapped via entdb.IsNotFound / entdb.IsConstraintError into app.NewApp*Error or models.NewGenericConflictError. (`if entdb.IsNotFound(err) { return appstripe.CustomerData{}, app.NewAppCustomerPreConditionError(input.AppID, app.AppTypeStripe, &input.CustomerID, "customer has no data for stripe app") }`)
**Config + New constructor with required-dependency validation** — New(Config) validates all injected deps (Client, AppService, CustomerService, SecretService, Logger) and supplies default Stripe client factories (stripeclient.NewStripeClient / NewStripeAppClient) when nil. asserts var _ appstripe.Adapter = (*adapter)(nil). (`func New(config Config) (appstripe.Adapter, error) { if err := config.Validate(); err != nil { ... } }`)
**Upsert via OnConflict with soft-delete-aware conflict target** — AppStripeCustomer upserts use sql.ConflictColumns(namespace, app_id, customer_id) plus sql.ConflictWhere(sql.IsNull(FieldDeletedAt)) so soft-deleted rows do not collide. (`OnConflict(sql.ConflictColumns(...), sql.ConflictWhere(sql.IsNull(appstripecustomerdb.FieldDeletedAt)))`)
**Webhook lookup intentionally ignores namespace** — GetWebhookSecret queries AppStripe by ID only (no namespace filter) because webhook callers have no namespace; trust derives from the signed payload secret. (`Where(appstripedb.ID(input.AppID)).Only(ctx) // no namespace filter, see comment`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | adapter struct, Config/New, Tx/WithTx/Self transaction plumbing | WithTx must copy ALL dependency fields (logger was omitted in the copy here — replicate the existing shape, do not drop fields silently). |
| `customer.go` | stripe customer-data CRUD (Get/Upsert/Delete) and createStripeCustomer | UpsertStripeCustomerData validates the payment method belongs to the stripe customer and calls appService.EnsureCustomer inside the tx; createStripeCustomer reaches the live Stripe API via stripeAppClientFactory. |
| `stripe.go` | app-level CRUD: CreateStripeApp, UpdateAPIKey, GetStripeAppData, DeleteStripeAppData, GetWebhookSecret, SetCustomerDefaultPaymentMethod, CreateCheckoutSession | UpdateAPIKey validates live/test-mode and stripe account match before persisting; secret writes go through secretService outside the ent tx (mapAppStripeData maps db row -> appstripe.AppData). |

## Anti-Patterns

- Using a.db directly inside a TransactingRepo closure instead of repo.db (breaks transaction binding).
- Returning raw ent errors instead of mapping IsNotFound/IsConstraintError to app.* / models.* typed errors.
- Adding a namespace filter to GetWebhookSecret's AppStripe query (webhook requests have no namespace).
- Skipping input.Validate() / models.NewGenericValidationError wrapping at the top of a public method.
- Calling the live Stripe API outside a factory (always go through stripeClientFactory / stripeAppClientFactory so test fakes can be injected).

## Decisions

- **Stripe client construction is injected via factories on Config rather than built inline** — Lets tests substitute fake Stripe clients and lets secret/livemode resolution happen per app instance.
- **Secret creation/webhook setup are NOT wrapped in the ent transaction** — They coordinate three remote systems (secret store, Stripe, DB); the code comments acknowledge this is intentionally non-transactional.

## Example: Adapter method: validate, transact, map ent errors

```
func (a *adapter) GetStripeAppData(ctx context.Context, input appstripe.GetStripeAppDataInput) (appstripe.AppData, error) {
	if err := input.Validate(); err != nil {
		return appstripe.AppData{}, models.NewGenericValidationError(err)
	}
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (appstripe.AppData, error) {
		dbApp, err := repo.db.AppStripe.Query().
			Where(appstripedb.Namespace(input.AppID.Namespace)).
			Where(appstripedb.ID(input.AppID.ID)).Only(ctx)
		if err != nil {
			if entdb.IsNotFound(err) { return appstripe.AppData{}, app.NewAppNotFoundError(input.AppID) }
			return appstripe.AppData{}, err
		}
		return mapAppStripeData(input.AppID, dbApp), nil
	})
}
```

<!-- archie:ai-end -->
