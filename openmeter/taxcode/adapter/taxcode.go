package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	taxcodedb "github.com/openmeterio/openmeter/openmeter/ent/db/taxcode"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (a *adapter) CreateTaxCode(ctx context.Context, input taxcode.CreateTaxCodeInput) (taxcode.TaxCode, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (taxcode.TaxCode, error) {
		if err := input.Validate(); err != nil {
			return taxcode.TaxCode{}, err
		}

		query := a.db.TaxCode.Create().
			SetNamespace(input.Namespace).
			SetKey(input.Key).
			SetName(input.Name).
			SetNillableDescription(input.Description).
			SetMetadata(input.Metadata)

		if len(input.AppMappings) > 0 {
			query = query.SetAppMappings(&input.AppMappings)
		}

		entity, err := query.Save(ctx)
		if err != nil {
			if db.IsConstraintError(err) {
				return taxcode.TaxCode{}, models.NewGenericConflictError(fmt.Errorf("tax code with the same key already exists"))
			}

			return taxcode.TaxCode{}, fmt.Errorf("failed to create tax code: %w", err)
		}

		return mapTaxCodeFromEntity(entity)
	})
}

func (a *adapter) UpdateTaxCode(ctx context.Context, input taxcode.UpdateTaxCodeInput) (taxcode.TaxCode, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (taxcode.TaxCode, error) {
		if err := input.Validate(); err != nil {
			return taxcode.TaxCode{}, err
		}

		query := a.db.TaxCode.UpdateOneID(input.ID).
			Where(taxcodedb.NamespaceEQ(input.Namespace)).
			Where(taxcodedb.DeletedAtIsNil()).
			SetName(input.Name).
			SetNillableDescription(input.Description).
			SetMetadata(input.Metadata)

		if len(input.AppMappings) > 0 {
			query = query.SetAppMappings(&input.AppMappings)
		} else {
			query = query.ClearAppMappings()
		}

		entity, err := query.Save(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return taxcode.TaxCode{}, taxcode.NewTaxCodeNotFoundError(input.ID)
			}

			return taxcode.TaxCode{}, fmt.Errorf("failed to update tax code: %w", err)
		}

		return mapTaxCodeFromEntity(entity)
	})
}

func (a *adapter) ListTaxCodes(ctx context.Context, input taxcode.ListTaxCodesInput) (pagination.Result[taxcode.TaxCode], error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (pagination.Result[taxcode.TaxCode], error) {
		if err := input.Validate(); err != nil {
			return pagination.Result[taxcode.TaxCode]{}, err
		}

		entities, err := a.db.TaxCode.Query().Where(taxcodedb.Namespace(input.Namespace)).Where(taxcodedb.DeletedAtIsNil()).Paginate(ctx, input.Page)
		if err != nil {
			return pagination.Result[taxcode.TaxCode]{}, fmt.Errorf("failed to list tax codes: %w", err)
		}

		return pagination.MapResultErr(entities, mapTaxCodeFromEntity)
	})
}

func (a *adapter) GetTaxCode(ctx context.Context, input taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (taxcode.TaxCode, error) {
		if err := input.Validate(); err != nil {
			return taxcode.TaxCode{}, err
		}

		entity, err := a.db.TaxCode.Query().Where(taxcodedb.Namespace(input.Namespace)).Where(taxcodedb.ID(input.ID)).Where(taxcodedb.DeletedAtIsNil()).Only(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return taxcode.TaxCode{}, taxcode.NewTaxCodeNotFoundError(input.ID)
			}

			return taxcode.TaxCode{}, fmt.Errorf("failed to get tax code: %w", err)
		}

		return mapTaxCodeFromEntity(entity)
	})
}

func (a *adapter) DeleteTaxCode(ctx context.Context, input taxcode.DeleteTaxCodeInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, a *adapter) error {
		if err := input.Validate(); err != nil {
			return err
		}

		err := a.db.TaxCode.UpdateOneID(input.ID).Where(taxcodedb.Namespace(input.Namespace)).Where(taxcodedb.DeletedAtIsNil()).SetDeletedAt(clock.Now()).Exec(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return taxcode.NewTaxCodeNotFoundError(input.ID)
			}

			return fmt.Errorf("failed to delete tax code: %w", err)
		}

		return nil
	})
}
