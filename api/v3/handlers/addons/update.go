package addons

import (
	"context"
	"fmt"
	"net/http"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	UpdateAddonRequest  = addon.UpdateAddonInput
	UpdateAddonResponse = apiv3.Addon
	UpdateAddonParams   = string
	UpdateAddonHandler  httptransport.HandlerWithArgs[UpdateAddonRequest, UpdateAddonResponse, string]
)

func (h *handler) UpdateAddon() UpdateAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, addonID UpdateAddonParams) (UpdateAddonRequest, error) {
			body := apiv3.UpsertAddonRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateAddonRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateAddonRequest{}, err
			}

			req, err := FromAPIUpsertAddonRequest(ns, addonID, body)
			if err != nil {
				return UpdateAddonRequest{}, err
			}

			req.IgnoreNonCriticalIssues = true

			return req, nil
		},
		func(ctx context.Context, request UpdateAddonRequest) (UpdateAddonResponse, error) {
			a, err := h.service.UpdateAddon(ctx, request)
			if err != nil {
				return UpdateAddonResponse{}, err
			}

			if a == nil {
				return UpdateAddonResponse{}, fmt.Errorf("failed to update add-on")
			}

			return ToAPIAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
