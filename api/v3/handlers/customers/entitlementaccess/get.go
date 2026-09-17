package customersentitlement

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetCustomerEntitlementAccessParams struct {
		CustomerID api.ULID
		FeatureKey api.ResourceKey
		Params     api.GetCustomerEntitlementAccessParams
	}
	GetCustomerEntitlementAccessResponse = api.BillingEntitlementAccessResult
	GetCustomerEntitlementAccessHandler  httptransport.HandlerWithArgs[GetCustomerEntitlementAccessRequest, GetCustomerEntitlementAccessResponse, GetCustomerEntitlementAccessParams]
)

type GetCustomerEntitlementAccessRequest struct {
	entitlement.GetCustomerEntitlementAccessInput
	Expands []api.BillingEntitlementAccessExpand
}

func (h *handler) GetCustomerEntitlementAccess() GetCustomerEntitlementAccessHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerEntitlementAccessParams) (GetCustomerEntitlementAccessRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerEntitlementAccessRequest{}, err
			}

			return GetCustomerEntitlementAccessRequest{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        params.CustomerID,
				},
				FeatureKey: params.FeatureKey,
				Expands:    lo.FromPtr(params.Params.Expand),
			}, nil
		},
		func(ctx context.Context, request GetCustomerEntitlementAccessRequest) (GetCustomerEntitlementAccessResponse, error) {
			access, err := h.entitlementService.GetCustomerEntitlementAccess(ctx, request.GetCustomerEntitlementAccessInput)
			if err != nil {
				return GetCustomerEntitlementAccessResponse{}, err
			}

			return mapEntitlementAccessToAPI(access, request.Expands...)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerEntitlementAccessResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-entitlement-access"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
