package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

var (
	ExampleFeatureKey       = "test-feature-1"
	ExampleFeatureKey2      = "test-feature-2"
	ExampleFeatureKey3      = "test-feature-3"
	ExampleFeatureMeterSlug = "meter1"
)

func ExampleFeature(meterID string) feature.CreateFeatureInputs {
	return feature.CreateFeatureInputs{
		Name:      "Example Feature",
		Key:       ExampleFeatureKey,
		Namespace: ExampleNamespace,
		MeterID:   lo.ToPtr(meterID),
	}
}

func ExampleFeature2(meterID string) feature.CreateFeatureInputs {
	return feature.CreateFeatureInputs{
		Name:      "Example Feature 2",
		Key:       ExampleFeatureKey2,
		Namespace: ExampleNamespace,
		MeterID:   lo.ToPtr(meterID),
	}
}

func ExampleFeature3(meterID string) feature.CreateFeatureInputs {
	return feature.CreateFeatureInputs{
		Name:      "Example Feature 3",
		Key:       ExampleFeatureKey3,
		Namespace: ExampleNamespace,
		MeterID:   lo.ToPtr(meterID),
	}
}

type testFeatureConnector struct {
	feature.FeatureConnector
}

func NewTestFeatureConnector(conn feature.FeatureConnector) *testFeatureConnector {
	return &testFeatureConnector{conn}
}

func (c *testFeatureConnector) CreateExampleFeatures(t *testing.T, meterID string) []feature.Feature {
	t.Helper()
	feat1, err := c.FeatureConnector.CreateFeature(context.Background(), ExampleFeature(meterID))
	if err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}
	feat2, err := c.FeatureConnector.CreateFeature(context.Background(), ExampleFeature2(meterID))
	if err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}
	feat3, err := c.FeatureConnector.CreateFeature(context.Background(), ExampleFeature3(meterID))
	if err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}
	return []feature.Feature{feat1, feat2, feat3}
}
