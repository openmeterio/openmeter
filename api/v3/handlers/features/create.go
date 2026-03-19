package features

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateFeatureRequest  = feature.CreateFeatureInputs
	CreateFeatureResponse = api.Feature
	CreateFeatureHandler  httptransport.Handler[CreateFeatureRequest, CreateFeatureResponse]
)

func (h *handler) CreateFeature() CreateFeatureHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateFeatureRequest, error) {
			body := api.CreateFeatureRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateFeatureRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateFeatureRequest{}, err
			}

			return convertCreateRequestToDomain(ns, body)
		},
		func(ctx context.Context, req CreateFeatureRequest) (CreateFeatureResponse, error) {
			created, err := h.connector.CreateFeature(ctx, req)
			if err != nil {
				return CreateFeatureResponse{}, err
			}

			return convertFeatureToAPI(created)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateFeatureResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-feature"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
