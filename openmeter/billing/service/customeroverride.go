package billingservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

var _ billing.CustomerOverrideService = (*Service)(nil)

func (s *Service) CreateCustomerOverride(ctx context.Context, input billing.CreateCustomerOverrideInput) (*billing.CustomerOverride, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return billing.WithTx(ctx, s.adapter, func(ctx context.Context, adapter billing.TxAdapter) (*billing.CustomerOverride, error) {
		existingOverride, err := adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Namespace:  input.Namespace,
			CustomerID: input.CustomerID,

			IncludeDeleted: true,
		})
		if err != nil {
			return nil, err
		}

		if existingOverride != nil {
			// We have an existing override, let's rather update it
			return adapter.UpdateCustomerOverride(ctx, billing.UpdateCustomerOverrideAdapterInput{
				UpdateCustomerOverrideInput: billing.UpdateCustomerOverrideInput{
					Namespace:  input.Namespace,
					CustomerID: input.CustomerID,

					ProfileID: input.ProfileID,
					UpdatedAt: existingOverride.UpdatedAt,

					Collection: input.Collection,
					Invoicing:  input.Invoicing,
					Payment:    input.Payment,
				},
				ResetDeletedAt: true,
			})
		}

		return adapter.CreateCustomerOverride(ctx, input)
	})
}

func (s *Service) UpdateCustomerOverride(ctx context.Context, input billing.UpdateCustomerOverrideInput) (*billing.CustomerOverride, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return billing.WithTx(ctx, s.adapter, func(ctx context.Context, adapter billing.TxAdapter) (*billing.CustomerOverride, error) {
		existingOverride, err := adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Namespace:  input.Namespace,
			CustomerID: input.CustomerID,
		})
		if err != nil {
			return nil, err
		}

		if existingOverride == nil {
			return nil, billing.NotFoundError{
				ID:     input.CustomerID,
				Entity: billing.EntityCustomerOverride,
				Err:    billing.ErrCustomerOverrideNotFound,
			}
		}

		if !existingOverride.UpdatedAt.Equal(input.UpdatedAt) {
			return nil, billing.UpdateAfterDeleteError{
				Err: billing.ErrCustomerOverrideConflict,
			}
		}

		return adapter.UpdateCustomerOverride(ctx, billing.UpdateCustomerOverrideAdapterInput{
			UpdateCustomerOverrideInput: input,
		})
	})
}

func (s *Service) GetCustomerOverride(ctx context.Context, input billing.GetCustomerOverrideInput) (*billing.CustomerOverride, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	override, err := s.adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
		Namespace:  input.Namespace,
		CustomerID: input.CustomerID,

		IncludeDeleted: false,
	})
	if err != nil {
		return nil, err
	}

	if override == nil {
		return nil, billing.NotFoundError{
			ID:     input.CustomerID,
			Entity: billing.EntityCustomerOverride,
			Err:    billing.ErrCustomerOverrideNotFound,
		}
	}

	return override, nil
}

func (s *Service) DeleteCustomerOverride(ctx context.Context, input billing.DeleteCustomerOverrideInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	return billing.WithTxNoValue(ctx, s.adapter, func(ctx context.Context, adapter billing.TxAdapter) error {
		existingOverride, err := adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Namespace:  input.Namespace,
			CustomerID: input.CustomerID,

			IncludeDeleted: true,
		})
		if err != nil {
			return err
		}

		if existingOverride == nil {
			return billing.NotFoundError{
				ID:     input.CustomerID,
				Entity: billing.EntityCustomerOverride,
				Err:    billing.ErrCustomerOverrideNotFound,
			}
		}

		if existingOverride.DeletedAt != nil {
			return billing.NotFoundError{
				ID:     input.CustomerID,
				Entity: billing.EntityCustomerOverride,
				Err:    billing.ErrCustomerOverrideAlreadyDeleted,
			}
		}

		return adapter.DeleteCustomerOverride(ctx, input)
	})
}

func (s *Service) GetProfileWithCustomerOverride(ctx context.Context, input billing.GetProfileWithCustomerOverrideInput) (*billing.ProfileWithCustomerDetails, error) {
	return billing.WithTx(ctx, s.adapter, func(ctx context.Context, adapter billing.TxAdapter) (*billing.ProfileWithCustomerDetails, error) {
		return s.getProfileWithCustomerOverride(ctx, adapter, input)
	})
}

func (s *Service) getProfileWithCustomerOverride(ctx context.Context, adapter billing.TxAdapter, input billing.GetProfileWithCustomerOverrideInput) (*billing.ProfileWithCustomerDetails, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	// TODO[later]: We need cross service transactions to include this in the same transaction as the calculation itself
	customer, err := s.customerService.GetCustomer(ctx, customerentity.GetCustomerInput{
		Namespace: input.Namespace,
		ID:        input.CustomerID,
	})
	if err != nil {
		// This popagates the not found error
		return nil, err
	}

	billingProfileWithOverrides, err := s.getProfileWithCustomerOverrideMerges(ctx, adapter, input)
	if err != nil {
		return nil, err
	}

	// Let's apply the customer specific overrides
	if customer.Timezone != nil {
		billingProfileWithOverrides.WorkflowConfig.Timezone = customer.Timezone
	}

	return &billing.ProfileWithCustomerDetails{
		Profile:  *billingProfileWithOverrides,
		Customer: *customer,
	}, nil
}

// getProfileWithCustomerOverrideMerges fetches the billing profile with the customer specific overrides applied,
// if any. If there are no overrides, it returns the default billing profile.
//
// This function does not perform validations or customer entity overrides.
func (s *Service) getProfileWithCustomerOverrideMerges(ctx context.Context, adapter billing.TxAdapter, input billing.GetProfileWithCustomerOverrideInput) (*billing.Profile, error) {
	override, err := adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
		Namespace:  input.Namespace,
		CustomerID: input.CustomerID,
	})
	if err != nil {
		return nil, err
	}

	if override == nil || override.DeletedAt != nil {
		// Let's fetch the default billing profile
		defaultProfile, err := adapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
			Namespace: input.Namespace,
		})
		if err != nil {
			return nil, err
		}

		if defaultProfile == nil {
			return nil, billing.NotFoundError{
				Entity: billing.EntityDefaultProfile,
				Err:    billing.ErrDefaultProfileNotFound,
			}
		}

		return defaultProfile, nil
	}

	// We have an active override, let's see what's the baseline profile
	baselineProfile := override.Profile
	if baselineProfile == nil {
		// Let's fetch the default billing profile
		baselineProfile, err = adapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
			Namespace: input.Namespace,
		})
		if err != nil {
			return nil, err
		}

		if baselineProfile == nil {
			return nil, billing.NotFoundError{
				Entity: billing.EntityDefaultProfile,
				Err:    billing.ErrDefaultProfileNotFound,
			}
		}
	}

	// We have the patches and the profile, let's merge them
	profile := baselineProfile.Merge(override)

	if err := profile.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return &profile, nil
}
