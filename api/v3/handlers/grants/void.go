package grants

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	VoidGrantRequest  = entitlement.VoidGrantInput
	VoidGrantResponse = any
	VoidGrantParams   struct {
		GrantID api.ULID
		Params  api.VoidGrantParams
	}
	VoidGrantHandler = httptransport.HandlerWithArgs[VoidGrantRequest, VoidGrantResponse, VoidGrantParams]
)

func (h *handler) VoidGrant() VoidGrantHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params VoidGrantParams) (VoidGrantRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return VoidGrantRequest{}, err
			}

			return VoidGrantRequest{
				GrantID: models.NamespacedID{
					Namespace: ns,
					ID:        params.GrantID,
				},
				At: params.Params.VoidedAt,
			}, nil
		},
		func(ctx context.Context, req VoidGrantRequest) (VoidGrantResponse, error) {
			return nil, h.service.VoidGrant(ctx, req)
		},
		commonhttp.EmptyResponseEncoder[VoidGrantResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("void-grant"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
