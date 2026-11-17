package rte

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
)

// Client reads stored state, registers demand and submits work to a running RTE.
// Its caller authorizes access to the namespace and key.
type Client interface {
	CreditBalances() BalanceClient[creditbalance.Key, creditbalance.Balance]
	MeteredEntitlements() BalanceClient[meteredentitlement.Key, meteredentitlement.Balance]
}

type BalanceClient[K Key, V any] interface {
	Get(context.Context, K) (State[V], error)
	// Requests acknowledge publication. A subsequent Get may still see the previous value.
	RequestCalculation(context.Context, K) error
	RequestRefresh(context.Context, K, FreshnessPolicy) error
	// Activate asks the worker to calculate a newly tracked balance and recalculate
	// it when its inputs change. The first result may arrive after Activate returns.
	Activate(context.Context, K) error
	// Deactivate stops tracking changes and keeps the last successful result.
	Deactivate(context.Context, K) error
}

// SyncClient runs operations to completion in a process that already hosts the engine.
type SyncClient interface {
	CreditBalances() SyncBalanceClient[creditbalance.Key, creditbalance.Balance]
	MeteredEntitlements() SyncBalanceClient[meteredentitlement.Key, meteredentitlement.Balance]
}

type SyncBalanceClient[K Key, V any] interface {
	Calculate(context.Context, K) (State[V], error)
	// EnsureFresh waits for freshness relative to source events RTE has processed.
	EnsureFresh(context.Context, K, FreshnessPolicy) error
}
