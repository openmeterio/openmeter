package charges

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateCustomerChargesRequest  = billingcharges.CreateInput
	CreateCustomerChargesResponse = api.BillingCharge
	CreateCustomerChargesParams   struct {
		CustomerID api.ULID
		Params     api.BillingCharge
	}
	CreateCustomerChargesHandler = httptransport.HandlerWithArgs[CreateCustomerChargesRequest, CreateCustomerChargesResponse, CreateCustomerChargesParams]
)

func (h *handler) CreateCustomerCharge() CreateCustomerChargesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args CreateCustomerChargesParams) (CreateCustomerChargesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCustomerChargesRequest{}, err
			}

			input := billingcharges.CreateInput{
				Namespace: ns,
			}

			// TODO mapping from params

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
		func(ctx context.Context, output CreateCustomerChargesRequest) (CreateCustomerChargesResponse, error) {
			return to, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerChargesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-customer-charges"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
