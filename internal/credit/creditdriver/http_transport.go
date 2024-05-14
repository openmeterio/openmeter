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
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Handler interface {
	GetFeature(ctx context.Context, w http.ResponseWriter, r *http.Request, featureID api.FeatureID)
	ListFeatures(ctx context.Context, w http.ResponseWriter, r *http.Request)
	CreateFeature(ctx context.Context, w http.ResponseWriter, r *http.Request)
	DeleteFeature(ctx context.Context, w http.ResponseWriter, r *http.Request, featureID api.FeatureID)
}

func NewHandler(
	creditConnector credit.Connector,
	meterRepository meter.Repository,
	namespaceDecoder namespacedriver.NamespaceDecoder,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		CreditConnector:  creditConnector,
		MeterRepository:  meterRepository,
		NamespaceDecoder: namespaceDecoder,
		Options:          options,
	}
}

type handler struct {
	CreditConnector  credit.Connector
	MeterRepository  meter.Repository
	NamespaceDecoder namespacedriver.NamespaceDecoder
	Options          []httptransport.HandlerOption
}

var _ Handler = (*handler)(nil)

type featureIDWithNamespace *namespacedriver.Wrapped[api.FeatureID]

func (h *handler) GetFeature(ctx context.Context, w http.ResponseWriter, r *http.Request, featureID api.FeatureID) {
	httptransport.NewHandler[featureIDWithNamespace, credit.Feature](
		func(ctx context.Context, r *http.Request) (featureIDWithNamespace, error) {
			return namespacedriver.Wrap(ctx, featureID, h.NamespaceDecoder)
		},
		func(ctx context.Context, request featureIDWithNamespace) (credit.Feature, error) {
			return h.CreditConnector.GetFeature(ctx, request.Namespace, request.Request)
		},
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
	).ServeHTTP(w, r)

}

type featureWithNamespace *namespacedriver.Wrapped[credit.Feature]

func (h *handler) CreateFeature(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	httptransport.NewHandler[featureWithNamespace, credit.Feature](
		func(ctx context.Context, r *http.Request) (featureWithNamespace, error) {
			featureIn := api.CreateFeatureJSONRequestBody{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &featureIn); err != nil {
				return nil, err
			}

			featureWithNS, err := namespacedriver.Wrap(ctx, featureIn, h.NamespaceDecoder)
			if err != nil {
				return nil, err
			}

			meter, err := h.MeterRepository.GetMeterByIDOrSlug(ctx, featureWithNS.Namespace, featureIn.MeterSlug)
			if err != nil {
				if _, ok := err.(*models.MeterNotFoundError); ok {
					return nil, commonhttp.NewHTTPError(
						http.StatusBadRequest,
						fmt.Errorf("meter not found: %s", featureIn.MeterSlug),
					)
				}

				return nil, err
			}

			if err := validateMeterAggregation(meter); err != nil {
				return nil, commonhttp.NewHTTPError(http.StatusBadRequest, err)
			}
			return featureWithNS, nil
		},
		func(ctx context.Context, in featureWithNamespace) (credit.Feature, error) {
			// Let's make sure we are not allowing the ID to be specified externally
			in.Request.ID = nil

			return h.CreditConnector.CreateFeature(ctx, in.Namespace, in.Request)
		},
		commonhttp.JSONResponseEncoderWithStatus[credit.Feature](http.StatusCreated),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("createFeature"),
		)...,
	).ServeHTTP(w, r)
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

type featureListWithNamespace *namespacedriver.Wrapped[credit.ListFeaturesParams]

func (h *handler) ListFeatures(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	httptransport.NewHandler[featureListWithNamespace, []credit.Feature](
		func(ctx context.Context, r *http.Request) (featureListWithNamespace, error) {
			// TODO: add get arguments (limit, offset, archived)
			return namespacedriver.Wrap(ctx, credit.ListFeaturesParams{}, h.NamespaceDecoder)
		},
		func(ctx context.Context, request featureListWithNamespace) ([]credit.Feature, error) {
			return h.CreditConnector.ListFeatures(ctx, request.Namespace, request.Request)
		},
		commonhttp.JSONResponseEncoder,
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("listFeatures"),
		)...,
	).ServeHTTP(w, r)
}

func (h *handler) DeleteFeature(ctx context.Context, w http.ResponseWriter, r *http.Request, featureID api.FeatureID) {
	httptransport.NewHandler[featureIDWithNamespace, any](
		func(ctx context.Context, r *http.Request) (featureIDWithNamespace, error) {
			featureIDWithNs, err := namespacedriver.Wrap(ctx, featureID, h.NamespaceDecoder)
			if err != nil {
				return nil, err
			}

			if _, err := h.CreditConnector.GetFeature(ctx, featureIDWithNs.Namespace, featureID); err != nil {
				return nil, err
			}
			return featureIDWithNs, nil
		},
		func(ctx context.Context, request featureIDWithNamespace) (any, error) {
			return nil,
				h.CreditConnector.DeleteFeature(ctx, request.Namespace, request.Request)
		},
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
	).ServeHTTP(w, r)
}
