package httpdriver

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestMapEventInvoiceToAPIDefaultTaxConfig(t *testing.T) {
	taxCode := &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01JTAXCODE0000000000000000A"},
		AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: "txcd_10000000"}},
	}

	invoice, err := billing.NewEventStandardInvoice(billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: "ns",
			ID:        "01JINVOICE0000000000000000A",
			Status:    billing.StandardInvoiceStatusDraftCreated,
			Workflow: billing.InvoiceWorkflow{
				Config: billing.WorkflowConfig{
					Collection: billing.DefaultWorkflowConfig.Collection,
					Invoicing: billing.InvoicingConfig{
						DefaultTaxConfig: &productcatalog.TaxCodeConfig{
							Behavior:  lo.ToPtr(productcatalog.InclusiveTaxBehavior),
							TaxCodeID: taxCode.ID,
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	t.Run("the event carries the tax code ID but no Stripe code", func(t *testing.T) {
		raw, err := json.Marshal(invoice.Invoice.Workflow.Config.Invoicing.DefaultTaxConfig)
		require.NoError(t, err)
		require.JSONEq(t, `{"behavior":"inclusive","tax_code_id":"`+taxCode.ID+`"}`, string(raw))
	})

	t.Run("the notification payload shows the Stripe code from the tax code", func(t *testing.T) {
		apiInvoice, err := MapEventInvoiceToAPI(invoice, map[string]*taxcode.TaxCode{taxCode.ID: taxCode})
		require.NoError(t, err)
		require.NotNil(t, apiInvoice.Workflow.Workflow.Invoicing)
		require.Equal(t, &api.TaxConfig{
			Behavior:  lo.ToPtr(api.TaxBehaviorInclusive),
			Stripe:    &api.StripeTaxConfig{Code: "txcd_10000000"},
			TaxCodeId: lo.ToPtr(taxCode.ID),
		}, apiInvoice.Workflow.Workflow.Invoicing.DefaultTaxConfig)
	})
}
