package featuremeter

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/ref"
)

type FeatureMeter struct {
	Feature feature.Feature
	Meter   *meter.Meter
}

type FeatureMeters interface {
	GetByKey(featureKey string, requireMeter bool) (FeatureMeter, error)
	GetByID(featureID string, requireMeter bool) (FeatureMeter, error)
	HasFeatureKey(featureKey string) bool
	HasFeatureID(featureID string) bool
	Resolve(ref FeatureMeterRef) (FeatureMeter, error)
}

type FeatureMeterCollection struct {
	ByKey map[string]FeatureMeter
	ByID  map[string]FeatureMeter
}

func (f FeatureMeterCollection) GetByKey(featureKey string, requireMeter bool) (FeatureMeter, error) {
	featureMeter, exists := f.ByKey[featureKey]
	if !exists {
		return FeatureMeter{}, models.NewGenericNotFoundError(fmt.Errorf("feature[%s] not found", featureKey))
	}

	if requireMeter && featureMeter.Meter == nil {
		return FeatureMeter{}, models.NewGenericValidationError(fmt.Errorf("feature[%s] has no meter associated", featureMeter.Feature.Key))
	}

	return featureMeter, nil
}

func (f FeatureMeterCollection) HasFeatureKey(featureKey string) bool {
	_, exists := f.ByKey[featureKey]

	return exists
}

func (f FeatureMeterCollection) HasFeatureID(featureID string) bool {
	_, exists := f.ByID[featureID]

	return exists
}

func (f FeatureMeterCollection) GetByID(featureID string, requireMeter bool) (FeatureMeter, error) {
	featureMeter, exists := f.ByID[featureID]
	if !exists {
		return FeatureMeter{}, models.NewGenericNotFoundError(fmt.Errorf("feature[%s] not found", featureID))
	}

	if requireMeter && featureMeter.Meter == nil {
		return FeatureMeter{}, models.NewGenericValidationError(fmt.Errorf("feature[%s] has no meter associated", featureMeter.Feature.Key))
	}

	return featureMeter, nil
}

type FeatureMeterRef struct {
	IDOrKey      ref.IDOrKey
	RequireMeter bool
}

func (f FeatureMeterCollection) Resolve(r FeatureMeterRef) (FeatureMeter, error) {
	var featureMeter FeatureMeter
	var err error

	switch {
	case r.IDOrKey.Key != "" && r.IDOrKey.ID != "":
		return FeatureMeter{}, fmt.Errorf("feature reference must have either key or ID, not both")
	case r.IDOrKey.Key != "":
		featureMeter, err = f.GetByKey(r.IDOrKey.Key, r.RequireMeter)
	case r.IDOrKey.ID != "":
		featureMeter, err = f.GetByID(r.IDOrKey.ID, r.RequireMeter)
	default:
		return FeatureMeter{}, fmt.Errorf("feature reference must have either key or ID")
	}

	return featureMeter, err
}
