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
	CreateAddonRequest  = addon.CreateAddonInput
	CreateAddonResponse = apiv3.Addon
	CreateAddonHandler  httptransport.Handler[CreateAddonRequest, CreateAddonResponse]
)

func (h *handler) CreateAddon() CreateAddonHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateAddonRequest, error) {
			body := apiv3.CreateAddonRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateAddonRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateAddonRequest{}, err
			}

			req, err := FromAPICreateAddonRequest(ns, body)
			if err != nil {
				return CreateAddonRequest{}, err
			}

			req.IgnoreNonCriticalIssues = true

			return req, nil
		},
		func(ctx context.Context, request CreateAddonRequest) (CreateAddonResponse, error) {
			a, err := h.service.CreateAddon(ctx, request)
			if err != nil {
				return CreateAddonResponse{}, err
			}

			if a == nil {
				return CreateAddonResponse{}, fmt.Errorf("failed to create add-on")
			}

			return ToAPIAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateAddonResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
