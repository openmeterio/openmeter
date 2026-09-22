package billingservice

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFireAndActivateReturnsNamedValidationIssueForUnavailableTrigger(t *testing.T) {
	stateMachine := allocateStateMachine()
	stateMachine.Invoice.Status = billing.StandardInvoiceStatusPaid

	err := stateMachine.FireAndActivate(t.Context(), billing.TriggerPaymentUncollectible)
	require.ErrorIs(t, err, billing.ErrInvoiceActionNotAvailable)

	var validationIssue billing.ValidationIssue
	require.ErrorAs(t, err, &validationIssue)

	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Equal(t, models.Annotations{
		billing.AttributeKeyInvoiceStatus:  billing.StandardInvoiceStatusPaid,
		billing.AttributeKeyInvoiceTrigger: billing.TriggerPaymentUncollectible,
	}, issues[0].Attributes)
	require.Equal(t, billing.StandardInvoiceStatusPaid, stateMachine.Invoice.Status)
}
