package entitlementdriverv2

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListCustomerEntitlementGrantsHandlerParams struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
		Params                    api.ListCustomerEntitlementGrantsV2Params
	}
	ListCustomerEntitlementGrantsHandlerRequest struct {
		CustomerIDOrKey           string
		EntitlementIdOrFeatureKey string
		Namespace                 string
		Params                    api.ListCustomerEntitlementGrantsV2Params
	}
	ListCustomerEntitlementGrantsHandlerResponse = api.GrantPaginatedResponse
	ListCustomerEntitlementGrantsHandler         = httptransport.HandlerWithArgs[ListCustomerEntitlementGrantsHandlerRequest, ListCustomerEntitlementGrantsHandlerResponse, ListCustomerEntitlementGrantsHandlerParams]
)

func (h *customerEntitlementHandler) ListCustomerEntitlementGrants() ListCustomerEntitlementGrantsHandler {
	return httptransport.NewHandlerWithArgs[ListCustomerEntitlementGrantsHandlerRequest, ListCustomerEntitlementGrantsHandlerResponse, ListCustomerEntitlementGrantsHandlerParams](
		func(ctx context.Context, r *http.Request, params ListCustomerEntitlementGrantsHandlerParams) (ListCustomerEntitlementGrantsHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerEntitlementGrantsHandlerRequest{}, err
			}

			return ListCustomerEntitlementGrantsHandlerRequest{
				CustomerIDOrKey:           params.CustomerIDOrKey,
				EntitlementIdOrFeatureKey: params.EntitlementIdOrFeatureKey,
				Namespace:                 ns,
			}, nil
		},
		func(ctx context.Context, request ListCustomerEntitlementGrantsHandlerRequest) (ListCustomerEntitlementGrantsHandlerResponse, error) {
			cust, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerIDOrKey: &customer.CustomerIDOrKey{
					Namespace: request.Namespace,
					IDOrKey:   request.CustomerIDOrKey,
				},
			})
			if err != nil {
				return ListCustomerEntitlementGrantsHandlerResponse{}, err
			}

			grants, err := h.balanceConnector.ListEntitlementGrants(ctx, request.Namespace, meteredentitlement.ListEntitlementGrantsParams{
				CustomerID:                cust.ID,
				EntitlementIDOrFeatureKey: request.EntitlementIdOrFeatureKey,
				OrderBy:                   grant.OrderBy(lo.CoalesceOrEmpty(string(lo.FromPtr(request.Params.OrderBy)), string(grant.OrderByDefault))),
				Order:                     sortx.Order(lo.CoalesceOrEmpty(string(lo.FromPtr(request.Params.Order)), string(sortx.OrderDefault))),
				Page: pagination.NewPage(
					lo.FromPtrOr(request.Params.Page, 1),
					lo.FromPtrOr(request.Params.PageSize, 100),
				),
			})
			if err != nil {
				return ListCustomerEntitlementGrantsHandlerResponse{}, err
			}

			mapped := pagination.MapPagedResponse(grants, func(grant meteredentitlement.EntitlementGrant) api.EntitlementGrant {
				return entitlementdriver.MapEntitlementGrantToAPI(&grant)
			})

			return ListCustomerEntitlementGrantsHandlerResponse{
				Items:      mapped.Items,
				Page:       mapped.Page.PageNumber,
				PageSize:   mapped.Page.PageSize,
				TotalCount: mapped.TotalCount,
			}, nil
		},
		commonhttp.JSONResponseEncoder[ListCustomerEntitlementGrantsHandlerResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listCustomerEntitlementGrantsV2"),
			httptransport.WithErrorEncoder(getErrorEncoder()),
		)...,
	)
}
