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

func (*spyCustomerTemplate) code() TransactionTemplateCode {
	return "test.spy"
}

func (*spyCustomerTemplate) resolve(context.Context, customer.CustomerID, ResolverDependencies) (ledger.TransactionInput, error) {
	return nil, nil
}

func (*spyCustomerTemplate) correct(CorrectionScope) ([]ledger.TransactionInput, error) {
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

func TestResolveTransactions_addsTemplateAnnotations(t *testing.T) {
	t.Parallel()

	input, err := annotateTemplateTransaction(&TransactionInput{}, IssueCustomerReceivableTemplate{}, ledger.TransactionDirectionForward)
	require.NoError(t, err)
	require.Equal(t, string(TemplateCodeIssueCustomerReceivable), input.Annotations()[ledger.AnnotationTransactionTemplateCode])
	require.Equal(t, string(ledger.TransactionDirectionForward), input.Annotations()[ledger.AnnotationTransactionDirection])
}
