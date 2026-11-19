package charges

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetChargeRequest  = billingcharges.GetByIDInput
	GetChargeResponse = apiv3models.BillingCharge
	GetChargeParams   struct {
		ChargeID apiv3models.ULID
		Params   apiv3models.GetChargeParams
	}
	GetChargeHandler = httptransport.HandlerWithArgs[GetChargeRequest, GetChargeResponse, GetChargeParams]
)

func (h *handler) GetCharge() GetChargeHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, _ *http.Request, args GetChargeParams) (GetChargeRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetChargeRequest{}, err
			}

			expands, err := fromAPIChargesExpands(ctx, args.Params.Expand)
			if err != nil {
				return GetChargeRequest{}, err
			}

			return GetChargeRequest{
				ChargeID: meta.ChargeID{Namespace: ns, ID: args.ChargeID},
				Expands:  expands,
			}, nil
		},
		func(ctx context.Context, request GetChargeRequest) (GetChargeResponse, error) {
			charge, err := h.service.GetCustomerCharge(ctx, request)
			if err != nil {
				return GetChargeResponse{}, err
			}

			return convertChargeToAPI(charge, request.Expands)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetChargeResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-charge"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
