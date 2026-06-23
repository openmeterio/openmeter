package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *Service) GetOrganizationDefaultTaxCodes(ctx context.Context, input taxcode.GetOrganizationDefaultTaxCodesInput) (taxcode.OrganizationDefaultTaxCodes, error) {
	if err := input.Validate(); err != nil {
		return taxcode.OrganizationDefaultTaxCodes{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.OrganizationDefaultTaxCodes, error) {
		return s.adapter.GetOrganizationDefaultTaxCodes(ctx, input)
	})
}

func (s *Service) UpsertOrganizationDefaultTaxCodes(ctx context.Context, input taxcode.UpsertOrganizationDefaultTaxCodesInput) (taxcode.OrganizationDefaultTaxCodes, error) {
	if err := input.Validate(); err != nil {
		return taxcode.OrganizationDefaultTaxCodes{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.OrganizationDefaultTaxCodes, error) {
		// Ensure both tax code IDs belong to the namespace and are not soft-deleted.
		if err := s.requireActiveTaxCode(ctx, input.Namespace, input.InvoicingTaxCodeID); err != nil {
			return taxcode.OrganizationDefaultTaxCodes{}, err
		}

		if err := s.requireActiveTaxCode(ctx, input.Namespace, input.CreditGrantTaxCodeID); err != nil {
			return taxcode.OrganizationDefaultTaxCodes{}, err
		}

		return s.adapter.UpsertOrganizationDefaultTaxCodes(ctx, input)
	})
}

// requireActiveTaxCode ensures the tax code belongs to the namespace and is not soft-deleted.
// GetTaxCode returns soft-deleted rows by ID (so billing can still resolve frozen Stripe
// mappings), so a deleted code must be rejected explicitly to prevent it from being
// designated as an organization default.
func (s *Service) requireActiveTaxCode(ctx context.Context, namespace, id string) error {
	tc, err := s.GetTaxCode(ctx, taxcode.GetTaxCodeInput{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: id},
	})
	if err != nil {
		return err
	}

	if tc.DeletedAt != nil {
		return taxcode.NewTaxCodeNotFoundError(id)
	}

	return nil
}
