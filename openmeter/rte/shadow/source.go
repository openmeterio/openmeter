package shadow

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/models"
)

type creditReadObserver interface {
	ObserveCreditRead(context.Context, customerbalance.GetBalanceServiceInput, time.Duration)
}

// ObserveCreditReads decorates the service consumed by the product facade.
// At this boundary currency identity and feature scope are resolved, and RTE
// uses its own undecorated source when replaying the request.
func ObserveCreditReads(service customerbalance.Service, observer creditReadObserver) customerbalance.Service {
	return observedCreditReads{Service: service, observer: observer}
}

type observedCreditReads struct {
	customerbalance.Service
	observer creditReadObserver
}

func (s observedCreditReads) GetBalance(ctx context.Context, input customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error) {
	if input.BalanceQuery.AsOf != nil || input.BalanceQuery.After != nil {
		return s.Service.GetBalance(ctx, input)
	}

	start := time.Now()
	balance, err := s.Service.GetBalance(ctx, input)
	if err == nil {
		s.observer.ObserveCreditRead(ctx, input, time.Since(start))
	}

	return balance, err
}

type meteredValueObserver interface {
	ObserveMeteredRead(context.Context, models.NamespacedID, time.Duration)
	ObserveMeteredCalculation(context.Context, models.NamespacedID, time.Duration)
}

// ObserveMeteredValues captures the metered portion of current access checks
// and worker calculations. The engine can still call GetEntitlementBalance
// directly to calculate a shadow result without submitting another request.
func ObserveMeteredValues(connector meteredentitlement.Connector, observer meteredValueObserver) meteredentitlement.Connector {
	return observedMeteredValues{Connector: connector, observer: observer}
}

type observedMeteredValues struct {
	meteredentitlement.Connector
	observer meteredValueObserver
}

func (s observedMeteredValues) GetValue(ctx context.Context, ent *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
	var record func(context.Context, models.NamespacedID, time.Duration)
	switch entitlement.CurrentValueOperationFromContext(ctx) {
	case entitlement.CurrentValueOperationRead:
		record = s.observer.ObserveMeteredRead
	case entitlement.CurrentValueOperationCalculate:
		record = s.observer.ObserveMeteredCalculation
	default:
		return s.Connector.GetValue(ctx, ent, at)
	}

	start := time.Now()
	value, err := s.Connector.GetValue(ctx, ent, at)
	if err == nil {
		record(ctx, models.NamespacedID{Namespace: ent.Namespace, ID: ent.ID}, time.Since(start))
	}

	return value, err
}
