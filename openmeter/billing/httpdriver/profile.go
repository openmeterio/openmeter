package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/api"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

type (
	CreateProfileRequest  = billing.CreateProfileInput
	CreateProfileResponse = api.BillingProfile
	CreateProfileHandler  httptransport.Handler[CreateProfileRequest, CreateProfileResponse]
)

func (h *handler) CreateProfile() CreateProfileHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateProfileRequest, error) {
			body := api.BillingProfileCreateInput{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateProfileRequest{}, fmt.Errorf("failed to decode request body: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateProfileRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			workflow, err := fromAPIBillingWorkflow(body.Workflow)
			if err != nil {
				return CreateProfileRequest{}, fmt.Errorf("failed to parse workflow: %w", err)
			}

			return CreateProfileRequest{
				Namespace: ns,

				Name:        body.Name,
				Description: body.Description,

				Metadata: lo.FromPtrOr(body.Metadata, map[string]string{}),

				Supplier: apiBillingPartyCreateToSupplierContact(body.Supplier),

				Default: body.Default,

				WorkflowConfig: workflow,
				Apps:           fromAPIBillingProfileCreateAppsInput(body.Apps),
			}, nil
		},
		func(ctx context.Context, request CreateProfileRequest) (CreateProfileResponse, error) {
			profile, err := h.service.CreateProfile(ctx, request)
			if err != nil {
				return CreateProfileResponse{}, fmt.Errorf("failed to create profile: %w", err)
			}

			return profile.ToAPI()
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateProfileResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("billingCreateProfile"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	GetProfileRequest  = billing.GetProfileInput
	GetProfileResponse = api.BillingProfile
	GetProfileHandler  httptransport.HandlerWithArgs[GetProfileRequest, GetProfileResponse, string]
)

func (h *handler) GetProfile() GetProfileHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, id string) (GetProfileRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetProfileRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetProfileRequest{
				Namespace: ns,
				ID:        id,
			}, nil
		},
		func(ctx context.Context, request GetProfileRequest) (GetProfileResponse, error) {
			profile, err := h.service.GetProfile(ctx, request)
			if err != nil {
				return GetProfileResponse{}, fmt.Errorf("failed to get profile: %w", err)
			}

			return profile.ToAPI()
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateProfileResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("billingGetProfile"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	ArchiveProfileRequest  = billing.DeleteProfileInput
	ArchiveProfileResponse = struct{}
	ArchiveProfileHandler  httptransport.HandlerWithArgs[ArchiveProfileRequest, ArchiveProfileResponse, string]
)

func (h *handler) ArchiveProfile() ArchiveProfileHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, id string) (ArchiveProfileRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ArchiveProfileRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return ArchiveProfileRequest{
				Namespace: ns,
				ID:        id,
			}, nil
		},
		func(ctx context.Context, request ArchiveProfileRequest) (ArchiveProfileResponse, error) {
			if err := h.service.DeleteProfile(ctx, request); err != nil {
				return ArchiveProfileResponse{}, fmt.Errorf("failed to archive profile: %w", err)
			}

			return ArchiveProfileResponse{}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ArchiveProfileResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("billingArchiveProfile"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	UpdateProfileRequest  = billing.UpdateProfileInput
	UpdateProfileResponse = api.BillingProfile
	UpdateProfileHandler  httptransport.HandlerWithArgs[UpdateProfileRequest, UpdateProfileResponse, string]
)

func (h *handler) UpdateProfile() UpdateProfileHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, id string) (UpdateProfileRequest, error) {
			body := api.BillingUpdateProfileJSONRequestBody{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateProfileRequest{}, fmt.Errorf("failed to decode request body: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateProfileRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			workflow, err := fromAPIBillingWorkflow(body.Workflow)
			if err != nil {
				return UpdateProfileRequest{}, fmt.Errorf("failed to parse workflow: %w", err)
			}

			return UpdateProfileRequest{
				Namespace: ns,
				ID:        id,

				Name:        body.Name,
				Description: body.Description,
				UpdatedAt:   body.UpdatedAt,

				Metadata: lo.FromPtrOr(body.Metadata, map[string]string{}),

				Supplier: apiBillingPartyToSupplierContact(body.Supplier),

				Default: body.Default,

				WorkflowConfig: workflow,
			}, nil
		},
		func(ctx context.Context, request UpdateProfileRequest) (UpdateProfileResponse, error) {
			profile, err := h.service.UpdateProfile(ctx, request)
			if err != nil {
				return UpdateProfileResponse{}, fmt.Errorf("failed to update profile: %w", err)
			}

			return profile.ToAPI()
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateProfileResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("billingArchiveProfile"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func apiBillingPartyCreateToSupplierContact(c api.BillingPartyCreate) billing.SupplierContact {
	out := billing.SupplierContact{
		Name: lo.FromPtrOr(c.Name, ""),
	}

	if c.Addresses == nil || len(*c.Addresses) == 0 {
		return out
	}

	if c.TaxId != nil {
		out.TaxCode = c.TaxId.Code
	}

	addr := (*c.Addresses)[0]

	out.Address = models.Address{
		Country:     (*models.CountryCode)(addr.Country),
		City:        addr.City,
		State:       addr.State,
		PostalCode:  addr.PostalCode,
		Line1:       addr.Line1,
		Line2:       addr.Line2,
		PhoneNumber: addr.PhoneNumber,
	}

	return out
}

func apiBillingPartyToSupplierContact(c api.BillingParty) billing.SupplierContact {
	out := billing.SupplierContact{
		ID:   c.Id,
		Name: lo.FromPtrOr(c.Name, ""),
	}

	if c.Addresses == nil || len(*c.Addresses) == 0 {
		return out
	}

	if c.TaxId != nil {
		out.TaxCode = c.TaxId.Code
	}

	addr := (*c.Addresses)[0]

	out.Address = models.Address{
		Country:     (*models.CountryCode)(addr.Country),
		City:        addr.City,
		State:       addr.State,
		PostalCode:  addr.PostalCode,
		Line1:       addr.Line1,
		Line2:       addr.Line2,
		PhoneNumber: addr.PhoneNumber,
	}

	return out
}

func fromAPIBillingProfileCreateAppsInput(i api.BillingProfileCreateAppsInput) billing.CreateProfileAppsInput {
	return billing.CreateProfileAppsInput{
		Tax:       fromAPIBillingAppIdOrType(i.Tax),
		Invoicing: fromAPIBillingAppIdOrType(i.Invoicing),
		Payment:   fromAPIBillingAppIdOrType(i.Payment),
	}
}

func fromAPIBillingAppIdOrType(i string) billing.AppReference {
	_, err := ulid.Parse(i)
	if err != nil {
		return billing.AppReference{
			Type: appentitybase.AppType(i),
		}
	}

	return billing.AppReference{
		ID: i,
	}
}

func fromAPIBillingWorkflow(i api.BillingWorkflow) (billing.WorkflowConfig, error) {
	def := billing.DefaultWorkflowConfig

	if i.Collection == nil {
		i.Collection = &api.BillingWorkflowCollectionSettings{}
	}

	if i.Invoicing == nil {
		i.Invoicing = &api.BillingWorkflowInvoicingSettings{}
	}

	if i.Payment == nil {
		i.Payment = &api.BillingWorkflowPaymentSettings{}
	}

	collInterval, err := parseDurationPtr(i.Collection.Interval, def.Collection.Interval)
	if err != nil {
		return billing.WorkflowConfig{}, fmt.Errorf("failed to parse collection interval: %w", err)
	}

	draftPeriod, err := parseDurationPtr(i.Invoicing.DraftPeriod, def.Invoicing.DraftPeriod)
	if err != nil {
		return billing.WorkflowConfig{}, fmt.Errorf("failed to parse draft period: %w", err)
	}

	dueAfter, err := parseDurationPtr(i.Invoicing.DueAfter, def.Invoicing.DueAfter)
	if err != nil {
		return billing.WorkflowConfig{}, fmt.Errorf("failed to parse due after: %w", err)
	}

	return billing.WorkflowConfig{
		ID:        i.Id,
		CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt,
		DeletedAt: i.DeletedAt,

		Collection: billing.CollectionConfig{
			Alignment: lo.FromPtrOr((*billing.AlignmentKind)(i.Collection.Alignment), def.Collection.Alignment),
			Interval:  collInterval,
		},

		Invoicing: billing.InvoicingConfig{
			AutoAdvance: lo.CoalesceOrEmpty(i.Invoicing.AutoAdvance, def.Invoicing.AutoAdvance),
			DraftPeriod: draftPeriod,
			DueAfter:    dueAfter,
		},

		Payment: billing.PaymentConfig{
			CollectionMethod: lo.FromPtrOr((*billing.CollectionMethod)(i.Payment.CollectionMethod), def.Payment.CollectionMethod),
		},
	}, nil
}

func parseDurationPtr(d *string, defaultDuration datex.Period) (datex.Period, error) {
	if d == nil {
		return defaultDuration, nil
	}

	return datex.ISOString(*d).Parse()
}
