package customersentitlement

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetCustomerEntitlementAccessParams struct {
		CustomerID api.ULID
		FeatureKey string
	}
	GetCustomerEntitlementAccessResponse = api.BillingEntitlementAccessCheckResult
	GetCustomerEntitlementAccessHandler  httptransport.HandlerWithArgs[entitlement.GetCustomerEntitlementAccessInput, GetCustomerEntitlementAccessResponse, GetCustomerEntitlementAccessParams]
)

func (h *handler) GetCustomerEntitlementAccess() GetCustomerEntitlementAccessHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerEntitlementAccessParams) (entitlement.GetCustomerEntitlementAccessInput, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return entitlement.GetCustomerEntitlementAccessInput{}, err
			}

			return entitlement.GetCustomerEntitlementAccessInput{
				CustomerID: customer.CustomerID{Namespace: ns, ID: params.CustomerID},
				FeatureKey: params.FeatureKey,
				At:         clock.Now(),
			}, nil
		},
		func(ctx context.Context, request entitlement.GetCustomerEntitlementAccessInput) (GetCustomerEntitlementAccessResponse, error) {
			access, err := h.entitlementService.GetCustomerEntitlementAccess(ctx, request)
			if err != nil {
				return GetCustomerEntitlementAccessResponse{}, err
			}

			return mapEntitlementAccessCheckToAPI(access)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCustomerEntitlementAccessResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-customer-entitlement-access"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
