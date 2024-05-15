package creditdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Handlers struct {
	GetFeature    GetFeatureHandler
	ListFeatures  ListFeaturesHandler
	CreateFeature CreateFeatureHandler
	DeleteFeature DeleteFeatureHandler
}

func New(
	creditConnector credit.Connector,
	meterRepository meter.Repository,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) Handlers {
	builder := &builder{
		CreditConnector:  creditConnector,
		MeterRepository:  meterRepository,
		NamespaceDecoder: namespaceDecoder,
		Options:          options,
	}

	return Handlers{
		GetFeature:    builder.GetFeature(),
		ListFeatures:  builder.ListFeatures(),
		CreateFeature: builder.CreateFeature(),
		DeleteFeature: builder.DeleteFeature(),
	}
}

type builder struct {
	CreditConnector  credit.Connector
	MeterRepository  meter.Repository
	NamespaceDecoder namespacedriver.NamespaceDecoder
	Options          []httptransport.HandlerOption
}

type GetFeatureHandler httptransport.HandlerWithArgs[credit.NamespacedID, credit.Feature, api.FeatureID]

func (h *builder) GetFeature() GetFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID api.FeatureID) (credit.NamespacedID, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return credit.NamespacedID{}, err
			}

			return credit.NamespacedID{
				Namespace: ns,
				ID:        featureID,
			}, nil
		},
		h.CreditConnector.GetFeature,
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			h.Options,
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

func (h *builder) CreateFeature() CreateFeatureHandler {
	return httptransport.NewHandler[credit.Feature, credit.Feature](
		func(ctx context.Context, r *http.Request) (credit.Feature, error) {
			featureIn := credit.Feature{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &featureIn); err != nil {
				return featureIn, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return  featureIn, err
			}

			featureIn.Namespace = ns

			meter, err := h.MeterRepository.GetMeterByIDOrSlug(ctx, featureIn.Namespace, featureIn.MeterSlug)
			if err != nil {
				if _, ok := err.(*models.MeterNotFoundError); ok {
					return  featureIn, commonhttp.NewHTTPError(
						http.StatusBadRequest,
						fmt.Errorf("meter not found: %s", featureIn.MeterSlug),
					)
				}

				return  featureIn, err
			}

			if err := validateMeterAggregation(meter); err != nil {
				return  featureIn, commonhttp.NewHTTPError(http.StatusBadRequest, err)
			}
			return featureIn, nil
		},
		h.CreditConnector.CreateFeature,
		commonhttp.JSONResponseEncoderWithStatus[credit.Feature](http.StatusCreated),
		httptransport.AppendOptions(
			h.Options,
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

type ListFeaturesHandler httptransport.Handler[credit.ListFeaturesParams, []credit.Feature]

func (h *builder) ListFeatures() ListFeaturesHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (credit.ListFeaturesParams, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return credit.ListFeaturesParams{}, err 
			}
			return credit.ListFeaturesParams{
				Namespace: ns,
			}, nil
		},
		h.CreditConnector.ListFeatures,
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("listFeatures"),
		)...,
	)
}

type DeleteFeatureHandler httptransport.HandlerWithArgs[credit.NamespacedID, any, api.FeatureID]

func (h *builder) DeleteFeature() DeleteFeatureHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, featureID api.FeatureID) (credit.NamespacedID, error) {
			id := credit.NamespacedID{
				ID: featureID,
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return id, err
			}

			id.Namespace = ns
			
			if _, err := h.CreditConnector.GetFeature(ctx, id); err != nil {
				return id, err
			}
			return id, nil
		},
		operation.AsNoResponseOperation(h.CreditConnector.DeleteFeature),
		func(ctx context.Context, w http.ResponseWriter, response any) error {
			w.WriteHeader(http.StatusNoContent)
			return nil
		},
		httptransport.AppendOptions(
			h.Options,
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
