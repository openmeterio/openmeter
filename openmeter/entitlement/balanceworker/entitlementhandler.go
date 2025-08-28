package balanceworker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker/filters"
	entitlementdriver "github.com/openmeterio/openmeter/openmeter/entitlement/driver"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
	"github.com/openmeterio/openmeter/pkg/convert"
	pkgmodels "github.com/openmeterio/openmeter/pkg/models"
)

type handleEntitlementEventOptions struct {
	// Source is the source of the event, e.g. the "subject" field from the upstream cloudevents event causing the change
	source string

	// EventAt is the time of the event, e.g. the "time" field from the upstream cloudevents event causing the change
	eventAt time.Time

	// SourceOperation is the operation that caused the entitlement change (if empty update is assumed)
	sourceOperation *snapshot.ValueOperationType

	rawIngestedEvents []serializer.CloudEventsKafkaPayload
}

func (o *handleEntitlementEventOptions) Validate() error {
	if o.eventAt.IsZero() {
		return errors.New("eventAt is required")
	}

	return nil
}

type handleOption func(*handleEntitlementEventOptions)

func WithSource(source string) handleOption {
	return func(o *handleEntitlementEventOptions) {
		o.source = source
	}
}

func WithEventAt(eventAt time.Time) handleOption {
	return func(o *handleEntitlementEventOptions) {
		o.eventAt = eventAt
	}
}

func WithSourceOperation(sourceOperation snapshot.ValueOperationType) handleOption {
	return func(o *handleEntitlementEventOptions) {
		o.sourceOperation = &sourceOperation
	}
}

func WithRawIngestedEvents(rawIngestedEvents []serializer.CloudEventsKafkaPayload) handleOption {
	return func(o *handleEntitlementEventOptions) {
		o.rawIngestedEvents = rawIngestedEvents
	}
}

func getOptions(opts ...handleOption) handleEntitlementEventOptions {
	options := handleEntitlementEventOptions{}

	for _, opt := range opts {
		opt(&options)
	}

	return options
}

func (w *Worker) handleEntitlementEvent(ctx context.Context, entitlementID pkgmodels.NamespacedID, options ...handleOption) (marshaler.Event, error) {
	calculatedAt := time.Now()

	opts := getOptions(options...)

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("handling entitlement event: %w", err)
	}

	inScope, err := w.filters.IsNamespaceInScope(ctx, entitlementID.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to check if entitlement is in scope: %w", err)
	}
	if !inScope {
		return nil, nil
	}

	entitlements, err := w.opts.Entitlement.Entitlement.ListEntitlements(ctx, entitlement.ListEntitlementsParams{
		Namespaces:     []string{entitlementID.Namespace},
		IDs:            []string{entitlementID.ID},
		IncludeDeleted: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement: %w", err)
	}

	if len(entitlements.Items) == 0 {
		// Given that rolled back transactions also fire events, we can expect that sometimes the entitlement is not found
		// we still need to retry, as if the originating transaction is running for a while, the entitlement might
		// appear after the transaction is committed.
		return nil, router.NewWarningLogSeverityError(fmt.Errorf("entitlement not found: %s", entitlementID.ID))
	}

	if len(entitlements.Items) > 1 {
		return nil, fmt.Errorf("multiple entitlements found: %s", entitlementID.ID)
	}

	entitlementEntity := entitlements.Items[0]

	inScope, err = w.filters.IsEntitlementInScope(ctx, filters.EntitlementFilterRequest{
		Entitlement: entitlementEntity,
		EventAt:     opts.eventAt,
		Operation:   lo.FromPtrOr(opts.sourceOperation, snapshot.ValueOperationUpdate),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check if entitlement is in scope: %w", err)
	}
	if !inScope {
		return nil, nil
	}
	return w.processEntitlementEntity(ctx, &entitlementEntity, calculatedAt, options...)
}

func (w *Worker) processEntitlementEntity(ctx context.Context, entitlementEntity *entitlement.Entitlement, calculatedAt time.Time, options ...handleOption) (marshaler.Event, error) {
	if entitlementEntity == nil {
		return nil, fmt.Errorf("entitlement entity is nil")
	}

	opts := getOptions(options...)

	if entitlementEntity.ActiveFrom != nil && entitlementEntity.ActiveFrom.After(calculatedAt) {
		// Not yet active entitlement we don't need to process it yet
		return nil, nil
	}

	if entitlementEntity.DeletedAt != nil ||
		(entitlementEntity.ActiveTo != nil && entitlementEntity.ActiveTo.Before(calculatedAt)) {
		// entitlement got deleted while processing changes => let's create a delete event so that we are not working

		snap, err := w.createDeletedSnapshotEvent(ctx,
			entitlement.EntitlementDeletedEvent{
				Entitlement: *entitlementEntity,
				Namespace: models.NamespaceID{
					ID: entitlementEntity.Namespace,
				},
			}, calculatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to create entitlement delete snapshot event: %w", err)
		}

		err = w.filters.RecordLastCalculation(ctx, filters.RecordLastCalculationRequest{
			Entitlement:  *entitlementEntity,
			CalculatedAt: calculatedAt,
			IsDeleted:    true,
		})
		if err != nil {
			// This is not critical, as worst case we are going to unnecessarily recalculate the entitlement
			// for the next event
			w.opts.Logger.WarnContext(ctx, "failed to record last calculation for deleted entitlement", "error", err, "entitlement", entitlementEntity.ID)
		}

		return snap, nil
	}

	// Reset events are always recalculated asOf the time of reset, so that we have a snapshot of initial grants
	// and overages.
	if lo.FromPtr(opts.sourceOperation) == snapshot.ValueOperationReset {
		if entitlementEntity.CurrentUsagePeriod == nil {
			return nil, fmt.Errorf("entitlement has no current usage period, cannot create snapshot event")
		}

		snap, err := w.createSnapshotEvent(ctx, entitlementEntity, entitlementEntity.CurrentUsagePeriod.From, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create entitlement update snapshot event: %w", err)
		}

		return snap, nil
	}

	snap, err := w.createSnapshotEvent(ctx, entitlementEntity, calculatedAt, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create entitlement update snapshot event: %w", err)
	}

	err = w.filters.RecordLastCalculation(ctx, filters.RecordLastCalculationRequest{
		Entitlement:  *entitlementEntity,
		CalculatedAt: calculatedAt,
	})
	if err != nil {
		// This is not critical, as worst case we are going to unnecessarily recalculate the entitlement
		// for the next event
		w.opts.Logger.WarnContext(ctx, "failed to record last calculation for entitlement", "error", err, "entitlement", entitlementEntity.ID)
	}

	return snap, nil
}

type snapshotToEventInput struct {
	Entitlement       *entitlement.Entitlement
	Feature           *feature.Feature
	Value             *snapshot.EntitlementValue
	CalculatedAt      time.Time
	Source            string
	OverrideOperation *snapshot.ValueOperationType
}

func (i *snapshotToEventInput) Validate() error {
	var errs []error

	if i.Value == nil {
		errs = append(errs, fmt.Errorf("entitlement value is required"))
	}

	if i.Entitlement == nil {
		errs = append(errs, fmt.Errorf("entitlement is required"))
	}

	if i.Feature == nil {
		errs = append(errs, fmt.Errorf("feature is required"))
	}

	if i.CalculatedAt.IsZero() {
		errs = append(errs, fmt.Errorf("calculatedAt is required"))
	}

	if i.Source == "" {
		errs = append(errs, fmt.Errorf("source is required"))
	}

	if i.OverrideOperation != nil {
		if err := i.OverrideOperation.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("overrideOperation is invalid: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (w *Worker) snapshotToEvent(ctx context.Context, in snapshotToEventInput) (marshaler.Event, error) {
	if err := in.Validate(); err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	sub, err := w.opts.Subject.GetByKey(ctx, pkgmodels.NamespacedKey{
		Namespace: in.Entitlement.Namespace,
		Key:       in.Entitlement.SubjectKey,
	})
	if err != nil {
		if !pkgmodels.IsGenericNotFoundError(err) {
			return nil, fmt.Errorf("failed to get subject: %w", err)
		}

		sub = subject.Subject{
			Key: in.Entitlement.SubjectKey,
		}
	}

	cus, err := w.opts.Customer.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace:  in.Entitlement.Namespace,
		SubjectKey: in.Entitlement.SubjectKey,
	})
	if err != nil {
		if !pkgmodels.IsGenericNotFoundError(err) {
			return nil, fmt.Errorf("failed to get customer: %w", err)
		}
	}

	return marshaler.WithSource(
		in.Source,
		snapshot.SnapshotEvent{
			Entitlement: *in.Entitlement,
			Namespace: models.NamespaceID{
				ID: in.Entitlement.Namespace,
			},
			Subject:   sub,
			Feature:   *in.Feature,
			Customer:  cus,
			Operation: lo.FromPtrOr(in.OverrideOperation, snapshot.ValueOperationUpdate),

			CalculatedAt: &in.CalculatedAt,

			Value:              in.Value,
			CurrentUsagePeriod: in.Entitlement.CurrentUsagePeriod,
		},
	), nil
}

func (w *Worker) createSnapshotEvent(ctx context.Context, entitlementEntity *entitlement.Entitlement, calculatedAt time.Time, opts handleEntitlementEventOptions) (marshaler.Event, error) {
	feat, err := w.opts.Entitlement.Feature.GetFeature(ctx, entitlementEntity.Namespace, entitlementEntity.FeatureID, feature.IncludeArchivedFeatureTrue)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	calculationStart := time.Now()

	value, err := w.opts.Entitlement.Entitlement.GetEntitlementValue(ctx, entitlementEntity.Namespace, entitlementEntity.SubjectKey, entitlementEntity.ID, calculatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlement value: %w", err)
	}

	if value == nil {
		return nil, fmt.Errorf("unexpected nil: entitlement value")
	}

	w.metricRecalculationTime.Record(ctx, time.Since(calculationStart).Milliseconds(), metric.WithAttributes(
		attribute.String(metricAttributeKeyEntitltementType, string(entitlementEntity.EntitlementType)),
	))

	mappedValues, err := entitlementdriver.MapEntitlementValueToAPI(value)
	if err != nil {
		return nil, fmt.Errorf("failed to map entitlement value: %w", err)
	}

	return w.snapshotToEvent(ctx, snapshotToEventInput{
		Entitlement:       entitlementEntity,
		Feature:           feat,
		Value:             convert.ToPointer((snapshot.EntitlementValue)(mappedValues)),
		CalculatedAt:      calculatedAt,
		Source:            opts.source,
		OverrideOperation: opts.sourceOperation,
	})
}

func (w *Worker) createDeletedSnapshotEvent(ctx context.Context, event entitlement.EntitlementDeletedEvent, calculationTime time.Time) (marshaler.Event, error) {
	feat, err := w.opts.Entitlement.Feature.GetFeature(ctx, event.Namespace.ID, event.FeatureID, feature.IncludeArchivedFeatureTrue)
	if err != nil {
		return nil, fmt.Errorf("failed to get feature: %w", err)
	}

	sub, err := w.opts.Subject.GetByKey(ctx, pkgmodels.NamespacedKey{
		Namespace: event.Namespace.ID,
		Key:       event.SubjectKey,
	})
	if err != nil {
		if !pkgmodels.IsGenericNotFoundError(err) {
			return nil, fmt.Errorf("failed to get subject: %w", err)
		}

		sub = subject.Subject{
			Key: event.SubjectKey,
		}
	}

	cus, err := w.opts.Customer.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace:  event.Entitlement.Namespace,
		SubjectKey: event.Entitlement.SubjectKey,
	})
	if err != nil {
		if !pkgmodels.IsGenericNotFoundError(err) {
			return nil, fmt.Errorf("failed to get customer: %w", err)
		}
	}

	return marshaler.WithSource(
		metadata.ComposeResourcePath(event.Namespace.ID, metadata.EntityEntitlement, event.ID),
		snapshot.SnapshotEvent{
			Entitlement: event.Entitlement,
			Namespace: models.NamespaceID{
				ID: event.Namespace.ID,
			},
			Subject:   sub,
			Feature:   *feat,
			Customer:  cus,
			Operation: snapshot.ValueOperationDelete,

			CalculatedAt: convert.ToPointer(calculationTime),

			CurrentUsagePeriod: event.CurrentUsagePeriod,
		},
	), nil
}
