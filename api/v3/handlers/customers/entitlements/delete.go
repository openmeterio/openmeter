package customersentitlements

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	DeleteCustomerEntitlementRequest  = entitlement.DeleteCustomerEntitlementInput
	DeleteCustomerEntitlementResponse = interface{}
	DeleteCustomerEntitlementParams   struct {
		CustomerID    apiv3models.ULID
		EntitlementID apiv3models.ULID
	}
	DeleteCustomerEntitlementHandler = httptransport.HandlerWithArgs[DeleteCustomerEntitlementRequest, DeleteCustomerEntitlementResponse, DeleteCustomerEntitlementParams]
)

func (h *handler) DeleteCustomerEntitlement() DeleteCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params DeleteCustomerEntitlementParams) (DeleteCustomerEntitlementRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteCustomerEntitlementRequest{}, err
			}

			return DeleteCustomerEntitlementRequest{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        params.CustomerID,
				},
				EntitlementID: params.EntitlementID,
			}, nil
		},
		func(ctx context.Context, req DeleteCustomerEntitlementRequest) (DeleteCustomerEntitlementResponse, error) {
			if err := h.service.DeleteCustomerEntitlement(ctx, req); err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteCustomerEntitlementResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-customer-entitlement"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
