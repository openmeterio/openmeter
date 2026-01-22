package billing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type InvoicingEventTestSuite struct {
	InvoicingTestSuite
}

func TestInvoicingEvent(t *testing.T) {
	suite.Run(t, new(InvoicingEventTestSuite))
}

func (s *InvoicingEventTestSuite) TestCreateInvoiceEvent() {
	// Given we have an invoice
	namespace := "ns-create-invoice-event"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	invoice := s.CreateDraftInvoice(s.T(), ctx, DraftInvoiceInput{
		Namespace: namespace,
		Customer:  s.CreateTestCustomer(namespace, "test-customer"),
	})

	// When we create an invoice created event
	event, err := billing.NewStandardInvoiceCreatedEvent(invoice)
	s.NoError(err)

	// Then the event should be be JSON marshallable
	marshaledInvoice, err := json.MarshalIndent(event, "", "  ")
	s.NoError(err)

	s.T().Logf("invoice created event: %s", string(marshaledInvoice))

	var unmarshaledEvent billing.StandardInvoiceCreatedEvent
	err = json.Unmarshal(marshaledInvoice, &unmarshaledEvent)
	s.NoError(err)

	// Then the event should contain the app bases as contextual information
	s.Equal(event.Apps.Tax.AppBase, invoice.Workflow.Apps.Tax.GetAppBase())
	s.Equal(event.Apps.Payment.AppBase, invoice.Workflow.Apps.Payment.GetAppBase())
	s.Equal(event.Apps.Invoicing.AppBase, invoice.Workflow.Apps.Invoicing.GetAppBase())
}
