package creditdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type GetFeatureHandler httptransport.HandlerWithArgs[credit.NamespacedFeatureID, credit.Feature, api.FeatureID]

func (b *builder) GetFeature() GetFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID api.FeatureID) (credit.NamespacedFeatureID, error) {
			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return credit.NamespacedFeatureID{}, err
			}

			return credit.NamespacedFeatureID{
				Namespace: ns,
				ID:        featureID,
			}, nil
		},
		b.CreditConnector.GetFeature,
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*credit.FeatureNotFoundError); ok {
					models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w)
					return true
				}
				return false
			}),
			httptransport.WithOperationName("getFeature"),
		)...,
	)

}

type CreateFeatureHandler httptransport.Handler[credit.Feature, credit.Feature]

func (b *builder) CreateFeature() CreateFeatureHandler {
	return httptransport.NewHandler[credit.Feature, credit.Feature](
		func(ctx context.Context, r *http.Request) (credit.Feature, error) {
			featureIn := credit.Feature{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &featureIn); err != nil {
				return featureIn, err
			}

			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return featureIn, err
			}

			featureIn.Namespace = ns

			meter, err := b.MeterRepository.GetMeterByIDOrSlug(ctx, featureIn.Namespace, featureIn.MeterSlug)
			if err != nil {
				if _, ok := err.(*models.MeterNotFoundError); ok {
					return featureIn, commonhttp.NewHTTPError(
						http.StatusBadRequest,
						fmt.Errorf("meter not found: %s", featureIn.MeterSlug),
					)
				}

				return featureIn, err
			}

			if err := validateMeterAggregation(meter); err != nil {
				return featureIn, commonhttp.NewHTTPError(http.StatusBadRequest, err)
			}
			return featureIn, nil
		},
		b.CreditConnector.CreateFeature,
		commonhttp.JSONResponseEncoderWithStatus[credit.Feature](http.StatusCreated),
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("createFeature"),
		)...,
	)
}

func validateMeterAggregation(meter models.Meter) error {
	switch meter.Aggregation {
	case models.MeterAggregationCount, models.MeterAggregationUniqueCount, models.MeterAggregationSum:
		return nil
	}

	return fmt.Errorf("meter %s's aggregation is %s but features can only be created for SUM, COUNT, UNIQUE_COUNT MeterRepository",
		meter.Slug,
		meter.Aggregation,
	)
}

type ListFeaturesHandler httptransport.HandlerWithArgs[credit.ListFeaturesParams, []credit.Feature, api.ListFeaturesParams]

func (b *builder) ListFeatures() ListFeaturesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, apiParams api.ListFeaturesParams) (credit.ListFeaturesParams, error) {
			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return credit.ListFeaturesParams{}, err
			}
			params := credit.ListFeaturesParams{
				Namespace:       ns,
				IncludeArchived: defaultx.WithDefault(apiParams.IncludeArchived, false),
				Offset:          defaultx.WithDefault(apiParams.Offset, DefaultLedgerQueryLimit),
				Limit:           defaultx.WithDefault(apiParams.Limit, DefaultLedgerQueryLimit),
				OrderBy:         defaultx.WithDefault((*credit.FeatureOrderBy)(apiParams.OrderBy), credit.FeatureOrderByID),
			}

			if params.Limit > MaxLedgerQueryLimit {
				return params, commonhttp.NewHTTPError(
					http.StatusBadRequest,
					fmt.Errorf("limit must be less than or equal to %d", MaxLedgerQueryLimit),
				)
			}

			return params, nil
		},
		b.CreditConnector.ListFeatures,
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("listFeatures"),
		)...,
	)
}

type DeleteFeatureHandler httptransport.HandlerWithArgs[credit.NamespacedFeatureID, any, api.FeatureID]

func (b *builder) DeleteFeature() DeleteFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID api.FeatureID) (credit.NamespacedFeatureID, error) {
			id := credit.NamespacedFeatureID{
				ID: featureID,
			}

			ns, err := b.resolveNamespace(ctx)
			if err != nil {
				return id, err
			}

			id.Namespace = ns

			if _, err := b.CreditConnector.GetFeature(ctx, id); err != nil {
				return id, err
			}
			return id, nil
		},
		operation.AsNoResponseOperation(b.CreditConnector.DeleteFeature),
		func(ctx context.Context, w http.ResponseWriter, response any) error {
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
		httptransport.AppendOptions(
			b.Options,
			httptransport.WithOperationName("deleteFeature"),
			httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) bool {
				if _, ok := err.(*credit.FeatureNotFoundError); ok {
					models.NewStatusProblem(ctx, err, http.StatusNotFound).Respond(w)
					return true
				}
				return false
			}),
		)...,
	)
}
