package features

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	CreateFeatureRequest  = feature.CreateFeatureInputs
	CreateFeatureResponse = api.Feature
	CreateFeatureHandler  httptransport.Handler[CreateFeatureRequest, CreateFeatureResponse]
)

func (h *handler) CreateFeature() CreateFeatureHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateFeatureRequest, error) {
			body := api.CreateFeatureRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateFeatureRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateFeatureRequest{}, err
			}

			// Resolve and validate the meter reference.
			var meterID *string
			if body.Meter != nil {
				if body.Meter.Id == "" {
					return CreateFeatureRequest{}, models.NewGenericValidationError(
						fmt.Errorf("meter id is required"),
					)
				}

				m, err := h.meterService.GetMeterByIDOrSlug(ctx, meter.GetMeterInput{
					Namespace: ns,
					IDOrSlug:  body.Meter.Id,
				})
				if err != nil {
					return CreateFeatureRequest{}, err
				}
				meterID = &m.ID

				// Validate meter filters.
				if body.Meter.Filters != nil {
					if err := validateMeterFilters(*body.Meter.Filters, m); err != nil {
						return CreateFeatureRequest{}, err
					}
				}
			}

			return convertCreateRequestToDomain(ns, body, meterID)
		},
		func(ctx context.Context, req CreateFeatureRequest) (CreateFeatureResponse, error) {
			created, err := h.connector.CreateFeature(ctx, req)
			if err != nil {
				return CreateFeatureResponse{}, err
			}

			return convertFeatureToAPI(created)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateFeatureResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-feature"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

// validateMeterFilters validates that filter keys exist in the meter's dimensions.
// The single-operator invariant on each filter value is enforced downstream by
// feature.MeterGroupByFilters.Validate.
func validateMeterFilters(filters map[string]api.QueryFilterStringMapItem, m meter.Meter) error {
	for k := range filters {
		if _, ok := m.GroupBy[k]; !ok {
			return models.NewGenericValidationError(
				fmt.Errorf("filter key %q is not a valid dimension of meter %q", k, m.Key),
			)
		}
	}
	return nil
}
