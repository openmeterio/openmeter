package customerscredits

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	VoidCreditGrantRequest  = creditgrant.VoidInput
	VoidCreditGrantResponse = apiv3models.BillingCreditGrant
	VoidCreditGrantParams   struct {
		CustomerID    apiv3models.ULID
		CreditGrantID apiv3models.ULID
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

			var body apiv3models.VoidCreditGrantRequest
			if err := request.ParseOptionalBody(r, &body); err != nil {
				return VoidCreditGrantRequest{}, err
			}

			return fromAPIVoidCreditGrantRequest(ns, args.CustomerID, args.CreditGrantID, body)
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
