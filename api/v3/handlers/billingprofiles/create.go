package billingprofiles

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateBillingProfileRequest  = billing.CreateProfileInput
	CreateBillingProfileResponse = api.BillingProfile
	CreateBillingProfileHandler  httptransport.Handler[CreateBillingProfileRequest, CreateBillingProfileResponse]
)

func (h *handler) CreateBillingProfile() CreateBillingProfileHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateBillingProfileRequest, error) {
			body := api.CreateBillingProfileRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateBillingProfileRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateBillingProfileRequest{}, err
			}

			return ConvertCreateBillingProfileRequestToCreateProfileInput(ns, body)
		},
		func(ctx context.Context, request CreateBillingProfileRequest) (CreateBillingProfileResponse, error) {
			profile, err := h.service.CreateProfile(ctx, request)
			if err != nil {
				return CreateBillingProfileResponse{}, err
			}

			if profile == nil {
				return CreateBillingProfileResponse{}, fmt.Errorf("failed to create billing profile")
			}

			return ConvertProfileToBillingProfile(*profile)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateBillingProfileResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-billing-profile"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
