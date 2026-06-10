package charges

import (
	"context"
	"errors"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateCustomerChargesRequest  = billingcharges.CreateInput
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

			input := billingcharges.CreateInput{
				Namespace: ns,
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

			switch discriminator {
			case string(api.BillingFlatFeeChargeTypeFlatFee):
				flatFee, err := body.AsCreateFlatFeeChargeRequest()
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: "unable to parse charge type",
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}

				intent, err := convertFlatFeeChargeAPIToIntent(param.CustomerID, flatFee)
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}

				input.Intents = append(input.Intents, intent)
			case string(api.BillingUsageBasedChargeTypeUsageBased):
				usageBasedFee, err := body.AsCreateUsageBasedChargeRequest()
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: "unable to parse charge type",
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}

				intent, err := convertUsageBaseChargeAPIToIntent(param.CustomerID, usageBasedFee)
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}

				input.Intents = append(input.Intents, intent)
			default:
				return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "params",
						Reason: "invalid charge params: unknown charge type discriminator",
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
			res, err := h.service.Create(ctx, request)
			if err != nil {
				return CreateCustomerChargesResponse{}, err
			}
			if len(res) < 1 {
				return CreateCustomerChargesResponse{}, errors.New("no charge created")
			}
			if len(res) > 1 {
				return CreateCustomerChargesResponse{}, errors.New("too many results")
			}
			return convertChargeToAPI(res[0])
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerChargesResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-customer-charges"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
