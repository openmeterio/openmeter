package charges

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateCustomerChargesRequest  = billingcharges.CreateCustomerChargeInput
	CreateCustomerChargesResponse = api.BillingCharge
	CreateCustomerChargesParams   struct {
		CustomerID api.ULID
	}
	CreateCustomerChargesHandler = httptransport.HandlerWithArgs[CreateCustomerChargesRequest, CreateCustomerChargesResponse, CreateCustomerChargesParams]
)

func (h *handler) CreateCustomerCharge() CreateCustomerChargesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, param CreateCustomerChargesParams) (CreateCustomerChargesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCustomerChargesRequest{}, err
			}

			body := api.CreateCustomerChargesJSONRequestBody{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateCustomerChargesRequest{}, err
			}

			discriminator, err := body.Discriminator()
			if err != nil {
				return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "params",
						Reason: "invalid charge params",
						Source: apierrors.InvalidParamSourceBody,
					},
				})
			}

			var input billingcharges.CreateCustomerChargeInput
			switch discriminator {
			case string(api.BillingChargeFlatFeeTypeFlatFee):
				flatFee, err := body.AsCreateChargeFlatFeeRequest()
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: "unable to parse charge type",
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}

				input, err = fromAPICreateChargeFlatFeeRequest(ns, param.CustomerID, flatFee)
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}
			case string(api.BillingChargeUsageBasedTypeUsageBased):
				usageBasedFee, err := body.AsCreateChargeUsageBasedRequest()
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: "unable to parse charge type",
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}

				input, err = fromAPICreateChargeUsageBasedRequest(ns, param.CustomerID, usageBasedFee)
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}
			default:
				err := fmt.Errorf("invalid charge type: %s", discriminator)
				return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "params",
						Reason: fmt.Errorf("invalid charge params: %w", err).Error(),
						Source: apierrors.InvalidParamSourceBody,
					},
				})
			}

			if err := input.Validate(); err != nil {
				return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "intents",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceBody,
					},
				})
			}

			return input, nil
		},
		func(ctx context.Context, request CreateCustomerChargesRequest) (CreateCustomerChargesResponse, error) {
			charge, err := h.service.CreateCustomerCharge(ctx, request)
			if err != nil {
				return CreateCustomerChargesResponse{}, err
			}

			return convertChargeToAPI(charge)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerChargesResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-customer-charges"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
