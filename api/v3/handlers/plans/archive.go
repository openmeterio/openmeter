package plans

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	ArchivePlanRequest  = plan.ArchivePlanInput
	ArchivePlanResponse = api.BillingPlan
	ArchivePlanParams   = string
	ArchivePlanHandler  httptransport.HandlerWithArgs[ArchivePlanRequest, ArchivePlanResponse, ArchivePlanParams]
)

func (h *handler) ArchivePlan() ArchivePlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID ArchivePlanParams) (ArchivePlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ArchivePlanRequest{}, err
			}

			return ArchivePlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
				EffectiveTo: clock.Now(),
			}, nil
		},
		func(ctx context.Context, request ArchivePlanRequest) (ArchivePlanResponse, error) {
			p, err := h.service.ArchivePlan(ctx, request)
			if err != nil {
				return ArchivePlanResponse{}, err
			}

			if p == nil {
				return ArchivePlanResponse{}, fmt.Errorf("failed to archive plan")
			}

			return ToAPIBillingPlan(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[ArchivePlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("archive-plan"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
