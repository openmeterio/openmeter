package transactions

import (
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func resolveCustomerFBOCreditPriority(configured *int) int {
	if configured != nil {
		return *configured
	}
	return ledger.DefaultCustomerFBOPriority
}
