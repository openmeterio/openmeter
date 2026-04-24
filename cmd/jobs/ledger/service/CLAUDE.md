# service

<!-- archie:ai-start -->

> Pure-Go backfill orchestration service (no Cobra, no Ent import in service.go) that pages through customers and provisions missing ledger accounts. Separated from the command to allow unit testing without a real database.

## Patterns

**Interface-driven dependencies** — Service depends on customerLister and accountProvisioner interfaces (unexported), not concrete types. EntCustomerLister and the real resolver are injected from the command layer. (`type customerLister interface { ListCustomers(ctx context.Context, input ListCustomersInput) (ListCustomersResult, error) }`)
**Config struct with Validate + constructor** — All dependencies are bundled in Config, validated in Config.Validate(), and the constructor returns (*Service, error). Do not add optional fields — add them to Config and validate. (`func NewService(cfg Config) (*Service, error) { if err := cfg.Validate(); err != nil { return nil, fmt.Errorf(...) } ... }`)
**RunInput.normalized() for defaults** — Caller-visible defaults (e.g. CustomerPageSize=0 → DefaultCustomerPageSize) are applied by a private normalized() method inside Run, not in the command layer or constructor. (`func (i RunInput) normalized() RunInput { if out.CustomerPageSize == 0 { out.CustomerPageSize = DefaultCustomerPageSize } ... }`)
**paginationv2.MAX_SAFE_ITER guard on cursor loop** — All cursor-paginated loops must check against paginationv2.MAX_SAFE_ITER and return an error if exhausted, preventing infinite loops on bad cursors. (`for iter := 0; iter < paginationv2.MAX_SAFE_ITER; iter++ { ... if !completed { return result, fmt.Errorf("max safe iter reached") } }`)
**Idempotent provisioning via Get-then-Create** — Always call GetCustomerAccounts / GetBusinessAccounts first. Only provision when the error carries ledger.ErrCodeCustomerAccountMissing / ErrCodeBusinessAccountMissing; other errors go through recordFailure. This makes the backfill re-runnable. (`if !hasValidationIssueCode(err, ledger.ErrCodeCustomerAccountMissing) { return s.recordFailure(...) }`)
**DryRun increments WouldProvision counters only** — When input.DryRun is true, the service increments BusinessWouldProvision / CustomersWouldProvision and returns without calling EnsureBusinessAccounts / CreateCustomerAccounts. (`if input.DryRun { result.CustomersWouldProvision++; return nil }`)
**recordFailure increments FailureCount and logs** — All errors go through recordFailure which increments result.FailureCount, logs via slog.Warn, and returns a wrapped error. ContinueOnError is checked in the caller loop, not inside recordFailure. (`func (s *Service) recordFailure(result *NamespaceResult, stage, customerID string, err error) error { result.FailureCount++; s.logger.Warn(...); return fmt.Errorf(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Core orchestration: Config/Service types, Run/runNamespace, ensureBusinessAccounts, ensureCustomerAccounts, recordFailure, hasValidationIssueCode. | No direct Ent imports — keep this file free of *entdb.Client so tests stay pure Go. Do not add new ledger error codes without updating hasValidationIssueCode. |
| `customer_lister_ent.go` | EntCustomerLister implements customerLister using Ent cursor queries with namespace + deleted-at + createdAt filters. | Uses query.Cursor(ctx, input.Cursor) — Ent cursor pagination, not offset. IncludeDeleted=false filters via DeletedAtIsNil OR DeletedAtGTE(now) to handle soft-deletes correctly. |
| `service_test.go` | Unit tests using fakeCustomerLister and fakeAccountProvisioner. Tests cover dry-run, cursor pagination, created-before cutoff, and continue-on-error. | Uses t.Context() not context.Background(). fakeCustomerLister implements cursor pagination manually — keep it consistent with real cursor semantics when adding tests. |

## Anti-Patterns

- Importing *entdb.Client directly in service.go — breaks unit testability
- Checking ContinueOnError inside recordFailure — callers own that decision
- Skipping the MAX_SAFE_ITER guard in new paginated loops
- Using offset pagination instead of cursor (paginationv2.Cursor) for large customer sets
- Using context.Background() in tests instead of t.Context()

## Decisions

- **Separate service package from Cobra command** — Enables pure-Go unit tests (no Ent, no Cobra, no real DB) covering dry-run, cursor paging, cutoff filtering, and error continuation without integration overhead.
- **Detect missing-account condition via ValidationIssue error codes, not nil checks** — ledger.ErrCodeCustomerAccountMissing / ErrCodeBusinessAccountMissing are structured error codes on ValidationIssue; treating any non-nil error as 'missing' would mask real failures.

## Example: Cursor-paginated provisioning loop with DryRun and ContinueOnError

```
for iter := 0; iter < paginationv2.MAX_SAFE_ITER; iter++ {
	res, err := s.customerLister.ListCustomers(ctx, ListCustomersInput{...Cursor: cursor})
	if err != nil {
		failure := s.recordFailure(&result, "list_customers", "", err)
		if input.ContinueOnError { return result, nil }
		return result, failure
	}
	for _, item := range res.Items {
		if err := s.ensureCustomerAccounts(ctx, input, &result, ...); err != nil {
			if !input.ContinueOnError { return result, err }
		}
	}
	if res.NextCursor == nil { completed = true; break }
	cursor = res.NextCursor
}
```

<!-- archie:ai-end -->
