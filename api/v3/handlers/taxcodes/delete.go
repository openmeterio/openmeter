package taxcodes

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	taxcode "github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	DeleteTaxCodeRequest  taxcode.DeleteTaxCodeInput
	DeleteTaxCodeResponse = interface{}
	DeleteTaxCodeParams   = string
	DeleteTaxCodeHandler  = httptransport.HandlerWithArgs[DeleteTaxCodeRequest, DeleteTaxCodeResponse, DeleteTaxCodeParams]
)

// DeleteTaxCode returns a handler for deleting a tax code.
func (h *handler) DeleteTaxCode() DeleteTaxCodeHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, taxCodeID DeleteTaxCodeParams) (DeleteTaxCodeRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteTaxCodeRequest{}, err
			}

			return DeleteTaxCodeRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        taxCodeID,
				},
			}, nil
		},
		func(ctx context.Context, request DeleteTaxCodeRequest) (DeleteTaxCodeResponse, error) {
			err := h.service.DeleteTaxCode(ctx, taxcode.DeleteTaxCodeInput{
				NamespacedID: models.NamespacedID{
					Namespace: request.Namespace,
					ID:        request.ID,
				},
			})
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[DeleteTaxCodeResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-tax-code"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
