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
	CreateCustomerEntitlementGrantRequest  = entitlement.CreateCustomerEntitlementGrantInput
	CreateCustomerEntitlementGrantResponse = apiv3models.BillingEntitlementGrant
	CreateCustomerEntitlementGrantParams   struct {
		CustomerID    apiv3models.ULID
		EntitlementID apiv3models.ULID
	}
	CreateCustomerEntitlementGrantHandler = httptransport.HandlerWithArgs[CreateCustomerEntitlementGrantRequest, CreateCustomerEntitlementGrantResponse, CreateCustomerEntitlementGrantParams]
)

func (h *handler) CreateCustomerEntitlementGrant() CreateCustomerEntitlementGrantHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params CreateCustomerEntitlementGrantParams) (CreateCustomerEntitlementGrantRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCustomerEntitlementGrantRequest{}, err
			}

			var body apiv3models.BillingEntitlementGrantCreateRequest
			if err := request.ParseBody(r, &body); err != nil {
				return CreateCustomerEntitlementGrantRequest{}, err
			}

			grantInput, err := fromAPIEntitlementGrantCreateRequest(body)
			if err != nil {
				return CreateCustomerEntitlementGrantRequest{}, apierrors.NewBadRequestError(ctx, err, nil)
			}

			return CreateCustomerEntitlementGrantRequest{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        params.CustomerID,
				},
				EntitlementID: params.EntitlementID,
				Grant:         grantInput,
			}, nil
		},
		func(ctx context.Context, req CreateCustomerEntitlementGrantRequest) (CreateCustomerEntitlementGrantResponse, error) {
			created, err := h.service.CreateCustomerEntitlementGrant(ctx, req)
			if err != nil {
				return CreateCustomerEntitlementGrantResponse{}, err
			}

			return ToAPIEntitlementGrant(created, clock.Now())
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerEntitlementGrantResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-customer-entitlement-grant"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
