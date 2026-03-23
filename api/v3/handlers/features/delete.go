package features

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	DeleteFeatureRequest  = models.NamespacedID
	DeleteFeatureResponse = interface{}
	DeleteFeatureParams   = string
	DeleteFeatureHandler  httptransport.HandlerWithArgs[DeleteFeatureRequest, DeleteFeatureResponse, DeleteFeatureParams]
)

func (h *handler) DeleteFeature() DeleteFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID DeleteFeatureParams) (DeleteFeatureRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteFeatureRequest{}, err
			}

			return DeleteFeatureRequest{
				Namespace: ns,
				ID:        featureID,
			}, nil
		},
		func(ctx context.Context, req DeleteFeatureRequest) (DeleteFeatureResponse, error) {
			err := h.connector.ArchiveFeature(ctx, req)
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteFeatureResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-feature"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
