package adapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
)

// stubFeatureConnector serves one fixed feature; QueryFeatureCost only calls GetFeature.
type stubFeatureConnector struct {
	feature feature.Feature
}

func (s stubFeatureConnector) CreateFeature(context.Context, feature.CreateFeatureInputs) (feature.Feature, error) {
	return feature.Feature{}, errors.New("not implemented")
}

func (s stubFeatureConnector) UpdateFeature(context.Context, feature.UpdateFeatureInputs) (feature.Feature, error) {
	return feature.Feature{}, errors.New("not implemented")
}

func (s stubFeatureConnector) ArchiveFeature(context.Context, models.NamespacedID) error {
	return errors.New("not implemented")
}

func (s stubFeatureConnector) ListFeatures(context.Context, feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
	return pagination.Result[feature.Feature]{}, errors.New("not implemented")
}

func (s stubFeatureConnector) GetFeature(context.Context, string, string, feature.IncludeArchivedFeature) (*feature.Feature, error) {
	f := s.feature

	return &f, nil
}

func (s stubFeatureConnector) ResolveFeatureMeters(context.Context, string, ...ref.IDOrKey) (feature.FeatureMeters, error) {
	return nil, errors.New("not implemented")
}

// stubMeterService serves one fixed meter; QueryFeatureCost only calls GetMeterByIDOrSlug.
type stubMeterService struct {
	meter meter.Meter
}

func (s stubMeterService) ListMeters(context.Context, meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
	return pagination.Result[meter.Meter]{}, errors.New("not implemented")
}

func (s stubMeterService) GetMeterByIDOrSlug(context.Context, meter.GetMeterInput) (meter.Meter, error) {
	return s.meter, nil
}

// TestQueryFeatureCostOptsIntoMeterCache pins the cost adapter as a designated meter
// cache opt-in call site: the params it hands to the streaming connector must carry
// Cachable=true without mutating the caller's input params, while billing paths construct
// their own params and must never set it.
func TestQueryFeatureCostOptsIntoMeterCache(t *testing.T) {
	m := meter.Meter{
		Key:           "meter-1",
		EventType:     "api-calls",
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
	}

	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
	streamingConnector.AddSimpleEvent(m.Key, 10, time.Now().Add(-time.Hour))

	costAdapter := New(
		stubFeatureConnector{feature: feature.Feature{
			Namespace: "test-ns",
			Key:       "feature-1",
			MeterID:   lo.ToPtr("meter-1"),
			UnitCost: &feature.UnitCost{
				Type:   feature.UnitCostTypeManual,
				Manual: &feature.ManualUnitCost{Amount: alpacadecimal.NewFromFloat(0.5)},
			},
		}},
		stubMeterService{meter: m},
		streamingConnector,
		nil,
	)

	from := time.Now().Add(-2 * time.Hour)
	to := time.Now()

	input := cost.QueryFeatureCostInput{
		Namespace: "test-ns",
		FeatureID: "feature-1",
		QueryParams: streaming.QueryParams{
			From: &from,
			To:   &to,
		},
	}

	_, err := costAdapter.QueryFeatureCost(t.Context(), input)
	require.NoError(t, err)

	captured := streamingConnector.CapturedQueryMeterParams()
	require.Len(t, captured, 1)
	require.True(t, captured[0].Cachable)

	// The opt-in must happen on the adapter's own copy: mutating the caller's params
	// would silently opt callers embedding these params into other queries into the cache.
	require.False(t, input.QueryParams.Cachable)
}
