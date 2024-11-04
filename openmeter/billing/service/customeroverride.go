package billingservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.CustomerOverrideService = (*Service)(nil)

func (s *Service) CreateCustomerOverride(ctx context.Context, input billing.CreateCustomerOverrideInput) (*billingentity.CustomerOverride, error) {
	if err := input.Validate(); err != nil {
		return nil, billingentity.ValidationError{
			Err: err,
		}
	}

	adapterOverride, err := entutils.TransactingRepo(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) (*billingentity.CustomerOverride, error) {
		existingOverride, err := txAdapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Customer: customerentity.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},

			IncludeDeleted: true,
		})
		if err != nil {
			return nil, err
		}

		// The user doesn't specified a profile, let's use the default
		if input.ProfileID == "" {
			defaultProfile, err := txAdapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
				Namespace: input.Namespace,
			})
			if err != nil {
				return nil, err
			}

			if defaultProfile == nil {
				return nil, billingentity.NotFoundError{
					Entity: billingentity.EntityDefaultProfile,
					Err:    billingentity.ErrDefaultProfileNotFound,
				}
			}

			input.ProfileID = defaultProfile.ID
		}

		if existingOverride != nil {
			// We have an existing override, let's rather update it
			return txAdapter.UpdateCustomerOverride(ctx, billing.UpdateCustomerOverrideAdapterInput{
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

		return txAdapter.CreateCustomerOverride(ctx, input)
	})
	if err != nil {
		return nil, err
	}

	return s.resolveCustomerOverride(ctx, adapterOverride)
}

func (s *Service) UpdateCustomerOverride(ctx context.Context, input billing.UpdateCustomerOverrideInput) (*billingentity.CustomerOverride, error) {
	if err := input.Validate(); err != nil {
		return nil, billingentity.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepo(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) (*billingentity.CustomerOverride, error) {
		existingOverride, err := txAdapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Customer: customerentity.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},
		})
		if err != nil {
			return nil, err
		}

		if existingOverride == nil {
			return nil, billingentity.NotFoundError{
				ID:     input.CustomerID,
				Entity: billingentity.EntityCustomerOverride,
				Err:    billingentity.ErrCustomerOverrideNotFound,
			}
		}

		if !existingOverride.UpdatedAt.Equal(input.UpdatedAt) {
			return nil, billingentity.UpdateAfterDeleteError{
				Err: billingentity.ErrCustomerOverrideConflict,
			}
		}

		override, err := txAdapter.UpdateCustomerOverride(ctx, billing.UpdateCustomerOverrideAdapterInput{
			UpdateCustomerOverrideInput: input,
		})
		if err != nil {
			return nil, err
		}

		return s.resolveCustomerOverride(ctx, override)
	})
}

func (s *Service) GetCustomerOverride(ctx context.Context, input billing.GetCustomerOverrideInput) (*billingentity.CustomerOverride, error) {
	if err := input.Validate(); err != nil {
		return nil, billingentity.ValidationError{
			Err: err,
		}
	}

	adapterOverride, err := s.adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
		Customer: customerentity.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		},

		IncludeDeleted: false,
	})
	if err != nil {
		return nil, err
	}

	if adapterOverride == nil {
		return nil, billingentity.NotFoundError{
			ID:     input.CustomerID,
			Entity: billingentity.EntityCustomerOverride,
			Err:    billingentity.ErrCustomerOverrideNotFound,
		}
	}

	return s.resolveCustomerOverride(ctx, adapterOverride)
}

func (s *Service) DeleteCustomerOverride(ctx context.Context, input billing.DeleteCustomerOverrideInput) error {
	if err := input.Validate(); err != nil {
		return billingentity.ValidationError{
			Err: err,
		}
	}

	return entutils.TransactingRepoWithNoValue(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) error {
		existingOverride, err := txAdapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Customer: customerentity.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},

			IncludeDeleted: true,
		})
		if err != nil {
			return err
		}

		if existingOverride == nil {
			return billingentity.NotFoundError{
				ID:     input.CustomerID,
				Entity: billingentity.EntityCustomerOverride,
				Err:    billingentity.ErrCustomerOverrideNotFound,
			}
		}

		if existingOverride.DeletedAt != nil {
			return billingentity.NotFoundError{
				ID:     input.CustomerID,
				Entity: billingentity.EntityCustomerOverride,
				Err:    billingentity.ErrCustomerOverrideAlreadyDeleted,
			}
		}

		return txAdapter.DeleteCustomerOverride(ctx, input)
	})
}

func (s *Service) GetProfileWithCustomerOverride(ctx context.Context, input billing.GetProfileWithCustomerOverrideInput) (*billingentity.ProfileWithCustomerDetails, error) {
	return entutils.TransactingRepo(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) (*billingentity.ProfileWithCustomerDetails, error) {
		return s.getProfileWithCustomerOverride(ctx, txAdapter, input)
	})
}

func (s *Service) getProfileWithCustomerOverride(ctx context.Context, adapter billing.Adapter, input billing.GetProfileWithCustomerOverrideInput) (*billingentity.ProfileWithCustomerDetails, error) {
	if err := input.Validate(); err != nil {
		return nil, billingentity.ValidationError{
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

	return &billingentity.ProfileWithCustomerDetails{
		Profile:  *billingProfileWithOverrides,
		Customer: *customer,
	}, nil
}

// getProfileWithCustomerOverrideMerges fetches the billing profile with the customer specific overrides applied,
// if any. If there are no overrides, it returns the default billing profile.
//
// This function does not perform validations or customer entity overrides.
func (s *Service) getProfileWithCustomerOverrideMerges(ctx context.Context, adapter billing.Adapter, input billing.GetProfileWithCustomerOverrideInput) (*billingentity.Profile, error) {
	adapterOverride, err := adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
		Customer: customerentity.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		},
	})
	if err != nil {
		return nil, err
	}

	override, err := s.resolveCustomerOverride(ctx, adapterOverride)
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
			return nil, billingentity.NotFoundError{
				Entity: billingentity.EntityDefaultProfile,
				Err:    billingentity.ErrDefaultProfileNotFound,
			}
		}

		return s.resolveProfileApps(ctx, defaultProfile)
	}

	// We have an active override, let's see what's the baseline profile
	baselineProfile := override.Profile
	if baselineProfile == nil {
		// Let's fetch the default billing profile
		defaultBaseProfile, err := adapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
			Namespace: input.Namespace,
		})
		if err != nil {
			return nil, err
		}

		baselineProfile, err = s.resolveProfileApps(ctx, defaultBaseProfile)
		if err != nil {
			return nil, err
		}

		if baselineProfile == nil {
			return nil, billingentity.NotFoundError{
				Entity: billingentity.EntityDefaultProfile,
				Err:    billingentity.ErrDefaultProfileNotFound,
			}
		}
	}

	// We have the patches and the profile, let's merge them
	profile := baselineProfile.Merge(override)

	if err := profile.Validate(); err != nil {
		return nil, billingentity.ValidationError{
			Err: err,
		}
	}

	return &profile, nil
}

func (s *Service) resolveCustomerOverride(ctx context.Context, input *billingentity.CustomerOverride) (*billingentity.CustomerOverride, error) {
	if input == nil {
		return nil, nil
	}

	out := *input

	if input.Profile != nil {
		profile, err := s.resolveProfileApps(ctx, &input.Profile.BaseProfile)
		if err != nil {
			return nil, err
		}

		out.Profile = profile
	}

	return &out, nil
}
