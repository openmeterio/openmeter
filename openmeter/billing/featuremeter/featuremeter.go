package featuremeter

import (
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/ref"
)

type FeatureMeter struct {
	Feature feature.Feature
	Meter   *meter.Meter
}

type FeatureReferenceKind string

const (
	FeatureReferenceKindLines   FeatureReferenceKind = "lines"
	FeatureReferenceKindCharges FeatureReferenceKind = "charges"
)

type FeatureReferenceIdentity struct {
	Kind FeatureReferenceKind
	ID   string
}

// FeatureReferenceGetter provides the feature and meter dependency of a billing entity.
// A nil reference means that the entity has no feature dependency.
type FeatureReferenceGetter interface {
	GetFeatureMeterRef() *FeatureMeterRef
}

// FeatureReferenceOwner provides the stable identity of the billing entity that
// owns a feature reference. Reference types without a stable identity must not
// implement this interface.
type FeatureReferenceOwner interface {
	GetFeatureMeterOwner() FeatureReferenceIdentity
}

type FeatureMeters interface {
	Get(reference FeatureReferenceGetter) (FeatureMeter, error)
	Has(reference FeatureReferenceGetter) bool
}

type FeatureMeterRef struct {
	IDOrKey      ref.IDOrKey
	RequireMeter bool
}
