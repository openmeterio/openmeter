package taxcodes

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	models "github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetTaxCodeRequest  = taxcode.GetTaxCodeInput
	GetTaxCodeResponse = api.BillingTaxCode
	GetTaxCodeParams   = string
	GetTaxCodeHandler  httptransport.HandlerWithArgs[GetTaxCodeRequest, GetTaxCodeResponse, GetTaxCodeParams]
)

func (h *handler) GetTaxCode() GetTaxCodeHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, taxCodeID GetTaxCodeParams) (GetTaxCodeRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetTaxCodeRequest{}, err
			}

			return GetTaxCodeRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        taxCodeID,
				},
			}, nil
		},
		func(ctx context.Context, request GetTaxCodeRequest) (GetTaxCodeResponse, error) {
			// Call the service to get the tax code
			taxCode, err := h.service.GetTaxCode(ctx, request)
			if err != nil {
				return GetTaxCodeResponse{}, err
			}
			// Convert to API response type
			return ConvertTaxCodeToAPITaxCode(taxCode)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetTaxCodeResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-tax-code"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
