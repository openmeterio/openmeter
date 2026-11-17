package client

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
)

type syncClient struct {
	engine rte.Service
	demand rte.DemandService
}

// NewSync borrows the host's engine and demand service. Calls use the caller's context.
func NewSync(engine rte.Service, demand rte.DemandService) (rte.SyncClient, error) {
	if engine == nil || demand == nil {
		return nil, errors.New("RTE sync client requires an engine and a demand service")
	}

	return &syncClient{engine: engine, demand: demand}, nil
}

func (c *syncClient) CreditBalances() rte.SyncBalanceClient[creditbalance.Key, creditbalance.Balance] {
	return syncBalances[creditbalance.Key, creditbalance.Balance]{engine: c.engine.CreditBalances(), demand: c.demand}
}

func (c *syncClient) MeteredEntitlements() rte.SyncBalanceClient[meteredentitlement.Key, meteredentitlement.Balance] {
	return syncBalances[meteredentitlement.Key, meteredentitlement.Balance]{engine: c.engine.MeteredEntitlements(), demand: c.demand}
}

type syncBalances[K rte.Key, V any] struct {
	engine rte.Calculation[K, V]
	demand rte.DemandService
}

func (b syncBalances[K, V]) Calculate(ctx context.Context, key K) (rte.State[V], error) {
	return b.engine.Calculate(ctx, key)
}

func (b syncBalances[K, V]) EnsureFresh(ctx context.Context, key K, freshness rte.FreshnessPolicy) error {
	target, err := rte.NewCalculationTarget(key)
	if err != nil {
		return err
	}

	return b.demand.EnsureFresh(ctx, freshness.EnsureFreshInput(target))
}
