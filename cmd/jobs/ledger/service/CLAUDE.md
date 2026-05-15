# service

<!-- archie:ai-start -->

> Pure-Go backfill orchestration service (no Cobra, no direct Ent import in service.go) that pages through customers and provisions missing ledger accounts. Separated from the command layer to allow unit testing without a real database.

## Patterns

**Interface-driven dependencies, no Ent in service.go** — Service depends on unexported customerLister and accountProvisioner interfaces. Concrete implementations (EntCustomerLister, real resolver) are injected from the command layer. Never import *entdb.Client in service.go. (`type customerLister interface { ListCustomers(ctx context.Context, input ListCustomersInput) (ListCustomersResult, error) }`)
**Config struct with Validate() and constructor returning error** — All dependencies bundled in Config, validated in Config.Validate(), constructor returns (*Service, error). Add new dependencies to Config and validate them; never add optional fields with zero-value fallback. (`func NewService(cfg Config) (*Service, error) { if err := cfg.Validate(); err != nil { return nil, fmt.Errorf("invalid backfill config: %w", err) }; return &Service{...}, nil }`)
**RunInput.normalized() for caller-visible defaults** — Input defaults (CustomerPageSize=0 → DefaultCustomerPageSize, CreatedBefore UTC normalization) are applied by a private normalized() method inside Run, not in the command layer or constructor. (`func (i RunInput) normalized() RunInput { out := i; if out.CustomerPageSize == 0 { out.CustomerPageSize = DefaultCustomerPageSize }; return out }`)
**paginationv2.MAX_SAFE_ITER guard on all cursor loops** — All cursor-paginated loops must check against paginationv2.MAX_SAFE_ITER and call recordFailure + return error when exhausted, preventing infinite loops on bad cursors. (`for iter := 0; iter < paginationv2.MAX_SAFE_ITER; iter++ { ...; if !completed { return result, s.recordFailure(&result, "paginate_customers", "", fmt.Errorf("max safe iter reached")) } }`)
**Idempotent Get-then-Create provisioning with ValidationIssue code detection** — Always call GetCustomerAccounts / GetBusinessAccounts first. Only provision when the error carries ledger.ErrCodeCustomerAccountMissing / ErrCodeBusinessAccountMissing via hasValidationIssueCode; other errors go through recordFailure. Makes the backfill safely re-runnable. (`if !hasValidationIssueCode(err, ledger.ErrCodeCustomerAccountMissing) { return s.recordFailure(result, "get_customer_accounts", customerID.ID, err) }`)
**DryRun increments WouldProvision counters only** — When input.DryRun is true, increment BusinessWouldProvision / CustomersWouldProvision and return without calling EnsureBusinessAccounts / CreateCustomerAccounts. (`if input.DryRun { result.CustomersWouldProvision++; return nil }`)
**recordFailure owns FailureCount increment and slog.Warn; ContinueOnError checked in caller** — recordFailure always increments result.FailureCount and logs via slog.Warn. Callers check input.ContinueOnError to decide whether to return the error or continue. (`failure := s.recordFailure(&result, "list_customers", "", err); if input.ContinueOnError { return result, nil }; return result, failure`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Core orchestration: Config/Service types, customerLister/accountProvisioner interfaces, Run/runNamespace/ensureBusinessAccounts/ensureCustomerAccounts/recordFailure/hasValidationIssueCode. | No direct Ent imports — keep this file free of *entdb.Client. Do not add new ledger error codes without updating hasValidationIssueCode. ContinueOnError must be checked by callers of recordFailure, not inside recordFailure itself. |
| `customer_lister_ent.go` | EntCustomerLister implements customerLister using Ent cursor queries (query.Cursor) with namespace + DeletedAt + CreatedAt filters. | Uses query.Cursor(ctx, input.Cursor) — Ent cursor pagination, not offset. IncludeDeleted=false filters via customerdb.Or(DeletedAtIsNil(), DeletedAtGTE(now)) for correct soft-delete semantics. CreatedBefore filter uses CreatedAtLT with UTC normalization. |
| `service_test.go` | Unit tests using fakeCustomerLister and fakeAccountProvisioner covering dry-run, cursor pagination, created-before cutoff, and continue-on-error. | Uses t.Context() not context.Background(). fakeCustomerLister implements cursor pagination manually — keep consistent with real cursor semantics (time+ID ordering) when adding tests. fakeAccountProvisioner uses ledger.ErrCustomerAccountMissing.WithAttrs(...) to emit real ValidationIssue codes. |

## Anti-Patterns

- Importing *entdb.Client directly in service.go — breaks unit testability
- Checking ContinueOnError inside recordFailure — callers own that decision
- Skipping the MAX_SAFE_ITER guard in any new paginated loop
- Using offset pagination instead of paginationv2.Cursor for customer iteration
- Using context.Background() in tests instead of t.Context()

## Decisions

- **Separate service package from Cobra command** — Enables pure-Go unit tests (no Ent, no Cobra, no real DB) covering dry-run, cursor paging, cutoff filtering, and error continuation without integration overhead.
- **Detect missing-account condition via ValidationIssue error codes, not nil checks** — ledger.ErrCodeCustomerAccountMissing / ErrCodeBusinessAccountMissing are structured error codes on ValidationIssue; treating any non-nil error as 'missing' would mask real failures from DB or network errors.

## Example: Cursor-paginated customer provisioning loop with DryRun and ContinueOnError

```
import (
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

var cursor *paginationv2.Cursor
completed := false
for iter := 0; iter < paginationv2.MAX_SAFE_ITER; iter++ {
	res, err := s.customerLister.ListCustomers(ctx, ListCustomersInput{Namespace: namespace, PageSize: input.CustomerPageSize, Cursor: cursor})
	if err != nil {
		failure := s.recordFailure(&result, "list_customers", "", err)
		if input.ContinueOnError { return result, nil }
		return result, failure
	}
// ...
```

<!-- archie:ai-end -->
