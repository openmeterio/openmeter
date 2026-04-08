package plans

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	PublishPlanRequest  = plan.PublishPlanInput
	PublishPlanResponse = api.BillingPlan
	PublishPlanParams   = string
	PublishPlanHandler  httptransport.HandlerWithArgs[PublishPlanRequest, PublishPlanResponse, PublishPlanParams]
)

func (h *handler) PublishPlan() PublishPlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID PublishPlanParams) (PublishPlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return PublishPlanRequest{}, err
			}

			return PublishPlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(clock.Now()),
				},
			}, nil
		},
		func(ctx context.Context, request PublishPlanRequest) (PublishPlanResponse, error) {
			p, err := h.service.PublishPlan(ctx, request)
			if err != nil {
				return PublishPlanResponse{}, err
			}

			if p == nil {
				return PublishPlanResponse{}, fmt.Errorf("failed to publish plan")
			}

			return FromPlan(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[PublishPlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("publish-plan"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
