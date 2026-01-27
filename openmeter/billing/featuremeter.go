package billing

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

type FeatureMeter struct {
	Feature feature.Feature
	Meter   *meter.Meter
}

type FeatureMeters map[string]FeatureMeter

func (f FeatureMeters) Get(featureKey string, dependsOnMeteredQuantity bool) (FeatureMeter, error) {
	featureMeter, exists := f[featureKey]
	if !exists {
		return FeatureMeter{}, &ErrSnapshotInvalidDatabaseState{
			Err: fmt.Errorf("feature[%s] not found", featureKey),
		}
	}

	if dependsOnMeteredQuantity && featureMeter.Meter == nil {
		return FeatureMeter{}, &ErrSnapshotInvalidDatabaseState{
			Err: fmt.Errorf("feature[%s] has no meter associated, but the line depends on metered quantity", featureMeter.Feature.Key),
		}
	}

	return featureMeter, nil
}
