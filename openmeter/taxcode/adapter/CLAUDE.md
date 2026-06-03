# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing taxcode.Repository. Owns all DB reads and writes for TaxCode and OrganizationDefaultTaxCodes entities, including JSONB containment queries and soft-delete semantics.

## Patterns

**TransactingRepo on every method** — Every exported method wraps its body in entutils.TransactingRepo or TransactingRepoWithNoValue, rebinding to any caller-supplied ctx-bound Ent transaction to prevent partial writes. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (taxcode.TaxCode, error) { ... })`)
**Tx/WithTx/Self triad** — adapter implements Tx(ctx) via HijackTx+NewTxDriver, WithTx(ctx, tx) via NewTxClientFromRawConfig, and Self() to satisfy the TxCreator+TxUser contract. All three required. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Soft-delete via SetDeletedAt** — DeleteTaxCode sets deleted_at to clock.Now() rather than a hard DELETE. Read queries add Where(taxcodedb.DeletedAtIsNil()). (`a.db.TaxCode.UpdateOneID(input.ID).Where(taxcodedb.NamespaceEQ(input.Namespace)).SetDeletedAt(clock.Now()).Exec(ctx)`)
**input.Validate() as first statement** — Every exported method calls input.Validate() before any DB access and returns immediately on error. (`if err := input.Validate(); err != nil { return taxcode.TaxCode{}, err }`)
**JSONB containment via raw sql.P predicate** — GetTaxCodeByAppMapping uses a raw sql.P predicate (app_mappings @> '...') because Ent has no native JSONB containment operator; the pattern is marshaled from []taxcode.TaxCodeAppMapping. (`s.Where(sql.P(func(b *sql.Builder) { b.Ident(taxcodedb.FieldAppMappings).WriteString(" @> ").Arg(string(pattern)) }))`)
**Ent error mapping to domain errors** — db.IsNotFound maps to NewTaxCodeNotFoundError; db.IsConstraintError maps to models.NewGenericConflictError. Never return raw Ent errors. (`if db.IsConstraintError(err) { return taxcode.TaxCode{}, models.NewGenericConflictError(...) }`)
**Config with models.Validator compile-time assertion** — Config implements models.Validator via var _ models.Validator = (*Config)(nil); New() calls config.Validate() before constructing the adapter. (`var _ models.Validator = (*Config)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter constructor, Config validation, and the Tx/WithTx/Self triad required by entutils.TransactingRepo. | All three transaction-rebinding methods must be present or entutils.TransactingRepo will fail to compile. |
| `taxcode.go` | CRUD operations (Create, Update, List, Get, GetByAppMapping, Delete) implementing taxcode.Repository. | GetTaxCodeByAppMapping uses a raw JSONB @> predicate; if TaxCodeAppMapping fields change, the marshaled JSON must stay compatible with the stored column format. |
| `organizationdefaulttaxcodes.go` | GetOrganizationDefaultTaxCodes and UpsertOrganizationDefaultTaxCodes with ON CONFLICT UPSERT and optional edge expansion. | Upsert uses sql.ConflictWhere(sql.IsNull(DeletedAt)) — the partial unique index requires this predicate or the conflict clause does not fire correctly. |
| `mapping.go` | MapTaxCodeFromEntity translates *db.TaxCode to taxcode.TaxCode. | lo.FromPtr(entity.AppMappings) silently returns a zero-value slice when the column is NULL — new nullable fields need the same nil-safe treatment. |

## Anti-Patterns

- Calling a.db.TaxCode.* directly inside a helper without entutils.TransactingRepo — the helper falls off the ctx-bound transaction
- Hard-deleting rows with .Remove() instead of setting deleted_at via SetDeletedAt
- Adding raw SQL outside a sql.P/sql.Selector wrapper — bypasses Ent's query builder
- Returning raw Ent errors without mapping via db.IsNotFound/db.IsConstraintError
- Skipping input.Validate() before DB access — the adapter is the last line of defense

## Decisions

- **entutils.TransactingRepo on every method, not only explicit tx callers** — Ent transactions are carried implicitly in ctx; any helper holding a raw *entdb.Client falls off the transaction unless it rebinds via TransactingRepo.
- **JSONB containment query via raw sql.Selector for GetTaxCodeByAppMapping** — Ent has no built-in @> operator for JSONB array containment; the raw predicate is the only way to express 'find rows whose app_mappings array contains this element'.

## Example: Add a new mutating Repository method following the existing pattern

```
import (
	"context"

	taxcodedb "github.com/openmeterio/openmeter/openmeter/ent/db/taxcode"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) ArchiveTaxCode(ctx context.Context, input taxcode.ArchiveTaxCodeInput) (taxcode.TaxCode, error) {
	if err := input.Validate(); err != nil {
		return taxcode.TaxCode{}, err
	}
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (taxcode.TaxCode, error) {
		// ... Ent mutation ...
	})
// ...
```

<!-- archie:ai-end -->
