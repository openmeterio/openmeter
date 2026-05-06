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

	return transaction.Run(ctx, s.orgDefaultsAdapter, func(ctx context.Context) (taxcode.OrganizationDefaultTaxCodes, error) {
		return s.orgDefaultsAdapter.GetOrganizationDefaultTaxCodes(ctx, input)
	})
}

func (s *Service) UpsertOrganizationDefaultTaxCodes(ctx context.Context, input taxcode.UpsertOrganizationDefaultTaxCodesInput) (taxcode.OrganizationDefaultTaxCodes, error) {
	if err := input.Validate(); err != nil {
		return taxcode.OrganizationDefaultTaxCodes{}, err
	}

	return transaction.Run(ctx, s.orgDefaultsAdapter, func(ctx context.Context) (taxcode.OrganizationDefaultTaxCodes, error) {
		// Ensure both tax code IDs belong to the namespace.
		if _, err := s.GetTaxCode(ctx, taxcode.GetTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: input.Namespace, ID: input.InvoicingTaxCodeID},
		}); err != nil {
			return taxcode.OrganizationDefaultTaxCodes{}, err
		}

		if _, err := s.GetTaxCode(ctx, taxcode.GetTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: input.Namespace, ID: input.CreditGrantTaxCodeID},
		}); err != nil {
			return taxcode.OrganizationDefaultTaxCodes{}, err
		}

		return s.orgDefaultsAdapter.UpsertOrganizationDefaultTaxCodes(ctx, input)
	})
}
