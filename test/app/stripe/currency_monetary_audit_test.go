package appstripe

import (
	"net/http"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	stripeapp "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// TestMonetaryAuditProviderCurrencyUnits preserves the invoice's economic value
// through the production Stripe app/client. Stripe deliberately differs from
// ISO currency precision for ISK and UGX: its API requires 500 to charge 5 ISK
// or 5 UGX, despite both currencies having zero accounting decimal places.
// Source: https://docs.stripe.com/currencies#special-cases.
//
// openmeter/app/stripe/calculator.go: NewStripeCalculator currently derives its
// multiplier from FiatCurrency.Details().Precision. RoundToAmount is shared by
// appinvoice.go's ordinary/commitment, discount, and credit item mappings on
// both creation and update. The correction belongs in this provider conversion:
// accounting precision must remain zero, while Stripe's API unit is 1/100.
// A synthetic invoice for 500,000 ISK/UGX consequently becomes a Stripe amount
// representing only 5,000 ISK/UGX. USD and JPY are controls for the ordinary
// two-decimal and zero-decimal cases. No exchange rate or rounding is involved.
//
// Reproduce: POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./test/app/stripe
// -run TestStripeInvoicing/TestMonetaryAuditProviderCurrencyUnits -count=1 -v
func (s *StripeInvoiceTestSuite) TestMonetaryAuditProviderCurrencyUnits() {
	for _, tc := range []struct {
		currency       currencyx.FiatCode
		expectedAmount int64
	}{
		{currency: "USD", expectedAmount: 50000000},
		{currency: "JPY", expectedAmount: 500000},
		{currency: "ISK", expectedAmount: 50000000},
		{currency: "UGX", expectedAmount: 50000000},
	} {
		s.Run(string(tc.currency), func() {
			// Given a persisted 500,000-unit invoice in the customer's own currency.
			ctx := s.T().Context()
			namespace := s.GetUniqueNamespace("stripe-currency-monetary-audit")
			period := timeutil.ClosedPeriod{
				From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
			}
			clock.FreezeTime(period.To.Add(time.Hour))
			defer clock.UnFreeze()

			sandboxApp := s.InstallSandboxApp(s.T(), namespace)
			s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID(), billingtest.WithManualApproval())
			s.TaxCodeEnv.ProvisionProviderDefaultTaxCode(s.T(), namespace)
			customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
				Namespace: namespace,
				CustomerMutate: customer.CustomerMutate{
					Key:          lo.ToPtr("synthetic-currency-audit-customer"),
					Name:         "Synthetic currency audit customer",
					Currency:     lo.ToPtr(currencyx.Code(tc.currency)),
					PrimaryEmail: lo.ToPtr("synthetic@example.com"),
					BillingAddress: &models.Address{
						Country: lo.ToPtr(models.CountryCode("US")),
					},
				},
			})
			s.Require().NoError(err)

			_, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
				Customer: customerEntity.GetID(),
				Currency: tc.currency,
				Lines: billing.NewCreatePendingInvoiceLines([]billing.GatheringLine{
					billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
						Namespace:     namespace,
						Period:        period,
						InvoiceAt:     period.To,
						ManagedBy:     billing.ManuallyManagedLine,
						Name:          "Synthetic currency audit service",
						PerUnitAmount: alpacadecimal.NewFromInt(500000),
						PaymentTerm:   productcatalog.InArrearsPaymentTerm,
						Currency:      tc.currency,
					}),
				}),
			})
			s.Require().NoError(err)

			invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: customerEntity.GetID(),
				AsOf:     lo.ToPtr(period.To),
			})
			s.Require().NoError(err)
			s.Require().Len(invoices, 1)

			persistedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: invoices[0].GetInvoiceID(),
				Expand:  billing.StandardInvoiceExpandAll,
			})
			s.Require().NoError(err)
			s.Require().Equal(tc.currency, persistedInvoice.Currency)
			s.Require().Equal(500000.0, persistedInvoice.Totals.Total.InexactFloat64())
			s.Require().Empty(persistedInvoice.ValidationIssues)

			installedApp, err := s.Fixture.setupApp(ctx, namespace)
			s.Require().NoError(err)
			_, err = s.Fixture.setupAppCustomerData(ctx, installedApp, customerEntity)
			s.Require().NoError(err)
			installedStripeApp, ok := installedApp.(stripeapp.App)
			s.Require().True(ok)

			provider := &stripeMonetaryAuditTransport{
				items:     make(map[string][]*stripe.InvoiceItem),
				responses: make(map[string][]byte),
			}
			previousHTTPClient := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{}).(*stripe.BackendImplementation).HTTPClient
			stripe.SetHTTPClient(&http.Client{Transport: provider})
			defer stripe.SetHTTPClient(previousHTTPClient)

			stripeInvoicingApp, err := stripeapp.New(installedStripeApp.Meta, stripeapp.AppConfig{
				Logger:                 omtestutils.NewDiscardLogger(s.T()),
				AppService:             s.AppService,
				BillingService:         s.BillingService,
				StripeAppClientFactory: stripeclient.NewStripeAppClient,
				StripeAppService:       s.AppStripeService,
				SecretService:          s.SecretService,
			})
			s.Require().NoError(err)

			// When the actual app and Stripe SDK encode the invoice into a request.
			result, err := stripeInvoicingApp.UpsertStandardInvoice(ctx, persistedInvoice)
			s.Require().NoError(err)
			stripeInvoiceID, ok := result.GetExternalID()
			s.Require().True(ok)
			s.Require().Len(provider.items[stripeInvoiceID], 1)
			s.Require().Equal(stripe.Currency(tc.currency), provider.currency)

			// Then provider units must represent exactly the persisted amount.
			providerAmount := provider.items[stripeInvoiceID][0].Amount
			s.T().Logf("currency=%s invoice_total=%s provider_amount=%d", tc.currency, persistedInvoice.Totals.Total, providerAmount)
			s.Equal(tc.expectedAmount, providerAmount, "Stripe's API unit convention can differ from ISO accounting precision")
		})
	}
}
