package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

var ExampleFeatureKey = "test-feature-1"

var ExampleFeatureMeterSlug = "meter1"

type testFeatureConnector struct {
	feature.FeatureConnector
}

func NewTestFeatureConnector(conn feature.FeatureConnector) *testFeatureConnector {
	return &testFeatureConnector{conn}
}

func (c *testFeatureConnector) CreateExampleFeature(t *testing.T) feature.Feature {
	feat, err := c.FeatureConnector.CreateFeature(context.Background(), feature.CreateFeatureInputs{
		Name:      "Example Feature",
		Key:       ExampleFeatureKey,
		Namespace: ExampleNamespace,
		MeterSlug: lo.ToPtr(ExampleFeatureMeterSlug),
	})
	if err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}
	return feat
}
