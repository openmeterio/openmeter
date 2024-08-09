package balanceworker

import (
	"context"
	"errors"
	"fmt"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/openmeterio/openmeter/internal/entitlement"
	entitlementdriver "github.com/openmeterio/openmeter/internal/entitlement/driver"
	"github.com/openmeterio/openmeter/internal/entitlement/snapshot"
	"github.com/openmeterio/openmeter/internal/event/metadata"
	"github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/internal/registry"
	"github.com/openmeterio/openmeter/internal/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/convert"
)

const (
	// defaultIncludeDeletedDuration is the default duration for which deleted entitlements are included in recalculation.
	// This ensures that the recent deleted snapshot events are also resent.
	defaultIncludeDeletedDuration = 24 * time.Hour

	defaultLRUCacheSize = 10_000
)

type RecalculatorOptions struct {
	Entitlement       *registry.Entitlement
	Namespace         string
	SubjectIDResolver SubjectIDResolver
	EventBus          eventbus.Publisher
}

type Recalculator struct {
	opts RecalculatorOptions

	featureCache   *lru.Cache[string, productcatalog.Feature]
	subjectIDCache *lru.Cache[string, string]
}

func NewRecalculator(opts RecalculatorOptions) (*Recalculator, error) {
	featureCache, err := lru.New[string, productcatalog.Feature](defaultLRUCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create feature cache: %w", err)
	}

	subjectIDCache, err := lru.New[string, string](defaultLRUCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create subject ID cache: %w", err)
	}

	return &Recalculator{
		opts:           opts,
		featureCache:   featureCache,
		subjectIDCache: subjectIDCache,
	}, nil
}

func (r *Recalculator) Recalculate(ctx context.Context) error {
	affectedEntitlements, err := r.opts.Entitlement.EntitlementRepo.ListEntitlements(
		ctx,
		entitlement.ListEntitlementsParams{
			Namespaces:          []string{r.opts.Namespace},
			IncludeDeleted:      true,
			IncludeDeletedAfter: time.Now().Add(-defaultIncludeDeletedDuration),
		})
	if err != nil {
		return err
	}

	return r.processEntitlements(ctx, affectedEntitlements.Items)
}

func (r *Recalculator) processEntitlements(ctx context.Context, entitlements []entitlement.Entitlement) error {
	var errs error
	for _, ent := range entitlements {
		if ent.DeletedAt != nil {
			err := r.sendEntitlementDeletedEvent(ctx, ent)
			if err != nil {
				errs = errors.Join(errs, err)
			}
		} else {
			err := r.sendEntitlementUpdatedEvent(ctx, ent)
			if err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}

	return errs
}

func (r *Recalculator) sendEntitlementDeletedEvent(ctx context.Context, ent entitlement.Entitlement) error {
	subjectID, err := r.getSubjectIDByKey(ctx, r.opts.Namespace, ent.SubjectKey)
	if err != nil {
		return err
	}

	feature, err := r.getFeature(ctx, ent.Namespace, ent.FeatureID)
	if err != nil {
		return err
	}

	event := marshaler.WithSource(
		metadata.ComposeResourcePath(ent.Namespace, metadata.EntityEntitlement, ent.ID),
		snapshot.SnapshotEvent{
			Entitlement: ent,
			Namespace: models.NamespaceID{
				ID: ent.Namespace,
			},
			Subject: models.SubjectKeyAndID{
				Key: ent.SubjectKey,
				ID:  subjectID,
			},
			Feature:   feature,
			Operation: snapshot.ValueOperationDelete,

			CalculatedAt: convert.ToPointer(time.Now().Add(-defaultClockDrift)),

			CurrentUsagePeriod: ent.CurrentUsagePeriod,
		},
	)

	return r.opts.EventBus.Publish(ctx, event)
}

func (r *Recalculator) sendEntitlementUpdatedEvent(ctx context.Context, ent entitlement.Entitlement) error {
	subjectID, err := r.getSubjectIDByKey(ctx, r.opts.Namespace, ent.SubjectKey)
	if err != nil {
		return err
	}

	feature, err := r.getFeature(ctx, ent.Namespace, ent.FeatureID)
	if err != nil {
		return err
	}

	calculatedAt := time.Now()

	value, err := r.opts.Entitlement.Entitlement.GetEntitlementValue(ctx, ent.Namespace, ent.SubjectKey, ent.ID, calculatedAt)
	if err != nil {
		return fmt.Errorf("failed to get entitlement value: %w", err)
	}

	mappedValues, err := entitlementdriver.MapEntitlementValueToAPI(value)
	if err != nil {
		return fmt.Errorf("failed to map entitlement value: %w", err)
	}

	event := marshaler.WithSource(
		metadata.ComposeResourcePath(ent.Namespace, metadata.EntityEntitlement, ent.ID),
		snapshot.SnapshotEvent{
			Entitlement: ent,
			Namespace: models.NamespaceID{
				ID: ent.Namespace,
			},
			Subject: models.SubjectKeyAndID{
				Key: ent.SubjectKey,
				ID:  subjectID,
			},
			Feature:   feature,
			Operation: snapshot.ValueOperationUpdate,

			CalculatedAt: &calculatedAt,

			Value:              convert.ToPointer((snapshot.EntitlementValue)(mappedValues)),
			CurrentUsagePeriod: ent.CurrentUsagePeriod,
		},
	)

	return r.opts.EventBus.Publish(ctx, event)
}

func (r *Recalculator) getSubjectIDByKey(ctx context.Context, ns, key string) (string, error) {
	if r.opts.SubjectIDResolver == nil {
		return "", nil
	}

	if id, ok := r.subjectIDCache.Get(key); ok {
		return id, nil
	}

	id, err := r.opts.SubjectIDResolver.GetSubjectIDByKey(ctx, ns, key)
	if err != nil {
		return "", err
	}

	r.subjectIDCache.Add(key, id)
	return id, nil
}

func (r *Recalculator) getFeature(ctx context.Context, ns, id string) (productcatalog.Feature, error) {
	if feature, ok := r.featureCache.Get(id); ok {
		return feature, nil
	}

	feature, err := r.opts.Entitlement.Feature.GetFeature(ctx, ns, id, productcatalog.IncludeArchivedFeatureTrue)
	if err != nil {
		return productcatalog.Feature{}, err
	}

	r.featureCache.Add(id, *feature)
	return *feature, nil
}
