package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingconvert "github.com/openmeterio/openmeter/openmeter/billing/apiconvert"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var _ ProfileHandler = (*handler)(nil)

type (
	CreateProfileRequest  = billing.CreateProfileInput
	CreateProfileResponse = api.BillingProfile
	CreateProfileHandler  httptransport.Handler[CreateProfileRequest, CreateProfileResponse]
)

func (h *handler) CreateProfile() CreateProfileHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateProfileRequest, error) {
			body := api.BillingProfileCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateProfileRequest{}, fmt.Errorf("failed to decode request body: %w", err)
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateProfileRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			workflow, err := fromAPIBillingWorkflowCreate(body.Workflow)
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
				Apps: billing.CreateProfileAppsInput{
					Tax: app.AppID{
						Namespace: ns,
						ID:        body.Apps.Tax,
					},
					Invoicing: app.AppID{
						Namespace: ns,
						ID:        body.Apps.Invoicing,
					},
					Payment: app.AppID{
						Namespace: ns,
						ID:        body.Apps.Payment,
					},
				},
			}, nil
		},
		func(ctx context.Context, request CreateProfileRequest) (CreateProfileResponse, error) {
			profile, err := h.service.CreateProfile(ctx, request)
			if err != nil {
				return CreateProfileResponse{}, fmt.Errorf("failed to create profile: %w", err)
			}

			return h.MapProfileToApi(ctx, profile)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateProfileResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("CreateBillingProfile"),
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
				Profile: billing.ProfileID{
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

			return h.MapProfileToApi(ctx, profile)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateProfileResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("GetBillingProfile"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	DeleteProfileRequest  = billing.DeleteProfileInput
	DeleteProfileResponse = struct{}
	DeleteProfileHandler  httptransport.HandlerWithArgs[DeleteProfileRequest, DeleteProfileResponse, string]
)

func (h *handler) DeleteProfile() DeleteProfileHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, id string) (DeleteProfileRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteProfileRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return DeleteProfileRequest{
				Namespace: ns,
				ID:        id,
			}, nil
		},
		func(ctx context.Context, request DeleteProfileRequest) (DeleteProfileResponse, error) {
			if err := h.service.DeleteProfile(ctx, request); err != nil {
				return DeleteProfileResponse{}, fmt.Errorf("failed to archive profile: %w", err)
			}

			return DeleteProfileResponse{}, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteProfileResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("DeleteBillingProfile"),
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
			body := api.BillingProfileReplaceUpdateWithWorkflow{}
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

			return h.MapProfileToApi(ctx, profile)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateProfileResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("UpdateBillingProfiles"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	ListProfilesRequest  = billing.ListProfilesInput
	ListProfilesResponse = api.BillingProfilePaginatedResponse
	ListProfilesParams   = api.ListBillingProfilesParams
	ListProfilesHandler  httptransport.HandlerWithArgs[ListProfilesRequest, ListProfilesResponse, ListProfilesParams]
)

func mapProfileOrderBy(orderBy api.BillingProfileOrderBy) billing.ProfileOrderBy {
	switch orderBy {
	case api.BillingProfileOrderByName:
		return billing.ProfileOrderByName
	case api.BillingProfileOrderByUpdatedAt:
		return billing.ProfileOrderByUpdatedAt
	case api.BillingProfileOrderByDefault:
		return billing.ProfileOrderByDefault
	default:
		return billing.ProfileOrderByCreatedAt
	}
}

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
				OrderBy:         mapProfileOrderBy(lo.FromPtrOr(params.OrderBy, api.BillingProfileOrderByCreatedAt)),
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
				apiProfile, err := h.MapProfileToApi(ctx, &profile)
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
			httptransport.WithOperationName("ListBillingProfiles"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func apiBillingPartyCreateToSupplierContact(c api.BillingParty) billing.SupplierContact {
	out := billing.SupplierContact{
		Name: lo.FromPtr(c.Name),
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
		Name: lo.FromPtr(c.Name),
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

func fromAPIBillingWorkflow(i api.BillingWorkflow) (billing.WorkflowConfig, error) {
	return fromAPIBillingWorkflowWithSubscriptionEndProrationModeDefault(i, "")
}

func fromAPIBillingWorkflowWithSubscriptionEndProrationModeDefault(i api.BillingWorkflow, defaultSubscriptionEndProrationMode billing.SubscriptionEndProrationMode) (billing.WorkflowConfig, error) {
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

	if i.Tax == nil {
		i.Tax = &api.BillingWorkflowTaxSettings{}
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

	subscriptionEndProrationMode := defaultSubscriptionEndProrationMode
	if i.Invoicing.SubscriptionEndProrationMode != nil {
		subscriptionEndProrationMode = billing.SubscriptionEndProrationMode(*i.Invoicing.SubscriptionEndProrationMode)
	}

	return billing.WorkflowConfig{
		Collection: billing.CollectionConfig{
			Alignment: def.Collection.Alignment,
			Interval:  collInterval,
		},

		Invoicing: billing.InvoicingConfig{
			AutoAdvance:                  lo.FromPtrOr(i.Invoicing.AutoAdvance, def.Invoicing.AutoAdvance),
			DraftPeriod:                  draftPeriod,
			DueAfter:                     dueAfter,
			ProgressiveBilling:           lo.FromPtrOr(i.Invoicing.ProgressiveBilling, def.Invoicing.ProgressiveBilling),
			SubscriptionEndProrationMode: subscriptionEndProrationMode,
			DefaultTaxConfig:             mapTaxConfigToEntity(i.Invoicing.DefaultTaxConfig),
		},

		Payment: billing.PaymentConfig{
			CollectionMethod: lo.FromPtrOr((*billing.CollectionMethod)(i.Payment.CollectionMethod), def.Payment.CollectionMethod),
		},

		Tax: billing.WorkflowTaxConfig{
			Enabled:  lo.FromPtrOr(i.Tax.Enabled, def.Tax.Enabled),
			Enforced: lo.FromPtrOr(i.Tax.Enforced, def.Tax.Enforced),
		},
	}, nil
}

func fromAPIBillingWorkflowCreate(i api.BillingWorkflowCreate) (billing.WorkflowConfig, error) {
	return fromAPIBillingWorkflowWithSubscriptionEndProrationModeDefault(
		api.BillingWorkflow(i),
		billing.DefaultWorkflowConfig.Invoicing.SubscriptionEndProrationMode,
	)
}

func parseDurationPtr(d *string, defaultDuration datetime.ISODuration) (datetime.ISODuration, error) {
	if d == nil {
		return defaultDuration, nil
	}

	return datetime.ISODurationString(*d).Parse()
}

func (h *handler) MapProfileToApi(ctx context.Context, p *billing.Profile) (api.BillingProfile, error) {
	if p == nil {
		return api.BillingProfile{}, errors.New("profile is nil")
	}

	workflowConfig, err := billingconvert.ToAPIBillingWorkflow(p.WorkflowConfig)
	if err != nil {
		return api.BillingProfile{}, fmt.Errorf("failed to map workflow config to API: %w", err)
	}

	out := api.BillingProfile{
		Id:        p.ID,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		DeletedAt: p.DeletedAt,

		Description: p.Description,
		Metadata:    convert.MapToPointer(p.Metadata),
		Default:     p.Default,

		Name:     p.Name,
		Supplier: billingconvert.ToAPIBillingParty(p.Supplier),
		Workflow: workflowConfig,
	}

	if p.Apps != nil {
		apps, err := billingconvert.ToAPIBillingProfileApps(p.Apps)
		if err != nil {
			return api.BillingProfile{}, fmt.Errorf("failed to map profile apps: %w", err)
		}

		if apps != nil {
			out.Apps = *apps
		}
	} else {
		apps, err := billingconvert.ToAPIBillingProfileAppReferences(p.AppReferences)
		if err != nil {
			return api.BillingProfile{}, fmt.Errorf("failed to map profile app references: %w", err)
		}

		if apps != nil {
			out.Apps = *apps
		}
	}

	return out, nil
}

func mapProfileExpandToEntity(expand []api.BillingProfileExpand) billing.ProfileExpand {
	if len(expand) == 0 {
		return billing.ProfileExpand{}
	}

	return billing.ProfileExpand{
		Apps: slices.Contains(expand, api.BillingProfileExpandApps),
	}
}
