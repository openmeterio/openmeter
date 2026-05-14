package taxcodes

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetOrganizationDefaultTaxCodesRequest  = taxcode.GetOrganizationDefaultTaxCodesInput
	GetOrganizationDefaultTaxCodesResponse = api.OrganizationDefaultTaxCodes
	GetOrganizationDefaultTaxCodesHandler  = httptransport.Handler[GetOrganizationDefaultTaxCodesRequest, GetOrganizationDefaultTaxCodesResponse]
)

func (h *handler) GetOrganizationDefaultTaxCodes() GetOrganizationDefaultTaxCodesHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (GetOrganizationDefaultTaxCodesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetOrganizationDefaultTaxCodesRequest{}, err
			}
			return GetOrganizationDefaultTaxCodesRequest{Namespace: ns}, nil
		},
		func(ctx context.Context, request GetOrganizationDefaultTaxCodesRequest) (GetOrganizationDefaultTaxCodesResponse, error) {
			cfg, err := h.service.GetOrganizationDefaultTaxCodes(ctx, request)
			if err != nil {
				return GetOrganizationDefaultTaxCodesResponse{}, err
			}
			return ToAPIOrganizationDefaultTaxCodes(cfg)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetOrganizationDefaultTaxCodesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-organization-default-tax-codes"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
