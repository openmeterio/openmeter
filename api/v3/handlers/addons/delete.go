package addons

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	DeleteAddonRequest  = addon.DeleteAddonInput
	DeleteAddonResponse = any
	DeleteAddonHandler  httptransport.HandlerWithArgs[DeleteAddonRequest, DeleteAddonResponse, string]
)

func (h *handler) DeleteAddon() DeleteAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID string) (DeleteAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteAddonRequest{}, err
			}

			return DeleteAddonRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        addonID,
				},
			}, nil
		},
		func(ctx context.Context, request DeleteAddonRequest) (DeleteAddonResponse, error) {
			err := h.service.DeleteAddon(ctx, request)
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteAddonResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
