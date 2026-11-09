package billingprofiles

import (
	"context"
	"fmt"
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
)

type (
	// CreateBillingProfileRequest keeps the default tax config in its raw legacy shape: resolving
	// a deprecated stripe code may create a Tax Code, which has to happen in the same transaction
	// as the create.
	CreateBillingProfileRequest struct {
		Input            billing.CreateProfileInput
		DefaultTaxConfig *productcatalog.TaxConfig
	}
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

			input, err := FromAPICreateBillingProfileRequest(ns, body)
			if err != nil {
				return CreateBillingProfileRequest{}, err
			}

			defaultTaxConfig, err := FromAPIBillingTaxConfigFull(lo.FromPtr(body.Workflow.Tax).DefaultTaxConfig)
			if err != nil {
				return CreateBillingProfileRequest{}, err
			}

			return CreateBillingProfileRequest{
				Input:            input,
				DefaultTaxConfig: defaultTaxConfig,
			}, nil
		},
		func(ctx context.Context, request CreateBillingProfileRequest) (CreateBillingProfileResponse, error) {
			var resp CreateBillingProfileResponse

			err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				input := request.Input

				// Nothing is stored yet, so the gate rejects any tax code and nothing gets
				// resolved or created; only behavior can get through.
				defaultTaxConfig, err := billinghttpdriver.ResolveProfileDefaultTaxConfig(ctx, resolver, input.Namespace, request.DefaultTaxConfig, nil)
				if err != nil {
					return err
				}

				input.WorkflowConfig.Invoicing.DefaultTaxConfig = defaultTaxConfig

				profile, err := h.service.CreateProfile(ctx, input)
				if err != nil {
					return err
				}

				if profile == nil {
					return fmt.Errorf("failed to create billing profile")
				}

				taxCodes, err := h.taxCodesForProfiles(ctx, profile.Namespace, *profile)
				if err != nil {
					return err
				}

				resp, err = ToAPIBillingProfile(*profile, taxCodes)

				return err
			})

			return resp, err
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateBillingProfileResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-billing-profile"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
