package transactions

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// spyCustomerTemplate implements CustomerTransactionTemplate and records Validate calls.
type spyCustomerTemplate struct {
	validateCalls int
}

func (s *spyCustomerTemplate) Validate() error {
	s.validateCalls++
	return nil
}

func (*spyCustomerTemplate) typeGuard() guard {
	return true
}

func (*spyCustomerTemplate) resolve(context.Context, customer.CustomerID, ResolverDependencies) (ledger.TransactionInput, error) {
	return nil, nil
}

var _ CustomerTransactionTemplate = (*spyCustomerTemplate)(nil)

func TestResolveTransactions_callsResolverValidate(t *testing.T) {
	t.Parallel()

	spy := &spyCustomerTemplate{}
	_, err := ResolveTransactions(
		t.Context(),
		ResolverDependencies{},
		ResolutionScope{
			CustomerID: customer.CustomerID{
				Namespace: "ns",
				ID:        "cust",
			},
		},
		spy,
	)
	require.NoError(t, err)
	require.Equal(t, 1, spy.validateCalls, "Resolver.Validate must be invoked for each template")
}
