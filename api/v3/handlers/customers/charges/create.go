package charges

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	// CreateCustomerChargesRequest keeps the tax config in its raw legacy shape: resolving
	// a deprecated stripe code may create a Tax Code, which has to happen in the same
	// transaction as the create.
	CreateCustomerChargesRequest struct {
		Input     billingcharges.CreateCustomerChargeInput
		TaxConfig *productcatalog.TaxConfig
	}
	CreateCustomerChargesResponse = apiv3models.BillingCharge
	CreateCustomerChargesParams   struct {
		CustomerID apiv3models.ULID
	}
	CreateCustomerChargesHandler = httptransport.HandlerWithArgs[CreateCustomerChargesRequest, CreateCustomerChargesResponse, CreateCustomerChargesParams]
)

func (h *handler) CreateCustomerCharge() CreateCustomerChargesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, param CreateCustomerChargesParams) (CreateCustomerChargesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCustomerChargesRequest{}, err
			}

			body := apiv3models.CreateCustomerChargesJSONRequestBody{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateCustomerChargesRequest{}, err
			}

			discriminator, err := body.Discriminator()
			if err != nil {
				return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "params",
						Reason: "invalid charge params",
						Source: apierrors.InvalidParamSourceBody,
					},
				})
			}

			var (
				input     billingcharges.CreateCustomerChargeInput
				taxConfig *productcatalog.TaxConfig
			)
			switch discriminator {
			case string(apiv3models.BillingChargeFlatFeeTypeFlatFee):
				flatFee, err := body.AsCreateChargeFlatFeeRequest()
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: "unable to parse charge type",
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}

				input, taxConfig, err = fromAPICreateChargeFlatFeeRequest(ns, param.CustomerID, flatFee)
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}
			case string(apiv3models.BillingChargeUsageBasedTypeUsageBased):
				usageBasedFee, err := body.AsCreateChargeUsageBasedRequest()
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: "unable to parse charge type",
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}

				if usageBasedFee.UnitConfig != nil && !h.unitConfigEnabled {
					err := models.NewGenericValidationError(errors.New("unit_config is not enabled on this deployment of OpenMeter"))

					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "unit_config",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}

				input, taxConfig, err = fromAPICreateChargeUsageBasedRequest(ns, param.CustomerID, usageBasedFee)
				if err != nil {
					return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "params",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceBody,
						},
					})
				}
			default:
				err := fmt.Errorf("invalid charge type: %s", discriminator)

				return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "params",
						Reason: fmt.Errorf("invalid charge params: %w", err).Error(),
						Source: apierrors.InvalidParamSourceBody,
					},
				})
			}

			if err := input.Validate(); err != nil {
				return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "intents",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceBody,
					},
				})
			}

			return CreateCustomerChargesRequest{
				Input:     input,
				TaxConfig: taxConfig,
			}, nil
		},
		func(ctx context.Context, request CreateCustomerChargesRequest) (CreateCustomerChargesResponse, error) {
			var resp CreateCustomerChargesResponse

			// The stripe.code each request carries is resolved to a Tax Code (creating missing
			// ones) in the same transaction the charge is created in, so a failed request
			// leaves no orphan Tax Code behind.
			err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				input := request.Input

				resolved, err := productcataloghttp.FromAPITaxConfig(ctx, resolver, input.Namespace, productcataloghttp.FromTaxConfig(lo.FromPtr(request.TaxConfig)))
				if err != nil {
					var resolveErr *productcataloghttp.TaxCodeResolutionError
					if errors.As(err, &resolveErr) {
						return err
					}

					return billing.ValidationError{Err: err}
				}

				if resolved != nil {
					input.TaxConfig = *resolved
				}

				charge, err := h.service.CreateCustomerCharge(ctx, input)
				if err != nil {
					return err
				}

				// No expands on the create path: every expandable relation
				// serializes as a reference.
				resp, err = convertChargeToAPI(charge, meta.ExpandNone)

				return err
			})
			if err != nil {
				return CreateCustomerChargesResponse{}, err
			}

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerChargesResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-customer-charges"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
