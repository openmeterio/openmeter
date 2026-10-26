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

var _ billing.CustomerBillingProfileService = (*Service)(nil)

func (s *Service) SetCustomerBillingProfile(ctx context.Context, input billing.SetCustomerBillingProfileInput) (billing.CustomerBillingProfile, error) {
	if err := input.Validate(); err != nil {
		return billing.CustomerBillingProfile{}, billing.ValidationError{Err: err}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.CustomerBillingProfile, error) {
		if _, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerID: &input.Customer}); err != nil {
			return billing.CustomerBillingProfile{}, fmt.Errorf("resolving customer for billing profile pin: %w", err)
		}

		profile, err := s.GetProfile(ctx, billing.GetProfileInput{
			Profile: billing.ProfileID{Namespace: input.Customer.Namespace, ID: input.ProfileID},
		})
		if err != nil {
			return billing.CustomerBillingProfile{}, fmt.Errorf("resolving billing profile for pin: %w", err)
		}

		if profile.DeletedAt != nil {
			return billing.CustomerBillingProfile{}, billing.NotFoundError{Err: billing.ErrProfileNotFound}
		}

		selection, err := s.adapter.SetCustomerBillingProfile(ctx, input)
		if err != nil {
			return billing.CustomerBillingProfile{}, fmt.Errorf("setting customer billing profile: %w", err)
		}

		return s.resolveCustomerBillingProfile(ctx, resolveCustomerBillingProfileInput{
			CustomerID: input.Customer,
			Selection:  selection,
		})
	})
}

func (s *Service) ClearCustomerBillingProfile(ctx context.Context, input billing.ClearCustomerBillingProfileInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{Err: err}
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		if _, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerID: &input.Customer}); err != nil {
			return fmt.Errorf("resolving customer for clearing billing profile: %w", err)
		}

		return s.adapter.ClearCustomerBillingProfile(ctx, input)
	})
}

func (s *Service) GetCustomerBillingProfile(ctx context.Context, input billing.GetCustomerBillingProfileInput) (billing.CustomerBillingProfile, error) {
	if err := input.Validate(); err != nil {
		return billing.CustomerBillingProfile{}, billing.ValidationError{Err: err}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.CustomerBillingProfile, error) {
		cust, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerID: &input.Customer})
		if err != nil {
			return billing.CustomerBillingProfile{}, err
		}

		selection, err := s.adapter.GetCustomerBillingProfile(ctx, input)
		if err != nil {
			return billing.CustomerBillingProfile{}, err
		}

		return s.resolveCustomerBillingProfile(ctx, resolveCustomerBillingProfileInput{
			CustomerID:         input.Customer,
			Selection:          selection,
			Expand:             input.Expand,
			CustomersByIdCache: map[string]customer.Customer{cust.ID: *cust},
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

	// Resolve the customer's effective profile.
	customerProfile, err := s.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
		Customer: input.CustomerID,
		Expand: billing.CustomerBillingProfileExpand{
			Apps: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error getting customer billing profile: %w", err)
	}

	// Get the apps from the effective profile
	invoicingApp := customerProfile.EffectiveProfile.Apps.Invoicing
	paymentApp := customerProfile.EffectiveProfile.Apps.Payment
	taxApp := customerProfile.EffectiveProfile.Apps.Tax

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

func (s *Service) ListCustomerBillingProfiles(ctx context.Context, input billing.ListCustomerBillingProfilesInput) (billing.ListCustomerBillingProfilesResult, error) {
	if err := input.Validate(); err != nil {
		return billing.ListCustomerBillingProfilesResult{}, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.ListCustomerBillingProfilesResult, error) {
		res, err := s.adapter.ListCustomerBillingProfiles(ctx, input)
		if err != nil {
			return billing.ListCustomerBillingProfilesResult{}, err
		}

		// For list let's not fetch the customers one by one, let's do it in a single query
		customersByID := make(map[string]customer.Customer, len(res.Items))
		if input.Expand.Customer {
			customers, err := s.customerService.ListCustomers(ctx, customer.ListCustomersInput{
				Namespace: input.Namespace,
				CustomerIDs: lo.Map(res.Items, func(selection billing.CustomerBillingProfileAdapterResult, _ int) string {
					return selection.CustomerID.ID
				}),
			})
			if err != nil {
				return billing.ListCustomerBillingProfilesResult{}, err
			}

			for _, c := range customers.Items {
				customersByID[c.ID] = c
			}
		}

		var defaultProfile *billing.Profile
		_, needDefaultProfile := lo.Find(res.Items, func(selection billing.CustomerBillingProfileAdapterResult) bool {
			return selection.AssignedProfile == nil
		})

		if needDefaultProfile {
			defaultProfile, err = s.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
				Namespace: input.Namespace,
			})
			if err != nil {
				return billing.ListCustomerBillingProfilesResult{}, err
			}
		}

		return pagination.MapResultErr(res, func(selection billing.CustomerBillingProfileAdapterResult) (billing.CustomerBillingProfile, error) {
			res, err := s.resolveCustomerBillingProfile(ctx, resolveCustomerBillingProfileInput{
				CustomerID:         selection.CustomerID,
				Selection:          selection,
				DefaultProfile:     defaultProfile,
				Expand:             input.Expand,
				CustomersByIdCache: customersByID,
			})
			if err != nil {
				return billing.CustomerBillingProfile{}, err
			}

			return res, nil
		})
	})
}

type resolveCustomerBillingProfileInput struct {
	CustomerID         customer.CustomerID
	Selection          billing.CustomerBillingProfileAdapterResult
	DefaultProfile     *billing.Profile
	Expand             billing.CustomerBillingProfileExpand
	CustomersByIdCache map[string]customer.Customer
}

func (i resolveCustomerBillingProfileInput) Validate() error {
	return i.CustomerID.Validate()
}

func (s *Service) resolveCustomerBillingProfile(ctx context.Context, input resolveCustomerBillingProfileInput) (billing.CustomerBillingProfile, error) {
	if err := input.Validate(); err != nil {
		return billing.CustomerBillingProfile{}, billing.ValidationError{Err: err}
	}

	profile := input.Selection.Profile
	if input.Selection.AssignedProfile == nil {
		profile = input.DefaultProfile
		if profile == nil {
			var err error
			profile, err = s.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{Namespace: input.CustomerID.Namespace})
			if err != nil {
				return billing.CustomerBillingProfile{}, err
			}
		}
	}

	if profile == nil {
		return billing.CustomerBillingProfile{}, billing.NotFoundError{Entity: billing.EntityDefaultProfile, Err: billing.ErrDefaultProfileNotFound}
	}

	if err := profile.Validate(); err != nil {
		return billing.CustomerBillingProfile{}, billing.ValidationError{Err: err}
	}

	result := billing.CustomerBillingProfile{
		CustomerID:       input.CustomerID,
		AssignedProfile:  input.Selection.AssignedProfile,
		EffectiveProfile: *profile,
		Expand:           input.Expand,
	}

	if input.Expand.Customer {
		if cached, ok := input.CustomersByIdCache[input.CustomerID.ID]; ok {
			result.Customer = &cached
		} else {
			cust, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerID: &input.CustomerID})
			if err != nil {
				return billing.CustomerBillingProfile{}, err
			}

			result.Customer = cust
		}
	}

	if input.Expand.Apps && result.EffectiveProfile.Apps == nil {
		expanded, err := s.resolveProfileApps(ctx, &result.EffectiveProfile.BaseProfile)
		if err != nil {
			return billing.CustomerBillingProfile{}, err
		}

		result.EffectiveProfile.Apps = expanded.Apps
	}

	return result, nil
}
