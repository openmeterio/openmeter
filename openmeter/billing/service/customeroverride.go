package billingservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/samber/lo"
)

var _ billing.CustomerOverrideService = (*Service)(nil)

func (s *Service) CreateCustomerOverride(ctx context.Context, input billing.CreateCustomerOverrideInput) (*billing.CustomerOverride, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	adapterOverride, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (*billing.CustomerOverride, error) {
		existingOverride, err := s.adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Customer: customer.CustomerID{
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
			defaultProfile, err := s.adapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
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

			input.ProfileID = defaultProfile.ID
		}

		if existingOverride != nil {
			// We have an existing override, let's rather update it
			return s.adapter.UpdateCustomerOverride(ctx, billing.UpdateCustomerOverrideAdapterInput{
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

		return s.adapter.CreateCustomerOverride(ctx, input)
	})
	if err != nil {
		return nil, err
	}

	return s.resolveCustomerOverride(ctx, adapterOverride)
}

func (s *Service) UpdateCustomerOverride(ctx context.Context, input billing.UpdateCustomerOverrideInput) (*billing.CustomerOverride, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	existingOverride, err := s.adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
		Customer: customer.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		},
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

	override, err := s.adapter.UpdateCustomerOverride(ctx, billing.UpdateCustomerOverrideAdapterInput{
		UpdateCustomerOverrideInput: input,
	})
	if err != nil {
		return nil, err
	}

	return s.resolveCustomerOverride(ctx, override)
}

func (s *Service) GetCustomerOverride(ctx context.Context, input billing.GetCustomerOverrideInput) (*billing.CustomerOverride, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	adapterOverride, err := s.adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
		Customer: customer.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		},

		IncludeDeleted: false,
	})
	if err != nil {
		return nil, err
	}

	if adapterOverride == nil {
		return nil, billing.NotFoundError{
			ID:     input.CustomerID,
			Entity: billing.EntityCustomerOverride,
			Err:    billing.ErrCustomerOverrideNotFound,
		}
	}

	return s.resolveCustomerOverride(ctx, adapterOverride)
}

func (s *Service) DeleteCustomerOverride(ctx context.Context, input billing.DeleteCustomerOverrideInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		existingOverride, err := s.adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Customer: customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},

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

		return s.adapter.DeleteCustomerOverride(ctx, input)
	})
}

func (s *Service) GetProfileWithCustomerOverride(ctx context.Context, input billing.GetProfileWithCustomerOverrideInput) (*billing.ProfileWithCustomerDetails, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*billing.ProfileWithCustomerDetails, error) {
		return s.getProfileWithCustomerOverride(ctx, s.adapter, input)
	})
}

func (s *Service) ListCustomerOverrides(ctx context.Context, input billing.ListCustomerOverridesInput) (billing.ListCustomerOverridesResult, error) {
	if err := input.Validate(); err != nil {
		return billing.ListCustomerOverridesResult{}, billing.ValidationError{
			Err: err,
		}
	}

	res, err := s.adapter.ListCustomerOverrides(ctx, input)
	if err != nil {
		return billing.ListCustomerOverridesResult{}, err
	}

	customersByID := make(map[string]customer.Customer, len(res.Items))
	if input.Expand.Customers {
		customers, err := s.customerService.ListCustomers(ctx, customer.ListCustomersInput{
			Namespace: input.Namespace,
			CustomerIDs: lo.Map(res.Items, func(override billing.CustomerOverrideWithAdapterProfile, _ int) string {
				return override.CustomerID
			}),
		})
		if err != nil {
			return billing.ListCustomerOverridesResult{}, err
		}

		for _, c := range customers.Items {
			customersByID[c.ID] = c
		}

	}

	return pagination.MapPagedResponseError(res, func(aOverride billing.CustomerOverrideWithAdapterProfile) (billing.CustomerOverrideWithMergedProfile, error) {
		out := billing.CustomerOverrideWithMergedProfile{
			CustomerOverride: aOverride.CustomerOverride,
		}

		if input.Expand.Customers {
			customer, ok := customersByID[aOverride.CustomerID]
			if !ok {
				return billing.CustomerOverrideWithMergedProfile{}, billing.NotFoundError{
					ID:     aOverride.CustomerID,
					Entity: billing.EntityCustomer,
					Err:    billing.ErrCustomerNotFound,
				}
			}

			out.Customer = &customer
		}

		if input.Expand.ProfileWithOverrides {
			// TODO
		}

		return out, nil
	})
}

func (s *Service) getProfileWithCustomerOverride(ctx context.Context, adapter billing.Adapter, input billing.GetProfileWithCustomerOverrideInput) (*billing.ProfileWithCustomerDetails, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	// TODO[later]: We need cross service transactions to include this in the same transaction as the calculation itself
	customer, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{
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

	return &billing.ProfileWithCustomerDetails{
		Profile:  *billingProfileWithOverrides,
		Customer: *customer,
	}, nil
}

// getProfileWithCustomerOverrideMerges fetches the billing profile with the customer specific overrides applied,
// if any. If there are no overrides, it returns the default billing profile.
//
// This function does not perform validations or customer entity overrides.
func (s *Service) getProfileWithCustomerOverrideMerges(ctx context.Context, adapter billing.Adapter, input billing.GetProfileWithCustomerOverrideInput) (*billing.Profile, error) {
	adapterOverride, err := adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
		Customer: customer.CustomerID{
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
			return nil, billing.NotFoundError{
				Entity: billing.EntityDefaultProfile,
				Err:    billing.ErrDefaultProfileNotFound,
			}
		}

		return s.resolveProfileApps(ctx, defaultProfile.BaseProfileOrEmpty())
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

		baselineProfile, err = s.resolveProfileApps(ctx, defaultBaseProfile.BaseProfileOrEmpty())
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

func (s *Service) resolveCustomerOverride(ctx context.Context, input *billing.CustomerOverride) (*billing.CustomerOverride, error) {
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
