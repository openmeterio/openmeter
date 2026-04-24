# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing taxcode.Repository. Owns all DB reads and writes for TaxCode entities including a raw JSONB containment query for app-mapping lookups.

## Patterns

**TransactingRepo on every write** — Every mutating method (Create, Update, Delete) wraps its body in entutils.TransactingRepo or TransactingRepoWithNoValue so the ctx-bound Ent transaction is honored. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (taxcode.TaxCode, error) { ... })`)
**Soft-delete via SetDeletedAt** — DeleteTaxCode sets deleted_at to clock.Now() rather than issuing a hard DELETE; read queries filter with Where(taxcodedb.DeletedAtIsNil()). (`a.db.TaxCode.UpdateOneID(input.ID).SetDeletedAt(clock.Now()).Exec(ctx)`)
**WithTx/Self/Tx triad for transaction rebinding** — adapter implements Tx(ctx), WithTx(ctx, tx *TxDriver) *adapter, and Self() *adapter to satisfy entutils.TransactingRepo's TxCreator contract. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Input.Validate() called before any DB access** — Every exported method calls input.Validate() as its first statement and returns early on error. (`if err := input.Validate(); err != nil { return taxcode.TaxCode{}, err }`)
**JSONB containment query via raw sql.Selector** — GetTaxCodeByAppMapping uses a raw sql.P predicate (app_mappings @> '...') because Ent has no native JSONB containment operator. (`s.Where(sql.P(func(b *sql.Builder) { b.Ident(taxcodedb.FieldAppMappings).WriteString(" @> ").Arg(string(pattern)) }))`)
**Constraint error mapped to GenericConflictError** — db.IsConstraintError(err) on create maps to models.NewGenericConflictError; db.IsNotFound maps to domain-specific NewTaxCodeNotFoundError. (`if db.IsConstraintError(err) { return taxcode.TaxCode{}, models.NewGenericConflictError(...) }`)
**Config struct with models.Validator compile-time assertion** — Config implements models.Validator via var _ models.Validator = (*Config)(nil); New() calls config.Validate() before constructing the adapter. (`var _ models.Validator = (*Config)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter constructor, Config validation, Tx/WithTx/Self triad. | Must implement all three transaction rebinding methods or entutils.TransactingRepo will fail to compile. |
| `taxcode.go` | All five CRUD operations plus GetTaxCodeByAppMapping implementing taxcode.Repository. | GetTaxCodeByAppMapping uses a raw JSONB @> predicate; if the taxcode.TaxCodeAppMapping struct fields change the marshal output must stay compatible with the stored column format. |
| `mapping.go` | Single MapTaxCodeFromEntity function translating *db.TaxCode to taxcode.TaxCode domain type. | lo.FromPtr(entity.AppMappings) silently returns a zero-value slice when the column is NULL — ensure new nullable fields get the same nil-safe treatment. |

## Anti-Patterns

- Calling a.db.TaxCode.* directly in a helper without wrapping in entutils.TransactingRepo — the helper falls off the ctx-bound transaction.
- Hard-deleting rows with .Remove() instead of setting deleted_at via SetDeletedAt.
- Adding raw SQL outside a sql.P / sql.Selector wrapper — bypasses Ent's query builder and breaks dialect portability.
- Returning a non-domain error (e.g. raw ent error) without mapping via db.IsNotFound / db.IsConstraintError.
- Skipping input.Validate() to save a round-trip — callers in the service layer also validate, but the adapter is the last defense before the DB.

## Decisions

- **entutils.TransactingRepo used on every write, not only explicit tx callers.** — Ent transactions are carried implicitly in ctx; any helper that holds a raw *entdb.Client falls off the transaction unless it rebinds via TransactingRepo.
- **JSONB containment query via raw sql.Selector for GetTaxCodeByAppMapping.** — Ent has no built-in @> operator for JSONB array containment; the raw predicate is the only way to express 'find rows whose app_mappings array contains this element'.

## Example: Add a new mutating Repository method following the existing pattern.

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	taxcodedb "github.com/openmeterio/openmeter/openmeter/ent/db/taxcode"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) ArchiveTaxCode(ctx context.Context, input taxcode.ArchiveTaxCodeInput) (taxcode.TaxCode, error) {
	if err := input.Validate(); err != nil {
		return taxcode.TaxCode{}, err
	}

// ...
```

<!-- archie:ai-end -->
