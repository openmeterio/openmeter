# taxcode

<!-- archie:ai-start -->

> Manages tax codes (with per-app-type mappings) used during invoice line processing. The root package owns the Service and Repository interfaces, all input/output types, and domain errors as ValidationIssue sentinels; adapter/ provides Ent/PostgreSQL persistence; service/ provides business logic including system-managed protection and get-or-create with conflict retry.

## Patterns

**ValidationIssue sentinels in errors.go** — Every domain error is a package-level models.ValidationIssue variable with an explicit ErrorCode, HTTP status attribute, and field path. Constructor functions (NewTaxCodeNotFoundError, NewTaxCodeByAppMappingNotFoundError) attach context attributes. (`var ErrTaxCodeNotFound = models.NewValidationIssue(ErrCodeTaxCodeNotFound, "tax code not found", commonhttp.WithHTTPStatusCodeAttribute(http.StatusNotFound))`)
**Validate() on every input type — compile-time assertion** — Every input struct implements models.Validator; the root service.go has compile-time var _ models.Validator = (*CreateTaxCodeInput)(nil) assertions for all inputs. (`var _ models.Validator = (*CreateTaxCodeInput)(nil)`)
**inputOptions embedded in mutable inputs for AllowAnnotations bypass** — UpdateTaxCodeInput and DeleteTaxCodeInput embed inputOptions{AllowAnnotations bool} to let internal callers (e.g. system sync) bypass the IsManagedBySystem guard. (`type UpdateTaxCodeInput struct { models.NamespacedID; ...; inputOptions }`)
**transaction.Run at service layer + TransactingRepo in adapter** — Service methods wrap adapter calls with transaction.Run for atomicity; each adapter method wraps Ent access with entutils.TransactingRepo so the ctx-bound transaction is honoured. (`return transaction.Run(ctx, s.repo, func(ctx context.Context) (TaxCode, error) { return s.repo.CreateTaxCode(ctx, input) })`)
**IsManagedBySystem guard in service mutating methods** — Before any Update/Delete, service checks tc.IsManagedBySystem() and returns ErrTaxCodeManagedBySystem unless input.AllowAnnotations is true. (`if tc.IsManagedBySystem() && !input.AllowAnnotations { return TaxCode{}, ErrTaxCodeManagedBySystem }`)
**JSONB containment query via raw sql.Selector in adapter** — GetTaxCodeByAppMapping uses a raw sql.Selector with JSONB @> operator for app-mapping lookups — not a standard Ent predicate. (`q.Where(sql.P(func(b *sql.Builder) { b.Ident("app_mappings").WriteOp(sql.OpContains).Arg(jsonBytes) }))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/taxcode/service.go` | Service interface, all input types with Validate(), compile-time models.Validator assertions, and inputOptions. | New input structs without Validate() or missing compile-time assertions; new mutating methods that skip the IsManagedBySystem guard. |
| `openmeter/taxcode/taxcode.go` | TaxCode domain type, TaxCodeAppMapping/TaxCodeAppMappings with uniqueness validation, IsManagedBySystem, GetAppMapping. | TaxCodeStripeRegexp format (txcd_\d{8}) must stay in sync with Stripe's tax code format. |
| `openmeter/taxcode/errors.go` | All ValidationIssue sentinels with HTTP status attributes and ErrorCode constants; IsTaxCodeNotFoundError predicate. | Returning raw errors instead of ValidationIssue sentinels — breaks HTTP status code mapping in the error encoder chain. |
| `openmeter/taxcode/repository.go` | Repository interface extending entutils.TxCreator — all adapter implementations must embed a Tx method. | New operations added to Service without a matching Repository method will compile but fail at runtime. |
| `openmeter/taxcode/adapter/adapter.go` | Ent adapter with TransactingRepo, soft-delete, and JSONB containment query. | Direct a.db.TaxCode.* calls without TransactingRepo wrapper fall off the ctx transaction. |
| `openmeter/taxcode/service/taxcode.go` | Concrete service implementation with transaction.Run, conflict retry in GetOrCreate, and system-managed guard. | Business logic (key derivation, conflict retry) belongs here, not in the adapter. |

## Anti-Patterns

- Calling adapter methods outside transaction.Run in the service layer — loses atomicity for multi-step operations
- Skipping IsManagedBySystem check in new mutating methods — allows external callers to overwrite system-synced records
- Implementing business logic (key derivation, conflict retry) inside the adapter instead of the service layer
- Adding raw SQL outside sql.P / sql.Selector wrapper in the adapter — bypasses Ent's query builder
- Importing app/common in testutils/ — creates import cycles and couples test setup to the full DI graph

## Decisions

- **transaction.Run at service layer in addition to entutils.TransactingRepo in adapter** — Service-level transaction.Run ensures multi-step operations (get-then-update, get-or-create with conflict retry) are atomic; adapter-level TransactingRepo ensures individual DB calls honour the ctx-bound transaction even when called directly.
- **AllowAnnotations flag on UpdateTaxCodeInput/DeleteTaxCodeInput** — System sync processes need to update system-managed tax codes without relaxing the public API surface — a typed flag on the input struct keeps the bypass explicit and auditable.

## Example: Creating a new tax code through the service

```
import (
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

tc, err := svc.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
	Namespace: ns,
	Key:       "standard",
	Name:      "Standard Rate",
	AppMappings: taxcode.TaxCodeAppMappings{
		{AppType: app.AppTypeStripe, TaxCode: "txcd_99999999"},
	},
})
```

<!-- archie:ai-end -->
