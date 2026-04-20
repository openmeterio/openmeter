package addons

import (
	"context"
	"fmt"
	"net/http"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	ArchiveAddonRequest  = addon.ArchiveAddonInput
	ArchiveAddonResponse = apiv3.Addon
	ArchiveAddonParams   = string
	ArchiveAddonHandler  httptransport.HandlerWithArgs[ArchiveAddonRequest, ArchiveAddonResponse, string]
)

func (h *handler) ArchiveAddon() ArchiveAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID ArchiveAddonParams) (ArchiveAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ArchiveAddonRequest{}, err
			}

			return ArchiveAddonRequest{
				NamespacedID: models.NamespacedID{Namespace: ns, ID: addonID},
				EffectiveTo:  clock.Now(),
			}, nil
		},
		func(ctx context.Context, request ArchiveAddonRequest) (ArchiveAddonResponse, error) {
			a, err := h.service.ArchiveAddon(ctx, request)
			if err != nil {
				return ArchiveAddonResponse{}, err
			}

			if a == nil {
				return ArchiveAddonResponse{}, fmt.Errorf("failed to archive add-on")
			}

			return ToAPIAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[ArchiveAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("archive-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
