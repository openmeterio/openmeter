package customersentitlements

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ResetCustomerEntitlementUsageRequest  = entitlement.ResetCustomerEntitlementUsageInput
	ResetCustomerEntitlementUsageResponse = interface{}
	ResetCustomerEntitlementUsageParams   struct {
		CustomerID    api.ULID
		EntitlementID api.ULID
	}
	ResetCustomerEntitlementUsageHandler = httptransport.HandlerWithArgs[ResetCustomerEntitlementUsageRequest, ResetCustomerEntitlementUsageResponse, ResetCustomerEntitlementUsageParams]
)

func (h *handler) ResetCustomerEntitlementUsage() ResetCustomerEntitlementUsageHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ResetCustomerEntitlementUsageParams) (ResetCustomerEntitlementUsageRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ResetCustomerEntitlementUsageRequest{}, err
			}

			var body api.ResetCustomerEntitlementUsageRequest
			if err := request.ParseOptionalBody(r, &body); err != nil {
				return ResetCustomerEntitlementUsageRequest{}, err
			}

			return ResetCustomerEntitlementUsageRequest{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        params.CustomerID,
				},
				EntitlementID:   params.EntitlementID,
				EffectiveAt:     body.EffectiveAt,
				RetainAnchor:    lo.FromPtr(body.RetainAnchor),
				PreserveOverage: body.PreserveOverage,
			}, nil
		},
		func(ctx context.Context, req ResetCustomerEntitlementUsageRequest) (ResetCustomerEntitlementUsageResponse, error) {
			return nil, h.service.ResetCustomerEntitlementUsage(ctx, req)
		},
		commonhttp.EmptyResponseEncoder[ResetCustomerEntitlementUsageResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("reset-customer-entitlement-usage"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
