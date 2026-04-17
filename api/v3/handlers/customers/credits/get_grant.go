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
	GetCreditGrantRequest  = creditgrant.GetInput
	GetCreditGrantResponse = api.BillingCreditGrant
	GetCreditGrantParams   struct {
		CustomerID    api.ULID
		CreditGrantID api.ULID
	}
	GetCreditGrantHandler httptransport.HandlerWithArgs[GetCreditGrantRequest, GetCreditGrantResponse, GetCreditGrantParams]
)

func (h *handler) GetCreditGrant() GetCreditGrantHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args GetCreditGrantParams) (GetCreditGrantRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetCreditGrantRequest{}, err
			}

			return GetCreditGrantRequest{
				Namespace:  ns,
				CustomerID: args.CustomerID,
				ChargeID:   args.CreditGrantID,
			}, nil
		},
		func(ctx context.Context, request GetCreditGrantRequest) (GetCreditGrantResponse, error) {
			charge, err := h.creditGrantService.Get(ctx, request)
			if err != nil {
				return GetCreditGrantResponse{}, err
			}

			return toAPIBillingCreditGrant(charge)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCreditGrantResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-credit-grant"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
