package credits

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/suite"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgerresolvers "github.com/openmeterio/openmeter/openmeter/ledger/resolvers"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

const USD = currencyx.Code(currency.USD)

type CreditsTestSuite struct {
	billingtest.BaseSuite

	Charges              charges.Service
	Ledger               ledger.Ledger
	LedgerAccountService ledgeraccount.Service
	LedgerResolver       *ledgerresolvers.AccountResolver
}

func TestCreditsTestSuite(t *testing.T) {
	suite.Run(t, new(CreditsTestSuite))
}

func (s *CreditsTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	logger := omtestutils.NewLogger(s.T())

	deps, err := ledgertestutils.InitDeps(s.DBClient, logger)
	s.NoError(err)

	s.Ledger = deps.HistoricalLedger
	s.LedgerAccountService = deps.AccountService
	s.LedgerResolver = deps.ResolversService

	stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{
		Client:                s.DBClient,
		Logger:                logger,
		BillingService:        s.BillingService,
		FeatureService:        s.FeatureService,
		StreamingConnector:    s.MockStreamingConnector,
		FlatFeeHandler:        ledgerchargeadapter.NewFlatFeeHandler(deps.HistoricalLedger, deps.ResolversService, deps.AccountService),
		CreditPurchaseHandler: ledgerchargeadapter.NewCreditPurchaseHandler(deps.HistoricalLedger, deps.ResolversService, deps.AccountService),
		UsageBasedHandler:     usagebased.UnimplementedHandler{},
	})
	s.NoError(err)
	s.Charges = stack.ChargesService
}

func (s *CreditsTestSuite) TestFlatFeeCreditThenInvoiceSanity() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-sanity-test")

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.createLedgerBackedCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	const (
		flatFeeName = "flat-fee"
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()

	clock.SetTime(setupAt)

	s.Run("the customer receives a promotional credit grant", func() {
		intent := s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(30),
			servicePeriod: timeutil.ClosedPeriod{
				From: setupAt,
				To:   setupAt,
			},
			settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		})

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())
		cpCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)

		// This should match the ledger's transaction group ID
		s.NotEmpty(cpCharge.State.CreditGrantRealization.TransactionGroupID)

		// LEDGER[galexi]:
		// - OnPromotionalCreditPurchase is called
		// - At this point the customer must have 30 USD promotional credits

		// Validate balances
		zeroCostBasis := alpacadecimal.Zero
		purchasedCostBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(float64(30), s.mustCustomerFBOBalance(cust.GetID(), USD, &zeroCostBasis).InexactFloat64())
		s.Equal(float64(0), s.mustCustomerFBOBalance(cust.GetID(), USD, &purchasedCostBasis).InexactFloat64())
	})

	var externalCreditPurchaseChargeID meta.ChargeID
	s.Run("and customer purchases 50 USD credits as 0.5 costbasis", func() {
		intent := s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(50),
			servicePeriod: timeutil.ClosedPeriod{
				From: setupAt,
				To:   setupAt,
			},
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  USD,
					CostBasis: alpacadecimal.NewFromFloat(0.5),
				},
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		})

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())
		cpCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)

		// This should match the ledger's transaction group ID
		s.NotEmpty(cpCharge.State.CreditGrantRealization.TransactionGroupID)

		// LEDGER[galexi]:
		// - OnCreditPurchaseInitiated is called
		// - At this point the customer must have 50 USD credits cost basis of 0.5

		// Validate balances
		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(float64(50), s.mustCustomerFBOBalance(cust.GetID(), USD, &costBasis).InexactFloat64())
		s.Equal(float64(-50), s.mustCustomerReceivableBalance(cust.GetID(), USD, &costBasis).InexactFloat64())

		externalCreditPurchaseChargeID = cpCharge.GetChargeID()
	})

	s.Run("the customer pays for the credit purchase - authorized", func() {
		updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           externalCreditPurchaseChargeID,
			TargetPaymentState: payment.StatusAuthorized,
		})
		s.NoError(err)

		// LEDGER[galexi]:
		// - OnCreditPurchasePaymentAuthorized is called

		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(payment.StatusAuthorized, updatedCharge.State.ExternalPaymentSettlement.Status)
		s.Equal(float64(-50), s.mustCustomerReceivableBalance(cust.GetID(), USD, &costBasis).InexactFloat64())
	})

	s.Run("the customer settles the credit purchase payment", func() {
		updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           externalCreditPurchaseChargeID,
			TargetPaymentState: payment.StatusSettled,
		})
		s.NoError(err)

		// LEDGER[galexi]:
		// - OnCreditPurchasePaymentSettled is called

		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(payment.StatusSettled, updatedCharge.State.ExternalPaymentSettlement.Status)
		s.Equal(float64(0), s.mustCustomerReceivableBalance(cust.GetID(), USD, &costBasis).InexactFloat64())
	})

	// TOTAL credits balance: 30 + 50 = 80 USD

	var flatFeeChargeID meta.ChargeID
	promoCostBasis := alpacadecimal.Zero
	externalCostBasis := alpacadecimal.NewFromFloat(0.5)
	type flatFeeLedgerSnapshot struct {
		promoFBO             alpacadecimal.Decimal
		externalFBO          alpacadecimal.Decimal
		promoReceivable      alpacadecimal.Decimal
		externalReceivable   alpacadecimal.Decimal
		totalOpenReceivable  alpacadecimal.Decimal
		accrued              alpacadecimal.Decimal
		authorizedReceivable alpacadecimal.Decimal
		totalWash            alpacadecimal.Decimal
		externalWash         alpacadecimal.Decimal
		earnings             alpacadecimal.Decimal
	}
	flatFeeStart := flatFeeLedgerSnapshot{
		promoFBO:             s.mustCustomerFBOBalance(cust.GetID(), USD, &promoCostBasis),
		externalFBO:          s.mustCustomerFBOBalance(cust.GetID(), USD, &externalCostBasis),
		promoReceivable:      s.mustCustomerReceivableBalance(cust.GetID(), USD, &promoCostBasis),
		externalReceivable:   s.mustCustomerReceivableBalance(cust.GetID(), USD, &externalCostBasis),
		totalOpenReceivable:  s.mustCustomerReceivableBalance(cust.GetID(), USD, nil),
		accrued:              s.mustCustomerAccruedBalance(cust.GetID(), USD),
		authorizedReceivable: s.mustCustomerAuthorizedReceivableBalance(cust.GetID(), USD, nil),
		totalWash:            s.mustWashBalance(ns, USD, nil),
		externalWash:         s.mustWashBalance(ns, USD, &externalCostBasis),
		earnings:             s.mustEarningsBalance(ns, USD),
	}
	assertDelta := func(label string, start, delta, actual alpacadecimal.Decimal) {
		s.T().Helper()
		expected := start.Add(delta)
		s.True(actual.Equal(expected), "%s: expected start %s + delta %s = %s, got %s", label, start.String(), delta.String(), expected.String(), actual.String())
	}

	s.Run("create new upcoming charge for flat fee", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              flatFeeName,
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: flatFeeName,
				}),
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(res[0].Type(), meta.ChargeTypeFlatFee)
		flatFeeCharge, err := res[0].AsFlatFeeCharge()
		s.NoError(err)

		// LEDGER[galexi]:
		// - This is a noop as this is in the future, so it just creates the charge + gathering line

		flatFeeChargeID = flatFeeCharge.GetChargeID()
	})

	var stdInvoiceID billing.InvoiceID
	var stdLineID billing.LineID
	clock.SetTime(servicePeriod.From)
	s.Run("invoice the charge", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.From),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice := invoices[0]
		s.DebugDumpStandardInvoice("invoice after invoice pending lines", invoice)

		s.Len(invoice.Lines.OrEmpty(), 1)
		stdLine := invoice.Lines.OrEmpty()[0]

		s.Equal(flatFeeChargeID.ID, *stdLine.ChargeID)
		stdLineID = stdLine.GetLineID()

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		s.Equal(flatFeeChargeID.ID, updatedFlatFeeCharge.ID)

		// LEDGER[galexi]:
		// - OnFlatFeeAssignedToInvoice is called with the pre tax total amount of USD 100
		// - Two credit realizations should happen for the two different credit types

		// Validate the credit realizations
		// The charge should have $80 realized as credits
		s.Len(updatedFlatFeeCharge.State.CreditRealizations, 2)
		promotionalCreditRealization := updatedFlatFeeCharge.State.CreditRealizations[0]
		s.Equal(float64(30), promotionalCreditRealization.Amount.InexactFloat64())

		customerCreditRealization := updatedFlatFeeCharge.State.CreditRealizations[1]
		s.Equal(float64(50), customerCreditRealization.Amount.InexactFloat64())

		assertDelta("promo FBO after invoice assignment", flatFeeStart.promoFBO, alpacadecimal.NewFromInt(-30), s.mustCustomerFBOBalance(cust.GetID(), USD, &promoCostBasis))
		assertDelta("external FBO after invoice assignment", flatFeeStart.externalFBO, alpacadecimal.NewFromInt(-50), s.mustCustomerFBOBalance(cust.GetID(), USD, &externalCostBasis))
		assertDelta("promo receivable after invoice assignment", flatFeeStart.promoReceivable, alpacadecimal.Zero, s.mustCustomerReceivableBalance(cust.GetID(), USD, &promoCostBasis))
		assertDelta("external receivable after invoice assignment", flatFeeStart.externalReceivable, alpacadecimal.Zero, s.mustCustomerReceivableBalance(cust.GetID(), USD, &externalCostBasis))
		assertDelta("total open receivable after invoice assignment", flatFeeStart.totalOpenReceivable, alpacadecimal.Zero, s.mustCustomerReceivableBalance(cust.GetID(), USD, nil))
		assertDelta("accrued after invoice assignment", flatFeeStart.accrued, alpacadecimal.NewFromInt(80), s.mustCustomerAccruedBalance(cust.GetID(), USD))
		assertDelta("authorized receivable after invoice assignment", flatFeeStart.authorizedReceivable, alpacadecimal.Zero, s.mustCustomerAuthorizedReceivableBalance(cust.GetID(), USD, nil))
		assertDelta("total wash after invoice assignment", flatFeeStart.totalWash, alpacadecimal.Zero, s.mustWashBalance(ns, USD, nil))
		assertDelta("external wash after invoice assignment", flatFeeStart.externalWash, alpacadecimal.Zero, s.mustWashBalance(ns, USD, &externalCostBasis))
		assertDelta("earnings after invoice assignment", flatFeeStart.earnings, alpacadecimal.Zero, s.mustEarningsBalance(ns, USD))

		stdInvoiceID = invoice.GetInvoiceID()
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	})

	s.Run("advance the invoice and authorize payment", func() {
		invoice, err := s.BillingService.ApproveInvoice(ctx, stdInvoiceID)
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		// LEDGER[galexi]:
		// - OnFlatFeeStandardInvoiceUsageAccrued is called with the service period and totals of USD 20 to be represented
		//   on the ledger
		// - OnFlatFeePaymentAuthorized is called (I cannot make this a two step process without creating a new app) with the USD 20

		// Invoice usage accrued callback should have been invoked
		accruedUsage := updatedFlatFeeCharge.State.AccruedUsage
		s.NotNil(accruedUsage)
		s.Equal(servicePeriod, accruedUsage.ServicePeriod, "service period should be the same as the input")
		s.False(accruedUsage.Mutable, "accrued usage should not be mutable")
		s.NotNil(accruedUsage.LineID, "line ID should be set")
		s.Equal(stdLineID.ID, *accruedUsage.LineID, "line ID should be the same as the standard line")
		s.Equal(float64(20), accruedUsage.Totals.Total.InexactFloat64(), "totals should be the same as the input")
		s.Equal(float64(80), accruedUsage.Totals.CreditsTotal.InexactFloat64(), "totals should be the same as the input")

		assertDelta("promo FBO after payment authorization", flatFeeStart.promoFBO, alpacadecimal.NewFromInt(-30), s.mustCustomerFBOBalance(cust.GetID(), USD, &promoCostBasis))
		assertDelta("external FBO after payment authorization", flatFeeStart.externalFBO, alpacadecimal.NewFromInt(-50), s.mustCustomerFBOBalance(cust.GetID(), USD, &externalCostBasis))
		assertDelta("promo receivable after payment authorization", flatFeeStart.promoReceivable, alpacadecimal.Zero, s.mustCustomerReceivableBalance(cust.GetID(), USD, &promoCostBasis))
		assertDelta("external receivable after payment authorization", flatFeeStart.externalReceivable, alpacadecimal.Zero, s.mustCustomerReceivableBalance(cust.GetID(), USD, &externalCostBasis))
		assertDelta("total open receivable after payment authorization", flatFeeStart.totalOpenReceivable, alpacadecimal.NewFromInt(-20), s.mustCustomerReceivableBalance(cust.GetID(), USD, nil))
		assertDelta("authorized receivable after payment authorization", flatFeeStart.authorizedReceivable, alpacadecimal.NewFromInt(20), s.mustCustomerAuthorizedReceivableBalance(cust.GetID(), USD, nil))
		assertDelta("accrued after payment authorization", flatFeeStart.accrued, alpacadecimal.NewFromInt(100), s.mustCustomerAccruedBalance(cust.GetID(), USD))
		assertDelta("total wash after payment authorization", flatFeeStart.totalWash, alpacadecimal.NewFromInt(-20), s.mustWashBalance(ns, USD, nil))
		assertDelta("external wash after payment authorization", flatFeeStart.externalWash, alpacadecimal.Zero, s.mustWashBalance(ns, USD, &externalCostBasis))
		assertDelta("earnings after payment authorization", flatFeeStart.earnings, alpacadecimal.Zero, s.mustEarningsBalance(ns, USD))
	})

	s.Run("payment is settled", func() {
		invoice, err := s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: stdInvoiceID,
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

		// LEDGER[galexi]:
		// - OnFlatFeePaymentSettled is called with the USD 20

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(meta.ChargeStatusFinal, updatedFlatFeeCharge.Status)

		assertDelta("promo receivable after payment settlement", flatFeeStart.promoReceivable, alpacadecimal.Zero, s.mustCustomerReceivableBalance(cust.GetID(), USD, &promoCostBasis))
		assertDelta("external receivable after payment settlement", flatFeeStart.externalReceivable, alpacadecimal.Zero, s.mustCustomerReceivableBalance(cust.GetID(), USD, &externalCostBasis))
		assertDelta("total open receivable after payment settlement", flatFeeStart.totalOpenReceivable, alpacadecimal.Zero, s.mustCustomerReceivableBalance(cust.GetID(), USD, nil))
		assertDelta("authorized receivable after payment settlement", flatFeeStart.authorizedReceivable, alpacadecimal.Zero, s.mustCustomerAuthorizedReceivableBalance(cust.GetID(), USD, nil))
		assertDelta("accrued after payment settlement", flatFeeStart.accrued, alpacadecimal.NewFromInt(100), s.mustCustomerAccruedBalance(cust.GetID(), USD))
		assertDelta("total wash after payment settlement", flatFeeStart.totalWash, alpacadecimal.NewFromInt(-20), s.mustWashBalance(ns, USD, nil))
		assertDelta("external wash after payment settlement", flatFeeStart.externalWash, alpacadecimal.Zero, s.mustWashBalance(ns, USD, &externalCostBasis))
		assertDelta("earnings after payment settlement", flatFeeStart.earnings, alpacadecimal.Zero, s.mustEarningsBalance(ns, USD))
	})
}

func (s *CreditsTestSuite) TestCreditPurchasePersistsPriority() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-creditpurchase-persists-priority")

	cust := s.createLedgerBackedCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	priority := 7
	at := datetime.MustParseTimeInLocation(s.T(), "2026-01-01T12:34:56Z", time.UTC).AsTime()

	intent := s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
		customer:      cust.GetID(),
		currency:      USD,
		amount:        alpacadecimal.NewFromInt(25),
		priority:      &priority,
		servicePeriod: timeutil.ClosedPeriod{From: at, To: at},
		settlement:    creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
	})

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: charges.ChargeIntents{
			intent,
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	cpCharge, err := res[0].AsCreditPurchaseCharge()
	s.NoError(err)
	s.NotNil(cpCharge.State.CreditGrantRealization)

	fetchedCharge, err := s.mustGetChargeByID(cpCharge.GetChargeID()).AsCreditPurchaseCharge()
	s.NoError(err)
	s.Equal(&priority, fetchedCharge.Intent.Priority)

	zeroCostBasis := alpacadecimal.Zero
	s.True(s.mustCustomerFBOBalanceWithPriority(cust.GetID(), USD, &zeroCostBasis, priority).Equal(alpacadecimal.NewFromInt(25)))
	s.True(s.mustCustomerFBOBalance(cust.GetID(), USD, &zeroCostBasis).Equal(alpacadecimal.Zero))
}

func (s *CreditsTestSuite) TestFlatFeeCreditOnlySanity() {
	ctx := context.Background()
	ns := s.GetUniqueNamespace("charges-sanity-test-credit-only")

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.createLedgerBackedCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	const (
		flatFeeName = "flat-fee-credit-only"
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	setupAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-31T00:00:00Z", time.UTC).AsTime()

	clock.SetTime(setupAt)

	s.Run("the customer receives a promotional credit grant", func() {
		intent := s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(30),
			servicePeriod: timeutil.ClosedPeriod{
				From: setupAt,
				To:   setupAt,
			},
			settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		})

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())
		cpCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.NotEmpty(cpCharge.State.CreditGrantRealization.TransactionGroupID)

		zeroCostBasis := alpacadecimal.Zero
		purchasedCostBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(float64(30), s.mustCustomerFBOBalance(cust.GetID(), USD, &zeroCostBasis).InexactFloat64())
		s.Equal(float64(0), s.mustCustomerFBOBalance(cust.GetID(), USD, &purchasedCostBasis).InexactFloat64())
	})

	var externalCreditPurchaseChargeID meta.ChargeID
	s.Run("and customer purchases 50 USD credits as 0.5 costbasis", func() {
		intent := s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromFloat(50),
			servicePeriod: timeutil.ClosedPeriod{
				From: setupAt,
				To:   setupAt,
			},
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  USD,
					CostBasis: alpacadecimal.NewFromFloat(0.5),
				},
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		})

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())
		cpCharge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.NotEmpty(cpCharge.State.CreditGrantRealization.TransactionGroupID)

		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(float64(50), s.mustCustomerFBOBalance(cust.GetID(), USD, &costBasis).InexactFloat64())
		s.Equal(float64(-50), s.mustCustomerReceivableBalance(cust.GetID(), USD, &costBasis).InexactFloat64())

		externalCreditPurchaseChargeID = cpCharge.GetChargeID()
	})

	s.Run("the customer pays for the credit purchase - authorized", func() {
		updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           externalCreditPurchaseChargeID,
			TargetPaymentState: payment.StatusAuthorized,
		})
		s.NoError(err)

		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(payment.StatusAuthorized, updatedCharge.State.ExternalPaymentSettlement.Status)
		s.Equal(float64(-50), s.mustCustomerReceivableBalance(cust.GetID(), USD, &costBasis).InexactFloat64())
	})

	s.Run("the customer settles the credit purchase payment", func() {
		updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           externalCreditPurchaseChargeID,
			TargetPaymentState: payment.StatusSettled,
		})
		s.NoError(err)

		costBasis := alpacadecimal.NewFromFloat(0.5)
		s.Equal(payment.StatusSettled, updatedCharge.State.ExternalPaymentSettlement.Status)
		s.Equal(float64(0), s.mustCustomerReceivableBalance(cust.GetID(), USD, &costBasis).InexactFloat64())
	})

	var flatFeeChargeID meta.ChargeID
	promoCostBasis := alpacadecimal.Zero
	externalCostBasis := alpacadecimal.NewFromFloat(0.5)
	type flatFeeLedgerSnapshot struct {
		promoFBO             alpacadecimal.Decimal
		externalFBO          alpacadecimal.Decimal
		unknownFBO           alpacadecimal.Decimal
		promoReceivable      alpacadecimal.Decimal
		externalReceivable   alpacadecimal.Decimal
		totalOpenReceivable  alpacadecimal.Decimal
		accrued              alpacadecimal.Decimal
		authorizedReceivable alpacadecimal.Decimal
		totalWash            alpacadecimal.Decimal
		externalWash         alpacadecimal.Decimal
		earnings             alpacadecimal.Decimal
	}
	flatFeeStart := flatFeeLedgerSnapshot{
		promoFBO:             s.mustCustomerFBOBalance(cust.GetID(), USD, &promoCostBasis),
		externalFBO:          s.mustCustomerFBOBalance(cust.GetID(), USD, &externalCostBasis),
		unknownFBO:           s.mustCustomerFBOBalance(cust.GetID(), USD, nil),
		promoReceivable:      s.mustCustomerReceivableBalance(cust.GetID(), USD, &promoCostBasis),
		externalReceivable:   s.mustCustomerReceivableBalance(cust.GetID(), USD, &externalCostBasis),
		totalOpenReceivable:  s.mustCustomerReceivableBalance(cust.GetID(), USD, nil),
		accrued:              s.mustCustomerAccruedBalance(cust.GetID(), USD),
		authorizedReceivable: s.mustCustomerAuthorizedReceivableBalance(cust.GetID(), USD, nil),
		totalWash:            s.mustWashBalance(ns, USD, nil),
		externalWash:         s.mustWashBalance(ns, USD, &externalCostBasis),
		earnings:             s.mustEarningsBalance(ns, USD),
	}
	assertDelta := func(label string, start, delta, actual alpacadecimal.Decimal) {
		s.T().Helper()
		expected := start.Add(delta)
		s.True(actual.Equal(expected), "%s: expected start %s + delta %s = %s, got %s", label, start.String(), delta.String(), expected.String(), actual.String())
	}

	s.Run("create new upcoming charge for flat fee", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditOnlySettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromFloat(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              flatFeeName,
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: flatFeeName,
				}),
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(res[0].Type(), meta.ChargeTypeFlatFee)
		flatFeeCharge, err := res[0].AsFlatFeeCharge()
		s.NoError(err)

		flatFeeChargeID = flatFeeCharge.GetChargeID()
		s.Equal(meta.ChargeStatusCreated, flatFeeCharge.Status)

		gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespaces: []string{ns},
			Customers:  []string{cust.ID},
			Currencies: []currencyx.Code{USD},
			Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
		})
		s.NoError(err)
		s.Len(gatheringInvoices.Items, 0)

		// Credit-only flat fees bypass invoice creation and are only allocated once the charge advances at InvoiceAt,
		// so creating the charge early should leave every ledger bucket untouched.
		assertDelta("promo FBO after credit-only create", flatFeeStart.promoFBO, alpacadecimal.Zero, s.mustCustomerFBOBalance(cust.GetID(), USD, &promoCostBasis))
		assertDelta("external FBO after credit-only create", flatFeeStart.externalFBO, alpacadecimal.Zero, s.mustCustomerFBOBalance(cust.GetID(), USD, &externalCostBasis))
		assertDelta("unknown FBO after credit-only create", flatFeeStart.unknownFBO, alpacadecimal.Zero, s.mustCustomerFBOBalance(cust.GetID(), USD, nil))
		assertDelta("authorized receivable after credit-only create", flatFeeStart.authorizedReceivable, alpacadecimal.Zero, s.mustCustomerAuthorizedReceivableBalance(cust.GetID(), USD, nil))
		assertDelta("total open receivable after credit-only create", flatFeeStart.totalOpenReceivable, alpacadecimal.Zero, s.mustCustomerReceivableBalance(cust.GetID(), USD, nil))
		assertDelta("accrued after credit-only create", flatFeeStart.accrued, alpacadecimal.Zero, s.mustCustomerAccruedBalance(cust.GetID(), USD))
		assertDelta("earnings after credit-only create", flatFeeStart.earnings, alpacadecimal.Zero, s.mustEarningsBalance(ns, USD))
	})

	clock.SetTime(servicePeriod.From)
	s.Run("advance the charge at invoice_at", func() {
		advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: cust.GetID(),
		})
		s.NoError(err)
		s.Len(advancedCharges, 1)

		advancedFlatFee, err := advancedCharges[0].AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatFeeChargeID.ID, advancedFlatFee.ID)
		s.Equal(meta.ChargeStatusFinal, advancedFlatFee.Status)
		// We expect three realizations here: promotional credit, purchased credit, and the synthetic shortfall coverage.
		s.Len(advancedFlatFee.State.CreditRealizations, 3)

		fetchedCharge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := fetchedCharge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(meta.ChargeStatusFinal, updatedFlatFeeCharge.Status)
		s.Len(updatedFlatFeeCharge.State.CreditRealizations, 3)

		gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespaces: []string{ns},
			Customers:  []string{cust.ID},
			Currencies: []currencyx.Code{USD},
			Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
		})
		s.NoError(err)
		s.Len(gatheringInvoices.Items, 0)

		// Credit-only advancement only performs the allocation step:
		// - existing credit buckets are consumed into accrued
		// - the uncovered remainder becomes open receivable immediately
		// - authorized receivable stays empty because no payment authorization happens
		// - wash and earnings stay unchanged because this flow never enters the invoice payment lifecycle
		assertDelta("promo FBO after credit-only advance", flatFeeStart.promoFBO, alpacadecimal.NewFromInt(-30), s.mustCustomerFBOBalance(cust.GetID(), USD, &promoCostBasis))
		assertDelta("external FBO after credit-only advance", flatFeeStart.externalFBO, alpacadecimal.NewFromInt(-50), s.mustCustomerFBOBalance(cust.GetID(), USD, &externalCostBasis))
		assertDelta("unknown FBO after credit-only advance", flatFeeStart.unknownFBO, alpacadecimal.Zero, s.mustCustomerFBOBalance(cust.GetID(), USD, nil))
		assertDelta("promo receivable after credit-only advance", flatFeeStart.promoReceivable, alpacadecimal.Zero, s.mustCustomerReceivableBalance(cust.GetID(), USD, &promoCostBasis))
		assertDelta("external receivable after credit-only advance", flatFeeStart.externalReceivable, alpacadecimal.Zero, s.mustCustomerReceivableBalance(cust.GetID(), USD, &externalCostBasis))
		assertDelta("total open receivable after credit-only advance", flatFeeStart.totalOpenReceivable, alpacadecimal.NewFromInt(-20), s.mustCustomerReceivableBalance(cust.GetID(), USD, nil))
		assertDelta("authorized receivable after credit-only advance", flatFeeStart.authorizedReceivable, alpacadecimal.Zero, s.mustCustomerAuthorizedReceivableBalance(cust.GetID(), USD, nil))
		s.True(
			s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, nil, ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.NewFromInt(-20)),
			"the uncovered credit_only shortfall should live in the exact open advance receivable route",
		)
		s.True(
			s.mustCustomerAccruedBalanceWithCostBasis(cust.GetID(), USD, nil).Equal(alpacadecimal.NewFromInt(20)),
			"the uncovered shortfall should also remain in unattributed accrued until a later purchase backfills it",
		)
		assertDelta("accrued after credit-only advance", flatFeeStart.accrued, alpacadecimal.NewFromInt(100), s.mustCustomerAccruedBalance(cust.GetID(), USD))
		assertDelta("total wash after credit-only advance", flatFeeStart.totalWash, alpacadecimal.Zero, s.mustWashBalance(ns, USD, nil))
		assertDelta("external wash after credit-only advance", flatFeeStart.externalWash, alpacadecimal.Zero, s.mustWashBalance(ns, USD, &externalCostBasis))
		assertDelta("earnings after credit-only advance", flatFeeStart.earnings, alpacadecimal.Zero, s.mustEarningsBalance(ns, USD))
	})

	s.Run("the customer later purchases credits and backfills the prior advance", func() {
		type backfillSnapshot struct {
			externalFBO            alpacadecimal.Decimal
			externalOpenReceivable alpacadecimal.Decimal
			advanceOpenReceivable  alpacadecimal.Decimal
			advanceAuthorized      alpacadecimal.Decimal
			externalAccrued        alpacadecimal.Decimal
			unattributedAccrued    alpacadecimal.Decimal
			totalAccrued           alpacadecimal.Decimal
			externalWash           alpacadecimal.Decimal
		}

		start := backfillSnapshot{
			externalFBO:            s.mustCustomerFBOBalance(cust.GetID(), USD, &externalCostBasis),
			externalOpenReceivable: s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, &externalCostBasis, ledger.TransactionAuthorizationStatusOpen),
			advanceOpenReceivable:  s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, nil, ledger.TransactionAuthorizationStatusOpen),
			advanceAuthorized:      s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, nil, ledger.TransactionAuthorizationStatusAuthorized),
			externalAccrued:        s.mustCustomerAccruedBalanceWithCostBasis(cust.GetID(), USD, &externalCostBasis),
			unattributedAccrued:    s.mustCustomerAccruedBalanceWithCostBasis(cust.GetID(), USD, nil),
			totalAccrued:           s.mustCustomerAccruedBalance(cust.GetID(), USD),
			externalWash:           s.mustWashBalance(ns, USD, &externalCostBasis),
		}

		const laterPurchaseAmount = 50
		clock.SetTime(servicePeriod.From.Add(time.Hour))

		intent := s.createCreditPurchaseIntent(createCreditPurchaseIntentInput{
			customer: cust.GetID(),
			currency: USD,
			amount:   alpacadecimal.NewFromInt(laterPurchaseAmount),
			servicePeriod: timeutil.ClosedPeriod{
				From: clock.Now(),
				To:   clock.Now(),
			},
			settlement: creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				GenericSettlement: creditpurchase.GenericSettlement{
					Currency:  USD,
					CostBasis: externalCostBasis,
				},
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			}),
		})

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				intent,
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		charge, err := res[0].AsCreditPurchaseCharge()
		s.NoError(err)
		s.NotEmpty(charge.State.CreditGrantRealization.TransactionGroupID)

		// Purchase initiation performs the whole attribution decision up front:
		// - the prior advance receivable is re-attributed into the purchased cost-basis bucket
		// - unattributed accrued is translated into the purchased cost-basis bucket
		// - only the remainder becomes newly issued purchased credit
		assertDelta("external FBO after later purchase initiation", start.externalFBO, alpacadecimal.NewFromInt(30), s.mustCustomerFBOBalance(cust.GetID(), USD, &externalCostBasis))
		s.True(
			s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, &externalCostBasis, ledger.TransactionAuthorizationStatusOpen).Equal(start.externalOpenReceivable.Sub(alpacadecimal.NewFromInt(50))),
			"the purchased cost-basis open receivable should now represent the full purchase amount",
		)
		s.True(
			s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, nil, ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero),
			"the prior advance receivable should be fully re-attributed out of the nil cost-basis bucket at initiation",
		)
		s.True(
			s.mustCustomerAccruedBalanceWithCostBasis(cust.GetID(), USD, nil).Equal(alpacadecimal.Zero),
			"the unattributed accrued bucket should be translated immediately during attribution",
		)
		s.True(
			s.mustCustomerAccruedBalanceWithCostBasis(cust.GetID(), USD, &externalCostBasis).Equal(start.externalAccrued.Add(alpacadecimal.NewFromInt(20))),
			"the backfilled portion should already be visible in the purchased cost-basis accrued bucket after initiation",
		)

		updatedCharge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           charge.GetChargeID(),
			TargetPaymentState: payment.StatusAuthorized,
		})
		s.NoError(err)
		s.Equal(payment.StatusAuthorized, updatedCharge.State.ExternalPaymentSettlement.Status)

		// Authorization now only stages settlement funding; attribution already happened during purchase initiation.
		s.True(
			s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, &externalCostBasis, ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.NewFromInt(50)),
			"the purchased amount should be visible in the exact authorized receivable route before settlement",
		)
		s.True(
			s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, nil, ledger.TransactionAuthorizationStatusAuthorized).Equal(start.advanceAuthorized),
			"the legacy advance route should still have no authorized staging",
		)

		updatedCharge, err = s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           charge.GetChargeID(),
			TargetPaymentState: payment.StatusSettled,
		})
		s.NoError(err)
		s.Equal(payment.StatusSettled, updatedCharge.State.ExternalPaymentSettlement.Status)

		// Settlement is now just the normal authorized -> open move in the purchased cost-basis bucket.
		// The earlier attribution stays intact, and the purchased receivable fully nets out here.
		s.True(
			s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, nil, ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero),
			"the exact open advance receivable bucket should stay cleared after initiation-time attribution",
		)
		s.True(
			s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, nil, ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.Zero),
			"the exact authorized advance bucket should stay empty",
		)
		s.True(
			s.mustCustomerAccruedBalanceWithCostBasis(cust.GetID(), USD, nil).Equal(alpacadecimal.Zero),
			"the unattributed accrued bucket should remain empty after initiation-time translation",
		)
		s.True(
			s.mustCustomerAccruedBalanceWithCostBasis(cust.GetID(), USD, &externalCostBasis).Equal(start.externalAccrued.Add(alpacadecimal.NewFromInt(20))),
			"the backfilled portion should remain attributed in the purchased cost-basis bucket",
		)
		s.True(
			s.mustCustomerFBOBalance(cust.GetID(), USD, &externalCostBasis).Equal(start.externalFBO.Add(alpacadecimal.NewFromInt(30))),
			"only the purchase remainder should stay behind as newly available credit",
		)
		s.True(
			s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, &externalCostBasis, ledger.TransactionAuthorizationStatusOpen).Equal(alpacadecimal.Zero),
			"the purchased cost-basis receivable should net back to zero after settlement and advance funding",
		)
		s.True(
			s.mustCustomerReceivableRouteBalance(cust.GetID(), USD, &externalCostBasis, ledger.TransactionAuthorizationStatusAuthorized).Equal(alpacadecimal.Zero),
			"the purchased authorized receivable route should be cleared by settlement",
		)
		s.True(
			s.mustCustomerAccruedBalance(cust.GetID(), USD).Equal(start.totalAccrued),
			"settlement should only translate accrued between buckets, not change the total accrued amount",
		)
		assertDelta("external wash after later purchase settlement", start.externalWash, alpacadecimal.NewFromInt(-50), s.mustWashBalance(ns, USD, &externalCostBasis))
	})
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

func (i *createMockChargeIntentInput) Validate() error {
	if i.price == nil {
		return errors.New("price is required")
	}

	if i.customer.Namespace == "" {
		return errors.New("customer namespace is required")
	}

	if i.customer.ID == "" {
		return errors.New("customer id is required")
	}

	if i.currency == "" {
		return errors.New("currency is required")
	}

	return nil
}

func (s *CreditsTestSuite) createMockChargeIntent(input createMockChargeIntentInput) charges.ChargeIntent {
	s.T().Helper()
	s.NoError(input.Validate())

	isFlatFee := input.price.Type() == productcatalog.FlatPriceType
	invoiceAt := input.servicePeriod.To

	if isFlatFee {
		price, err := input.price.AsFlat()
		s.NoError(err)

		switch price.PaymentTerm {
		case productcatalog.InAdvancePaymentTerm:
			invoiceAt = input.servicePeriod.From
		case productcatalog.InArrearsPaymentTerm:
			invoiceAt = input.servicePeriod.To
		default:
			s.T().Fatalf("invalid payment term: %s", price.PaymentTerm)
		}
	}

	intentMeta := meta.Intent{
		Name:              input.name,
		ManagedBy:         input.managedBy,
		ServicePeriod:     input.servicePeriod,
		FullServicePeriod: input.servicePeriod,
		BillingPeriod:     input.servicePeriod,
		UniqueReferenceID: lo.EmptyableToPtr(input.uniqueReferenceID),
		CustomerID:        input.customer.ID,
		Currency:          input.currency,
	}

	if isFlatFee {
		price, err := input.price.AsFlat()
		s.NoError(err)

		flatFeeIntent := flatfee.Intent{
			Intent:         intentMeta,
			PaymentTerm:    price.PaymentTerm,
			FeatureKey:     input.featureKey,
			InvoiceAt:      invoiceAt,
			SettlementMode: lo.CoalesceOrEmpty(input.settlementMode, productcatalog.InvoiceOnlySettlementMode),

			AmountBeforeProration: price.Amount,
		}
		return charges.NewChargeIntent(flatFeeIntent)
	}

	usageBasedIntent := usagebased.Intent{
		Intent:         intentMeta,
		Price:          *input.price,
		InvoiceAt:      invoiceAt,
		SettlementMode: lo.CoalesceOrEmpty(input.settlementMode, productcatalog.InvoiceOnlySettlementMode),
		FeatureKey:     input.featureKey,
	}

	return charges.NewChargeIntent(usageBasedIntent)
}

func (s *CreditsTestSuite) createLedgerBackedCustomer(ns string, subjectKey string) *customer.Customer {
	s.T().Helper()

	_, err := s.LedgerResolver.EnsureBusinessAccounts(context.Background(), ns)
	s.NoError(err)

	cust := s.CreateTestCustomer(ns, subjectKey)
	_, err = s.LedgerResolver.CreateCustomerAccounts(context.Background(), cust.GetID())
	s.NoError(err)

	return cust
}

func (s *CreditsTestSuite) mustCustomerFBOBalance(customerID customer.CustomerID, code currencyx.Code, costBasis *alpacadecimal.Decimal) alpacadecimal.Decimal {
	return s.mustCustomerFBOBalanceWithPriority(customerID, code, costBasis, ledger.DefaultCustomerFBOPriority)
}

func (s *CreditsTestSuite) mustCustomerFBOBalanceWithPriority(customerID customer.CustomerID, code currencyx.Code, costBasis *alpacadecimal.Decimal, priority int) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	subAccount, err := customerAccounts.FBOAccount.GetSubAccountForRoute(s.T().Context(), ledger.CustomerFBORouteParams{
		Currency:       code,
		CostBasis:      costBasis,
		CreditPriority: priority,
	})
	s.NoError(err)

	balance, err := subAccount.GetBalance(s.T().Context())
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditsTestSuite) mustCustomerReceivableBalance(customerID customer.CustomerID, code currencyx.Code, costBasis *alpacadecimal.Decimal) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := customerAccounts.ReceivableAccount.GetBalance(s.T().Context(), ledger.RouteFilter{
		Currency:                       code,
		CostBasis:                      routeFilterCostBasis(costBasis),
		TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
	})
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditsTestSuite) mustCustomerAuthorizedReceivableBalance(customerID customer.CustomerID, code currencyx.Code, costBasis *alpacadecimal.Decimal) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := customerAccounts.ReceivableAccount.GetBalance(s.T().Context(), ledger.RouteFilter{
		Currency:                       code,
		CostBasis:                      routeFilterCostBasis(costBasis),
		TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusAuthorized),
	})
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditsTestSuite) mustCustomerAccruedBalance(customerID customer.CustomerID, code currencyx.Code) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	balance, err := customerAccounts.AccruedAccount.GetBalance(s.T().Context(), ledger.RouteFilter{
		Currency: code,
	})
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditsTestSuite) mustCustomerAccruedBalanceWithCostBasis(customerID customer.CustomerID, code currencyx.Code, costBasis *alpacadecimal.Decimal) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	subAccount, err := customerAccounts.AccruedAccount.GetSubAccountForRoute(s.T().Context(), ledger.CustomerAccruedRouteParams{
		Currency:  code,
		CostBasis: costBasis,
	})
	s.NoError(err)

	balance, err := subAccount.GetBalance(s.T().Context())
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditsTestSuite) mustCustomerReceivableRouteBalance(customerID customer.CustomerID, code currencyx.Code, costBasis *alpacadecimal.Decimal, status ledger.TransactionAuthorizationStatus) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	subAccount, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(s.T().Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       code,
		CostBasis:                      costBasis,
		TransactionAuthorizationStatus: status,
	})
	s.NoError(err)

	balance, err := subAccount.GetBalance(s.T().Context())
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditsTestSuite) mustWashBalance(namespace string, code currencyx.Code, costBasis *alpacadecimal.Decimal) alpacadecimal.Decimal {
	s.T().Helper()

	businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(s.T().Context(), namespace)
	s.NoError(err)

	balance, err := businessAccounts.WashAccount.GetBalance(s.T().Context(), ledger.RouteFilter{
		Currency:  code,
		CostBasis: routeFilterCostBasis(costBasis),
	})
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditsTestSuite) mustEarningsBalance(namespace string, code currencyx.Code) alpacadecimal.Decimal {
	s.T().Helper()

	businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(s.T().Context(), namespace)
	s.NoError(err)

	balance, err := businessAccounts.EarningsAccount.GetBalance(s.T().Context(), ledger.RouteFilter{
		Currency: code,
	})
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditsTestSuite) mustGetChargeByID(chargeID meta.ChargeID) charges.Charge {
	s.T().Helper()
	charge, err := s.Charges.GetByID(s.T().Context(), charges.GetByIDInput{
		ChargeID: chargeID,
		Expands:  meta.Expands{meta.ExpandRealizations},
	})
	s.NoError(err)
	return charge
}

func routeFilterCostBasis(costBasis *alpacadecimal.Decimal) mo.Option[*alpacadecimal.Decimal] {
	if costBasis == nil {
		return mo.None[*alpacadecimal.Decimal]()
	}

	return mo.Some(costBasis)
}

type createCreditPurchaseIntentInput struct {
	customer      customer.CustomerID
	currency      currencyx.Code
	amount        alpacadecimal.Decimal
	effectiveAt   *time.Time
	priority      *int
	servicePeriod timeutil.ClosedPeriod
	settlement    creditpurchase.Settlement
}

func (i createCreditPurchaseIntentInput) Validate() error {
	if err := i.customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if i.currency == "" {
		return errors.New("currency is required")
	}

	if !i.amount.IsPositive() {
		return errors.New("amount must be positive")
	}

	if err := i.servicePeriod.Validate(); err != nil {
		return fmt.Errorf("service period: %w", err)
	}

	if err := i.settlement.Validate(); err != nil {
		return fmt.Errorf("settlement: %w", err)
	}

	return nil
}

func (s *CreditsTestSuite) createCreditPurchaseIntent(input createCreditPurchaseIntentInput) charges.ChargeIntent {
	s.T().Helper()
	s.NoError(input.Validate())

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
		EffectiveAt:  input.effectiveAt,
		Priority:     input.priority,
		Settlement:   input.settlement,
	})
}
