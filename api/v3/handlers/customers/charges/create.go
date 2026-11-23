package charges

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	// CreateCustomerChargesRequest carries the deprecated stripe.code separately: resolving it
	// may create a Tax Code, which has to happen in the same transaction as the create.
	CreateCustomerChargesRequest struct {
		Input         billingcharges.CreateCustomerChargeInput
		StripeTaxCode string
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
				input        billingcharges.CreateCustomerChargeInput
				apiTaxConfig *apiv3models.BillingTaxConfig
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

				apiTaxConfig = flatFee.TaxConfig

				input, err = fromAPICreateChargeFlatFeeRequest(ns, param.CustomerID, flatFee)
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

				apiTaxConfig = usageBasedFee.TaxConfig

				input, err = fromAPICreateChargeUsageBasedRequest(ns, param.CustomerID, usageBasedFee)
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

			taxConfig, stripeTaxCode, err := fromAPIChargeTaxConfig(apiTaxConfig)
			if err != nil {
				return CreateCustomerChargesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "tax_config",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceBody,
					},
				})
			}

			input.TaxConfig = taxConfig

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
				Input:         input,
				StripeTaxCode: stripeTaxCode,
			}, nil
		},
		func(ctx context.Context, request CreateCustomerChargesRequest) (CreateCustomerChargesResponse, error) {
			var resp CreateCustomerChargesResponse

			// Resolving a deprecated stripe.code may create its Tax Code; doing it in the create's
			// transaction means a failed request leaves no new Tax Code behind.
			err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				input := request.Input

				if request.StripeTaxCode != "" {
					tc, err := resolver.ResolveOrBackfillTaxCode(ctx, productcatalog.ResolveOrBackfillTaxCodeInput{
						Namespace:  input.Namespace,
						StripeCode: request.StripeTaxCode,
					})
					if err != nil {
						return fmt.Errorf("resolving stripe tax code %s: %w", request.StripeTaxCode, err)
					}

					input.TaxConfig.TaxCodeID = tc.ID
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
