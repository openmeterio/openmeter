package billingservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ billing.CustomerOverrideService = (*Service)(nil)

func (s *Service) UpsertCustomerOverride(ctx context.Context, input billing.UpsertCustomerOverrideInput) (billing.CustomerOverrideWithDetails, error) {
	def := billing.CustomerOverrideWithDetails{}

	if err := input.Validate(); err != nil {
		return def, billing.ValidationError{
			Err: err,
		}
	}

	adapterOverride, err := transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.CustomerOverrideWithDetails, error) {
		existingOverride, err := s.adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Customer: customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},

			IncludeDeleted: true,
		})
		if err != nil {
			return def, err
		}

		var upsertedOverride *billing.CustomerOverride
		if existingOverride != nil {
			// We have an existing override, let's rather update it
			upsertedOverride, err = s.adapter.UpdateCustomerOverride(ctx, input)
			if err != nil {
				return def, err
			}
		} else {
			upsertedOverride, err = s.adapter.CreateCustomerOverride(ctx, input)
			if err != nil {
				return def, err
			}
		}

		return s.resolveCustomerOverrideWithDetails(ctx, resolveCustomerOverrideWithDetailsInput{
			CustomerID: customer.CustomerID{
				Namespace: input.Namespace,
				ID:        input.CustomerID,
			},
			Override: upsertedOverride,
		})
	})
	if err != nil {
		return def, err
	}

	return adapterOverride, nil
}

func (s *Service) GetCustomerOverride(ctx context.Context, input billing.GetCustomerOverrideInput) (billing.CustomerOverrideWithDetails, error) {
	def := billing.CustomerOverrideWithDetails{}

	if err := input.Validate(); err != nil {
		return def, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.CustomerOverrideWithDetails, error) {
		if _, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{
			CustomerID: &input.Customer,
		}); err != nil {
			return def, err
		}

		adapterOverride, err := s.adapter.GetCustomerOverride(ctx, billing.GetCustomerOverrideAdapterInput{
			Customer: input.Customer,

			IncludeDeleted: false,
		})
		if err != nil {
			return def, err
		}

		return s.resolveCustomerOverrideWithDetails(ctx, resolveCustomerOverrideWithDetailsInput{
			CustomerID: input.Customer,
			Override:   adapterOverride,
			Expand:     input.Expand,
		})
	})
}

// TODO: remove this once legacy API is removed
// GetCustomerApp returns the app for a customer, it will return the first app found for the app type
func (s *Service) GetCustomerApp(ctx context.Context, input billing.GetCustomerAppInput) (app.App, error) {
	// Validate the input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Get the default billingprofile for the customer
	customerOverride, err := s.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: input.CustomerID,
		Expand: billing.CustomerOverrideExpand{
			Apps: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error getting customer billing profile: %w", err)
	}

	// Get the apps from the merged profile
	invoicingApp := customerOverride.MergedProfile.Apps.Invoicing
	paymentApp := customerOverride.MergedProfile.Apps.Payment
	taxApp := customerOverride.MergedProfile.Apps.Tax

	// Lookup if any of the apps matches the provided app type
	var resolvedApp app.App

	if invoicingApp.GetType() == input.AppType {
		resolvedApp = invoicingApp
	} else if paymentApp.GetType() == input.AppType {
		resolvedApp = paymentApp
	} else if taxApp.GetType() == input.AppType {
		resolvedApp = taxApp
	} else {
		return nil, models.NewGenericPreConditionFailedError(
			fmt.Errorf("no %s app found in billing profile for customer %s", input.AppType, input.CustomerID.ID),
		)
	}

	// For now enforce that the app type is the same for all apps
	// TODO: Remove this once we support multiple app types per billing profile
	if invoicingApp.GetType() != paymentApp.GetType() || invoicingApp.GetType() != taxApp.GetType() {
		return nil, models.NewGenericPreConditionFailedError(
			fmt.Errorf(
				"app type is not the same for all apps: invoicing: %s, payment: %s, tax: %s",
				invoicingApp.GetType(),
				paymentApp.GetType(),
				taxApp.GetType(),
			),
		)
	}

	return resolvedApp, nil
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
				Namespace: input.Customer.Namespace,
				ID:        input.Customer.ID,
			},

			IncludeDeleted: true,
		})
		if err != nil {
			return err
		}

		if existingOverride == nil {
			return billing.NotFoundError{
				ID:     input.Customer.ID,
				Entity: billing.EntityCustomerOverride,
				Err:    billing.ErrCustomerOverrideNotFound,
			}
		}

		if existingOverride.DeletedAt != nil {
			return billing.NotFoundError{
				ID:     input.Customer.ID,
				Entity: billing.EntityCustomerOverride,
				Err:    billing.ErrCustomerOverrideAlreadyDeleted,
			}
		}

		return s.adapter.DeleteCustomerOverride(ctx, input)
	})
}

func (s *Service) ListCustomerOverrides(ctx context.Context, input billing.ListCustomerOverridesInput) (billing.ListCustomerOverridesResult, error) {
	if err := input.Validate(); err != nil {
		return billing.ListCustomerOverridesResult{}, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.ListCustomerOverridesResult, error) {
		res, err := s.adapter.ListCustomerOverrides(ctx, input)
		if err != nil {
			return billing.ListCustomerOverridesResult{}, err
		}

		// For list let's not fetch the customers one by one, let's do it in a single query
		customersByID := make(map[string]customer.Customer, len(res.Items))
		if input.Expand.Customer {
			customers, err := s.customerService.ListCustomers(ctx, customer.ListCustomersInput{
				Namespace: input.Namespace,
				CustomerIDs: lo.Map(res.Items, func(override billing.CustomerOverrideWithCustomerID, _ int) string {
					return override.CustomerID.ID
				}),
			})
			if err != nil {
				return billing.ListCustomerOverridesResult{}, err
			}

			for _, c := range customers.Items {
				customersByID[c.ID] = c
			}
		}

		var defaultProfile *billing.Profile
		// Let's see if we need to fetch the default profile

		_, needDefaultProfile := lo.Find(res.Items, func(override billing.CustomerOverrideWithCustomerID) bool {
			return override.CustomerOverride == nil || override.CustomerOverride.Profile == nil
		})

		if needDefaultProfile {
			defaultProfile, err = s.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
				Namespace: input.Namespace,
			})
			if err != nil {
				return billing.ListCustomerOverridesResult{}, err
			}
		}

		return pagination.MapResultErr(res, func(override billing.CustomerOverrideWithCustomerID) (billing.CustomerOverrideWithDetails, error) {
			res, err := s.resolveCustomerOverrideWithDetails(ctx, resolveCustomerOverrideWithDetailsInput{
				CustomerID:         override.CustomerID,
				Override:           override.CustomerOverride,
				DefaultProfile:     defaultProfile,
				Expand:             input.Expand,
				CustomersByIdCache: customersByID,
			})
			if err != nil {
				return billing.CustomerOverrideWithDetails{}, err
			}

			return res, nil
		})
	})
}

type resolveCustomerOverrideWithDetailsInput struct {
	CustomerID     customer.CustomerID
	Override       *billing.CustomerOverride
	DefaultProfile *billing.Profile
	Expand         billing.CustomerOverrideExpand

	CustomersByIdCache map[string]customer.Customer
}

func (v resolveCustomerOverrideWithDetailsInput) GetCustomerFromCache(id string) (customer.Customer, bool) {
	if v.CustomersByIdCache == nil {
		return customer.Customer{}, false
	}

	customer, found := v.CustomersByIdCache[id]
	return customer, found
}

func (v *resolveCustomerOverrideWithDetailsInput) GetDefaultProfile(ctx context.Context, svc *Service) (*billing.Profile, error) {
	if v.DefaultProfile != nil {
		return v.DefaultProfile, nil
	}

	return svc.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
		Namespace: v.CustomerID.Namespace,
	})
}

func (v resolveCustomerOverrideWithDetailsInput) Validate() error {
	if err := v.CustomerID.Validate(); err != nil {
		return billing.ValidationError{
			Err: fmt.Errorf("invalid customer ID: %w", err),
		}
	}

	return nil
}

func (s *Service) resolveCustomerOverrideWithDetails(ctx context.Context, in resolveCustomerOverrideWithDetailsInput) (billing.CustomerOverrideWithDetails, error) {
	def := billing.CustomerOverrideWithDetails{}

	if err := in.Validate(); err != nil {
		return def, err
	}

	details, err := s.resolveProfileWorkflow(ctx, in)
	if err != nil {
		return def, err
	}

	if details.Expand.Customer {
		if cachedCustomer, ok := in.GetCustomerFromCache(in.CustomerID.ID); ok {
			details.Customer = &cachedCustomer
		} else {
			cust, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: &in.CustomerID,
			})
			if err != nil {
				return def, err
			}

			details.Customer = cust
		}
	}

	if details.Expand.Apps {
		profileWithApps, err := s.resolveProfileApps(ctx, &details.MergedProfile.BaseProfile)
		if err != nil {
			return def, err
		}

		details.MergedProfile.Apps = profileWithApps.Apps
	}

	return details, nil
}

// resolveProfileWorkflow resolves the profile workflow for a customer override, popuplates the baseline profile and the
// merged profile for the output.
func (s *Service) resolveProfileWorkflow(ctx context.Context, in resolveCustomerOverrideWithDetailsInput) (billing.CustomerOverrideWithDetails, error) {
	def := billing.CustomerOverrideWithDetails{}

	// If there is no override or it's deleted, let's use the default profile
	if in.Override == nil || in.Override.DeletedAt != nil {
		defaultProfile, err := in.GetDefaultProfile(ctx, s)
		if err != nil {
			return def, err
		}

		if defaultProfile == nil {
			return def, billing.NotFoundError{
				Entity: billing.EntityDefaultProfile,
				Err:    billing.ErrDefaultProfileNotFound,
			}
		}

		return billing.CustomerOverrideWithDetails{
			Expand:        in.Expand,
			MergedProfile: *defaultProfile,
		}, nil
	}

	// We have an active override, let's see what's the baseline profile
	baselineProfile := in.Override.Profile
	if baselineProfile == nil {
		// Let's fetch the default billing profile
		defaultProfile, err := in.GetDefaultProfile(ctx, s)
		if err != nil {
			return def, err
		}

		if defaultProfile == nil {
			return def, billing.NotFoundError{
				Entity: billing.EntityDefaultProfile,
				Err:    billing.ErrDefaultProfileNotFound,
			}
		}

		baselineProfile = defaultProfile
	}

	// We have the patches and the profile, let's merge them
	profile := baselineProfile.Merge(in.Override)

	if err := profile.Validate(); err != nil {
		return def, billing.ValidationError{
			Err: err,
		}
	}

	return billing.CustomerOverrideWithDetails{
		Expand:           in.Expand,
		CustomerOverride: in.Override,
		MergedProfile:    profile,
	}, nil
}
