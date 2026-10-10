package entitlements

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	customersentitlements "github.com/openmeterio/openmeter/api/v3/handlers/customers/entitlements"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetEntitlementRequest  = entitlement.GetEntitlementByIDInput
	GetEntitlementResponse = api.BillingEntitlement
	GetEntitlementParams   struct {
		EntitlementID api.ULID
	}
	GetEntitlementHandler = httptransport.HandlerWithArgs[GetEntitlementRequest, GetEntitlementResponse, GetEntitlementParams]
)

func (h *handler) GetEntitlement() GetEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetEntitlementParams) (GetEntitlementRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetEntitlementRequest{}, err
			}

			return GetEntitlementRequest{
				Namespace:     ns,
				EntitlementID: params.EntitlementID,
			}, nil
		},
		func(ctx context.Context, req GetEntitlementRequest) (GetEntitlementResponse, error) {
			ent, err := h.service.GetEntitlementByID(ctx, req)
			if err != nil {
				return GetEntitlementResponse{}, err
			}

			return customersentitlements.ToAPIBillingEntitlement(ent)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetEntitlementResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-entitlement"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
