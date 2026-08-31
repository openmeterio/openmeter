package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	EditSubscriptionRequest struct {
		ID             models.NamespacedID
		Customizations []subscription.Patch
		Timing         subscription.Timing
	}
	EditSubscriptionResponse = api.BillingSubscription
	EditSubscriptionParams   = string
	EditSubscriptionHandler  = httptransport.HandlerWithArgs[EditSubscriptionRequest, EditSubscriptionResponse, EditSubscriptionParams]
)

func (h *handler) EditSubscription() EditSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subscriptionID EditSubscriptionParams) (EditSubscriptionRequest, error) {
			body := api.BillingSubscriptionEdit{}
			if err := request.ParseBody(r, &body); err != nil {
				return EditSubscriptionRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return EditSubscriptionRequest{}, err
			}

			if len(body.Customizations) == 0 {
				reason := "at least one customization is required"
				return EditSubscriptionRequest{}, apierrors.NewBadRequestError(
					ctx,
					errors.New(reason),
					[]apierrors.InvalidParameter{
						{
							Field:  "customizations",
							Reason: reason,
							Source: apierrors.InvalidParamSourceBody,
							Rule:   "required",
						},
					},
				)
			}

			patches := make([]subscription.Patch, 0, len(body.Customizations))
			for idx, op := range body.Customizations {
				p, err := FromAPIBillingSubscriptionEditOperation(op)
				if err != nil {
					return EditSubscriptionRequest{}, fmt.Errorf("failed to map customization at index %d: %w", idx, err)
				}

				patches = append(patches, p)
			}

			// Timing defaults to immediate when omitted, matching cancel/change.
			timing := subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}
			if body.Timing != nil {
				timing, err = FromAPIBillingSubscriptionEditTiming(*body.Timing)
				if err != nil {
					return EditSubscriptionRequest{}, err
				}
			}

			return EditSubscriptionRequest{
				ID:             models.NamespacedID{Namespace: ns, ID: subscriptionID},
				Customizations: patches,
				Timing:         timing,
			}, nil
		},
		func(ctx context.Context, req EditSubscriptionRequest) (EditSubscriptionResponse, error) {
			view, err := h.subscriptionWorkflowService.EditRunning(ctx, req.ID, req.Customizations, req.Timing)
			if err != nil {
				return EditSubscriptionResponse{}, err
			}

			return ToAPIBillingSubscription(view)
		},
		commonhttp.JSONResponseEncoderWithStatus[EditSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("edit-subscription"),
			httptransport.WithErrorEncoder(editSubscriptionErrorEncoder()),
		)...,
	)
}

// editSubscriptionErrorEncoder maps the domain patch-application errors that
// EditRunning surfaces to their HTTP statuses, then defers to the generic v3
// encoder. These error types are not part of the models error framework, so
// without this the generic encoder falls through to a 500 for otherwise ordinary
// client mistakes (e.g. "cannot add phase in the past", "phase already exists").
// This mirrors the v1 subscription error encoder.
func editSubscriptionErrorEncoder() encoder.ErrorEncoder {
	generic := apierrors.GenericErrorEncoder()

	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[*subscription.PatchConflictError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PatchForbiddenError](ctx, http.StatusForbidden, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PatchValidationError](ctx, http.StatusBadRequest, err, w) ||
			generic(ctx, err, w, r)
	}
}
