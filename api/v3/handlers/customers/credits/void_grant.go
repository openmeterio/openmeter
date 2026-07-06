package customerscredits

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	VoidCreditGrantRequest  = creditgrant.VoidInput
	VoidCreditGrantResponse = api.BillingCreditGrant
	VoidCreditGrantParams   struct {
		CustomerID    api.ULID
		CreditGrantID api.ULID
	}
	VoidCreditGrantHandler = httptransport.HandlerWithArgs[VoidCreditGrantRequest, VoidCreditGrantResponse, VoidCreditGrantParams]
)

func (h *handler) VoidCreditGrant() VoidCreditGrantHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args VoidCreditGrantParams) (VoidCreditGrantRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return VoidCreditGrantRequest{}, err
			}

			return VoidCreditGrantRequest{
				Namespace:  ns,
				CustomerID: args.CustomerID,
				ChargeID:   args.CreditGrantID,
			}, nil
		},
		func(ctx context.Context, request VoidCreditGrantRequest) (VoidCreditGrantResponse, error) {
			grant, err := h.creditGrantService.Void(ctx, request)
			if err != nil {
				return VoidCreditGrantResponse{}, err
			}

			return toAPIBillingCreditGrant(grant)
		},
		commonhttp.JSONResponseEncoderWithStatus[VoidCreditGrantResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("void-credit-grant"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
