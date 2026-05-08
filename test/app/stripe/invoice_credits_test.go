package appstripe

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *StripeInvoiceTestSuite) TestUsageBasedCreditThenInvoiceProgressiveBillingCreditAllocation() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("stripe-credits-usagebased-progressive-credit-then-invoice")

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	midPeriodInvoiceAt := datetime.MustParseTimeInLocation(t, "2026-01-16T00:00:00Z", time.UTC).AsTime()
	costBasis := alpacadecimal.NewFromInt(1)

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()
	defer s.MockStreamingConnector.Reset()

	var (
		partialInvoice billing.StandardInvoice
		finalInvoice   billing.StandardInvoice
	)

	customInvoicing := s.SetupCustomInvoicing(ns)

	stripeApp, err := s.Fixture.setupApp(ctx, ns)
	s.NoError(err)
	stripeInvoicingApp, err := billing.GetApp(stripeApp)
	s.NoError(err)

	cust := s.createStripeLedgerBackedCustomer(ctx, ns, "test-subject")
	customerData, err := s.Fixture.setupAppCustomerData(ctx, stripeApp, cust)
	s.NoError(err)

	s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	t.Run("given settled purchased credits and a credit-then-invoice usage charge", func(t *testing.T) {
		creditPurchaseRes, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
					customer:      cust.GetID(),
					currency:      currencyx.Code("USD"),
					amount:        alpacadecimal.NewFromInt(7),
					servicePeriod: timeutil.ClosedPeriod{From: setupAt, To: setupAt},
					settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
						GenericSettlement: creditpurchase.GenericSettlement{
							Currency:  currencyx.Code("USD"),
							CostBasis: costBasis,
						},
						InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
					}),
				}),
			},
		})
		s.NoError(err)
		s.Len(creditPurchaseRes, 1)

		creditPurchaseCharge, err := creditPurchaseRes[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.mustSettleExternalCreditPurchase(ctx, creditPurchaseCharge.GetChargeID())

		usageChargeRes, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       currencyx.Code("USD"),
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					name:              "usage-based-progressive-credit-then-invoice",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "usage-based-progressive-credit-then-invoice",
					featureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(usageChargeRes, 1)

		usageBasedCharge, err := usageChargeRes[0].AsUsageBasedCharge()
		s.NoError(err)
		s.Equal(usagebased.RatingEngineDelta, usageBasedCharge.State.RatingEngine)
	})

	t.Run("when the first progressive invoice is synced to stripe with credits", func(t *testing.T) {
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-10T00:00:00Z", time.UTC).AsTime(),
		)

		clock.FreezeTime(midPeriodInvoiceAt)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(midPeriodInvoiceAt),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		partialInvoice = invoices[0]

		clock.FreezeTime(partialInvoice.DefaultCollectionAtForStandardInvoice())
		partialInvoice, err = s.BillingService.AdvanceInvoice(ctx, partialInvoice.GetInvoiceID())
		s.NoError(err)
		s.Len(partialInvoice.Lines.OrEmpty(), 1)

		partialLine := partialInvoice.Lines.OrEmpty()[0]
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, partialLine.Totals)
		s.Equal(float64(5), partialLine.CreditsApplied.SumAmount(lo.Must(currencyx.Code("USD").Calculator())).InexactFloat64())

		s.expectStripeInvoiceCreate(stripeApp.GetID(), cust.GetID(), partialInvoice.ID, customerData.StripeCustomerID, "stripe-partial-invoice-id")
		s.expectStripeInvoiceAddLines("stripe-partial-invoice-id", []expectedStripeInvoiceItem{
			{
				Amount:      500,
				Description: "usage-based-progressive-credit-then-invoice: usage in period (5 x $1)",
				Type:        "line",
			},
			{
				Amount:      -500,
				Description: "credits applied for usage-based-progressive-credit-then-invoice: usage in period",
				Type:        "credit",
			},
		})

		stripePartialInvoice := lo.Must(partialInvoice.RemoveCircularReferences())
		upsertResult, err := stripeInvoicingApp.UpsertStandardInvoice(ctx, stripePartialInvoice)
		s.NoError(err)
		externalID, ok := upsertResult.GetExternalID()
		s.True(ok)
		s.Equal("stripe-partial-invoice-id", externalID)
		s.StripeAppClient.AssertExpectations(t)

		partialInvoice, err = s.BillingService.ApproveInvoice(ctx, partialInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, partialInvoice.Status)

		partialInvoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: partialInvoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, partialInvoice.Status)
	})

	t.Run("when the final invoice is synced to stripe with remaining credits", func(t *testing.T) {
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			15,
			datetime.MustParseTimeInLocation(t, "2026-01-25T00:00:00Z", time.UTC).AsTime(),
		)

		clock.FreezeTime(servicePeriod.To)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		finalInvoice = invoices[0]

		clock.FreezeTime(finalInvoice.DefaultCollectionAtForStandardInvoice())
		finalInvoice, err = s.BillingService.AdvanceInvoice(ctx, finalInvoice.GetInvoiceID())
		s.NoError(err)
		s.Len(finalInvoice.Lines.OrEmpty(), 1)

		finalLine := finalInvoice.Lines.OrEmpty()[0]
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       15,
			CreditsTotal: 2,
			Total:        13,
		}, finalLine.Totals)
		s.Equal(float64(2), finalLine.CreditsApplied.SumAmount(lo.Must(currencyx.Code("USD").Calculator())).InexactFloat64())

		s.expectStripeInvoiceCreate(stripeApp.GetID(), cust.GetID(), finalInvoice.ID, customerData.StripeCustomerID, "stripe-final-invoice-id")
		s.expectStripeInvoiceAddLines("stripe-final-invoice-id", []expectedStripeInvoiceItem{
			{
				Amount:      1500,
				Description: "usage-based-progressive-credit-then-invoice: usage in period (15 x $1)",
				Type:        "line",
			},
			{
				Amount:      -200,
				Description: "credits applied for usage-based-progressive-credit-then-invoice: usage in period",
				Type:        "credit",
			},
		})

		stripeFinalInvoice := lo.Must(finalInvoice.RemoveCircularReferences())
		upsertResult, err := stripeInvoicingApp.UpsertStandardInvoice(ctx, stripeFinalInvoice)
		s.NoError(err)
		externalID, ok := upsertResult.GetExternalID()
		s.True(ok)
		s.Equal("stripe-final-invoice-id", externalID)
		s.StripeAppClient.AssertExpectations(t)
	})
}

type expectedStripeInvoiceItem struct {
	Amount      int64
	Description string
	Type        string
}

func (s *StripeInvoiceTestSuite) expectStripeInvoiceCreate(appID app.AppID, customerID customer.CustomerID, invoiceID string, stripeCustomerID string, stripeInvoiceID string) {
	s.StripeAppClient.
		On("CreateInvoice", stripeclient.CreateInvoiceInput{
			AppID:               appID,
			CustomerID:          customerID,
			InvoiceID:           invoiceID,
			AutomaticTaxEnabled: true,
			CollectionMethod:    billing.CollectionMethodChargeAutomatically,
			Currency:            currencyx.Code("USD"),
			StripeCustomerID:    stripeCustomerID,
		}).
		Once().
		Return(&stripe.Invoice{
			ID:       stripeInvoiceID,
			Customer: &stripe.Customer{ID: stripeCustomerID},
			Currency: "USD",
			Lines:    &stripe.InvoiceLineItemList{Data: []*stripe.InvoiceLineItem{}},
		}, nil)
}

func (s *StripeInvoiceTestSuite) expectStripeInvoiceAddLines(stripeInvoiceID string, expectedItems []expectedStripeInvoiceItem) {
	s.StripeAppClient.
		On("AddInvoiceLines", mock.MatchedBy(func(input stripeclient.AddInvoiceLinesInput) bool {
			if input.StripeInvoiceID != stripeInvoiceID {
				return false
			}

			if len(input.Lines) != len(expectedItems) {
				return false
			}

			expectedByDescription := lo.KeyBy(expectedItems, func(item expectedStripeInvoiceItem) string {
				return item.Description
			})

			for _, line := range input.Lines {
				if line.Description == nil || line.Amount == nil {
					return false
				}

				expected, ok := expectedByDescription[*line.Description]
				if !ok {
					return false
				}

				if *line.Amount != expected.Amount {
					return false
				}

				if line.Metadata["om_line_type"] != expected.Type {
					return false
				}

				if line.Metadata["om_line_id"] == "" {
					return false
				}
			}

			return true
		})).
		Once().
		Return([]stripeclient.StripeInvoiceItemWithLineID{}, nil)
}

type createMockChargeIntentInput struct {
	customer          customer.CustomerID
	currency          currencyx.Code
	servicePeriod     timeutil.ClosedPeriod
	price             *productcatalog.Price
	featureKey        string
	name              string
	settlementMode    productcatalog.SettlementMode
	managedBy         billing.InvoiceLineManagedBy
	uniqueReferenceID string
}

func (s *StripeInvoiceTestSuite) createMockChargeIntent(input createMockChargeIntentInput) charges.ChargeIntent {
	s.T().Helper()
	s.Require().NotNil(input.price)
	s.Require().NoError(input.customer.Validate())
	s.Require().NotEmpty(input.currency)

	return charges.NewChargeIntent(usagebased.Intent{
		Intent: meta.Intent{
			Name:              input.name,
			ManagedBy:         input.managedBy,
			ServicePeriod:     input.servicePeriod,
			FullServicePeriod: input.servicePeriod,
			BillingPeriod:     input.servicePeriod,
			UniqueReferenceID: lo.EmptyableToPtr(input.uniqueReferenceID),
			CustomerID:        input.customer.ID,
			Currency:          input.currency,
		},
		Price:          *input.price,
		InvoiceAt:      input.servicePeriod.To,
		SettlementMode: lo.CoalesceOrEmpty(input.settlementMode, productcatalog.InvoiceOnlySettlementMode),
		FeatureKey:     input.featureKey,
	})
}

type createCreditPurchaseIntentInput struct {
	customer      customer.CustomerID
	currency      currencyx.Code
	amount        alpacadecimal.Decimal
	servicePeriod timeutil.ClosedPeriod
	settlement    creditpurchase.Settlement
}

func (s *StripeInvoiceTestSuite) createCreditPurchaseIntent(input createCreditPurchaseIntentInput) charges.ChargeIntent {
	s.T().Helper()
	s.Require().NoError(input.customer.Validate())
	s.Require().NotEmpty(input.currency)
	s.Require().True(input.amount.IsPositive())
	s.Require().NoError(input.servicePeriod.Validate())
	s.Require().NoError(input.settlement.Validate())

	return charges.NewChargeIntent(creditpurchase.Intent{
		Intent: meta.Intent{
			Name:              "Credit Purchase",
			ManagedBy:         billing.ManuallyManagedLine,
			CustomerID:        input.customer.ID,
			Currency:          input.currency,
			ServicePeriod:     input.servicePeriod,
			BillingPeriod:     input.servicePeriod,
			FullServicePeriod: input.servicePeriod,
		},
		CreditAmount: input.amount,
		Settlement:   input.settlement,
	})
}

func (s *StripeInvoiceTestSuite) createStripeLedgerBackedCustomer(ctx context.Context, ns string, subjectKey string) *customer.Customer {
	s.T().Helper()

	_, err := s.LedgerResolver.EnsureBusinessAccounts(ctx, ns)
	s.NoError(err)

	cust := s.CreateTestCustomer(ns, subjectKey)
	_, err = s.LedgerResolver.CreateCustomerAccounts(ctx, cust.GetID())
	s.NoError(err)

	return cust
}

func (s *StripeInvoiceTestSuite) mustSettleExternalCreditPurchase(ctx context.Context, chargeID meta.ChargeID) {
	s.T().Helper()

	updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
		ChargeID:           chargeID,
		TargetPaymentState: payment.StatusAuthorized,
	})
	s.NoError(err)
	s.Equal(payment.StatusAuthorized, updatedCharge.Realizations.ExternalPaymentSettlement.Status)

	updatedCharge, err = s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
		ChargeID:           chargeID,
		TargetPaymentState: payment.StatusSettled,
	})
	s.NoError(err)
	s.Equal(payment.StatusSettled, updatedCharge.Realizations.ExternalPaymentSettlement.Status)
}
