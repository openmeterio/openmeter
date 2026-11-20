package billing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingconvert "github.com/openmeterio/openmeter/openmeter/billing/apiconvert"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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

	// And the invoice line references a Tax Code that has a Stripe mapping
	taxCode := s.TaxCodeEnv.GetOrCreateByAppMapping(s.T(), namespace, app.AppTypeStripe, "txcd_10000000")

	invoice, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      invoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceSystem,
		EditFn: func(invoice *billing.StandardInvoice) error {
			for _, line := range invoice.Lines.OrEmpty() {
				line.TaxConfig = &productcatalog.TaxCodeConfig{
					Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
					TaxCodeID: taxCode.ID,
				}
			}

			return nil
		},
	})
	s.Require().NoError(err)

	taxCodes, err := billingconvert.ResolveInvoiceTaxCodes(ctx, s.TaxCodeResolver, billingconvert.ResolveInvoiceTaxCodesInput{
		Namespace:     namespace,
		StandardLines: invoice.Lines.OrEmpty(),
	})
	s.Require().NoError(err)

	// When we create an invoice created event
	event, err := billing.NewStandardInvoiceCreatedEvent(invoice, taxCodes)
	s.NoError(err)

	// Then the event should be be JSON marshallable
	marshaledInvoice, err := json.MarshalIndent(event, "", "  ")
	s.NoError(err)

	s.T().Logf("invoice created event: %s", string(marshaledInvoice))

	var unmarshaledEvent billing.StandardInvoiceCreatedEvent
	err = json.Unmarshal(marshaledInvoice, &unmarshaledEvent)
	s.NoError(err)

	// Then the event keeps the legacy taxOverrides shape: Stripe code and the embedded Tax Code
	var rawEvent struct {
		Invoice struct {
			Lines []struct {
				TaxOverrides struct {
					TaxCodeID string `json:"tax_code_id"`
					Stripe    struct {
						Code string `json:"code"`
					} `json:"stripe"`
					TaxCode struct {
						ID string `json:"id"`
					} `json:"tax_code"`
				} `json:"taxOverrides"`
			} `json:"lines"`
		} `json:"invoice"`
	}
	s.Require().NoError(json.Unmarshal(marshaledInvoice, &rawEvent))
	s.Require().NotEmpty(rawEvent.Invoice.Lines)

	for _, line := range rawEvent.Invoice.Lines {
		s.Equal(taxCode.ID, line.TaxOverrides.TaxCodeID)
		s.Equal("txcd_10000000", line.TaxOverrides.Stripe.Code)
		s.Equal(taxCode.ID, line.TaxOverrides.TaxCode.ID)
	}

	// Then the event should contain the app bases as contextual information
	s.Equal(event.Apps.Tax.AppBase, invoice.Workflow.Apps.Tax.GetAppBase())
	s.Equal(event.Apps.Payment.AppBase, invoice.Workflow.Apps.Payment.GetAppBase())
	s.Equal(event.Apps.Invoicing.AppBase, invoice.Workflow.Apps.Invoicing.GetAppBase())
}
