package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	metered "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/rte"
	ingestevents "github.com/openmeterio/openmeter/openmeter/sink/flushhandler/ingestnotification/events"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Config struct {
	Client *entdb.Client
	Logger *slog.Logger
}

func (c Config) Validate() error {
	var errs []error
	if c.Client == nil {
		errs = append(errs, errors.New("postgres client is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func New(config Config) (rte.UpstreamResolver, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &resolver{db: config.Client, logger: config.Logger}, nil
}

type resolver struct {
	db     *entdb.Client
	logger *slog.Logger
}

var eventFactories = newEventFactories()

// NewEvent allocates a fresh source payload for decoding. Commands are excluded:
// another consumer group can handle them before their effects commit.
func NewEvent(name string) (marshaler.Event, bool) {
	factory, ok := eventFactories[name]
	if !ok {
		return nil, false
	}

	return factory(), true
}

func newEventFactories() map[string]func() marshaler.Event {
	prototypes := []marshaler.Event{
		ingestevents.EventBatchedIngest{},
		ledger.TransactionGroupCommittedEvent{},
		charges.ChargeInputsChangedEvent{},
		entitlement.EntitlementCreatedEventV2{},
		entitlement.EntitlementDeletedEventV2{},
		grant.CreatedEvent{},
		grant.CreatedEventV2{},
		grant.VoidedEvent{},
		grant.VoidedEventV2{},
		metered.EntitlementResetEvent{},
		metered.EntitlementResetEventV3{},
		customer.CustomerCreateEvent{},
		customer.CustomerUpdateEvent{},
		customer.CustomerDeleteEvent{},
		feature.FeatureCreateEvent{},
		feature.FeatureUpdateEvent{},
		feature.FeatureArchiveEvent{},
		meter.MeterCreateEvent{},
		meter.MeterUpdateEvent{},
		meter.MeterDeleteEvent{},
		subscription.CreatedEvent{},
		subscription.UpdatedEvent{},
		subscription.DeletedEvent{},
		subscription.CancelledEvent{},
		subscription.ContinuedEvent{},
	}
	factories := make(map[string]func() marshaler.Event, len(prototypes))
	for _, prototype := range prototypes {
		eventType := reflect.TypeOf(prototype)
		factories[prototype.EventName()] = func() marshaler.Event {
			return reflect.New(eventType).Interface().(marshaler.Event)
		}
	}

	return factories
}

func (r *resolver) Resolve(ctx context.Context, event marshaler.Event) ([]rte.DemandSelector, error) {
	if event == nil {
		return nil, models.NewGenericValidationError(errors.New("source event is required"))
	}

	// Group handlers decode pointers; direct callers may already have a value.
	value := reflect.ValueOf(event)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, models.NewGenericValidationError(errors.New("source event is required"))
		}

		if decoded, ok := value.Elem().Interface().(marshaler.Event); ok {
			event = decoded
		}
	}

	// Validate the original payload before reading fields from versioned literals.
	if err := event.Validate(); err != nil {
		return nil, err
	}

	switch e := event.(type) {
	case charges.ChargeInputsChangedEvent:
		return []rte.DemandSelector{{Namespace: e.Namespace, Kind: rte.CalculationKindCreditBalance, OwnerIDs: []string{e.CustomerID}}}, nil
	case ledger.TransactionGroupCommittedEvent:
		return r.query(ctx, ledgerQuery{namespace: e.Namespace, groupID: e.GroupID, now: clock.Now().UTC()})
	case ingestevents.EventBatchedIngest:
		if len(e.MeterSlugs) == 0 || len(e.RawEvents) == 0 {
			return nil, nil
		}

		timestamps := make([]time.Time, 0, len(e.RawEvents))
		for _, raw := range e.RawEvents {
			timestamps = append(timestamps, time.Unix(raw.Time, 0).UTC())
		}

		return r.query(ctx, ingestQuery{namespace: e.Namespace.ID, subject: e.SubjectKey, meters: e.MeterSlugs, timestamps: timestamps, now: clock.Now().UTC()})
	case entitlement.EntitlementCreatedEventV2:
		return entitlementSelector(e.Namespace.ID, e.Entitlement.ID), nil
	case entitlement.EntitlementDeletedEventV2:
		return entitlementSelector(e.Namespace.ID, e.Entitlement.ID), nil
	case grant.CreatedEvent:
		return entitlementSelector(e.Namespace.ID, e.OwnerID), nil
	case grant.CreatedEventV2:
		return entitlementSelector(e.Namespace.ID, e.Grant.OwnerID), nil
	case grant.VoidedEvent:
		return entitlementSelector(e.Namespace.ID, e.OwnerID), nil
	case grant.VoidedEventV2:
		return entitlementSelector(e.Namespace.ID, e.Grant.OwnerID), nil
	case metered.EntitlementResetEvent:
		return entitlementSelector(e.Namespace.ID, e.EntitlementID), nil
	case metered.EntitlementResetEventV3:
		return entitlementSelector(e.Namespace.ID, e.EntitlementID), nil
	case customer.CustomerCreateEvent:
		return r.forCustomer(ctx, e.Customer.Namespace, e.Customer.ID)
	case customer.CustomerUpdateEvent:
		return r.forCustomer(ctx, e.Customer.Namespace, e.Customer.ID)
	case customer.CustomerDeleteEvent:
		return r.forCustomer(ctx, e.Customer.Namespace, e.Customer.ID)
	case feature.FeatureCreateEvent:
		return r.query(ctx, featureQuery{namespace: e.Feature.Namespace, featureID: e.Feature.ID, now: clock.Now().UTC()})
	case feature.FeatureUpdateEvent:
		return r.query(ctx, featureQuery{namespace: e.Feature.Namespace, featureID: e.Feature.ID, now: clock.Now().UTC()})
	case feature.FeatureArchiveEvent:
		return r.query(ctx, featureQuery{namespace: e.Feature.Namespace, featureID: e.Feature.ID, now: clock.Now().UTC()})
	case meter.MeterCreateEvent:
		return r.query(ctx, meterQuery{namespace: e.Meter.Namespace, meterID: e.Meter.ID, now: clock.Now().UTC()})
	case meter.MeterUpdateEvent:
		return r.query(ctx, meterQuery{namespace: e.Meter.Namespace, meterID: e.Meter.ID, now: clock.Now().UTC()})
	case meter.MeterDeleteEvent:
		return r.query(ctx, meterQuery{namespace: e.Meter.Namespace, meterID: e.Meter.ID, now: clock.Now().UTC()})
	case subscription.CreatedEvent:
		return r.forCustomer(ctx, e.Subscription.Namespace, e.Customer.ID)
	case subscription.UpdatedEvent:
		return r.forCustomer(ctx, e.UpdatedView.Subscription.Namespace, e.UpdatedView.Customer.ID)
	case subscription.CancelledEvent:
		return r.forCustomer(ctx, e.Subscription.Namespace, e.Customer.ID)
	case subscription.DeletedEvent:
		return r.forCustomer(ctx, e.Subscription.Namespace, e.Customer.ID)
	case subscription.ContinuedEvent:
		return r.forCustomer(ctx, e.Subscription.Namespace, e.Customer.ID)
	default:
		return nil, nil
	}
}

func entitlementSelector(namespace, id string) []rte.DemandSelector {
	return []rte.DemandSelector{{Namespace: namespace, Kind: rte.CalculationKindMeteredEntitlement, OwnerIDs: []string{id}}}
}

func (r *resolver) forCustomer(ctx context.Context, namespace, id string) ([]rte.DemandSelector, error) {
	return r.query(ctx, customerQuery{namespace: namespace, customerID: id, now: clock.Now().UTC()})
}

type sourceQuery interface{ SQL() (string, []any) }

func (r *resolver) query(ctx context.Context, query sourceQuery) ([]rte.DemandSelector, error) {
	statement, args := query.SQL()
	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("resolve affected RTE demand: %w", err)
	}

	defer func() {
		if err := rows.Close(); err != nil {
			r.logger.ErrorContext(ctx, "close RTE upstream rows", "error", err)
		}
	}()
	var selectors []rte.DemandSelector
	type scope struct {
		namespace string
		kind      rte.CalculationKind
	}
	indexes := map[scope]int{}
	for rows.Next() {
		var namespace string
		var target rte.CalculationTarget
		var key []byte
		if err := rows.Scan(&namespace, &target.Kind, &key); err != nil {
			return nil, fmt.Errorf("scan affected RTE demand: %w", err)
		}

		target.Key = json.RawMessage(key)
		if err := target.Validate(); err != nil {
			return nil, fmt.Errorf("stored RTE demand key: %w", err)
		}

		group := scope{namespace: namespace, kind: target.Kind}
		index, ok := indexes[group]
		if !ok {
			index = len(selectors)
			indexes[group] = index
			selectors = append(selectors, rte.DemandSelector{Namespace: namespace, Kind: target.Kind})
		}

		selectors[index].Targets = append(selectors[index].Targets, target)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate affected RTE demand: %w", err)
	}

	return selectors, nil
}
