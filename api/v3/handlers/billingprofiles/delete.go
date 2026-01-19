package billingprofiles

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	DeleteBillingProfileRequest struct {
		Namespace string
		ProfileID string
	}
	DeleteBillingProfileResponse = interface{}
	DeleteBillingProfileParams   = string
	DeleteBillingProfileHandler  httptransport.HandlerWithArgs[DeleteBillingProfileRequest, DeleteBillingProfileResponse, DeleteBillingProfileParams]
)

func (h *handler) DeleteBillingProfile() DeleteBillingProfileHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, profileID DeleteBillingProfileParams) (DeleteBillingProfileRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteBillingProfileRequest{}, err
			}

			return DeleteBillingProfileRequest{
				Namespace: ns,
				ProfileID: profileID,
			}, nil
		},
		func(ctx context.Context, request DeleteBillingProfileRequest) (DeleteBillingProfileResponse, error) {
			err := h.service.DeleteProfile(ctx, billing.DeleteProfileInput(models.NamespacedID{
				Namespace: request.Namespace,
				ID:        request.ProfileID,
			}))
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[DeleteBillingProfileResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-billing-profile"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
