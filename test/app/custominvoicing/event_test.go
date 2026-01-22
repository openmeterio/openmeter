package custominvoicing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type CustomInvoicingEventTestSuite struct {
	CustomInvoicingTestSuite
}

func TestInvoicingEvent(t *testing.T) {
	suite.Run(t, new(CustomInvoicingEventTestSuite))
}

func (s *CustomInvoicingEventTestSuite) TestCreateInvoiceEvent() {
	// Given we have an invoice
	namespace := "ns-create-invoice-event-custom-invoicing"
	ctx := context.Background()

	s.setupDefaultBillingProfile(ctx, namespace, appcustominvoicing.Configuration{
		EnableDraftSyncHook:   true,
		EnableIssuingSyncHook: true,
	})

	invoice := s.CreateDraftInvoice(s.T(), ctx, billingtest.DraftInvoiceInput{
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

	// Let's validate the app data unmarshaling
	meta := appcustominvoicing.Meta{}
	s.NoError(meta.FromEventAppData(event.Apps.Invoicing))
	s.Equal(invoice.Workflow.Apps.Invoicing.GetID(), meta.GetID())
	s.True(meta.Configuration.EnableDraftSyncHook)
	s.True(meta.Configuration.EnableIssuingSyncHook)
}
