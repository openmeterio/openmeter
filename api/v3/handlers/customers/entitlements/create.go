package customersentitlements

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateCustomerEntitlementRequest  = entitlement.CreateCustomerEntitlementInput
	CreateCustomerEntitlementResponse = api.BillingEntitlement
	CreateCustomerEntitlementParams   struct {
		CustomerID api.ULID
	}
	CreateCustomerEntitlementHandler = httptransport.HandlerWithArgs[CreateCustomerEntitlementRequest, CreateCustomerEntitlementResponse, CreateCustomerEntitlementParams]
)

func (h *handler) CreateCustomerEntitlement() CreateCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params CreateCustomerEntitlementParams) (CreateCustomerEntitlementRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCustomerEntitlementRequest{}, err
			}

			var body api.CreateEntitlementRequest
			if err := request.ParseBody(r, &body); err != nil {
				return CreateCustomerEntitlementRequest{}, err
			}

			customerID := customer.CustomerID{
				Namespace: ns,
				ID:        params.CustomerID,
			}

			req, err := fromAPICreateEntitlementRequest(customerID, body, clock.Now())
			if err != nil {
				return CreateCustomerEntitlementRequest{}, apierrors.NewBadRequestError(ctx, err, nil)
			}

			return req, nil
		},
		func(ctx context.Context, req CreateCustomerEntitlementRequest) (CreateCustomerEntitlementResponse, error) {
			created, err := h.service.CreateCustomerEntitlement(ctx, req)
			if err != nil {
				return CreateCustomerEntitlementResponse{}, err
			}

			return ToAPIBillingEntitlement(created)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerEntitlementResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-customer-entitlement"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
