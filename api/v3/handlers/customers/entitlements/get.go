package customersentitlements

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetCustomerEntitlementRequest  = entitlement.GetCustomerEntitlementInput
	GetCustomerEntitlementResponse = api.BillingEntitlement
	GetCustomerEntitlementParams   struct {
		CustomerID    api.ULID
		EntitlementID api.ULID
	}
	GetCustomerEntitlementHandler = httptransport.HandlerWithArgs[GetCustomerEntitlementRequest, GetCustomerEntitlementResponse, GetCustomerEntitlementParams]
)

func (h *handler) GetCustomerEntitlement() GetCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerEntitlementParams) (GetCustomerEntitlementRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCustomerEntitlementRequest{}, err
			}

			return GetCustomerEntitlementRequest{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        params.CustomerID,
				},
				EntitlementID: params.EntitlementID,
			}, nil
		},
		func(ctx context.Context, req GetCustomerEntitlementRequest) (GetCustomerEntitlementResponse, error) {
			ent, err := h.service.GetCustomerEntitlement(ctx, req)
			if err != nil {
				return GetCustomerEntitlementResponse{}, err
			}

			return ToAPIBillingEntitlement(ent)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerEntitlementResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-entitlement"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
