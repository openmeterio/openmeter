package rte

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
)

type Service interface {
	// RefreshDemand executes a claimed calculation. Only this path uses checkpoints.
	RefreshDemand(context.Context, RefreshDemandInput) error

	CreditBalances() Calculation[creditbalance.Key, creditbalance.Balance]
	MeteredEntitlements() Calculation[meteredentitlement.Key, meteredentitlement.Balance]
}

type RefreshDemandInput struct {
	Target         CalculationTarget
	UseCheckpoints bool
}
