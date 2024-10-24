package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appshttpdriver "github.com/openmeterio/openmeter/openmeter/app/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
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

			return MapProfileToApi(profile)
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
	GetProfileParams struct {
		ID     string
		Expand []api.BillingProfileExpand
	}
	GetProfileRequest  = billing.GetProfileInput
	GetProfileResponse = api.BillingProfile
	GetProfileHandler  httptransport.HandlerWithArgs[GetProfileRequest, GetProfileResponse, GetProfileParams]
)

func (h *handler) GetProfile() GetProfileHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetProfileParams) (GetProfileRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetProfileRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetProfileRequest{
				Profile: models.NamespacedID{
					Namespace: ns,
					ID:        params.ID,
				},
				Expand: mapProfileExpandToEntity(params.Expand),
			}, nil
		},
		func(ctx context.Context, request GetProfileRequest) (GetProfileResponse, error) {
			profile, err := h.service.GetProfile(ctx, request)
			if err != nil {
				return GetProfileResponse{}, fmt.Errorf("failed to get profile: %w", err)
			}

			return MapProfileToApi(profile)
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

			return MapProfileToApi(profile)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateProfileResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("billingArchiveProfile"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	ListProfilesRequest  = billing.ListProfilesInput
	ListProfilesResponse = api.ProfilePaginatedResponse
	ListProfilesParams   = api.BillingListProfilesParams
	ListProfilesHandler  httptransport.HandlerWithArgs[ListProfilesRequest, ListProfilesResponse, ListProfilesParams]
)

func (h *handler) ListProfiles() ListProfilesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListProfilesParams) (ListProfilesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListProfilesRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return ListProfilesRequest{
				Namespace:       ns,
				IncludeArchived: lo.FromPtrOr(params.IncludeArchived, DefaultIncludeArchived),
				OrderBy:         lo.FromPtrOr(params.OrderBy, api.BillingProfileOrderByCreatedAt),
				Order:           sortx.Order(lo.FromPtrOr(params.Order, api.SortOrderDESC)),

				Page: pagination.Page{
					PageSize:   defaultx.WithDefault(params.PageSize, DefaultPageSize),
					PageNumber: defaultx.WithDefault(params.Page, DefaultPageNumber),
				},
			}, nil
		},
		func(ctx context.Context, request ListProfilesRequest) (ListProfilesResponse, error) {
			profiles, err := h.service.ListProfiles(ctx, request)
			if err != nil {
				return ListProfilesResponse{}, fmt.Errorf("failed to list profiles: %w", err)
			}

			out := ListProfilesResponse{
				Items:      make([]api.BillingProfile, 0, len(profiles.Items)),
				Page:       profiles.Page.PageNumber,
				PageSize:   profiles.Page.PageSize,
				TotalCount: profiles.TotalCount,
			}

			for _, profile := range profiles.Items {
				apiProfile, err := MapProfileToApi(&profile)
				if err != nil {
					return ListProfilesResponse{}, fmt.Errorf("failed to convert profile to API: %w", err)
				}

				out.Items = append(out.Items, apiProfile)
			}

			return out, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListProfilesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("billingListProfile"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func apiBillingPartyCreateToSupplierContact(c api.BillingPartyCreate) billingentity.SupplierContact {
	out := billingentity.SupplierContact{
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

func apiBillingPartyToSupplierContact(c api.BillingParty) billingentity.SupplierContact {
	out := billingentity.SupplierContact{
		ID:   lo.FromPtrOr(c.Id, ""),
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

func fromAPIBillingAppIdOrType(i string) billingentity.AppReference {
	_, err := ulid.Parse(i)
	if err != nil {
		return billingentity.AppReference{
			Type: appentitybase.AppType(i),
		}
	}

	return billingentity.AppReference{
		ID: i,
	}
}

func fromAPIBillingWorkflow(i api.BillingWorkflow) (billingentity.WorkflowConfig, error) {
	def := billingentity.DefaultWorkflowConfig

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
		return billingentity.WorkflowConfig{}, fmt.Errorf("failed to parse collection interval: %w", err)
	}

	draftPeriod, err := parseDurationPtr(i.Invoicing.DraftPeriod, def.Invoicing.DraftPeriod)
	if err != nil {
		return billingentity.WorkflowConfig{}, fmt.Errorf("failed to parse draft period: %w", err)
	}

	dueAfter, err := parseDurationPtr(i.Invoicing.DueAfter, def.Invoicing.DueAfter)
	if err != nil {
		return billingentity.WorkflowConfig{}, fmt.Errorf("failed to parse due after: %w", err)
	}

	return billingentity.WorkflowConfig{
		ID:        i.Id,
		CreatedAt: i.CreatedAt,
		UpdatedAt: i.UpdatedAt,
		DeletedAt: i.DeletedAt,

		Collection: billingentity.CollectionConfig{
			Alignment: lo.FromPtrOr((*billingentity.AlignmentKind)(i.Collection.Alignment), def.Collection.Alignment),
			Interval:  collInterval,
		},

		Invoicing: billingentity.InvoicingConfig{
			AutoAdvance: lo.CoalesceOrEmpty(i.Invoicing.AutoAdvance, def.Invoicing.AutoAdvance),
			DraftPeriod: draftPeriod,
			DueAfter:    dueAfter,
		},

		Payment: billingentity.PaymentConfig{
			CollectionMethod: lo.FromPtrOr((*billingentity.CollectionMethod)(i.Payment.CollectionMethod), def.Payment.CollectionMethod),
		},
	}, nil
}

func parseDurationPtr(d *string, defaultDuration datex.Period) (datex.Period, error) {
	if d == nil {
		return defaultDuration, nil
	}

	return datex.ISOString(*d).Parse()
}

func MapProfileToApi(p *billingentity.Profile) (api.BillingProfile, error) {
	if p == nil {
		return api.BillingProfile{}, errors.New("profile is nil")
	}

	out := api.BillingProfile{
		Id:        p.ID,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		DeletedAt: p.DeletedAt,

		Description: p.Description,
		Metadata:    (*api.Metadata)(lo.EmptyableToPtr(p.Metadata)),
		Default:     p.Default,

		Name:     p.Name,
		Supplier: mapSupplierContactToAPI(p.Supplier),
		Workflow: mapWorkflowConfigToAPI(p.WorkflowConfig),
	}

	if p.Apps != nil {
		apps, err := mapProfileAppsToAPI(p.Apps)
		if err != nil {
			return api.BillingProfile{}, fmt.Errorf("failed to map profile apps: %w", err)
		}

		if apps != nil {
			out.Apps = *apps
		}
	} else {
		apps, err := mapProfileAppReferencesToAPI(p.AppReferences)
		if err != nil {
			return api.BillingProfile{}, fmt.Errorf("failed to map profile app references: %w", err)
		}

		if apps != nil {
			out.Apps = *apps
		}
	}

	return out, nil
}

func mapProfileAppsToAPI(a *billingentity.ProfileApps) (*api.BillingProfileAppsOrReference, error) {
	if a == nil {
		return nil, nil
	}

	tax, err := appshttpdriver.MapAppToAPI(a.Tax)
	if err != nil {
		return nil, fmt.Errorf("cannot map tax app: %w", err)
	}

	invoicing, err := appshttpdriver.MapAppToAPI(a.Invoicing)
	if err != nil {
		return nil, fmt.Errorf("cannot map invoicing app: %w", err)
	}

	payment, err := appshttpdriver.MapAppToAPI(a.Payment)
	if err != nil {
		return nil, fmt.Errorf("cannot map payment app: %w", err)
	}

	apps := api.BillingProfileApps{
		Tax:       tax,
		Invoicing: invoicing,
		Payment:   payment,
	}

	out := api.BillingProfileAppsOrReference{}

	if err := out.FromBillingProfileApps(apps); err != nil {
		return nil, fmt.Errorf("failed to convert apps to API: %w", err)
	}

	return &out, nil
}

func mapProfileAppReferencesToAPI(a *billingentity.ProfileAppReferences) (*api.BillingProfileAppsOrReference, error) {
	if a == nil {
		return nil, nil
	}

	apps := api.BillingProfileAppReferences{
		Tax: api.AppReference{
			Id: a.Tax.ID,
		},
		Invoicing: api.AppReference{
			Id: a.Invoicing.ID,
		},
		Payment: api.AppReference{
			Id: a.Payment.ID,
		},
	}

	out := api.BillingProfileAppsOrReference{}

	if err := out.FromBillingProfileAppReferences(apps); err != nil {
		return nil, fmt.Errorf("failed to convert apps to API: %w", err)
	}

	return &out, nil
}

func mapProfileExpandToEntity(expand []api.BillingProfileExpand) billing.ProfileExpand {
	if len(expand) == 0 {
		return billing.ProfileExpand{}
	}

	if slices.Contains(expand, api.BillingProfileExpandAll) {
		return billing.ProfileExpand{
			Apps: true,
		}
	}

	return billing.ProfileExpand{
		Apps: slices.Contains(expand, api.BillingProfileExpandApps),
	}
}

func mapSupplierContactToAPI(c billingentity.SupplierContact) api.BillingParty {
	a := c.Address

	out := api.BillingParty{
		Name: lo.EmptyableToPtr(c.Name),
		Addresses: lo.ToPtr([]api.Address{
			{
				Country:     (*string)(a.Country),
				PostalCode:  a.PostalCode,
				State:       a.State,
				City:        a.City,
				Line1:       a.Line1,
				Line2:       a.Line2,
				PhoneNumber: a.PhoneNumber,
			},
		}),
	}

	if c.TaxCode != nil {
		out.TaxId = &api.BillingTaxIdentity{
			Code: c.TaxCode,
		}
	}

	return out
}

func mapWorkflowConfigToAPI(c billingentity.WorkflowConfig) api.BillingWorkflow {
	return api.BillingWorkflow{
		Id:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		DeletedAt: c.DeletedAt,

		Collection: &api.BillingWorkflowCollectionSettings{
			Alignment: (*api.BillingWorkflowCollectionAlignment)(lo.EmptyableToPtr(c.Collection.Alignment)),
			Interval:  lo.EmptyableToPtr(c.Collection.Interval.String()),
		},

		Invoicing: &api.BillingWorkflowInvoicingSettings{
			AutoAdvance: c.Invoicing.AutoAdvance,
			DraftPeriod: lo.EmptyableToPtr(c.Invoicing.DraftPeriod.String()),
			DueAfter:    lo.EmptyableToPtr(c.Invoicing.DueAfter.String()),
		},

		Payment: &api.BillingWorkflowPaymentSettings{
			CollectionMethod: (*api.BillingWorkflowCollectionMethod)(lo.EmptyableToPtr(string(c.Payment.CollectionMethod))),
		},
	}
}

func mapWorkflowConfigSettingsToAPI(c billingentity.WorkflowConfig) api.BillingWorkflowSettings {
	return api.BillingWorkflowSettings{
		Collection: &api.BillingWorkflowCollectionSettings{
			Alignment: (*api.BillingWorkflowCollectionAlignment)(lo.EmptyableToPtr(c.Collection.Alignment)),
			Interval:  lo.EmptyableToPtr(c.Collection.Interval.String()),
		},

		Invoicing: &api.BillingWorkflowInvoicingSettings{
			AutoAdvance: c.Invoicing.AutoAdvance,
			DraftPeriod: lo.EmptyableToPtr(c.Invoicing.DraftPeriod.String()),
			DueAfter:    lo.EmptyableToPtr(c.Invoicing.DueAfter.String()),
		},

		Payment: &api.BillingWorkflowPaymentSettings{
			CollectionMethod: (*api.BillingWorkflowCollectionMethod)(lo.EmptyableToPtr(string(c.Payment.CollectionMethod))),
		},
	}
}
