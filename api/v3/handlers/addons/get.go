package addons

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetAddonRequest       = addon.GetAddonInput
	GetAddonRequestParams = string
	GetAddonResponse      = apiv3models.Addon
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

			if a == nil {
				return GetAddonResponse{}, fmt.Errorf("failed to get add-on: %s", request.ID)
			}

			return ToAPIAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
