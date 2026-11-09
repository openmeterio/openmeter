package billingprofiles

import (
	"context"
	"errors"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billinghttpdriver "github.com/openmeterio/openmeter/openmeter/billing/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	UpdateBillingProfileRequest struct {
		Namespace string
		ProfileID string
		Profile   billing.UpdateProfileInput
		// DefaultTaxConfig is kept raw for the same reason as in CreateBillingProfileRequest.
		DefaultTaxConfig *productcatalog.TaxConfig
	}
	UpdateBillingProfileParams   = string
	UpdateBillingProfileResponse = api.BillingProfile
	UpdateBillingProfileHandler  httptransport.HandlerWithArgs[UpdateBillingProfileRequest, UpdateBillingProfileResponse, UpdateBillingProfileParams]
)

func (h *handler) UpdateBillingProfile() UpdateBillingProfileHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, profileID UpdateBillingProfileParams) (UpdateBillingProfileRequest, error) {
			body := api.UpsertBillingProfileRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateBillingProfileRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateBillingProfileRequest{}, err
			}

			profile, err := FromAPIUpsertBillingProfileRequest(models.NamespacedID{
				Namespace: ns,
				ID:        profileID,
			}, body)
			if err != nil {
				return UpdateBillingProfileRequest{}, err
			}

			defaultTaxConfig, err := FromAPIBillingTaxConfigFull(lo.FromPtr(body.Workflow.Tax).DefaultTaxConfig)
			if err != nil {
				return UpdateBillingProfileRequest{}, err
			}

			req := UpdateBillingProfileRequest{
				Namespace:        ns,
				ProfileID:        profileID,
				Profile:          profile,
				DefaultTaxConfig: defaultTaxConfig,
			}

			return req, nil
		},
		func(ctx context.Context, request UpdateBillingProfileRequest) (UpdateBillingProfileResponse, error) {
			var resp UpdateBillingProfileResponse

			err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				input := request.Profile

				stored, err := h.service.GetProfile(ctx, billing.GetProfileInput{
					Profile: input.ProfileID(),
				})
				if err != nil {
					return err
				}

				// The gate compares against the stored default the way responses show it.
				var storedDefaultTaxConfig *productcatalog.TaxConfig
				if cfg := stored.WorkflowConfig.Invoicing.DefaultTaxConfig; cfg != nil {
					storedTaxCodes, err := h.taxCodesForProfiles(ctx, request.Namespace, *stored)
					if err != nil {
						return err
					}

					storedDefaultTaxConfig = lo.ToPtr(cfg.ToLegacyTaxConfig(storedTaxCodes))
				}

				defaultTaxConfig, err := billinghttpdriver.ResolveProfileDefaultTaxConfig(ctx, resolver, request.Namespace, request.DefaultTaxConfig, storedDefaultTaxConfig)
				if err != nil {
					return err
				}

				input.WorkflowConfig.Invoicing.DefaultTaxConfig = defaultTaxConfig

				updatedProfile, err := h.service.UpdateProfile(ctx, input)
				if err != nil {
					return err
				}

				if updatedProfile == nil {
					return errors.New("failed to update billing profile")
				}

				taxCodes, err := h.taxCodesForProfiles(ctx, request.Namespace, *updatedProfile)
				if err != nil {
					return err
				}

				resp, err = ToAPIBillingProfile(*updatedProfile, taxCodes)

				return err
			})

			return resp, err
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateBillingProfileResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-billing-profile"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
