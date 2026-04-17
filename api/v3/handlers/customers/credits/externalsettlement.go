package customerscredits

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	UpdateCreditGrantExternalSettlementRequest  = creditgrant.UpdateExternalSettlementInput
	UpdateCreditGrantExternalSettlementResponse = api.BillingCreditGrant
	UpdateCreditGrantExternalSettlementParams   struct {
		CustomerID    api.ULID
		CreditGrantID api.ULID
	}
	UpdateCreditGrantExternalSettlementHandler = httptransport.HandlerWithArgs[UpdateCreditGrantExternalSettlementRequest, UpdateCreditGrantExternalSettlementResponse, UpdateCreditGrantExternalSettlementParams]
)

func (h *handler) UpdateCreditGrantExternalSettlement() UpdateCreditGrantExternalSettlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args UpdateCreditGrantExternalSettlementParams) (UpdateCreditGrantExternalSettlementRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateCreditGrantExternalSettlementRequest{}, err
			}

			var body api.UpdateCreditGrantExternalSettlementRequest
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateCreditGrantExternalSettlementRequest{}, err
			}

			req, err := convertAPIUpdateCreditGrantExternalSettlementRequest(ns, args.CustomerID, args.CreditGrantID, body)
			if err != nil {
				return UpdateCreditGrantExternalSettlementRequest{}, err
			}

			return req, nil
		},
		func(ctx context.Context, request UpdateCreditGrantExternalSettlementRequest) (UpdateCreditGrantExternalSettlementResponse, error) {
			charge, err := h.creditGrantService.UpdateExternalSettlement(ctx, request)
			if models.IsGenericNotFoundError(err) {
				return UpdateCreditGrantExternalSettlementResponse{}, apierrors.NewNotFoundError(ctx, err, "credit grant")
			}
			if err != nil {
				return UpdateCreditGrantExternalSettlementResponse{}, err
			}

			return convertCreditGrant(charge)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateCreditGrantExternalSettlementResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-credit-grant-external-settlement"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
