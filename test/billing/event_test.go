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

	s.InstallSandboxApp(s.T(), namespace)

	createProfileInput := MinimalCreateProfileInputTemplate
	createProfileInput.Namespace = namespace

	_, err := s.BillingService.CreateProfile(ctx, createProfileInput)
	s.NoError(err)
	invoice := s.createDraftInvoice(s.T(), ctx, draftInvoiceInput{
		Customer: s.CreateTestCustomer(namespace, "test-customer"),
	})

	// When we create an invoice created event
	event := billing.NewInvoiceCreatedEvent(invoice)

	// Then the event should be be JSON marshallable
	marshaledInvoice, err := json.MarshalIndent(event, "", "  ")
	s.NoError(err)

	s.T().Logf("invoice created event: %s", string(marshaledInvoice))

	var unmarshaledEvent billing.InvoiceCreatedEvent
	err = json.Unmarshal(marshaledInvoice, &unmarshaledEvent)
	s.NoError(err)

	// Then the event should contain the app bases as contextual information
	s.Equal(event.AppBases.Tax, invoice.Workflow.Apps.Tax.GetAppBase())
	s.Equal(event.AppBases.Payment, invoice.Workflow.Apps.Payment.GetAppBase())
	s.Equal(event.AppBases.Invocing, invoice.Workflow.Apps.Invoicing.GetAppBase())
}
