package testutils

import (
	"testing"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

func NewTestFeature(t *testing.T, namespace string) feature.CreateFeatureInputs {
	t.Helper()

	return feature.CreateFeatureInputs{
		Name:      "Test",
		Key:       "test",
		Namespace: namespace,
	}
}

func NewTestFeatureFromMeter(t *testing.T, meter *meter.Meter) feature.CreateFeatureInputs {
	t.Helper()

	return feature.CreateFeatureInputs{
		Name:                meter.Key,
		Key:                 meter.Key,
		Namespace:           meter.Namespace,
		MeterSlug:           lo.ToPtr(meter.Key),
		MeterGroupByFilters: feature.ConvertMapStringToMeterGroupByFilters(meter.GroupBy),
		Metadata:            map[string]string{},
	}
}
