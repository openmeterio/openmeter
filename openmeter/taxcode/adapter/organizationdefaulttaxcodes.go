package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	orgdefaultsdb "github.com/openmeterio/openmeter/openmeter/ent/db/organizationdefaulttaxcodes"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ taxcode.OrganizationDefaultTaxCodesRepository = (*adapter)(nil)

// NewOrganizationDefaultTaxCodesAdapter creates an adapter satisfying OrganizationDefaultTaxCodesRepository.
func NewOrganizationDefaultTaxCodesAdapter(config Config) (taxcode.OrganizationDefaultTaxCodesRepository, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db:     config.Client,
		logger: config.Logger,
	}, nil
}

func (a *adapter) GetOrganizationDefaultTaxCodes(ctx context.Context, input taxcode.GetOrganizationDefaultTaxCodesInput) (taxcode.OrganizationDefaultTaxCodes, error) {
	if err := input.Validate(); err != nil {
		return taxcode.OrganizationDefaultTaxCodes{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (taxcode.OrganizationDefaultTaxCodes, error) {
		entity, err := a.db.OrganizationDefaultTaxCodes.Query().
			Where(orgdefaultsdb.NamespaceEQ(input.Namespace)).
			Where(orgdefaultsdb.DeletedAtIsNil()).
			WithInvoicingTaxCode().
			WithCreditGrantTaxCode().
			Only(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return taxcode.OrganizationDefaultTaxCodes{}, taxcode.NewOrganizationDefaultTaxCodesNotFoundError(input.Namespace)
			}

			return taxcode.OrganizationDefaultTaxCodes{}, fmt.Errorf("failed to get organization default tax codes: %w", err)
		}

		return mapOrganizationDefaultTaxCodesFromEntity(entity)
	})
}

func (a *adapter) UpsertOrganizationDefaultTaxCodes(ctx context.Context, input taxcode.UpsertOrganizationDefaultTaxCodesInput) (taxcode.OrganizationDefaultTaxCodes, error) {
	if err := input.Validate(); err != nil {
		return taxcode.OrganizationDefaultTaxCodes{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (taxcode.OrganizationDefaultTaxCodes, error) {
		existing, err := a.db.OrganizationDefaultTaxCodes.Query().
			Where(orgdefaultsdb.NamespaceEQ(input.Namespace)).
			Where(orgdefaultsdb.DeletedAtIsNil()).
			Only(ctx)
		if err != nil && !db.IsNotFound(err) {
			return taxcode.OrganizationDefaultTaxCodes{}, fmt.Errorf("failed to query organization default tax codes: %w", err)
		}

		if existing != nil {
			_, err = a.db.OrganizationDefaultTaxCodes.UpdateOneID(existing.ID).
				SetInvoicingTaxCodeID(input.InvoicingTaxCodeID).
				SetCreditGrantTaxCodeID(input.CreditGrantTaxCodeID).
				Save(ctx)
			if err != nil {
				if db.IsConstraintError(err) {
					return taxcode.OrganizationDefaultTaxCodes{}, models.NewGenericConflictError(fmt.Errorf("invalid tax code reference"))
				}

				return taxcode.OrganizationDefaultTaxCodes{}, fmt.Errorf("failed to update organization default tax codes: %w", err)
			}
		} else {
			_, err = a.db.OrganizationDefaultTaxCodes.Create().
				SetNamespace(input.Namespace).
				SetInvoicingTaxCodeID(input.InvoicingTaxCodeID).
				SetCreditGrantTaxCodeID(input.CreditGrantTaxCodeID).
				Save(ctx)
			if err != nil {
				if db.IsConstraintError(err) {
					return taxcode.OrganizationDefaultTaxCodes{}, models.NewGenericConflictError(fmt.Errorf("invalid tax code reference"))
				}

				return taxcode.OrganizationDefaultTaxCodes{}, fmt.Errorf("failed to create organization default tax codes: %w", err)
			}
		}

		return a.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{
			Namespace: input.Namespace,
		})
	})
}

func mapOrganizationDefaultTaxCodesFromEntity(entity *db.OrganizationDefaultTaxCodes) (taxcode.OrganizationDefaultTaxCodes, error) {
	invoicingEdge, err := entity.Edges.InvoicingTaxCodeOrErr()
	if err != nil {
		return taxcode.OrganizationDefaultTaxCodes{}, fmt.Errorf("failed to load invoicing_tax_code edge: %w", err)
	}

	creditGrantEdge, err := entity.Edges.CreditGrantTaxCodeOrErr()
	if err != nil {
		return taxcode.OrganizationDefaultTaxCodes{}, fmt.Errorf("failed to load credit_grant_tax_code edge: %w", err)
	}

	invoicingTaxCode, err := MapTaxCodeFromEntity(invoicingEdge)
	if err != nil {
		return taxcode.OrganizationDefaultTaxCodes{}, err
	}

	creditGrantTaxCode, err := MapTaxCodeFromEntity(creditGrantEdge)
	if err != nil {
		return taxcode.OrganizationDefaultTaxCodes{}, err
	}

	return taxcode.OrganizationDefaultTaxCodes{
		NamespacedID: models.NamespacedID{
			Namespace: entity.Namespace,
			ID:        entity.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		InvoicingTaxCode:   invoicingTaxCode,
		CreditGrantTaxCode: creditGrantTaxCode,
	}, nil
}
