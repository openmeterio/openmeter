package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerhttpdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListCustomerOverridesRequest  = billing.ListCustomerOverridesInput
	ListCustomerOverridesResponse = api.BillingProfileCustomerOverrideWithDetailsPaginatedResponse
	ListCustomerOverridesParams   = api.ListBillingProfileCustomerOverridesParams
	ListCustomerOverridesHandler  httptransport.HandlerWithArgs[ListCustomerOverridesRequest, ListCustomerOverridesResponse, ListCustomerOverridesParams]
)

func (h *handler) ListCustomerOverrides() ListCustomerOverridesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, input ListCustomerOverridesParams) (ListCustomerOverridesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerOverridesRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			expand, err := h.mapCustomerOverrideExpandToEntity(input.Expand)
			if err != nil {
				return ListCustomerOverridesRequest{}, err
			}

			return ListCustomerOverridesRequest{
				Namespace:                     ns,
				BillingProfiles:               lo.FromPtr(input.BillingProfile),
				IncludeAllCustomers:           lo.FromPtr(input.IncludeAllCustomers),
				CustomersWithoutPinnedProfile: lo.FromPtr(input.CustomersWithoutPinnedProfile),
				CustomerIDs:                   lo.FromPtr(input.CustomerId),
				CustomerName:                  lo.FromPtr(input.CustomerName),
				CustomerKey:                   lo.FromPtr(input.CustomerKey),
				CustomerPrimaryEmail:          lo.FromPtr(input.CustomerPrimaryEmail),

				Expand:  expand,
				OrderBy: billing.CustomerOverrideOrderBy(lo.FromPtrOr(input.OrderBy, api.BillingProfileCustomerOverrideOrderByOrderingOrderBy(billing.DefaultCustomerOverrideOrderBy))),
				Order:   sortx.Order(lo.FromPtrOr(input.Order, api.SortOrderDESC)),

				Page: pagination.Page{
					PageSize:   lo.FromPtr(input.PageSize),
					PageNumber: lo.FromPtr(input.Page),
				},
			}, nil
		},
		func(ctx context.Context, input ListCustomerOverridesRequest) (ListCustomerOverridesResponse, error) {
			overrides, err := h.service.ListCustomerOverrides(ctx, input)
			if err != nil {
				return ListCustomerOverridesResponse{}, err
			}

			res := ListCustomerOverridesResponse{
				Items:      make([]api.BillingProfileCustomerOverrideWithDetails, 0, len(overrides.Items)),
				Page:       overrides.Page.PageNumber,
				PageSize:   overrides.Page.PageSize,
				TotalCount: overrides.TotalCount,
			}

			for _, override := range overrides.Items {
				override, err := h.mapCustomerOverrideToAPI(ctx, override)
				if err != nil {
					return ListCustomerOverridesResponse{}, err
				}

				res.Items = append(res.Items, override)
			}

			return res, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomerOverridesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("ListCustomerOverrides"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	GetCustomerOverrideRequest  = billing.GetCustomerOverrideInput
	GetCustomerOverrideResponse = api.BillingProfileCustomerOverrideWithDetails
	GetCustomerOverrideParams   struct {
		CustomerId string `json:"customerId"`
		Expand     *api.BillingProfileListCustomerOverridesParamsExpand
	}
	GetCustomerOverrideHandler httptransport.HandlerWithArgs[GetCustomerOverrideRequest, GetCustomerOverrideResponse, GetCustomerOverrideParams]
)

func (h *handler) GetCustomerOverride() GetCustomerOverrideHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, input GetCustomerOverrideParams) (GetCustomerOverrideRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerOverrideRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			expand, err := h.mapCustomerOverrideExpandToEntity(input.Expand)
			if err != nil {
				return GetCustomerOverrideRequest{}, err
			}

			return GetCustomerOverrideRequest{
				Customer: customer.CustomerID{
					Namespace: ns,
					ID:        input.CustomerId,
				},
				Expand: expand,
			}, nil
		},
		func(ctx context.Context, input GetCustomerOverrideRequest) (GetCustomerOverrideResponse, error) {
			override, err := h.service.GetCustomerOverride(ctx, input)
			if err != nil {
				return GetCustomerOverrideResponse{}, err
			}

			res, err := h.mapCustomerOverrideToAPI(ctx, override)
			if err != nil {
				return GetCustomerOverrideResponse{}, err
			}

			return res, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerOverrideResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("GetCustomerOverride"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	UpsertCustomerOverrideRequest  = billing.UpsertCustomerOverrideInput
	UpsertCustomerOverrideResponse = api.BillingProfileCustomerOverrideWithDetails
	UpsertCustomerOverrideParams   struct {
		CustomerId string `json:"customerId"`
	}
	UpsertCustomerOverrideHandler httptransport.HandlerWithArgs[UpsertCustomerOverrideRequest, UpsertCustomerOverrideResponse, UpsertCustomerOverrideParams]
)

func (h *handler) UpsertCustomerOverride() UpsertCustomerOverrideHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, input UpsertCustomerOverrideParams) (UpsertCustomerOverrideRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpsertCustomerOverrideRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			body := api.BillingProfileCustomerOverrideCreate{}

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpsertCustomerOverrideRequest{}, err
			}

			return UpsertCustomerOverrideRequest{
				Namespace:  ns,
				CustomerID: input.CustomerId,

				ProfileID: lo.FromPtr(body.BillingProfileId),
			}, nil
		},
		func(ctx context.Context, input UpsertCustomerOverrideRequest) (UpsertCustomerOverrideResponse, error) {
			override, err := h.service.UpsertCustomerOverride(ctx, input)
			if err != nil {
				return UpsertCustomerOverrideResponse{}, err
			}

			return h.mapCustomerOverrideToAPI(ctx, override)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpsertCustomerOverrideResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("UpsertCustomerOverride"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	DeleteCustomerOverrideRequest  = billing.DeleteCustomerOverrideInput
	DeleteCustomerOverrideResponse = struct{}
	DeleteCustomerOverrideParams   struct {
		CustomerId string `json:"customerId"`
	}
	DeleteCustomerOverrideHandler httptransport.HandlerWithArgs[DeleteCustomerOverrideRequest, DeleteCustomerOverrideResponse, DeleteCustomerOverrideParams]
)

func (h *handler) DeleteCustomerOverride() DeleteCustomerOverrideHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, input DeleteCustomerOverrideParams) (DeleteCustomerOverrideRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteCustomerOverrideRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return DeleteCustomerOverrideRequest{
				Customer: customer.CustomerID{
					Namespace: ns,
					ID:        input.CustomerId,
				},
			}, nil
		},
		func(ctx context.Context, input DeleteCustomerOverrideRequest) (DeleteCustomerOverrideResponse, error) {
			err := h.service.DeleteCustomerOverride(ctx, input)
			if err != nil {
				return DeleteCustomerOverrideResponse{}, err
			}

			return DeleteCustomerOverrideResponse{}, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteCustomerOverrideResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("DeleteCustomerOverride"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func (h *handler) mapCustomerOverrideExpandToEntity(in *api.BillingProfileListCustomerOverridesParamsExpand) (billing.CustomerOverrideExpand, error) {
	if in == nil {
		return billing.CustomerOverrideExpand{}, nil
	}

	expand := billing.CustomerOverrideExpand{}

	for _, e := range *in {
		switch e {
		case api.BillingProfileCustomerOverrideExpandApps:
			expand.Apps = true
		case api.BillingProfileCustomerOverrideExpandCustomer:
			expand.Customer = true
		default:
			return billing.CustomerOverrideExpand{}, models.NewGenericValidationError(fmt.Errorf("invalid expand: %s", e))
		}
	}

	return expand, nil
}

func (h *handler) mapCustomerOverrideToAPI(ctx context.Context, in billing.CustomerOverrideWithDetails) (api.BillingProfileCustomerOverrideWithDetails, error) {
	res := api.BillingProfileCustomerOverrideWithDetails{}

	if in.CustomerOverride != nil {
		var profileID *string

		if in.CustomerOverride.Profile != nil && in.CustomerOverride.Profile.ID != "" {
			profileID = &in.CustomerOverride.Profile.ID
		}

		res.CustomerOverride = &api.BillingProfileCustomerOverride{
			// TODO[later]: add workflow mappings here

			BillingProfileId: profileID,
			CustomerId:       in.CustomerOverride.ID,

			CreatedAt: in.CustomerOverride.CreatedAt,
			UpdatedAt: in.CustomerOverride.UpdatedAt,
		}
	}

	if in.Customer != nil {
		customer, err := customerhttpdriver.CustomerToAPI(*in.Customer, nil, nil)
		if err != nil {
			return res, err
		}

		res.Customer = &customer
	}

	res.BaseBillingProfileId = in.MergedProfile.ID

	profile, err := h.mapCustomerProfileToAPI(ctx, in.MergedProfile)
	if err != nil {
		return res, err
	}

	res.CustomerProfile = &profile

	return res, nil
}

func (h *handler) mapCustomerProfileToAPI(ctx context.Context, in billing.Profile) (api.BillingCustomerProfile, error) {
	profile, err := h.MapProfileToApi(ctx, &in)
	if err != nil {
		return api.BillingCustomerProfile{}, err
	}

	return api.BillingCustomerProfile{
		Supplier: profile.Supplier,
		Workflow: profile.Workflow,
		Apps:     profile.Apps,
	}, nil
}
