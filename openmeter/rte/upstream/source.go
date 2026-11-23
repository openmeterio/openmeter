package upstream

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	metered "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type sourceEvent struct {
	newEvent func() marshaler.Event
	resolve  func(*resolver, context.Context, marshaler.Event) ([]rte.DemandSelector, error)
}

// The worker decodes into *T. Direct callers may already hold a T.
func registerSource[T marshaler.Event, P interface {
	*T
	marshaler.Event
}](
	events map[string]sourceEvent,
	resolve func(*resolver, context.Context, T) ([]rte.DemandSelector, error),
) {
	var event T
	events[event.EventName()] = sourceEvent{
		newEvent: func() marshaler.Event { return P(new(T)) },
		resolve: func(r *resolver, ctx context.Context, event marshaler.Event) ([]rte.DemandSelector, error) {
			switch e := event.(type) {
			case T:
				return resolve(r, ctx, e)
			case P:
				return resolve(r, ctx, *e)
			default:
				return nil, nil
			}
		},
	}
}

var sourceEvents = registeredSources()

// NewEvent allocates a fresh source payload for decoding. Commands are excluded
// because a different consumer group may handle them before their effects commit.
func NewEvent(name string) (marshaler.Event, bool) {
	source, ok := sourceEvents[name]
	if !ok {
		return nil, false
	}

	return source.newEvent(), true
}

func registeredSources() map[string]sourceEvent {
	events := make(map[string]sourceEvent)

	registerSource(events, func(_ *resolver, _ context.Context, e charges.CustomerChargesChangedEvent) ([]rte.DemandSelector, error) {
		return []rte.DemandSelector{{Namespace: e.Namespace, Kind: rte.CalculationKindCreditBalance, OwnerIDs: []string{e.CustomerID}}}, nil
	})

	registerSource(events, func(r *resolver, ctx context.Context, e ledger.TransactionGroupCommittedEvent) ([]rte.DemandSelector, error) {
		return r.query(ctx, ledgerQuery{namespace: e.Namespace, groupID: e.GroupID, now: clock.Now().UTC()})
	})

	registerSource(events, (*resolver).forIngest)

	registerSource(events, func(_ *resolver, _ context.Context, e entitlement.EntitlementCreatedEventV2) ([]rte.DemandSelector, error) {
		return entitlementSelector(e.Namespace.ID, e.Entitlement.ID), nil
	})
	registerSource(events, func(_ *resolver, _ context.Context, e entitlement.EntitlementDeletedEventV2) ([]rte.DemandSelector, error) {
		return entitlementSelector(e.Namespace.ID, e.Entitlement.ID), nil
	})

	registerSource(events, func(_ *resolver, _ context.Context, e grant.CreatedEvent) ([]rte.DemandSelector, error) {
		return entitlementSelector(e.Namespace.ID, e.OwnerID), nil
	})
	registerSource(events, func(_ *resolver, _ context.Context, e grant.CreatedEventV2) ([]rte.DemandSelector, error) {
		return entitlementSelector(e.Namespace.ID, e.Grant.OwnerID), nil
	})
	registerSource(events, func(_ *resolver, _ context.Context, e grant.VoidedEvent) ([]rte.DemandSelector, error) {
		return entitlementSelector(e.Namespace.ID, e.OwnerID), nil
	})
	registerSource(events, func(_ *resolver, _ context.Context, e grant.VoidedEventV2) ([]rte.DemandSelector, error) {
		return entitlementSelector(e.Namespace.ID, e.Grant.OwnerID), nil
	})

	registerSource(events, func(_ *resolver, _ context.Context, e metered.EntitlementResetEvent) ([]rte.DemandSelector, error) {
		return entitlementSelector(e.Namespace.ID, e.EntitlementID), nil
	})
	registerSource(events, func(_ *resolver, _ context.Context, e metered.EntitlementResetEventV3) ([]rte.DemandSelector, error) {
		return entitlementSelector(e.Namespace.ID, e.EntitlementID), nil
	})

	registerSource(events, func(r *resolver, ctx context.Context, e customer.CustomerCreateEvent) ([]rte.DemandSelector, error) {
		return r.forCustomer(ctx, e.Customer.Namespace, e.Customer.ID)
	})
	registerSource(events, func(r *resolver, ctx context.Context, e customer.CustomerUpdateEvent) ([]rte.DemandSelector, error) {
		return r.forCustomer(ctx, e.Customer.Namespace, e.Customer.ID)
	})
	registerSource(events, func(r *resolver, ctx context.Context, e customer.CustomerDeleteEvent) ([]rte.DemandSelector, error) {
		return r.forCustomer(ctx, e.Customer.Namespace, e.Customer.ID)
	})

	registerSource(events, func(r *resolver, ctx context.Context, e feature.FeatureCreateEvent) ([]rte.DemandSelector, error) {
		return r.query(ctx, featureQuery{namespace: e.Feature.Namespace, featureID: e.Feature.ID, now: clock.Now().UTC()})
	})
	registerSource(events, func(r *resolver, ctx context.Context, e feature.FeatureUpdateEvent) ([]rte.DemandSelector, error) {
		return r.query(ctx, featureQuery{namespace: e.Feature.Namespace, featureID: e.Feature.ID, now: clock.Now().UTC()})
	})
	registerSource(events, func(r *resolver, ctx context.Context, e feature.FeatureArchiveEvent) ([]rte.DemandSelector, error) {
		return r.query(ctx, featureQuery{namespace: e.Feature.Namespace, featureID: e.Feature.ID, now: clock.Now().UTC()})
	})

	registerSource(events, func(r *resolver, ctx context.Context, e meter.MeterCreateEvent) ([]rte.DemandSelector, error) {
		return r.query(ctx, meterQuery{namespace: e.Meter.Namespace, meterID: e.Meter.ID, now: clock.Now().UTC()})
	})
	registerSource(events, func(r *resolver, ctx context.Context, e meter.MeterUpdateEvent) ([]rte.DemandSelector, error) {
		return r.query(ctx, meterQuery{namespace: e.Meter.Namespace, meterID: e.Meter.ID, now: clock.Now().UTC()})
	})
	registerSource(events, func(r *resolver, ctx context.Context, e meter.MeterDeleteEvent) ([]rte.DemandSelector, error) {
		return r.query(ctx, meterQuery{namespace: e.Meter.Namespace, meterID: e.Meter.ID, now: clock.Now().UTC()})
	})

	registerSource(events, func(r *resolver, ctx context.Context, e subscription.CreatedEvent) ([]rte.DemandSelector, error) {
		return r.forCustomer(ctx, e.Subscription.Namespace, e.Customer.ID)
	})
	registerSource(events, func(r *resolver, ctx context.Context, e subscription.UpdatedEvent) ([]rte.DemandSelector, error) {
		return r.forCustomer(ctx, e.UpdatedView.Subscription.Namespace, e.UpdatedView.Customer.ID)
	})
	registerSource(events, func(r *resolver, ctx context.Context, e subscription.CancelledEvent) ([]rte.DemandSelector, error) {
		return r.forCustomer(ctx, e.Subscription.Namespace, e.Customer.ID)
	})
	registerSource(events, func(r *resolver, ctx context.Context, e subscription.DeletedEvent) ([]rte.DemandSelector, error) {
		return r.forCustomer(ctx, e.Subscription.Namespace, e.Customer.ID)
	})
	registerSource(events, func(r *resolver, ctx context.Context, e subscription.ContinuedEvent) ([]rte.DemandSelector, error) {
		return r.forCustomer(ctx, e.Subscription.Namespace, e.Customer.ID)
	})

	return events
}
