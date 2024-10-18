package billingservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ billing.ProfileService = (*Service)(nil)

func (s *Service) CreateProfile(ctx context.Context, input billing.CreateProfileInput) (*billingentity.Profile, error) {
	input = input.WithDefaults()

	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return Transaction(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) (*billingentity.Profile, error) {
		// Given that we have multiple constraints let's validate those here for better error reporting
		if input.Default {
			defaultProfile, err := txAdapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
				Namespace: input.Namespace,
			})
			if err != nil {
				return nil, err
			}

			if defaultProfile != nil {
				return nil, billing.ValidationError{
					Err: fmt.Errorf("%w [id=%s]", billing.ErrDefaultProfileAlreadyExists, defaultProfile.ID),
				}
			}
		}

		// let's resolve the applications
		taxApp, err := s.validateAppReference(ctx, input.Namespace, input.Apps.Tax, appentitybase.CapabilityTypeCalculateTax)
		if err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("error resolving tax app: %w", err),
			}
		}

		invocingApp, err := s.validateAppReference(ctx, input.Namespace, input.Apps.Invoicing, appentitybase.CapabilityTypeInvoiceCustomers)
		if err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("error resolving tax app: %w", err),
			}
		}

		paymentsApp, err := s.validateAppReference(ctx, input.Namespace, input.Apps.Payment, appentitybase.CapabilityTypeCollectPayments)
		if err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("error resolving tax app: %w", err),
			}
		}

		input.Apps = billing.CreateProfileAppsInput{
			Tax:       taxApp.Reference,
			Invoicing: invocingApp.Reference,
			Payment:   paymentsApp.Reference,
		}

		profile, err := txAdapter.CreateProfile(ctx, input)
		if err != nil {
			return nil, err
		}

		if err := profile.Validate(); err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("error validating profile: %w", err),
			}
		}

		return s.resolveBaseProfile(ctx, profile)
	})
}

func (s *Service) validateAppReference(ctx context.Context, ns string, ref billingentity.AppReference, capabilities ...appentitybase.CapabilityType) (*resolvedAppReference, error) {
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
	Reference billingentity.AppReference
	App       appentity.App
}

func (s *Service) resolveAppReference(ctx context.Context, ns string, ref billingentity.AppReference) (*resolvedAppReference, error) {
	if ref.ID != "" {
		app, err := s.appService.GetApp(ctx, appentity.GetAppInput{
			Namespace: ns,
			ID:        ref.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot find application[id=%s]: %w", ref.ID, err)
		}

		return &resolvedAppReference{
			Reference: billingentity.AppReference{
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
			Reference: billingentity.AppReference{
				ID: app.GetID().ID,
			},
			App: app,
		}, nil
	}

	return nil, fmt.Errorf("invalid app reference: %v", ref)
}

func (s *Service) GetDefaultProfile(ctx context.Context, input billing.GetDefaultProfileInput) (*billingentity.Profile, error) {
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

	return s.resolveBaseProfile(ctx, profile)
}

func (s *Service) GetProfile(ctx context.Context, input billing.GetProfileInput) (*billingentity.Profile, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	profile, err := s.adapter.GetProfile(ctx, input)
	if err != nil {
		return nil, err
	}

	return s.resolveBaseProfile(ctx, profile)
}

func (s *Service) DeleteProfile(ctx context.Context, input billing.DeleteProfileInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	return TransactionWithNoValue(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) error {
		profile, err := txAdapter.GetProfile(ctx, billing.GetProfileInput(input))
		if err != nil {
			return err
		}

		if profile == nil {
			return billing.ValidationError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrProfileNotFound, input.ID),
			}
		}

		if profile.DeletedAt != nil {
			return billing.ValidationError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrProfileAlreadyDeleted, profile.ID),
			}
		}

		referringCustomerIDs, err := txAdapter.GetCustomerOverrideReferencingProfile(ctx, billing.HasCustomerOverrideReferencingProfileAdapterInput(input))
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

		return txAdapter.DeleteProfile(ctx, billing.DeleteProfileInput{
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

	response := pagination.PagedResponse[billingentity.Profile]{
		Page:       profiles.Page,
		TotalCount: profiles.TotalCount,
		Items:      make([]billingentity.Profile, 0, len(profiles.Items)),
	}

	for _, profile := range profiles.Items {
		resolvedProfile, err := s.resolveBaseProfile(ctx, &profile)
		if err != nil {
			return billing.ListProfilesResult{}, fmt.Errorf("error resolving profile: %w", err)
		}

		response.Items = append(response.Items, *resolvedProfile)
	}

	return response, nil
}

func (s *Service) UpdateProfile(ctx context.Context, input billing.UpdateProfileInput) (*billingentity.Profile, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return Transaction(ctx, s.adapter, func(ctx context.Context, txAdapter billing.Adapter) (*billingentity.Profile, error) {
		profile, err := txAdapter.GetProfile(ctx, billing.GetProfileInput{
			Namespace: input.Namespace,
			ID:        input.ID,
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

		if !profile.UpdatedAt.Equal(input.UpdatedAt) {
			return nil, billing.UpdateAfterDeleteError{
				Err: fmt.Errorf("%w [id=%s]", billing.ErrProfileConflict, input.ID),
			}
		}

		if !profile.Default && input.Default {
			defaultProfile, err := txAdapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
				Namespace: input.Namespace,
			})
			if err != nil {
				return nil, err
			}

			if defaultProfile != nil && defaultProfile.ID != input.ID {
				return nil, billing.ValidationError{
					Err: fmt.Errorf("%w [id=%s]", billing.ErrDefaultProfileAlreadyExists, defaultProfile.ID),
				}
			}
		}

		profile, err = txAdapter.UpdateProfile(ctx, billing.UpdateProfileAdapterInput{
			TargetState:      billingentity.BaseProfile(input),
			WorkflowConfigID: profile.WorkflowConfig.ID,
		})
		if err != nil {
			return nil, err
		}

		if err := profile.Validate(); err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("error validating profile: %w", err),
			}
		}

		return s.resolveBaseProfile(ctx, profile)
	})
}

func (s *Service) resolveBaseProfile(ctx context.Context, input *billingentity.BaseProfile) (*billingentity.Profile, error) {
	if input == nil {
		return nil, nil
	}

	out := billingentity.Profile{
		BaseProfile: *input,
	}

	out.Apps = &billingentity.ProfileApps{}

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
		return nil, fmt.Errorf("cannot resolve tax app: %w", err)
	}
	out.Apps.Invoicing = invoiceApp

	paymentApp, err := s.appService.GetApp(ctx, appentity.GetAppInput{
		Namespace: out.Namespace,
		ID:        input.AppReferences.Payment.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot resolve tax app: %w", err)
	}
	out.Apps.Payment = paymentApp

	return &out, nil
}
