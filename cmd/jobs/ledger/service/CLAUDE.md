# service

<!-- archie:ai-start -->

> Backend service for the ledger account backfill job. `Service.Run` paginates customers in a namespace and ensures business + per-customer ledger accounts exist, supporting dry-run, continue-on-error, created-before cutoff, and include-deleted modes; returns a `NamespaceResult` of counters.

## Patterns

**Config-validated service constructor** — `NewService(Config)` validates required dependencies (customerLister, accountProvisioner) via `Config.Validate()` and returns `(*Service, error)`; dependencies are unexported interfaces so tests can inject fakes. (`func NewService(cfg Config) (*Service, error) { if err := cfg.Validate(); err != nil {...} }`)
**Interface-typed collaborators** — `customerLister` and `accountProvisioner` are local interfaces. accountProvisioner exposes Get/Create CustomerAccounts and Get/Ensure BusinessAccounts; the backfillaccounts command satisfies it with the concrete resolver. (`type accountProvisioner interface { CreateCustomerAccounts(ctx, customer.CustomerID) (ledger.CustomerAccounts, error); ... }`)
**Validate + normalize input structs** — RunInput and ListCustomersInput both have `Validate() error`; RunInput also has `normalized()` that applies DefaultCustomerPageSize and forces CreatedBefore to UTC before use. (`input := in.normalized(); result, err := s.runNamespace(ctx, input, input.Namespace)`)
**Cursor pagination with MAX_SAFE_ITER guard** — runNamespace loops up to paginationv2.MAX_SAFE_ITER, calling customerLister.ListCustomers with a *paginationv2.Cursor; breaks when items empty or NextCursor is nil, and records a paginate_customers failure if the loop never completes. (`for iter := 0; iter < paginationv2.MAX_SAFE_ITER; iter++ { ... cursor = res.NextCursor }`)
**Missing-account detection via ValidationIssue codes** — ensureBusiness/CustomerAccounts call Get first; only ErrCodeBusinessAccountMissing / ErrCodeCustomerAccountMissing (matched by hasValidationIssueCode using models.AsValidationIssues) trigger provisioning. Any other error is a recorded failure. (`if !hasValidationIssueCode(err, ledger.ErrCodeCustomerAccountMissing) { return s.recordFailure(...) }`)
**Counter accumulation + recordFailure** — All outcomes mutate a *NamespaceResult counter (BusinessProvisioned, CustomersWouldProvision, etc.). recordFailure increments FailureCount, logs a warn with namespace/stage/customer_id, and returns a stage-wrapped error. (`result.CustomersProvisioned++`)
**DryRun short-circuits writes** — When input.DryRun is set, ensure* increments the *WouldProvision counter and returns before calling EnsureBusinessAccounts / CreateCustomerAccounts. (`if input.DryRun { result.CustomersWouldProvision++; return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Core service: Config/RunInput/RunOutput/NamespaceResult types, NewService, Run -> runNamespace -> ensureBusinessAccounts/ensureCustomerAccounts, hasValidationIssueCode, recordFailure. | Get-before-provision relies on ledger missing-account validation codes; a non-validation error must NOT be treated as 'missing'. ContinueOnError returns (result, nil) at the Run level even on failures, so callers must inspect FailureCount. |
| `customer_lister_ent.go` | EntCustomerLister implementing customerLister via entClient.Customer.Query with namespace filter, optional deleted/created-before filters, and Cursor pagination. | Soft-delete filtering uses DeletedAtIsNil OR DeletedAtGTE(now) (clock.Now().UTC()); CreatedBefore uses CreatedAtLT with UTC. Nil items from paged.Items are skipped. |
| `service_test.go` | Table-style tests using fakeCustomerLister and fakeAccountProvisioner to cover dry-run, multi-page cursoring, created-before cutoff, and continue-on-error. | Fakes emit ledger.ErrCustomerAccountMissing/ErrBusinessAccountMissing.WithAttrs to drive the provisioning branch; tests use t.Context() and newCustomer() built from models.NewManagedResource. |

## Anti-Patterns

- Treating any error from GetCustomerAccounts/GetBusinessAccounts as 'missing' instead of matching the specific validation issue code.
- Writing accounts while DryRun is set, or incrementing provisioned counters in dry-run mode.
- Removing the MAX_SAFE_ITER bound or breaking the cursor advance, risking an infinite pagination loop.
- Returning early on the first customer failure when ContinueOnError is requested (Run/runNamespace must keep going and only accumulate FailureCount).
- Using time without UTC normalization for CreatedBefore / cursor comparisons.

## Decisions

- **Detect missing accounts by inspecting validation issue codes on the Get error rather than a sentinel boolean.** — Reuses the ledger package's structured ErrCode* validation errors so provisioning only triggers on a genuine 'account missing' condition, not transient/db errors.
- **Counters are accumulated into NamespaceResult and surfaced via RunOutput even on error.** — Operational job needs a printable summary of scanned/provisioned/failed customers regardless of whether the run aborted or continued on error.

## Example: Ensure a single customer's ledger accounts (get-then-provision with dry-run + validation-code gating)

```
func (s *Service) ensureCustomerAccounts(ctx context.Context, input RunInput, result *NamespaceResult, customerID customer.CustomerID) error {
	_, err := s.accountProvisioner.GetCustomerAccounts(ctx, customerID)
	if err == nil {
		result.CustomersAlreadyProvisioned++
		return nil
	}
	if !hasValidationIssueCode(err, ledger.ErrCodeCustomerAccountMissing) {
		return s.recordFailure(result, "get_customer_accounts", customerID.ID, err)
	}
	if input.DryRun {
		result.CustomersWouldProvision++
		return nil
	}
	if _, err = s.accountProvisioner.CreateCustomerAccounts(ctx, customerID); err != nil {
		return s.recordFailure(result, "create_customer_accounts", customerID.ID, err)
// ...
```

<!-- archie:ai-end -->
