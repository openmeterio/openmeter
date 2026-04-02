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

func (*spyCustomerTemplate) correct(context.Context, CorrectionScope, ResolverDependencies) ([]ledger.TransactionInput, error) {
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
	require.Equal(t, 1, spy.validateCalls, "TransactionTemplate.Validate must be invoked for each template")
}

type annotatedCustomerTemplate struct{}

func (annotatedCustomerTemplate) Validate() error {
	return nil
}

func (annotatedCustomerTemplate) typeGuard() guard {
	return true
}

func (annotatedCustomerTemplate) resolve(_ context.Context, _ customer.CustomerID, _ ResolverDependencies) (ledger.TransactionInput, error) {
	return &TransactionInput{}, nil
}

func (annotatedCustomerTemplate) correct(context.Context, CorrectionScope, ResolverDependencies) ([]ledger.TransactionInput, error) {
	return nil, nil
}

func TestResolveTransactions_addsTemplateAnnotations(t *testing.T) {
	t.Parallel()

	inputs, err := ResolveTransactions(
		t.Context(),
		ResolverDependencies{},
		ResolutionScope{
			CustomerID: customer.CustomerID{
				Namespace: "ns",
				ID:        "cust",
			},
		},
		annotatedCustomerTemplate{},
	)
	require.NoError(t, err)
	require.Len(t, inputs, 1)
	require.Equal(t, "annotatedCustomerTemplate", inputs[0].Annotations()[ledger.AnnotationTransactionTemplateName])
	require.Equal(t, string(ledger.TransactionDirectionForward), inputs[0].Annotations()[ledger.AnnotationTransactionDirection])
}
