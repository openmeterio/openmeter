package appstripe

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v80"
	stripego "github.com/stripe/stripe-go/v80/client"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// Set this to a Stripe test-mode secret key to run the scenario against a real Stripe account
// instead of the client mock:
//
//	POSTGRES_HOST=127.0.0.1 OPENMETER_TEST_STRIPE_SECRET_KEY=sk_test_... \
//	  go test ./test/app/stripe/ -run 'TestStripeInvoicing/TestSubscriptionZeroUsageInvoiceWithCredits' -count=1 -v
//
// The test then creates its own Stripe customer with a default card and logs the Stripe invoice
// ID at the end so it can be inspected in the dashboard. Nothing is cleaned up in Stripe.
const stripeSecretKeyEnv = "OPENMETER_TEST_STRIPE_SECRET_KEY"

// TestSubscriptionZeroUsageInvoiceWithCredits reproduces a customer on a charge-automatically
// Stripe profile, with promotional credits and a credit-then-invoice subscription of several
// tiny per-unit prices, who has no usage at all for the first month. The expected outcome in
// OpenMeter is a $0 invoice; the test shows what that invoice looks like on the Stripe side.
func (s *StripeInvoiceTestSuite) TestSubscriptionZeroUsageInvoiceWithCredits() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("stripe-subscription-zero-usage-credits")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(t, ns)

	stripeSecretKey := os.Getenv(stripeSecretKeyEnv)
	realStripe := stripeSecretKey != ""

	start := datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime()
	periodEnd := start.AddDate(0, 1, 0)

	clock.FreezeTime(start)
	defer clock.UnFreeze()
	defer s.MockStreamingConnector.Reset()

	var (
		stripeApp        app.App
		cust             = s.createStripeLedgerBackedCustomer(ctx, ns, "zero-usage-subject")
		stripeCustomerID string
		subsView         subscription.SubscriptionView
		invoice          billing.StandardInvoice
	)

	s.Run("given a charge-automatically stripe profile and a customer falling back to the stripe default payment method", func() {
		apiKey := TestStripeAPIKey
		if realStripe {
			apiKey = stripeSecretKey
		}

		var err error
		stripeApp, err = s.Fixture.setupAppWithAPIKey(ctx, ns, apiKey)
		s.Require().NoError(err)

		s.ProvisionBillingProfile(ctx, ns, stripeApp.GetID())

		if realStripe {
			stripeCustomerID = s.createRealStripeCustomerWithDefaultCard(stripeSecretKey, cust.Name)

			s.Require().NoError(stripeApp.UpsertCustomerData(ctx, app.UpsertAppInstanceCustomerDataInput{
				CustomerID: cust.GetID(),
				Data:       appstripe.CustomerData{StripeCustomerID: stripeCustomerID},
			}))

			return
		}

		customerData, err := s.Fixture.setupAppCustomerData(ctx, stripeApp, cust)
		s.Require().NoError(err)
		stripeCustomerID = customerData.StripeCustomerID

		s.StripeAppClient.
			On("GetCustomer", stripeCustomerID).
			Return(stripeclient.StripeCustomer{
				StripeCustomerID: stripeCustomerID,
				DefaultPaymentMethod: &stripeclient.StripePaymentMethod{
					ID: "pm_stripe_default",
					BillingAddress: &models.Address{
						Country:    lo.ToPtr(models.CountryCode("US")),
						PostalCode: lo.ToPtr("94103"),
					},
				},
			}, nil)

		s.StripeAppClient.
			On("CreateInvoice", mock.Anything).
			Once().
			Return(&stripe.Invoice{
				ID:       "in_zero_usage",
				Customer: &stripe.Customer{ID: stripeCustomerID},
				Currency: "usd",
				Lines:    &stripe.InvoiceLineItemList{Data: []*stripe.InvoiceLineItem{}},
			}, nil)

		s.StripeAppClient.
			On("ListInvoiceLineItems", "in_zero_usage").
			Maybe().
			Return([]*stripe.InvoiceLineItem{}, nil)

		s.StripeAppClient.
			On("UpdateInvoice", mock.Anything).
			Maybe().
			Return(&stripe.Invoice{
				ID:       "in_zero_usage",
				Customer: &stripe.Customer{ID: stripeCustomerID},
				Currency: "usd",
				Lines:    &stripe.InvoiceLineItemList{Data: []*stripe.InvoiceLineItem{}},
			}, nil)

		// Stripe finalizes a $0 charge-automatically invoice straight into paid and creates no
		// PaymentIntent for it.
		s.StripeAppClient.
			On("FinalizeInvoice", stripeclient.FinalizeInvoiceInput{
				AutoAdvance:     true,
				StripeInvoiceID: "in_zero_usage",
			}).
			Once().
			Return(&stripe.Invoice{
				ID:        "in_zero_usage",
				Customer:  &stripe.Customer{ID: stripeCustomerID},
				Currency:  "usd",
				Number:    "ZERO-0001",
				Status:    stripe.InvoiceStatusPaid,
				AmountDue: 0,
				Total:     0,
				Lines:     &stripe.InvoiceLineItemList{Data: []*stripe.InvoiceLineItem{}},
			}, nil)
	})

	s.Run("given 300 USD of promotional credits", func() {
		_, err := s.CreditGrant.Create(ctx, creditgrant.CreateInput{
			Namespace:     ns,
			CustomerID:    cust.ID,
			Name:          "Promotional credits",
			Currency:      currencyx.Code(currency.USD),
			Amount:        alpacadecimal.NewFromInt(300),
			FundingMethod: creditgrant.FundingMethodNone,
		})
		s.Require().NoError(err)

		s.Equal(float64(300), s.customerFBOBalance(cust).InexactFloat64())
	})

	s.Run("given a credit-then-invoice subscription with tiny per-unit prices and no usage", func() {
		features := s.setupZeroUsageMeteredFeatures(t, ns, start, []string{"tokens-input", "tokens-output", "embeddings", "images"})

		unitPrices := []string{"0.00007778", "0.00001234", "0.00000556", "0.00003333"}
		rateCards := make(productcatalog.RateCards, 0, len(features))
		for i, f := range features {
			rateCards = append(rateCards, &productcatalog.UsageBasedRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:     f.Key,
					Name:    f.Key,
					Feature: productcatalog.NewFeatureReference(lo.ToPtr(f.ID), lo.ToPtr(f.Key)),
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.RequireFromString(unitPrices[i]),
					}),
				},
				BillingCadence: datetime.MustParseDuration(t, "P1M"),
			})
		}

		p, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
			NamespacedModel: models.NamespacedModel{Namespace: ns},
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Name:           "Pay as you go",
					Key:            "pay-as-you-go",
					Version:        1,
					Currency:       currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					BillingCadence: datetime.MustParseDuration(t, "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
				Phases: []productcatalog.Phase{
					{
						PhaseMeta: productcatalog.PhaseMeta{Name: "default", Key: "default"},
						RateCards: rateCards,
					},
				},
			},
		})
		s.Require().NoError(err)

		subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, ns, productcatalogsubscription.PlanRefInput{
			Key:     p.Key,
			Version: lo.ToPtr(1),
		})
		s.Require().NoError(err)

		subsView, err = s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{Custom: lo.ToPtr(start)},
				Name:   "zero-usage",
			},
			Namespace:  ns,
			CustomerID: cust.ID,
		}, subscriptionPlan)
		s.Require().NoError(err)

		s.Require().NoError(s.SubscriptionSync.SyncByView(ctx, subsView, start.Add(time.Minute)))

		gathering, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespace: ns,
			Customers: []string{cust.ID},
			Expand:    billing.NewGatheringInvoiceExpands(billing.GatheringInvoiceExpandLines),
		})
		s.Require().NoError(err)
		s.Require().Len(gathering.Items, 1)
		s.Len(gathering.Items[0].Lines.OrEmpty(), len(features))
	})

	s.Run("when the billing period ends and the invoice is advanced", func() {
		clock.FreezeTime(periodEnd)
		s.Require().NoError(s.SubscriptionSync.SyncByView(ctx, subsView, periodEnd))

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(periodEnd),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		invoice = invoices[0]

		invoice = s.advanceStandardInvoiceUntilSettled(ctx, invoice)
	})

	s.Run("then openmeter settles the zero total invoice without creating one in stripe", func() {
		s.Len(invoice.Lines.OrEmpty(), 4)
		for _, line := range invoice.Lines.OrEmpty() {
			s.Equal(float64(0), line.GetQuantity().InexactFloat64(), line.Name)
			s.Equal(float64(0), line.Totals.Total.InexactFloat64(), line.Name)
			s.Equal(float64(0), line.Totals.CreditsTotal.InexactFloat64(), line.Name)
		}

		s.RequireTotals(billingtest.ExpectedTotals{Amount: 0, CreditsTotal: 0, Total: 0}, invoice.Totals)
		s.Empty(invoice.ValidationIssues)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		s.Equal(float64(300), s.customerFBOBalance(cust).InexactFloat64())

		// The Stripe id is written only from a successful CreateInvoice during sync, so a stored
		// one is proof the zero total invoice reached Stripe.
		s.Empty(invoice.ExternalIDs.Invoicing, "zero total invoice must not be created in stripe for customer %s", stripeCustomerID)
	})
}

// advanceStandardInvoiceUntilSettled drives an auto-advancing invoice through collection, the
// draft period and issuing the way the worker would, moving the frozen clock past each wait.
func (s *StripeInvoiceTestSuite) advanceStandardInvoiceUntilSettled(ctx context.Context, invoice billing.StandardInvoice) billing.StandardInvoice {
	s.T().Helper()

	for range 6 {
		s.T().Logf("invoice %s status %s", invoice.ID, invoice.Status)

		switch invoice.Status {
		case billing.StandardInvoiceStatusDraftWaitingForCollection:
			clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice().Add(time.Second))
		case billing.StandardInvoiceStatusDraftWaitingAutoApproval:
			s.Require().NotNil(invoice.DraftUntil)
			clock.FreezeTime(invoice.DraftUntil.Add(time.Second))
		case billing.StandardInvoiceStatusPaymentProcessingPending, billing.StandardInvoiceStatusPaid:
			return invoice
		}

		advanced, err := s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)

		if advanced.Status == invoice.Status {
			s.Require().Failf("invoice did not advance", "status stuck at %s with issues %v", invoice.Status, invoice.ValidationIssues)
		}

		invoice = advanced
	}

	s.Require().Failf("invoice did not settle", "last status %s with issues %v", invoice.Status, invoice.ValidationIssues)

	return invoice
}

func (s *StripeInvoiceTestSuite) customerFBOBalance(cust *customer.Customer) alpacadecimal.Decimal {
	s.T().Helper()
	ctx := s.T().Context()

	accounts, err := s.LedgerResolver.GetCustomerAccounts(ctx, cust.GetID())
	s.Require().NoError(err)

	balance, err := s.BalanceQuerier.GetAccountBalance(ctx, accounts.FBOAccount, ledger.RouteFilter{
		Currency:       currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
		CostBasis:      mo.None[*alpacadecimal.Decimal](),
		CreditPriority: lo.ToPtr(ledger.DefaultCustomerFBOPriority),
	}, ledger.BalanceQuery{})
	s.Require().NoError(err)

	return balance
}

// setupZeroUsageMeteredFeatures registers one sum meter per key and seeds each with a single
// zero event before the period, so the streaming mock knows the meters but reports no usage.
//
// t must be the top-level test: the meters are removed in its cleanup, and s.T() inside an
// s.Run subtest would tear them down as soon as that subtest returns.
func (s *StripeInvoiceTestSuite) setupZeroUsageMeteredFeatures(t *testing.T, ns string, before time.Time, keys []string) []feature.Feature {
	t.Helper()
	ctx := t.Context()

	meters := make([]meter.Meter, 0, len(keys))
	for _, key := range keys {
		meters = append(meters, meter.Meter{
			ManagedResource: models.ManagedResource{
				ID:              ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{Namespace: ns},
				ManagedModel:    models.ManagedModel{CreatedAt: before, UpdatedAt: before},
				Name:            key,
			},
			Key:           key,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		})
	}

	s.Require().NoError(s.MeterAdapter.ReplaceMeters(ctx, meters))
	t.Cleanup(func() {
		s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}))
	})

	features := make([]feature.Feature, 0, len(meters))
	for _, m := range meters {
		s.MockStreamingConnector.AddSimpleEvent(m.Key, 0, before.Add(-time.Minute))

		f, err := s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: ns,
			Name:      m.Key,
			Key:       m.Key,
			MeterID:   lo.ToPtr(m.ID),
		})
		s.Require().NoError(err)

		features = append(features, f)
	}

	return features
}

// createRealStripeCustomerWithDefaultCard provisions a test-mode Stripe customer whose invoice
// default payment method is a Visa test card with a US billing address, matching what the
// charge-automatically validation expects from a production customer.
func (s *StripeInvoiceTestSuite) createRealStripeCustomerWithDefaultCard(secretKey string, name string) string {
	s.T().Helper()

	sc := &stripego.API{}
	sc.Init(secretKey, nil)

	address := &stripe.AddressParams{
		Line1:      lo.ToPtr("123 Market St"),
		City:       lo.ToPtr("San Francisco"),
		State:      lo.ToPtr("CA"),
		PostalCode: lo.ToPtr("94103"),
		Country:    lo.ToPtr("US"),
	}

	stripeCustomer, err := sc.Customers.New(&stripe.CustomerParams{
		Name:    lo.ToPtr(name + " (openmeter zero usage test)"),
		Email:   lo.ToPtr("zero-usage@openmeter.test"),
		Address: address,
	})
	s.Require().NoError(err)

	paymentMethod, err := sc.PaymentMethods.New(&stripe.PaymentMethodParams{
		Type: lo.ToPtr(string(stripe.PaymentMethodTypeCard)),
		Card: &stripe.PaymentMethodCardParams{Token: lo.ToPtr("tok_visa")},
		BillingDetails: &stripe.PaymentMethodBillingDetailsParams{
			Name:    lo.ToPtr(name),
			Email:   lo.ToPtr("zero-usage@openmeter.test"),
			Address: address,
		},
	})
	s.Require().NoError(err)

	_, err = sc.PaymentMethods.Attach(paymentMethod.ID, &stripe.PaymentMethodAttachParams{
		Customer: lo.ToPtr(stripeCustomer.ID),
	})
	s.Require().NoError(err)

	_, err = sc.Customers.Update(stripeCustomer.ID, &stripe.CustomerParams{
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: lo.ToPtr(paymentMethod.ID),
		},
	})
	s.Require().NoError(err)

	return stripeCustomer.ID
}
