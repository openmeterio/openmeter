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
	CustomerID                            = string
	ListCustomerEntitlementAccessRequest  = entitlement.ListCustomerEntitlementAccessInput
	ListCustomerEntitlementAccessResponse = api.ListCustomerEntitlementAccessResponseData
	ListCustomerEntitlementAccessHandler  httptransport.HandlerWithArgs[ListCustomerEntitlementAccessRequest, ListCustomerEntitlementAccessResponse, CustomerID]
)

func (h *handler) ListCustomerEntitlementAccess() ListCustomerEntitlementAccessHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID CustomerID) (ListCustomerEntitlementAccessRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerEntitlementAccessRequest{}, err
			}

			return ListCustomerEntitlementAccessRequest{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        customerID,
				},
			}, nil
		},
		func(ctx context.Context, request ListCustomerEntitlementAccessRequest) (ListCustomerEntitlementAccessResponse, error) {
			items, err := h.entitlementService.ListCustomerEntitlementAccess(ctx, request)
			if err != nil {
				return ListCustomerEntitlementAccessResponse{}, err
			}

			data, err := lo.MapErr(items, func(item entitlement.CustomerEntitlementAccess, _ int) (api.BillingEntitlementValueResult, error) {
				return mapEntitlementAccessToAPI(item)
			})
			if err != nil {
				return ListCustomerEntitlementAccessResponse{}, err
			}

			return ListCustomerEntitlementAccessResponse{
				Data: data,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomerEntitlementAccessResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-customer-entitlement-access"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
