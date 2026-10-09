package transactions

import (
	"context"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
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

func TestDecoratedTransactionInput_AsGroupInput(t *testing.T) {
	id := ulid.Make().String()
	entryID := ulid.Make().String()
	original := &TransactionInput{entryInputs: []*EntryInput{{}}}
	input := ledger.WithEntryInputs(original, ledger.WithEntryID(original.EntryInputs()[0], entryID))
	annotations := models.Annotations{"source": "decorated input"}
	groupAnnotations := models.Annotations{"group": "decorated group"}

	requireDecoratedGroup := func(t *testing.T, input ledger.TransactionInput) {
		t.Helper()

		group := input.AsGroupInput("ns-test", groupAnnotations)
		require.Equal(t, "ns-test", group.Namespace())
		require.Equal(t, groupAnnotations, group.Annotations())
		require.Len(t, group.Transactions(), 1)

		transaction := group.Transactions()[0]
		require.Equal(t, id, transaction.AssignedID())
		require.Equal(t, annotations, transaction.Annotations())
		require.Len(t, transaction.EntryInputs(), 1)
		require.Equal(t, entryID, transaction.EntryInputs()[0].AssignedID())
	}

	t.Run("IDs before annotations", func(t *testing.T) {
		// given: annotations wrap an already identified transaction
		decorated := WithAnnotations(ledger.WithTransactionID(input, id), annotations)

		// when/then: grouping retains both wrappers' data
		requireDecoratedGroup(t, decorated)
	})

	t.Run("annotations before IDs", func(t *testing.T) {
		// given: ID assignment wraps an already annotated transaction
		decorated := ledger.WithTransactionID(WithAnnotations(input, annotations), id)

		// when/then: grouping retains both wrappers' data
		requireDecoratedGroup(t, decorated)
	})
}
