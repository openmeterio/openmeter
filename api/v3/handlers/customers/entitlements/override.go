package customersentitlements

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	OverrideCustomerEntitlementRequest  = entitlement.OverrideCustomerEntitlementInput
	OverrideCustomerEntitlementResponse = apiv3models.BillingEntitlement
	OverrideCustomerEntitlementParams   struct {
		CustomerID    apiv3models.ULID
		EntitlementID apiv3models.ULID
	}
	OverrideCustomerEntitlementHandler = httptransport.HandlerWithArgs[OverrideCustomerEntitlementRequest, OverrideCustomerEntitlementResponse, OverrideCustomerEntitlementParams]
)

func (h *handler) OverrideCustomerEntitlement() OverrideCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params OverrideCustomerEntitlementParams) (OverrideCustomerEntitlementRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return OverrideCustomerEntitlementRequest{}, err
			}

			var body apiv3models.CreateEntitlementRequest
			if err := request.ParseBody(r, &body); err != nil {
				return OverrideCustomerEntitlementRequest{}, err
			}

			customerID := customer.CustomerID{
				Namespace: ns,
				ID:        params.CustomerID,
			}

			create, err := fromAPICreateEntitlementRequest(customerID, body, clock.Now())
			if err != nil {
				return OverrideCustomerEntitlementRequest{}, apierrors.NewBadRequestError(ctx, err, nil)
			}

			return OverrideCustomerEntitlementRequest{
				CustomerID:    customerID,
				EntitlementID: params.EntitlementID,
				Entitlement:   create.Entitlement,
				Grants:        create.Grants,
			}, nil
		},
		func(ctx context.Context, req OverrideCustomerEntitlementRequest) (OverrideCustomerEntitlementResponse, error) {
			overridden, err := h.service.OverrideCustomerEntitlement(ctx, req)
			if err != nil {
				return OverrideCustomerEntitlementResponse{}, err
			}

			return ToAPIBillingEntitlement(overridden)
		},
		commonhttp.JSONResponseEncoderWithStatus[OverrideCustomerEntitlementResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("override-customer-entitlement"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
