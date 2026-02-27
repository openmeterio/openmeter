package taxcodes

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	UpdateTaxCodeRequest  = taxcode.UpdateTaxCodeInput
	UpdateTaxCodeResponse = api.BillingTaxCode
	UpdateTaxCodeParams   = string
	UpdateTaxCodeHandler  = httptransport.HandlerWithArgs[UpdateTaxCodeRequest, UpdateTaxCodeResponse, UpdateTaxCodeParams]
)

// UpdateTaxCode returns a handler for updating a tax code.
func (h *handler) UpdateTaxCode() UpdateTaxCodeHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, taxCodeID UpdateTaxCodeParams) (UpdateTaxCodeRequest, error) {
			body := api.UpsertTaxCodeRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateTaxCodeRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateTaxCodeRequest{}, err
			}

			req, err := ConvertFromUpsertTaxCodeRequestToUpdateTaxCodeInput(models.NamespacedID{
				Namespace: ns,
				ID:        taxCodeID,
			}, body)
			if err != nil {
				return UpdateTaxCodeRequest{}, err
			}

			return req, nil
		},
		func(ctx context.Context, request UpdateTaxCodeRequest) (UpdateTaxCodeResponse, error) {
			t, err := h.service.UpdateTaxCode(ctx, request)
			if err != nil {
				return UpdateTaxCodeResponse{}, err
			}

			return ConvertTaxCodeToAPITaxCode(t)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateTaxCodeResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("upsert-tax-code"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
