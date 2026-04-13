package addons

import (
	"context"
	"net/http"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetAddonRequest       = addon.GetAddonInput
	GetAddonRequestParams = string
	GetAddonResponse      = apiv3.Addon
	GetAddonHandler       httptransport.HandlerWithArgs[GetAddonRequest, GetAddonResponse, GetAddonRequestParams]
)

func (h *handler) GetAddon() GetAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID GetAddonRequestParams) (GetAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetAddonRequest{}, err
			}

			return GetAddonRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        addonID,
				},
			}, nil
		},
		func(ctx context.Context, request GetAddonRequest) (GetAddonResponse, error) {
			a, err := h.service.GetAddon(ctx, request)
			if err != nil {
				return GetAddonResponse{}, err
			}

			return ConvertFromAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
