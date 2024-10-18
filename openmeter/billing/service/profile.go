package billingservice

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

var _ billing.ProfileService = (*Service)(nil)

func (s *Service) CreateProfile(ctx context.Context, input billing.CreateProfileInput) (*billing.Profile, error) {
	input = input.WithDefaults()

	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return billing.WithTx(ctx, s.adapter, func(ctx context.Context, adapter billing.TxAdapter) (*billing.Profile, error) {
		// Given that we have multiple constraints let's validate those here for better error reporting
		if input.Default {
			defaultProfile, err := adapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
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

		// TODO[later]: align tx handling with apps tx handling (the next gen thing)
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

		profile, err := adapter.CreateProfile(ctx, input)
		if err != nil {
			return nil, err
		}

		if err := profile.Validate(); err != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("error validating profile: %w", err),
			}
		}

		return s.resolveAdapterProfile(ctx, profile)
	})
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

	return s.resolveAdapterProfile(ctx, profile)
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

	return s.resolveAdapterProfile(ctx, profile)
}

func (s *Service) DeleteProfile(ctx context.Context, input billing.DeleteProfileInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	return billing.WithTxNoValue(ctx, s.adapter, func(ctx context.Context, adapter billing.TxAdapter) error {
		profile, err := s.adapter.GetProfile(ctx, billing.GetProfileInput(input))
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

		referringCustomerIDs, err := adapter.GetCustomerOverrideReferencingProfile(ctx, billing.HasCustomerOverrideReferencingProfileAdapterInput(input))
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

		return adapter.DeleteProfile(ctx, billing.DeleteProfileInput{
			Namespace: input.Namespace,
			ID:        profile.ID,
		})
	})
}

func (s *Service) UpdateProfile(ctx context.Context, input billing.UpdateProfileInput) (*billing.Profile, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return billing.WithTx(ctx, s.adapter, func(ctx context.Context, adapter billing.TxAdapter) (*billing.Profile, error) {
		profile, err := adapter.GetProfile(ctx, billing.GetProfileInput{
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
			defaultProfile, err := adapter.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
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

		profile, err = adapter.UpdateProfile(ctx, billing.UpdateProfileAdapterInput{
			TargetState:      billing.Profile(input),
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

		return s.resolveAdapterProfile(ctx, profile)
	})
}

func (s *Service) resolveAdapterProfile(ctx context.Context, input *billing.AdapterProfile) (*billing.Profile, error) {
	if input == nil {
		return nil, nil
	}

	out := input.Profile
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
