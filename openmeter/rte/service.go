package rte

import (
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
)

type Service interface {
	CreditBalances() Calculation[creditbalance.Key, creditbalance.Balance]
	MeteredEntitlements() Calculation[meteredentitlement.Key, meteredentitlement.Balance]
}
