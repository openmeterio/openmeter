package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingconvert "github.com/openmeterio/openmeter/openmeter/billing/apiconvert"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
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
	// CreateProfileRequest keeps the default tax config unmapped: mapping it may create Tax Codes
	// for legacy stripe.code values, which has to happen in the same transaction as the create.
	CreateProfileRequest struct {
		Input            billing.CreateProfileInput
		DefaultTaxConfig *apiv1models.TaxConfig
	}
	CreateProfileResponse = apiv1models.BillingProfile
	CreateProfileHandler  httptransport.Handler[CreateProfileRequest, CreateProfileResponse]
)

func (h *handler) CreateProfile() CreateProfileHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateProfileRequest, error) {
			body := apiv1models.BillingProfileCreate{}
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
				Input: billing.CreateProfileInput{
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
				},
				DefaultTaxConfig: lo.FromPtr(body.Workflow.Invoicing).DefaultTaxConfig,
			}, nil
		},
		func(ctx context.Context, request CreateProfileRequest) (CreateProfileResponse, error) {
			var resp CreateProfileResponse

			err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				input := request.Input

				// Nothing is stored yet, so the gate rejects any tax code and nothing gets resolved
				// or created; only behavior can get through.
				defaultTaxConfig, err := billingconvert.ResolveProfileDefaultTaxConfig(ctx, resolver, input.Namespace, mapTaxConfigToEntity(request.DefaultTaxConfig), nil)
				if err != nil {
					return err
				}

				input.WorkflowConfig.Invoicing.DefaultTaxConfig = defaultTaxConfig

				profile, err := h.service.CreateProfile(ctx, input)
				if err != nil {
					return fmt.Errorf("failed to create profile: %w", err)
				}

				resp, err = h.mapProfileWithTaxCodesToAPI(ctx, profile)

				return err
			})

			return resp, err
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
		Expand []apiv1models.BillingProfileExpand
	}
	GetProfileRequest  = billing.GetProfileInput
	GetProfileResponse = apiv1models.BillingProfile
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

			return h.mapProfileWithTaxCodesToAPI(ctx, profile)
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
	// UpdateProfileRequest keeps the default tax config unmapped for the same reason as
	// CreateProfileRequest.
	UpdateProfileRequest struct {
		Input            billing.UpdateProfileInput
		DefaultTaxConfig *apiv1models.TaxConfig
	}
	UpdateProfileResponse = apiv1models.BillingProfile
	UpdateProfileHandler  httptransport.HandlerWithArgs[UpdateProfileRequest, UpdateProfileResponse, string]
)

func (h *handler) UpdateProfile() UpdateProfileHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, id string) (UpdateProfileRequest, error) {
			body := apiv1models.BillingProfileReplaceUpdateWithWorkflow{}
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
				Input: billing.UpdateProfileInput{
					Namespace: ns,
					ID:        id,

					Name:        body.Name,
					Description: body.Description,

					Metadata: lo.FromPtrOr(body.Metadata, map[string]string{}),

					Supplier: apiBillingPartyToSupplierContact(body.Supplier),

					Default: body.Default,

					WorkflowConfig: workflow,
				},
				DefaultTaxConfig: lo.FromPtr(body.Workflow.Invoicing).DefaultTaxConfig,
			}, nil
		},
		func(ctx context.Context, request UpdateProfileRequest) (UpdateProfileResponse, error) {
			var resp UpdateProfileResponse

			err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				input := request.Input

				stored, err := h.service.GetProfile(ctx, billing.GetProfileInput{
					Profile: input.ProfileID(),
				})
				if err != nil {
					return fmt.Errorf("failed to get profile: %w", err)
				}

				storedDefaultTaxConfig, err := billingconvert.LegacyProfileDefaultTaxConfig(ctx, h.taxCodeResolver, input.Namespace, stored.WorkflowConfig)
				if err != nil {
					return err
				}

				defaultTaxConfig, err := billingconvert.ResolveProfileDefaultTaxConfig(ctx, resolver, input.Namespace, mapTaxConfigToEntity(request.DefaultTaxConfig), storedDefaultTaxConfig)
				if err != nil {
					return err
				}

				input.WorkflowConfig.Invoicing.DefaultTaxConfig = defaultTaxConfig

				profile, err := h.service.UpdateProfile(ctx, input)
				if err != nil {
					return fmt.Errorf("failed to update profile: %w", err)
				}

				resp, err = h.mapProfileWithTaxCodesToAPI(ctx, profile)

				return err
			})

			return resp, err
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
	ListProfilesResponse = apiv1models.BillingProfilePaginatedResponse
	ListProfilesParams   = apiv1models.ListBillingProfilesParams
	ListProfilesHandler  httptransport.HandlerWithArgs[ListProfilesRequest, ListProfilesResponse, ListProfilesParams]
)

func mapProfileOrderBy(orderBy apiv1models.BillingProfileOrderBy) billing.ProfileOrderBy {
	switch orderBy {
	case apiv1models.BillingProfileOrderByName:
		return billing.ProfileOrderByName
	case apiv1models.BillingProfileOrderByUpdatedAt:
		return billing.ProfileOrderByUpdatedAt
	case apiv1models.BillingProfileOrderByDefault:
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
				OrderBy:         mapProfileOrderBy(lo.FromPtrOr(params.OrderBy, apiv1models.BillingProfileOrderByCreatedAt)),
				Order:           sortx.Order(lo.FromPtrOr(params.Order, apiv1models.SortOrderDESC)),

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
				Items:      make([]apiv1models.BillingProfile, 0, len(profiles.Items)),
				Page:       profiles.Page.PageNumber,
				PageSize:   profiles.Page.PageSize,
				TotalCount: profiles.TotalCount,
			}

			taxCodes, err := billingconvert.ResolveWorkflowTaxCodes(ctx, h.taxCodeResolver, request.Namespace,
				lo.Map(profiles.Items, func(p billing.Profile, _ int) billing.WorkflowConfig {
					return p.WorkflowConfig
				})...)
			if err != nil {
				return ListProfilesResponse{}, err
			}

			for _, profile := range profiles.Items {
				apiProfile, err := mapProfileToAPI(&profile, taxCodes)
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

func apiBillingPartyCreateToSupplierContact(c apiv1models.BillingParty) billing.SupplierContact {
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

func apiBillingPartyToSupplierContact(c apiv1models.BillingParty) billing.SupplierContact {
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

func fromAPIBillingWorkflow(i apiv1models.BillingWorkflow) (billing.WorkflowConfig, error) {
	return fromAPIBillingWorkflowWithSubscriptionEndProrationModeDefault(i, "")
}

func fromAPIBillingWorkflowWithSubscriptionEndProrationModeDefault(i apiv1models.BillingWorkflow, defaultSubscriptionEndProrationMode billing.SubscriptionEndProrationMode) (billing.WorkflowConfig, error) {
	def := billing.DefaultWorkflowConfig

	if i.Collection == nil {
		i.Collection = &apiv1models.BillingWorkflowCollectionSettings{}
	}

	if i.Invoicing == nil {
		i.Invoicing = &apiv1models.BillingWorkflowInvoicingSettings{}
	}

	if i.Payment == nil {
		i.Payment = &apiv1models.BillingWorkflowPaymentSettings{}
	}

	if i.Tax == nil {
		i.Tax = &apiv1models.BillingWorkflowTaxSettings{}
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
			// The default tax config is mapped by the handlers: it needs the stored profile and
			// the tax code resolver.
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

func fromAPIBillingWorkflowCreate(i apiv1models.BillingWorkflowCreate) (billing.WorkflowConfig, error) {
	return fromAPIBillingWorkflowWithSubscriptionEndProrationModeDefault(
		apiv1models.BillingWorkflow(i),
		billing.DefaultWorkflowConfig.Invoicing.SubscriptionEndProrationMode,
	)
}

func parseDurationPtr(d *string, defaultDuration datetime.ISODuration) (datetime.ISODuration, error) {
	if d == nil {
		return defaultDuration, nil
	}

	return datetime.ISODurationString(*d).Parse()
}

func (h *handler) mapProfileWithTaxCodesToAPI(ctx context.Context, p *billing.Profile) (apiv1models.BillingProfile, error) {
	if p == nil {
		return apiv1models.BillingProfile{}, errors.New("profile is nil")
	}

	taxCodes, err := billingconvert.ResolveWorkflowTaxCodes(ctx, h.taxCodeResolver, p.Namespace, p.WorkflowConfig)
	if err != nil {
		return apiv1models.BillingProfile{}, err
	}

	return mapProfileToAPI(p, taxCodes)
}

func mapProfileToAPI(p *billing.Profile, taxCodes map[string]*taxcode.TaxCode) (apiv1models.BillingProfile, error) {
	if p == nil {
		return apiv1models.BillingProfile{}, errors.New("profile is nil")
	}

	workflowConfig, err := billingconvert.ToAPIBillingWorkflow(p.WorkflowConfig, taxCodes)
	if err != nil {
		return apiv1models.BillingProfile{}, fmt.Errorf("failed to map workflow config to API: %w", err)
	}

	out := apiv1models.BillingProfile{
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
			return apiv1models.BillingProfile{}, fmt.Errorf("failed to map profile apps: %w", err)
		}

		if apps != nil {
			out.Apps = *apps
		}
	} else {
		apps, err := billingconvert.ToAPIBillingProfileAppReferences(p.AppReferences)
		if err != nil {
			return apiv1models.BillingProfile{}, fmt.Errorf("failed to map profile app references: %w", err)
		}

		if apps != nil {
			out.Apps = *apps
		}
	}

	return out, nil
}

func mapProfileExpandToEntity(expand []apiv1models.BillingProfileExpand) billing.ProfileExpand {
	if len(expand) == 0 {
		return billing.ProfileExpand{}
	}

	return billing.ProfileExpand{
		Apps: slices.Contains(expand, apiv1models.BillingProfileExpandApps),
	}
}
