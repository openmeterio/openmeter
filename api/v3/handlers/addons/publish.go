package addons

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	PublishAddonRequest  = addon.PublishAddonInput
	PublishAddonResponse = apiv3.Addon
	PublishAddonParams   = string
	PublishAddonHandler  httptransport.HandlerWithArgs[PublishAddonRequest, PublishAddonResponse, string]
)

func (h *handler) PublishAddon() PublishAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID PublishAddonParams) (PublishAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return PublishAddonRequest{}, err
			}

			return PublishAddonRequest{
				NamespacedID: models.NamespacedID{Namespace: ns, ID: addonID},
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(clock.Now()),
				},
			}, nil
		},
		func(ctx context.Context, request PublishAddonRequest) (PublishAddonResponse, error) {
			a, err := h.service.PublishAddon(ctx, request)
			if err != nil {
				return PublishAddonResponse{}, err
			}

			if a == nil {
				return PublishAddonResponse{}, fmt.Errorf("failed to publish add-on")
			}

			return ToAPIAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[PublishAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("publish-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
