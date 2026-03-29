package credits

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/app/common"
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
	"github.com/openmeterio/openmeter/openmeter/ledger/routingrules"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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

type lazyQuerier struct {
	querier ledger.Querier
}

func (l *lazyQuerier) SumEntries(ctx context.Context, query ledger.Query) (ledger.QuerySummedResult, error) {
	return l.querier.SumEntries(ctx, query)
}

func (s *CreditsTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	ledgerLocker, err := common.NewLocker(slog.Default())
	s.NoError(err)

	lq := &lazyQuerier{}
	accountRepo := common.NewLedgerAccountRepo(s.DBClient)
	accountLiveServices := ledgeraccount.AccountLiveServices{
		Locker:  ledgerLocker,
		Querier: lq,
	}
	accountService := common.NewLedgerAccountService(accountRepo, accountLiveServices)
	historicalRepo := common.NewLedgerHistoricalRepo(s.DBClient)
	historicalLedger := common.NewLedgerHistoricalLedger(
		historicalRepo,
		accountService,
		ledgerLocker,
		routingrules.DefaultValidator,
	)
	lq.querier = historicalLedger

	resolversRepo := common.NewLedgerResolversRepo(s.DBClient)
	accountResolver := common.NewLedgerResolversService(accountService, resolversRepo)

	s.Ledger = historicalLedger
	s.LedgerAccountService = accountService
	s.LedgerResolver = accountResolver

	stack, err := chargestestutils.NewServices(s.T(), chargestestutils.Config{
		Client:                s.DBClient,
		Logger:                slog.Default(),
		BillingService:        s.BillingService,
		FeatureService:        s.FeatureService,
		StreamingConnector:    s.MockStreamingConnector,
		FlatFeeHandler:        ledgerchargeadapter.NewFlatFeeHandler(historicalLedger, accountResolver, accountService),
		CreditPurchaseHandler: ledgerchargeadapter.NewCreditPurchaseHandler(historicalLedger, accountResolver, accountService),
		UsageBasedHandler:     usagebased.UnimplementedHandler{},
	})
	s.NoError(err)
	s.Charges = stack.ChargesService
}

func (s *CreditsTestSuite) TestFlatFeePartialCreditRealizations() {
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
		invoices, err := s.Charges.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
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
		assertDelta("accrued after payment authorization", flatFeeStart.accrued, alpacadecimal.Zero, s.mustCustomerAccruedBalance(cust.GetID(), USD))
		assertDelta("total wash after payment authorization", flatFeeStart.totalWash, alpacadecimal.NewFromInt(-20), s.mustWashBalance(ns, USD, nil))
		assertDelta("external wash after payment authorization", flatFeeStart.externalWash, alpacadecimal.Zero, s.mustWashBalance(ns, USD, &externalCostBasis))
		assertDelta("earnings after payment authorization", flatFeeStart.earnings, alpacadecimal.NewFromInt(100), s.mustEarningsBalance(ns, USD))
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
		assertDelta("accrued after payment settlement", flatFeeStart.accrued, alpacadecimal.Zero, s.mustCustomerAccruedBalance(cust.GetID(), USD))
		assertDelta("total wash after payment settlement", flatFeeStart.totalWash, alpacadecimal.NewFromInt(-20), s.mustWashBalance(ns, USD, nil))
		assertDelta("external wash after payment settlement", flatFeeStart.externalWash, alpacadecimal.Zero, s.mustWashBalance(ns, USD, &externalCostBasis))
		assertDelta("earnings after payment settlement", flatFeeStart.earnings, alpacadecimal.NewFromInt(100), s.mustEarningsBalance(ns, USD))
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

	cust := s.CreateTestCustomer(ns, subjectKey)
	_, err := s.LedgerResolver.CreateCustomerAccounts(context.Background(), cust.GetID())
	s.NoError(err)

	return cust
}

func (s *CreditsTestSuite) mustCustomerFBOBalance(customerID customer.CustomerID, code currencyx.Code, costBasis *alpacadecimal.Decimal) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	subAccount, err := customerAccounts.FBOAccount.GetSubAccountForRoute(s.T().Context(), ledger.CustomerFBORouteParams{
		Currency:       code,
		CostBasis:      costBasis,
		CreditPriority: ledger.DefaultCustomerFBOPriority,
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

	subAccount, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(s.T().Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       code,
		CostBasis:                      costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusOpen,
	})
	s.NoError(err)

	balance, err := subAccount.GetBalance(s.T().Context())
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditsTestSuite) mustCustomerAuthorizedReceivableBalance(customerID customer.CustomerID, code currencyx.Code, costBasis *alpacadecimal.Decimal) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	subAccount, err := customerAccounts.ReceivableAccount.GetSubAccountForRoute(s.T().Context(), ledger.CustomerReceivableRouteParams{
		Currency:                       code,
		CostBasis:                      costBasis,
		TransactionAuthorizationStatus: ledger.TransactionAuthorizationStatusAuthorized,
	})
	s.NoError(err)

	balance, err := subAccount.GetBalance(s.T().Context())
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditsTestSuite) mustCustomerAccruedBalance(customerID customer.CustomerID, code currencyx.Code) alpacadecimal.Decimal {
	s.T().Helper()

	customerAccounts, err := s.LedgerResolver.GetCustomerAccounts(s.T().Context(), customerID)
	s.NoError(err)

	subAccount, err := customerAccounts.AccruedAccount.GetSubAccountForRoute(s.T().Context(), ledger.CustomerAccruedRouteParams{
		Currency: code,
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

	subAccount, err := businessAccounts.WashAccount.GetSubAccountForRoute(s.T().Context(), ledger.BusinessRouteParams{
		Currency:  code,
		CostBasis: costBasis,
	})
	s.NoError(err)

	balance, err := subAccount.GetBalance(s.T().Context())
	s.NoError(err)

	return balance.Settled()
}

func (s *CreditsTestSuite) mustEarningsBalance(namespace string, code currencyx.Code) alpacadecimal.Decimal {
	s.T().Helper()

	businessAccounts, err := s.LedgerResolver.GetBusinessAccounts(s.T().Context(), namespace)
	s.NoError(err)

	subAccount, err := businessAccounts.EarningsAccount.GetSubAccountForRoute(s.T().Context(), ledger.BusinessRouteParams{
		Currency: code,
	})
	s.NoError(err)

	balance, err := subAccount.GetBalance(s.T().Context())
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

type createCreditPurchaseIntentInput struct {
	customer      customer.CustomerID
	currency      currencyx.Code
	amount        alpacadecimal.Decimal
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
		Settlement:   input.settlement,
	})
}
