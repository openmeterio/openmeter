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

var ExampleFeature = feature.CreateFeatureInputs{
	Name:      "Example Feature",
	Key:       ExampleFeatureKey,
	Namespace: ExampleNamespace,
	MeterSlug: lo.ToPtr(ExampleFeatureMeterSlug),
}

var ExampleFeature2 = feature.CreateFeatureInputs{
	Name:      "Example Feature 2",
	Key:       ExampleFeatureKey2,
	Namespace: ExampleNamespace,
	MeterSlug: lo.ToPtr(ExampleFeatureMeterSlug),
}

var ExampleFeature3 = feature.CreateFeatureInputs{
	Name:      "Example Feature 3",
	Key:       ExampleFeatureKey3,
	Namespace: ExampleNamespace,
	MeterSlug: lo.ToPtr(ExampleFeatureMeterSlug),
}

type testFeatureConnector struct {
	feature.FeatureConnector
}

func NewTestFeatureConnector(conn feature.FeatureConnector) *testFeatureConnector {
	return &testFeatureConnector{conn}
}

func (c *testFeatureConnector) CreateExampleFeatures(t *testing.T) []feature.Feature {
	t.Helper()
	feat1, err := c.FeatureConnector.CreateFeature(context.Background(), ExampleFeature)
	if err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}
	feat2, err := c.FeatureConnector.CreateFeature(context.Background(), ExampleFeature2)
	if err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}
	feat3, err := c.FeatureConnector.CreateFeature(context.Background(), ExampleFeature3)
	if err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}
	return []feature.Feature{feat1, feat2, feat3}
}
