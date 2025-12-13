package customersentitlement

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CustomerID                            = string
	ListCustomerEntitlementAccessResponse = api.ListCustomerEntitlementAccessResponseData
	ListCustomerEntitlementAccessHandler  httptransport.HandlerWithArgs[ListCustomerEntitlementAccessRequest, ListCustomerEntitlementAccessResponse, CustomerID]
)

type ListCustomerEntitlementAccessRequest struct {
	CustomerID customer.CustomerID
}

func (h *handler) ListCustomerEntitlementAccess() ListCustomerEntitlementAccessHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID CustomerID) (ListCustomerEntitlementAccessRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerEntitlementAccessRequest{}, err
			}

			req := ListCustomerEntitlementAccessRequest{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        customerID,
				},
			}

			return req, nil
		},
		func(ctx context.Context, request ListCustomerEntitlementAccessRequest) (ListCustomerEntitlementAccessResponse, error) {
			// Get the customer
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: &request.CustomerID,
			})
			if err != nil {
				return ListCustomerEntitlementAccessResponse{}, err
			}

			if cus != nil && cus.IsDeleted() {
				return ListCustomerEntitlementAccessResponse{},
					apierrors.NewPreconditionFailedError(ctx,
						fmt.Sprintf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID),
					)
			}

			// Get the access
			access, err := h.entitlementService.GetAccess(ctx, cus.Namespace, cus.ID)
			if err != nil {
				return ListCustomerEntitlementAccessResponse{}, err
			}

			// Convert the access to the API response
			items := make([]api.BillingEntitlementAccessResult, 0, len(access.Entitlements))
			for featureKey, entitlement := range access.Entitlements {
				found, item, err := mapEntitlementValueToAPI(featureKey, entitlement.Value)
				if err != nil {
					return ListCustomerEntitlementAccessResponse{}, err
				}

				if !found {
					continue
				}

				items = append(items, item)
			}

			// Sort the items by feature key
			sort.Slice(items, func(i, j int) bool {
				return items[i].FeatureKey < items[j].FeatureKey
			})

			// Return the response
			return ListCustomerEntitlementAccessResponse{
				Data: items,
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
