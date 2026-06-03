# taxcode

<!-- archie:ai-start -->

> Manages tax codes (with per-app-type mappings) used during invoice line processing. The root package owns the Service and Repository interfaces, input/output types, and ValidationIssue error sentinels; adapter/ provides Ent/PostgreSQL persistence; service/ provides business logic (system-managed protection, get-or-create with conflict retry); testutils/ wires a full stack without app/common.

## Patterns

**ValidationIssue sentinels in errors.go** — Every domain error is a package-level models.ValidationIssue with an ErrorCode, HTTP status attribute, and field path; constructor helpers (NewTaxCodeNotFoundError, NewTaxCodeByAppMappingNotFoundError) attach context attrs. (`var ErrTaxCodeNotFound = models.NewValidationIssue(ErrCodeTaxCodeNotFound, "tax code not found", commonhttp.WithHTTPStatusCodeAttribute(http.StatusNotFound))`)
**Validate() + compile-time models.Validator assertion on every input** — Each input struct implements models.Validator with a var _ models.Validator = (*XInput)(nil) assertion in service.go. (`var _ models.Validator = (*CreateTaxCodeInput)(nil)`)
**transaction.Run at service layer + TransactingRepo in adapter** — Service methods wrap adapter calls in transaction.Run for atomicity; each adapter method wraps Ent access with entutils.TransactingRepo so the ctx-bound tx is honoured. (`return transaction.Run(ctx, s.repo, func(ctx context.Context) (TaxCode, error) { return s.repo.CreateTaxCode(ctx, input) })`)
**IsManagedBySystem guard with AllowAnnotations bypass** — Before Update/Delete the service checks tc.IsManagedBySystem() and returns ErrTaxCodeManagedBySystem unless input.AllowAnnotations (from embedded inputOptions) is true. (`if tc.IsManagedBySystem() && !input.AllowAnnotations { return TaxCode{}, ErrTaxCodeManagedBySystem }`)
**JSONB containment query via raw sql.Selector** — GetTaxCodeByAppMapping uses a raw sql.P/sql.Selector with the JSONB @> operator for app-mapping lookups, not a standard Ent predicate. (`q.Where(sql.P(func(b *sql.Builder){ b.Ident("app_mappings").WriteOp(sql.OpContains).Arg(jsonBytes) }))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/taxcode/service.go` | Service interface, all input types with Validate(), compile-time Validator assertions, inputOptions{AllowAnnotations}. | New inputs without Validate()/assertions; new mutating methods skipping the IsManagedBySystem guard. |
| `openmeter/taxcode/taxcode.go` | TaxCode domain type, TaxCodeAppMapping(s) with uniqueness validation, IsManagedBySystem, GetAppMapping. | TaxCodeStripeRegexp (txcd_\d{8}) must stay in sync with Stripe's tax code format. |
| `openmeter/taxcode/errors.go` | ValidationIssue sentinels + ErrorCode constants + IsTaxCodeNotFoundError predicate. | Returning raw errors instead of sentinels breaks HTTP status mapping. |
| `openmeter/taxcode/repository.go` | Repository interface extending entutils.TxCreator. | Service methods added without a matching Repository method compile but fail at runtime. |
| `openmeter/taxcode/namespacehandler.go` | namespace.Handler implementation for taxcode namespace lifecycle. | Namespace provisioning side-effects must stay idempotent. |

## Anti-Patterns

- Calling adapter methods outside transaction.Run in the service layer — loses atomicity for get-then-mutate flows
- Skipping the IsManagedBySystem check in new mutating methods — lets external callers overwrite system-synced records
- Implementing business logic (key derivation, conflict retry) in the adapter instead of the service layer
- Adding raw SQL outside a sql.P/sql.Selector wrapper in the adapter — bypasses Ent's query builder
- Importing app/common in testutils/ — creates import cycles and couples test setup to the full DI graph

## Decisions

- **transaction.Run at the service layer in addition to TransactingRepo in the adapter** — Service-level transaction.Run keeps multi-step operations (get-or-create with retry) atomic; adapter-level TransactingRepo keeps individual DB calls on the ctx-bound transaction even when called directly.
- **AllowAnnotations flag on Update/Delete inputs** — System-sync processes must update system-managed tax codes without relaxing the public API; a typed flag keeps the bypass explicit and auditable.

<!-- archie:ai-end -->
