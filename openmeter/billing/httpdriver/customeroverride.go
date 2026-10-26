package httpdriver

import (
	"context"
	"errors"
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
	ListCustomerOverridesRequest  = billing.ListCustomerBillingProfilesInput
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

			expand, err := h.mapCustomerBillingProfileExpandToEntity(input.Expand)
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
				OrderBy: billing.CustomerBillingProfileOrderBy(lo.FromPtrOr(input.OrderBy, api.BillingProfileCustomerOverrideOrderByOrderingOrderBy(billing.DefaultCustomerBillingProfileOrderBy))),
				Order:   sortx.Order(lo.FromPtrOr(input.Order, api.SortOrderDESC)),

				Page: pagination.Page{
					PageSize:   lo.FromPtr(input.PageSize),
					PageNumber: lo.FromPtr(input.Page),
				},
			}, nil
		},
		func(ctx context.Context, input ListCustomerOverridesRequest) (ListCustomerOverridesResponse, error) {
			overrides, err := h.service.ListCustomerBillingProfiles(ctx, input)
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
	GetCustomerOverrideRequest  = billing.GetCustomerBillingProfileInput
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

			expand, err := h.mapCustomerBillingProfileExpandToEntity(input.Expand)
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
			override, err := h.service.GetCustomerBillingProfile(ctx, input)
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
	UpsertCustomerOverrideRequest struct {
		Customer  customer.CustomerID
		ProfileID *string
	}
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
				Customer:  customer.CustomerID{Namespace: ns, ID: input.CustomerId},
				ProfileID: body.BillingProfileId,
			}, nil
		},
		func(ctx context.Context, input UpsertCustomerOverrideRequest) (UpsertCustomerOverrideResponse, error) {
			var profile billing.CustomerBillingProfile
			var err error
			if input.ProfileID == nil {
				// Preserve the pin if the effective profile cannot be resolved for the response.
				err = h.service.WithLock(ctx, input.Customer, func(ctx context.Context) error {
					clearErr := h.service.ClearCustomerBillingProfile(ctx, billing.ClearCustomerBillingProfileInput{Customer: input.Customer})
					if clearErr != nil && !errors.Is(clearErr, billing.ErrCustomerOverrideNotFound) {
						return clearErr
					}

					var resolveErr error
					profile, resolveErr = h.service.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{Customer: input.Customer})

					return resolveErr
				})
			} else {
				profile, err = h.service.SetCustomerBillingProfile(ctx, billing.SetCustomerBillingProfileInput{
					Customer:  input.Customer,
					ProfileID: *input.ProfileID,
				})
			}

			if err != nil {
				return UpsertCustomerOverrideResponse{}, err
			}

			return h.mapCustomerOverrideToAPI(ctx, profile)
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
	DeleteCustomerOverrideRequest  = billing.ClearCustomerBillingProfileInput
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
			err := h.service.ClearCustomerBillingProfile(ctx, input)
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

func (h *handler) mapCustomerBillingProfileExpandToEntity(in *api.BillingProfileListCustomerOverridesParamsExpand) (billing.CustomerBillingProfileExpand, error) {
	if in == nil {
		return billing.CustomerBillingProfileExpand{}, nil
	}

	expand := billing.CustomerBillingProfileExpand{}

	for _, e := range *in {
		switch e {
		case api.BillingProfileCustomerOverrideExpandApps:
			expand.Apps = true
		case api.BillingProfileCustomerOverrideExpandCustomer:
			expand.Customer = true
		default:
			return billing.CustomerBillingProfileExpand{}, models.NewGenericValidationError(fmt.Errorf("invalid expand: %s", e))
		}
	}

	return expand, nil
}

func (h *handler) mapCustomerOverrideToAPI(ctx context.Context, in billing.CustomerBillingProfile) (api.BillingProfileCustomerOverrideWithDetails, error) {
	res := api.BillingProfileCustomerOverrideWithDetails{}

	if in.AssignedProfile != nil {
		res.CustomerOverride = &api.BillingProfileCustomerOverride{
			BillingProfileId: &in.AssignedProfile.ProfileID,
			CustomerId:       in.CustomerID.ID,
			CreatedAt:        in.AssignedProfile.CreatedAt,
			UpdatedAt:        in.AssignedProfile.UpdatedAt,
		}
	}

	if in.Customer != nil {
		customer, err := customerhttpdriver.CustomerToAPI(*in.Customer, nil, nil)
		if err != nil {
			return res, err
		}

		res.Customer = &customer
	}

	res.BaseBillingProfileId = in.EffectiveProfile.ID

	profile, err := h.mapCustomerProfileToAPI(ctx, in.EffectiveProfile)
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
