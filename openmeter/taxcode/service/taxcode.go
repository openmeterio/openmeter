package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s *Service) CreateTaxCode(ctx context.Context, input taxcode.CreateTaxCodeInput) (taxcode.TaxCode, error) {
	if err := input.Validate(); err != nil {
		return taxcode.TaxCode{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) {
		tc, err := s.adapter.CreateTaxCode(ctx, input)
		if err != nil {
			return taxcode.TaxCode{}, err
		}

		if err = s.hooks.PostCreate(ctx, &tc); err != nil {
			return taxcode.TaxCode{}, err
		}

		// TODO: add event publishing

		return tc, nil
	})
}

func (s *Service) UpdateTaxCode(ctx context.Context, input taxcode.UpdateTaxCodeInput) (taxcode.TaxCode, error) {
	if err := input.Validate(); err != nil {
		return taxcode.TaxCode{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) {
		tc, err := s.adapter.GetTaxCode(ctx, taxcode.GetTaxCodeInput{NamespacedID: input.NamespacedID})
		if err != nil {
			return taxcode.TaxCode{}, err
		}

		if tc.IsDeleted() {
			return taxcode.TaxCode{}, models.NewGenericNotFoundError(taxcode.ErrTaxCodeNotFound)
		}

		if tc.IsManagedBySystem() && !input.AllowAnnotations {
			return taxcode.TaxCode{}, models.NewGenericConflictError(taxcode.ErrTaxCodeManagedBySystem)
		}

		if err = s.hooks.PreUpdate(ctx, &tc); err != nil {
			return taxcode.TaxCode{}, err
		}

		tc, err = s.adapter.UpdateTaxCode(ctx, input)
		if err != nil {
			return taxcode.TaxCode{}, err
		}

		if err = s.hooks.PostUpdate(ctx, &tc); err != nil {
			return taxcode.TaxCode{}, err
		}

		// TODO: add event publishing

		return tc, nil
	})
}

func (s *Service) ListTaxCodes(ctx context.Context, input taxcode.ListTaxCodesInput) (pagination.Result[taxcode.TaxCode], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[taxcode.TaxCode]{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[taxcode.TaxCode], error) {
		return s.adapter.ListTaxCodes(ctx, input)
	})
}

func (s *Service) GetTaxCode(ctx context.Context, input taxcode.GetTaxCodeInput) (taxcode.TaxCode, error) {
	if err := input.Validate(); err != nil {
		return taxcode.TaxCode{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) {
		return s.adapter.GetTaxCode(ctx, input)
	})
}

func (s *Service) GetTaxCodeByAppMapping(ctx context.Context, input taxcode.GetTaxCodeByAppMappingInput) (taxcode.TaxCode, error) {
	if err := input.Validate(); err != nil {
		return taxcode.TaxCode{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) {
		return s.adapter.GetTaxCodeByAppMapping(ctx, input)
	})
}

// GetOrCreateByAppMapping looks up a TaxCode by its app mapping. If none exists,
// it creates one with a key derived from the app-specific code.
func (s *Service) GetOrCreateByAppMapping(ctx context.Context, input taxcode.GetOrCreateByAppMappingInput) (taxcode.TaxCode, error) {
	if err := input.Validate(); err != nil {
		return taxcode.TaxCode{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (taxcode.TaxCode, error) {
		// Try to find an existing TaxCode with this app mapping.
		tc, err := s.adapter.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput(input))
		if err != nil && !taxcode.IsTaxCodeNotFoundError(err) {
			return taxcode.TaxCode{}, err
		}

		if err == nil { // If taxcode is returned let's just return it to the caller
			return tc, nil
		}

		// Not found — create a new TaxCode.
		key := fmt.Sprintf("%s_%s", input.AppType, input.TaxCode)

		tc, err = s.adapter.CreateTaxCode(ctx, taxcode.CreateTaxCodeInput{
			Namespace: input.Namespace,
			Key:       key,
			Name:      input.TaxCode,
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: input.AppType, TaxCode: input.TaxCode},
			},
		})
		if err != nil {
			// Another request may have created it concurrently.
			if models.IsGenericConflictError(err) {
				tc, retryErr := s.adapter.GetTaxCodeByAppMapping(ctx, taxcode.GetTaxCodeByAppMappingInput(input))
				if retryErr != nil {
					if taxcode.IsTaxCodeNotFoundError(retryErr) {
						// The key derived from this Stripe code exists but its app mapping was changed
						// after auto-creation (orphaned key). Avoid poisoning the pg tx.
						return taxcode.TaxCode{}, fmt.Errorf("resolving orphaned tax code key for %q: %w", input.TaxCode, taxcode.ErrTaxCodeOrphanedKey)
					}
					return taxcode.TaxCode{}, retryErr
				}
				return tc, nil
			}

			return taxcode.TaxCode{}, err
		}

		if err = s.hooks.PostCreate(ctx, &tc); err != nil {
			return taxcode.TaxCode{}, err
		}

		return tc, nil
	})
}

func (s *Service) DeleteTaxCode(ctx context.Context, input taxcode.DeleteTaxCodeInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		existing, err := s.adapter.GetTaxCode(ctx, taxcode.GetTaxCodeInput{NamespacedID: input.NamespacedID})
		if err != nil {
			return err
		}

		if existing.IsDeleted() {
			return nil
		}

		if existing.IsManagedBySystem() && !input.AllowAnnotations {
			return models.NewGenericConflictError(taxcode.ErrTaxCodeManagedBySystem)
		}

		defaults, err := s.adapter.GetOrganizationDefaultTaxCodes(ctx, taxcode.GetOrganizationDefaultTaxCodesInput{Namespace: input.NamespacedID.Namespace})
		if err != nil {
			return err
		}

		if defaults.CreditGrantTaxCodeID == existing.ID || defaults.InvoicingTaxCodeID == existing.ID {
			return models.NewGenericConflictError(taxcode.ErrTaxCodeIsOrganizationDefault)
		}

		if err = s.hooks.PreDelete(ctx, &existing); err != nil {
			return err
		}

		err = s.adapter.DeleteTaxCode(ctx, input)
		if err != nil {
			return err
		}

		deleted, err := s.adapter.GetTaxCode(ctx, taxcode.GetTaxCodeInput{NamespacedID: input.NamespacedID})
		if err != nil {
			return err
		}

		if err = s.hooks.PostDelete(ctx, &deleted); err != nil {
			return err
		}

		return nil
	})
}
