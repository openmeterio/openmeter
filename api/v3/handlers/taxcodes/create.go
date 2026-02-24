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
)

type (
	CreateTaxCodeRequest  = taxcode.CreateTaxCodeInput
	CreateTaxCodeResponse = api.BillingTaxCode
	CreateTaxCodeHandler  httptransport.Handler[CreateTaxCodeRequest, CreateTaxCodeResponse]
)

// CreateTaxCode returns a new httptransport.Handler for creating a tax code.
func (h *handler) CreateTaxCode() CreateTaxCodeHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateTaxCodeRequest, error) {
			body := api.CreateTaxCodeRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateTaxCodeRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateTaxCodeRequest{}, err
			}

			req, err := ConvertFromCreateTaxCodeRequestToCreateTaxCodeInput(ns, body)
			if err != nil {
				return CreateTaxCodeRequest{}, err
			}
			return req, nil
		},
		func(ctx context.Context, request CreateTaxCodeRequest) (CreateTaxCodeResponse, error) {
			t, err := h.service.CreateTaxCode(ctx, request)
			if err != nil {
				return CreateTaxCodeResponse{}, err
			}

			resp, err := ConvertTaxCodeToAPITaxCode(t)
			if err != nil {
				return CreateTaxCodeResponse{}, err
			}

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateTaxCodeResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-tax-code"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
