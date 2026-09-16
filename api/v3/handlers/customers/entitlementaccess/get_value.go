package customersentitlement

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetCustomerEntitlementValueParams struct {
		CustomerID    api.ULID
		EntitlementID api.ULID
		Params        api.GetCustomerEntitlementValueParams
	}
	GetCustomerEntitlementValueResponse = api.BillingEntitlementAccessResult
	GetCustomerEntitlementValueHandler  httptransport.HandlerWithArgs[GetCustomerEntitlementValueRequest, GetCustomerEntitlementValueResponse, GetCustomerEntitlementValueParams]
)

type GetCustomerEntitlementValueRequest struct {
	entitlement.GetCustomerEntitlementValueInput
	Expands []api.BillingEntitlementAccessExpand
}

func (h *handler) GetCustomerEntitlementValue() GetCustomerEntitlementValueHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerEntitlementValueParams) (GetCustomerEntitlementValueRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerEntitlementValueRequest{}, err
			}

			return GetCustomerEntitlementValueRequest{
				GetCustomerEntitlementValueInput: entitlement.GetCustomerEntitlementValueInput{
					CustomerID: customer.CustomerID{
						Namespace: ns,
						ID:        params.CustomerID,
					},
					EntitlementID: params.EntitlementID,
					At:            lo.FromPtrOr(params.Params.At, clock.Now()),
				},
				Expands: lo.FromPtr(params.Params.Expand),
			}, nil
		},
		func(ctx context.Context, request GetCustomerEntitlementValueRequest) (GetCustomerEntitlementValueResponse, error) {
			access, err := h.entitlementService.GetCustomerEntitlementValue(ctx, request.GetCustomerEntitlementValueInput)
			if err != nil {
				return GetCustomerEntitlementValueResponse{}, err
			}

			return mapEntitlementAccessToAPI(access, request.Expands...)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerEntitlementValueResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-entitlement-value"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
