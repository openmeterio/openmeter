package llmcost

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	DeleteOverrideRequest  = llmcost.DeleteOverrideInput
	DeleteOverrideResponse = interface{}
	DeleteOverrideHandler  = httptransport.HandlerWithArgs[DeleteOverrideRequest, DeleteOverrideResponse, apiv3models.ULID]
)

func (h *handler) DeleteOverride() DeleteOverrideHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, overrideID apiv3models.ULID) (DeleteOverrideRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteOverrideRequest{}, err
			}

			return DeleteOverrideRequest{
				ID:        overrideID,
				Namespace: ns,
			}, nil
		},
		func(ctx context.Context, req DeleteOverrideRequest) (DeleteOverrideResponse, error) {
			err := h.service.DeleteOverride(ctx, req)
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteOverrideResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-llm-cost-override"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
