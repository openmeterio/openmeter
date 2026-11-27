package apiconvert

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestMapInvoiceLineToAPIStripeCode(t *testing.T) {
	const liveStripeCode = "txcd_live"

	taxCode := &taxcode.TaxCode{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01JTAXCODE0000000000000000A"},
		AppMappings:  taxcode.TaxCodeAppMappings{{AppType: app.AppTypeStripe, TaxCode: liveStripeCode}},
	}
	taxCodes := productcatalog.TaxCodesByID{taxCode.ID: taxCode}

	tests := []struct {
		name           string
		invoicingApp   app.AppType
		stamp          *string
		wantStripeCode string
	}{
		{
			name:           "a stamped line on a Stripe invoice shows the stamp, not the live mapping",
			invoicingApp:   app.AppTypeStripe,
			stamp:          lo.ToPtr("txcd_stamped"),
			wantStripeCode: "txcd_stamped",
		},
		{
			name:           "a line stamped without a code on a Stripe invoice shows no code",
			invoicingApp:   app.AppTypeStripe,
			stamp:          lo.ToPtr(""),
			wantStripeCode: "",
		},
		{
			name:           "an unstamped line on a Stripe invoice shows no code though the Tax Code maps to one now",
			invoicingApp:   app.AppTypeStripe,
			wantStripeCode: "",
		},
		{
			name:           "a line on a non-Stripe invoice shows the live mapping",
			invoicingApp:   app.AppTypeCustomInvoicing,
			stamp:          lo.ToPtr("txcd_stamped"),
			wantStripeCode: liveStripeCode,
		},
		{
			name:           "a line on an invoice with an unknown invoicing app shows the live mapping",
			stamp:          lo.ToPtr("txcd_stamped"),
			wantStripeCode: liveStripeCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a line with one detailed line and the stamp state of the test case
			line := &billing.StandardLine{
				StandardLineBase: billing.StandardLineBase{
					ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
						Namespace: "ns",
						ID:        "01JLINE0000000000000000000A",
						Name:      "line",
					}),
					ManagedBy: billing.ManuallyManagedLine,
					Engine:    billing.LineEngineTypeInvoice,
					TaxConfig: &productcatalog.TaxCodeConfig{
						Behavior:  lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
						TaxCodeID: taxCode.ID,
					},
					InvoicingAppTaxCode: tt.stamp,
				},
				UsageBased: &billing.UsageBasedLine{
					Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(10),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
				},
				DetailedLines: billing.DetailedLines{{}},
			}

			// when: the line is mapped to v1
			apiLine, err := mapInvoiceLineToAPI(line, taxCodes, tt.invoicingApp)
			require.NoError(t, err)

			// then: the line, its rate card and its detailed lines show the same tax config
			var wantStripe *apiv1models.StripeTaxConfig
			if tt.wantStripeCode != "" {
				wantStripe = &apiv1models.StripeTaxConfig{Code: tt.wantStripeCode}
			}

			want := &apiv1models.TaxConfig{
				Behavior:  lo.ToPtr(apiv1models.TaxBehaviorExclusive),
				Stripe:    wantStripe,
				TaxCodeId: lo.ToPtr(taxCode.ID),
			}

			require.Equal(t, want, apiLine.TaxConfig)
			require.NotNil(t, apiLine.RateCard)
			require.Equal(t, want, apiLine.RateCard.TaxConfig)
			require.NotNil(t, apiLine.Children)
			require.Len(t, *apiLine.Children, 1)
			require.Equal(t, want, (*apiLine.Children)[0].TaxConfig)
		})
	}
}
