package billingservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
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
		// Given that we have multiple constraints let's validate those here for better error reporting
		if input.Default {
			oldDefaultProfile, err := s.adapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
				Namespace: input.Namespace,
			})
			if err != nil {
				return nil, fmt.Errorf("error fetching default profile: %w", err)
			}

			if oldDefaultProfile != nil {
				oldDefaultProfile.Default = false

				_, err := s.adapter.UpdateProfile(ctx, billing.UpdateProfileAdapterInput{
					TargetState:      oldDefaultProfile.BaseProfile,
					WorkflowConfigID: oldDefaultProfile.WorkflowConfigID,
				})
				if err != nil {
					return nil, err
				}
			}
		}

		resolvedApps, err := s.resolveApps(ctx, input.Namespace, input.Apps)
		if err != nil {
			return nil, err
		}

		input.Apps = billing.ProfileAppReferences{
			Tax:       resolvedApps.Tax.Reference,
			Invoicing: resolvedApps.Invoicing.Reference,
			Payment:   resolvedApps.Payment.Reference,
		}

		if resolvedApps.Tax.App.GetType() != resolvedApps.Invoicing.App.GetType() ||
			resolvedApps.Tax.App.GetType() != resolvedApps.Payment.App.GetType() {
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

		return s.resolveProfileApps(ctx, profile)
	})
}

type resolvedApps struct {
	Tax       *resolvedAppReference
	Invoicing *resolvedAppReference
	Payment   *resolvedAppReference
}

func (s *Service) resolveApps(ctx context.Context, ns string, apps billing.ProfileAppReferences) (*resolvedApps, error) {
	taxApp, err := s.validateAppReference(ctx, ns, apps.Tax, appentitybase.CapabilityTypeCalculateTax)
	if err != nil {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("error resolving tax app: %w", err),
		}
	}

	invocingApp, err := s.validateAppReference(ctx, ns, apps.Invoicing, appentitybase.CapabilityTypeInvoiceCustomers)
	if err != nil {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("error resolving invocing app: %w", err),
		}
	}

	paymentsApp, err := s.validateAppReference(ctx, ns, apps.Payment, appentitybase.CapabilityTypeCollectPayments)
	if err != nil {
		return nil, billing.ValidationError{
			Err: fmt.Errorf("error resolving payments app: %w", err),
		}
	}

	return &resolvedApps{
		Tax:       taxApp,
		Invoicing: invocingApp,
		Payment:   paymentsApp,
	}, nil
}

func (s *Service) validateAppReference(ctx context.Context, ns string, ref billing.AppReference, capabilities ...appentitybase.CapabilityType) (*resolvedAppReference, error) {
	if err := ref.Validate(); err != nil {
		return nil, fmt.Errorf("invalid app reference: %w", err)
	}

	resolved, err := s.resolveAppReference(ctx, ns, ref)
	if err != nil {
		return nil, err
	}

	if err := resolved.App.ValidateCapabilities(capabilities...); err != nil {
		return nil, err
	}

	return resolved, nil
}

type resolvedAppReference struct {
	Reference billing.AppReference
	App       appentity.App
}

func (s *Service) resolveAppReference(ctx context.Context, ns string, ref billing.AppReference) (*resolvedAppReference, error) {
	if ref.ID != "" {
		app, err := s.appService.GetApp(ctx, appentity.GetAppInput{
			Namespace: ns,
			ID:        ref.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot find application[id=%s]: %w", ref.ID, err)
		}

		return &resolvedAppReference{
			Reference: billing.AppReference{
				ID: app.GetID().ID,
			},
			App: app,
		}, nil
	}

	if ref.Type != "" {
		app, err := s.appService.GetDefaultApp(ctx, appentity.GetDefaultAppInput{
			Namespace: ns,
			Type:      ref.Type,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot find default application[type=%s]: %w", ref.Type, err)
		}

		return &resolvedAppReference{
			Reference: billing.AppReference{
				ID: app.GetID().ID,
			},
			App: app,
		}, nil
	}

	return nil, fmt.Errorf("invalid app reference: %v", ref)
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

		if profile == nil {
			return billing.ValidationError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrProfileNotFound, input.ID),
			}
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
					lo.Map(referringCustomerIDs, func(item customerentity.CustomerID, _ int) string {
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

	response := pagination.PagedResponse[billing.Profile]{
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

		if profile == nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrProfileNotFound, input.ID),
			}
		}

		if profile.DeletedAt != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrProfileAlreadyDeleted, input.ID),
			}
		}

		if !profile.Default && input.Default {
			oldDefaultProfile, err := s.adapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
				Namespace: input.Namespace,
			})
			if err != nil {
				return nil, err
			}

			if oldDefaultProfile != nil {
				if err := s.adapter.UnsetDefaultProfile(ctx, oldDefaultProfile.ProfileID()); err != nil {
					return nil, err
				}
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

		return s.resolveProfileApps(ctx, updatedProfile)
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
			Tax:       billing.AppReference{Type: appentitybase.AppTypeSandbox},
			Invoicing: billing.AppReference{Type: appentitybase.AppTypeSandbox},
			Payment:   billing.AppReference{Type: appentitybase.AppTypeSandbox},
		},
	})
	if err != nil {
		return fmt.Errorf("error creating default profile: %w", err)
	}
	return nil
}

func (s *Service) IsAppUsed(ctx context.Context, appID appentitybase.AppID) (bool, error) {
	return s.adapter.IsAppUsed(ctx, appID)
}

func (s *Service) resolveProfileApps(ctx context.Context, input *billing.BaseProfile) (*billing.Profile, error) {
	if input == nil {
		return nil, nil
	}

	out := billing.Profile{
		BaseProfile: *input,
	}

	out.Apps = &billing.ProfileApps{}

	taxApp, err := s.appService.GetApp(ctx, appentity.GetAppInput{
		Namespace: out.Namespace,
		ID:        input.AppReferences.Tax.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot resolve tax app: %w", err)
	}
	out.Apps.Tax = taxApp

	invoiceApp, err := s.appService.GetApp(ctx, appentity.GetAppInput{
		Namespace: out.Namespace,
		ID:        input.AppReferences.Invoicing.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot resolve invoicing app: %w", err)
	}
	out.Apps.Invoicing = invoiceApp

	paymentApp, err := s.appService.GetApp(ctx, appentity.GetAppInput{
		Namespace: out.Namespace,
		ID:        input.AppReferences.Payment.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot resolve payments app: %w", err)
	}
	out.Apps.Payment = paymentApp

	return &out, nil
}
