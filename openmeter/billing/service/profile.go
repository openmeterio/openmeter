package billingservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ billing.ProfileService = (*Service)(nil)

func (s *Service) CreateProfile(ctx context.Context, input billing.CreateProfileInput) (*billing.Profile, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*billing.Profile, error) {
		var demotedProfile *demoteDefaultProfileResult

		// Given that we have multiple constraints let's validate those here for better error reporting
		if input.Default {
			var err error

			demotedProfile, err = s.demoteDefaultProfile(ctx, input.Namespace)
			if err != nil {
				return nil, err
			}
		}

		input.Apps = billing.ProfileAppReferences{
			Tax:       input.Apps.Tax,
			Invoicing: input.Apps.Invoicing,
			Payment:   input.Apps.Payment,
		}

		// Resolve the apps
		taxApp, err := s.appService.GetApp(ctx, app.GetAppInput{
			Namespace: input.Namespace,
			ID:        input.Apps.Tax.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot resolve tax app: %w", err)
		}

		invoicingApp, err := s.appService.GetApp(ctx, app.GetAppInput{
			Namespace: input.Namespace,
			ID:        input.Apps.Invoicing.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot resolve invoicing app: %w", err)
		}

		paymentApp, err := s.appService.GetApp(ctx, app.GetAppInput{
			Namespace: input.Namespace,
			ID:        input.Apps.Payment.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot resolve payment app: %w", err)
		}

		if taxApp.GetType() != invoicingApp.GetType() ||
			taxApp.GetType() != paymentApp.GetType() {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("all apps must be of the same type"),
			}
		}

		profile, err := s.adapter.CreateProfile(ctx, input)
		if err != nil {
			return nil, err
		}

		if err := profile.Validate(); err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("error validating profile: %w", err),
			}
		}

		profileWithApps, err := s.resolveProfileApps(ctx, profile)
		if err != nil {
			return nil, err
		}

		if demotedProfile != nil {
			if err := s.handleDefaultProfileChange(ctx, defaultProfileChangeInput{
				Old: demotedProfile,
				New: profileWithApps,
			}); err != nil {
				return nil, err
			}
		}

		return profileWithApps, nil
	})
}

type resolvedApps struct {
	Tax       app.App
	Invoicing app.App
	Payment   app.App
}

func (s *Service) GetDefaultProfile(ctx context.Context, input billing.GetDefaultProfileInput) (*billing.Profile, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	profile, err := s.adapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
		Namespace: input.Namespace,
	})
	if err != nil {
		return nil, err
	}

	return s.resolveProfileApps(ctx, profile.BaseProfileOrEmpty())
}

func (s *Service) GetProfile(ctx context.Context, input billing.GetProfileInput) (*billing.Profile, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	profile, err := s.adapter.GetProfile(ctx, input)
	if err != nil {
		return nil, err
	}

	if input.Expand.Apps {
		return s.resolveProfileApps(ctx, profile.BaseProfileOrEmpty())
	}

	return &billing.Profile{
		BaseProfile: profile.BaseProfile,
	}, nil
}

func (s *Service) DeleteProfile(ctx context.Context, input billing.DeleteProfileInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		profile, err := s.adapter.GetProfile(ctx, billing.GetProfileInput{
			Profile: input,
		})
		if err != nil {
			return err
		}

		// Already deleted profiles cannot be deleted again
		if profile.DeletedAt != nil {
			return billing.ValidationError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrProfileAlreadyDeleted, profile.ID),
			}
		}

		// Default profiles cannot be deleted
		if profile.Default {
			return billing.ValidationError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrDefaultProfileCannotBeDeleted, profile.ID),
			}
		}

		referringCustomerIDs, err := s.adapter.GetCustomerOverrideReferencingProfile(ctx, input)
		if err != nil {
			return err
		}

		if len(referringCustomerIDs) > 0 {
			return billing.ValidationError{
				Err: fmt.Errorf("%w [profile_id=%s, customer_ids=%v]",
					billing.ErrProfileReferencedByOverrides,
					input.ID,
					lo.Map(referringCustomerIDs, func(item customer.CustomerID, _ int) string {
						return item.ID
					}),
				),
			}
		}

		return s.adapter.DeleteProfile(ctx, billing.DeleteProfileInput{
			Namespace: input.Namespace,
			ID:        profile.ID,
		})
	})
}

func (s *Service) ListProfiles(ctx context.Context, input billing.ListProfilesInput) (billing.ListProfilesResult, error) {
	if err := input.Validate(); err != nil {
		return billing.ListProfilesResult{}, billing.ValidationError{
			Err: err,
		}
	}

	profiles, err := s.adapter.ListProfiles(ctx, input)
	if err != nil {
		return billing.ListProfilesResult{}, err
	}

	response := pagination.Result[billing.Profile]{
		Page:       profiles.Page,
		TotalCount: profiles.TotalCount,
		Items:      make([]billing.Profile, 0, len(profiles.Items)),
	}

	for _, profile := range profiles.Items {
		finalProfile := billing.Profile{
			BaseProfile: profile,
		}

		if input.Expand.Apps {
			resolvedProfile, err := s.resolveProfileApps(ctx, &profile)
			if err != nil {
				return billing.ListProfilesResult{}, fmt.Errorf("error resolving profile: %w", err)
			}
			finalProfile = *resolvedProfile
		}
		response.Items = append(response.Items, finalProfile)
	}

	return response, nil
}

func (s *Service) UpdateProfile(ctx context.Context, input billing.UpdateProfileInput) (*billing.Profile, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*billing.Profile, error) {
		profile, err := s.adapter.GetProfile(ctx, billing.GetProfileInput{
			Profile: input.ProfileID(),
		})
		if err != nil {
			return nil, err
		}

		if profile.DeletedAt != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrProfileAlreadyDeleted, input.ID),
			}
		}

		var demotedProfile *demoteDefaultProfileResult

		if !profile.Default && input.Default {
			var err error

			demotedProfile, err = s.demoteDefaultProfile(ctx, input.Namespace)
			if err != nil {
				return nil, err
			}
		}

		if profile.Default && !input.Default {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrDefaultProfileCannotBeUnset, input.ID),
			}
		}

		updatedProfile, err := s.adapter.UpdateProfile(ctx, billing.UpdateProfileAdapterInput{
			TargetState:      billing.BaseProfile(input),
			WorkflowConfigID: profile.WorkflowConfigID,
		})
		if err != nil {
			return nil, err
		}

		if err := profile.Validate(); err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("error validating profile: %w", err),
			}
		}

		profileWithApps, err := s.resolveProfileApps(ctx, updatedProfile)
		if err != nil {
			return nil, err
		}

		if demotedProfile != nil {
			if err := s.handleDefaultProfileChange(ctx, defaultProfileChangeInput{
				Old: demotedProfile,
				New: profileWithApps,
			}); err != nil {
				return nil, err
			}
		}

		return profileWithApps, nil
	})
}

func (s *Service) ProvisionDefaultBillingProfile(ctx context.Context, namespace string) error {
	profile, err := s.adapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
		Namespace: namespace,
	})
	if err != nil {
		return err
	}

	if profile != nil {
		return nil
	}

	// Sandbox apps
	sandboxAppList, err := s.appService.ListApps(ctx, app.ListAppInput{
		Namespace: namespace,
		Type:      lo.ToPtr(app.AppTypeSandbox),
	})
	if err != nil {
		return fmt.Errorf("error fetching sandbox apps: %w", err)
	}

	if len(sandboxAppList.Items) == 0 {
		return fmt.Errorf("no sandbox apps found")
	}

	sandboxApp := sandboxAppList.Items[0]

	// Create the default profile
	_, err = s.CreateProfile(ctx, billing.CreateProfileInput{
		Namespace:   namespace,
		Name:        "openmeter-sandbox",
		Description: lo.ToPtr("Default profile for OpenMeter sandbox"),
		Supplier: billing.SupplierContact{
			Name: "OpenMeter",
			Address: models.Address{
				Country: lo.ToPtr(models.CountryCode("US")),
			},
		},
		WorkflowConfig: billing.DefaultWorkflowConfig,
		Default:        true,
		Apps: billing.ProfileAppReferences{
			Tax:       sandboxApp.GetID(),
			Invoicing: sandboxApp.GetID(),
			Payment:   sandboxApp.GetID(),
		},
	})
	if err != nil {
		return fmt.Errorf("error creating default profile: %w", err)
	}
	return nil
}

func (s *Service) IsAppUsed(ctx context.Context, appID app.AppID) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.IsAppUsed(ctx, appID)
	})
}

func (s *Service) resolveProfileApps(ctx context.Context, input *billing.BaseProfile) (*billing.Profile, error) {
	if input == nil {
		return nil, nil
	}

	out := billing.Profile{
		BaseProfile: *input,
	}

	out.Apps = &billing.ProfileApps{}

	taxApp, err := s.appService.GetApp(ctx, app.GetAppInput{
		Namespace: out.Namespace,
		ID:        input.AppReferences.Tax.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot resolve tax app: %w", err)
	}
	out.Apps.Tax = taxApp

	invoiceApp, err := s.appService.GetApp(ctx, app.GetAppInput{
		Namespace: out.Namespace,
		ID:        input.AppReferences.Invoicing.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot resolve invoicing app: %w", err)
	}
	out.Apps.Invoicing = invoiceApp

	paymentApp, err := s.appService.GetApp(ctx, app.GetAppInput{
		Namespace: out.Namespace,
		ID:        input.AppReferences.Payment.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot resolve payments app: %w", err)
	}
	out.Apps.Payment = paymentApp

	return &out, nil
}

type demoteDefaultProfileResult struct {
	Profile *billing.Profile

	CustomersWithPaidSubscription []customer.CustomerID
}

func (s *Service) demoteDefaultProfile(ctx context.Context, ns string) (*demoteDefaultProfileResult, error) {
	if ns == "" {
		return nil, errors.New("namespace is required")
	}

	defaultProfile, err := s.adapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
		Namespace: ns,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching default profile: %w", err)
	}

	// Nothing to do if there is no default profile
	if defaultProfile == nil {
		return nil, nil
	}

	defaultProfile.Default = false

	customerIDsWithPaidSubscription, err := s.adapter.GetUnpinnedCustomerIDsWithPaidSubscription(ctx, billing.GetUnpinnedCustomerIDsWithPaidSubscriptionInput{
		Namespace: ns,
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching customer IDs with paid subscription: %w", err)
	}

	updatedBaseProfile, err := s.adapter.UpdateProfile(ctx, billing.UpdateProfileAdapterInput{
		TargetState:      defaultProfile.BaseProfile,
		WorkflowConfigID: defaultProfile.WorkflowConfigID,
	})
	if err != nil {
		return nil, err
	}

	updatedProfile, err := s.resolveProfileApps(ctx, updatedBaseProfile)
	if err != nil {
		return nil, fmt.Errorf("error resolving profile apps: %w", err)
	}

	return &demoteDefaultProfileResult{
		Profile:                       updatedProfile,
		CustomersWithPaidSubscription: customerIDsWithPaidSubscription,
	}, nil
}

type defaultProfileChangeInput struct {
	Old *demoteDefaultProfileResult
	New *billing.Profile
}

func (i defaultProfileChangeInput) Validate() error {
	if i.Old == nil || i.New == nil {
		return errors.New("old or new default profile is nil")
	}

	if i.Old.Profile.Apps == nil || i.Old.Profile.Apps.Invoicing == nil {
		return fmt.Errorf("old profile has no invoicing app")
	}

	if i.New.Apps == nil || i.New.Apps.Invoicing == nil {
		return fmt.Errorf("new profile has no invoicing app")
	}

	return nil
}

func (s *Service) handleDefaultProfileChange(ctx context.Context, input defaultProfileChangeInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	oldProfile := input.Old.Profile
	newProfile := input.New

	if oldProfile.Apps.Invoicing.GetType() == newProfile.Apps.Invoicing.GetType() {
		s.logger.InfoContext(ctx, "no need to unassign customers from old default profile as invoicing app type is the same",
			"old_profile_id", oldProfile.ID,
			"new_profile_id", input.New.ID,
			"namespace", oldProfile.Namespace,
			"invoicing_app_type", oldProfile.Apps.Invoicing.GetType(),
		)
		return nil
	}

	// The app type have changed (right now we are checking the invoicing app type as we do not support mix and match setups)
	// let's pin the customers with paid subscriptions to the old profile.
	if len(input.Old.CustomersWithPaidSubscription) > 0 {
		err := s.adapter.BulkAssignCustomersToProfile(ctx, billing.BulkAssignCustomersToProfileInput{
			ProfileID:   oldProfile.ProfileID(),
			CustomerIDs: input.Old.CustomersWithPaidSubscription,
		})
		if err != nil {
			return fmt.Errorf("error pinning customers to old profile: %w", err)
		}

		// Let's log all the customer IDs (even if this can be a long list) so that we can use the logs
		// to troubleshoot issues if any arise.
		s.logger.InfoContext(ctx, "automatically pinned customers with paid subscription to the old default billing profile",
			"old_profile_id", oldProfile.ID,
			"new_profile_id", newProfile.ID,
			"namespace", oldProfile.Namespace,
			"invoicing_app_type_old", oldProfile.Apps.Invoicing.GetType(),
			"invoicing_app_type_new", newProfile.Apps.Invoicing.GetType(),
			"customer_ids", strings.Join(
				lo.Map(input.Old.CustomersWithPaidSubscription, func(item customer.CustomerID, _ int) string {
					return item.ID
				}),
				",",
			),
		)
	}

	return nil
}

func (s *Service) ResolveAppIDFromBillingProfile(ctx context.Context, namespace string, customerId *customer.CustomerID) (app.AppID, error) {
	var appID app.AppID

	// If the customer ID is provided, resolve billing profile based on the customer
	if customerId != nil {
		billingProfile, err := s.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: *customerId,
			Expand: billing.CustomerOverrideExpand{
				Apps: true,
			},
		})
		if err != nil {
			return appID, fmt.Errorf("failed to get billing profile: %w", err)
		}

		if billingProfile.MergedProfile.Apps == nil {
			return appID, fmt.Errorf("apps are not expanded in merged billing profile")
		}

		if billingProfile.MergedProfile.Apps.Payment.GetType() != app.AppTypeStripe {
			return appID, models.NewGenericNotFoundError(
				fmt.Errorf("customer has a billing profile, but the payment app is not a stripe app"),
			)
		}

		return billingProfile.MergedProfile.Apps.Payment.GetID(), nil
	}

	// If the customer ID is not provided, resolve billing profile from namespace
	// We list all billing profiles to be able to give a better error message
	billingProfileList, err := s.ListProfiles(ctx, billing.ListProfilesInput{
		Namespace: namespace,
		Expand:    billing.ProfileExpand{Apps: true},
	})
	if err != nil {
		return appID, fmt.Errorf("failed to get billing profile: %w", err)
	}

	// Find the billing profile with the stripe payment app
	// Prioritize the default profile
	var stripeApps []app.App
	var foundDefault bool

	for _, profile := range billingProfileList.Items {
		if foundDefault {
			break
		}

		if profile.Apps == nil {
			return appID, fmt.Errorf("billing profile apps are not expanded")
		}

		if profile.Apps.Payment.GetType() == app.AppTypeStripe {
			appID = profile.Apps.Payment.GetID()
			stripeApps = append(stripeApps, profile.Apps.Payment)

			if profile.Default {
				foundDefault = true
			}
		}
	}

	// If no default profile is found, return an error
	if !foundDefault {
		// If there is no stripe app, return an error
		if len(stripeApps) == 0 {
			return appID, models.NewGenericNotFoundError(
				fmt.Errorf("no stripe billing profile found, please create a billing profile with a stripe app"),
			)
		} else {
			return appID, models.NewGenericNotFoundError(
				fmt.Errorf("you have stripe billing profiles, but none is marked as default, provide the app id in the request"),
			)
		}
	}

	return appID, nil
}
