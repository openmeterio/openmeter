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
	UpsertOrganizationDefaultTaxCodesRequest = taxcode.UpsertOrganizationDefaultTaxCodesInput
	UpsertOrganizationDefaultTaxCodesResponse = api.OrganizationDefaultTaxCodes
	UpsertOrganizationDefaultTaxCodesHandler  = httptransport.Handler[UpsertOrganizationDefaultTaxCodesRequest, UpsertOrganizationDefaultTaxCodesResponse]
)

func (h *handler) UpsertOrganizationDefaultTaxCodes() UpsertOrganizationDefaultTaxCodesHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (UpsertOrganizationDefaultTaxCodesRequest, error) {
			body := api.UpdateOrganizationDefaultTaxCodesJSONRequestBody{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpsertOrganizationDefaultTaxCodesRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpsertOrganizationDefaultTaxCodesRequest{}, err
			}

			return FromAPIUpdateOrganizationDefaultTaxCodesRequest(ns, body)
		},
		func(ctx context.Context, request UpsertOrganizationDefaultTaxCodesRequest) (UpsertOrganizationDefaultTaxCodesResponse, error) {
			cfg, err := h.service.UpsertOrganizationDefaultTaxCodes(ctx, request)
			if err != nil {
				return UpsertOrganizationDefaultTaxCodesResponse{}, err
			}
			return ToAPIOrganizationDefaultTaxCodes(cfg)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpsertOrganizationDefaultTaxCodesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("upsert-organization-default-tax-codes"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
