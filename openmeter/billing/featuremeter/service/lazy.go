package featuremeterservice

import (
	"context"

	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/pkg/syncx"
	"github.com/samber/lo"
)

type featureMeterReferenceSnapshot struct {
	reference *billingfeaturemeter.FeatureMeterRef
}

func (s featureMeterReferenceSnapshot) GetFeatureMeterRef() *billingfeaturemeter.FeatureMeterRef {
	return s.reference
}

type identifiedFeatureMeterReferenceSnapshot struct {
	featureMeterReferenceSnapshot
	identity billingfeaturemeter.FeatureReferenceIdentity
}

func (s identifiedFeatureMeterReferenceSnapshot) GetFeatureMeterOwner() billingfeaturemeter.FeatureReferenceIdentity {
	return s.identity
}

type lazyFeatureMeters struct {
	ctx     context.Context
	resolve func(context.Context) (billingfeaturemeter.FeatureMeters, error)
}

func (l *lazyFeatureMeters) Get(reference billingfeaturemeter.FeatureReferenceGetter) (billingfeaturemeter.FeatureMeter, error) {
	featureMeters, err := l.resolve(l.ctx)
	if err != nil {
		return billingfeaturemeter.FeatureMeter{}, err
	}

	return featureMeters.Get(reference)
}

func (l *lazyFeatureMeters) Has(reference billingfeaturemeter.FeatureReferenceGetter) bool {
	featureMeters, err := l.resolve(l.ctx)
	if err != nil {
		return false
	}

	return featureMeters.Has(reference)
}

// ResolveLazy snapshots the requested references and resolves their feature-meter
// collection when it is first queried.
func (r Resolver) ResolveLazy[T billingfeaturemeter.FeatureReferenceGetter](ctx context.Context, namespace string, targets ...T) billingfeaturemeter.FeatureMeters {
	references := lo.Map(targets, func(target T, _ int) billingfeaturemeter.FeatureReferenceGetter {
		return snapshotFeatureMeterReference(target)
	})

	return &lazyFeatureMeters{
		ctx: ctx,
		resolve: syncx.OnceValues(func(ctx context.Context) (billingfeaturemeter.FeatureMeters, error) {
			return r.Resolve(ctx, namespace, references...)
		}),
	}
}

func snapshotFeatureMeterReference(reference billingfeaturemeter.FeatureReferenceGetter) billingfeaturemeter.FeatureReferenceGetter {
	snapshot := featureMeterReferenceSnapshot{reference: reference.GetFeatureMeterRef().CloneIfPresent()}
	owner, ok := reference.(billingfeaturemeter.FeatureReferenceOwner)
	if !ok {
		return snapshot
	}
	identity := owner.GetFeatureMeterOwner()
	if identity.Kind == "" || identity.ID == "" {
		return snapshot
	}

	return identifiedFeatureMeterReferenceSnapshot{
		featureMeterReferenceSnapshot: snapshot,
		identity:                      identity,
	}
}
