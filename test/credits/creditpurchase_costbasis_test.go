package credits

import (
	"context"
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
	creditpurchaseadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/adapter"
	creditpurchaseservice "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	omtestutils "github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestCreditPurchaseCostBasisSuite(t *testing.T) {
	suite.Run(t, new(CreditPurchaseCostBasisSuite))
}

type CreditPurchaseCostBasisSuite struct {
	BaseSuite

	creditPurchaseService creditpurchase.Service
}

func (s *CreditPurchaseCostBasisSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	logger := omtestutils.NewLogger(s.T())
	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: s.DBClient,
		Logger: logger,
	})
	s.Require().NoError(err)

	adapter, err := creditpurchaseadapter.New(creditpurchaseadapter.Config{
		Client:      s.DBClient,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	s.Require().NoError(err)

	handler, err := ledgerchargeadapter.NewCreditPurchaseHandler(
		s.Ledger,
		s.BalanceQuerier,
		s.LedgerResolver,
		s.LedgerAccountService,
		s.BreakageService,
		enttx.NewCreator(s.DBClient),
	)
	s.Require().NoError(err)

	s.creditPurchaseService, err = creditpurchaseservice.New(creditpurchaseservice.Config{
		Adapter:     adapter,
		Handler:     handler,
		Lineage:     creditPurchaseCostBasisLineage{},
		MetaAdapter: metaAdapter,
		Currencies:  s.CurrencyService,
	})
	s.Require().NoError(err)

	s.Require().NoError(s.BillingService.DeregisterLineEngine(billing.LineEngineTypeChargeCreditPurchase))
	s.Require().NoError(s.BillingService.RegisterLineEngine(s.creditPurchaseService.GetLineEngine()))
}

func (s *CreditPurchaseCostBasisSuite) TestDynamicInvoiceSettlementLifecycle() {
	// given:
	// - a dynamic invoice-settled custom-currency credit purchase
	// - a cost basis effective at the service-period start and a newer cost basis
	// when:
	// - billing materializes, authorizes, and settles the invoice
	// then:
	// - the state machine pins the threshold cost basis before granting credits
	// - invoicing and every ledger realization use that resolved cost basis
	t := s.T()
	ctx := t.Context()
	namespace := s.GetUniqueNamespace("dynamic-invoice-credit-purchase")
	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()

	var (
		fixture  dynamicCreditPurchaseFixture
		chargeID meta.ChargeID
		invoice  billing.StandardInvoice
	)

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	s.Run("given a dynamic invoice credit purchase with historical cost bases", func() {
		// This phase verifies that creation persists the dynamic intent without
		// resolving it or booking the credit grant.
		customInvoicing := s.SetupCustomInvoicing(namespace)
		_ = s.ProvisionBillingProfile(ctx, namespace, customInvoicing.App.GetID(),
			billingtest.WithProgressiveBilling(),
			billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "PT1H")),
			billingtest.WithManualApproval(),
		)

		fixture = s.provisionDynamicCreditPurchaseFixture(namespace)
		created, err := s.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
			Namespace: namespace,
			Intent:    fixture.newIntent(creditpurchase.NewInvoiceSettlement()),
		})
		s.Require().NoError(err)
		charge := created.Charge
		chargeID = charge.GetChargeID()
		s.Equal(creditpurchase.StatusCreated, charge.Status)
		s.Require().NotNil(charge.State.ChargeCostBasisID)
		s.Nil(charge.State.ResolvedCostBasis)
		s.Nil(charge.Realizations.CreditGrantRealization)
		s.Require().NotNil(created.GatheringLineToCreate)
		gatheringPrice, err := created.GatheringLineToCreate.Price.AsFlat()
		s.Require().NoError(err)
		s.Equal(alpacadecimal.Zero, gatheringPrice.Amount)

		_, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: fixture.customerID,
			Currency: currencyx.FiatCode(fixture.settlementCurrency),
			Lines:    []billing.GatheringLine{*created.GatheringLineToCreate},
		})
		s.Require().NoError(err)

		persisted, err := s.MustGetChargeByID(chargeID).AsCreditPurchaseCharge()
		s.Require().NoError(err)
		s.Nil(persisted.State.ResolvedCostBasis)
		s.Equal(alpacadecimal.Zero, s.mustAccountBalance(fixture.customerAccounts.FBOAccount, ledger.RouteFilter{
			Currency:          fixture.customCurrency.Reference(),
			CostBasisCurrency: mo.Some(&fixture.settlementCurrency),
			CostBasis:         mo.Some(&fixture.resolvedRate),
			CreditPriority:    lo.ToPtr(ledger.DefaultCustomerFBOPriority),
		}))
	})

	s.Run("when billing materializes the invoice line", func() {
		// This phase verifies that invoice creation resolves the threshold cost
		// basis before the line and granted-credit ledger route are finalized.
		clock.FreezeTime(fixture.servicePeriod.From)
		now := clock.Now()
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: fixture.customerID,
			AsOf:     &now,
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		invoice = invoices[0]

		lines := invoice.Lines.OrEmpty()
		s.Require().Len(lines, 1)
		line := lines[0]
		s.Equal(currencyx.FiatCode(fixture.settlementCurrency), line.Currency)
		s.Equal(fixture.fiatAmount, line.Totals.Amount)
		s.Equal(fixture.fiatAmount, line.Totals.Total)
		s.Require().Len(line.DetailedLines, 1)
		s.Equal(fixture.creditAmount, line.DetailedLines[0].Quantity)
		s.Equal(fixture.resolvedRate, line.DetailedLines[0].PerUnitAmount)

		charge, err := s.MustGetChargeByID(chargeID).AsCreditPurchaseCharge()
		s.Require().NoError(err)
		s.Equal(creditpurchase.StatusActivePaymentPending, charge.Status)
		s.Require().NotNil(charge.Realizations.CreditGrantRealization)
		s.Require().NotNil(charge.State.ResolvedCostBasis)
		s.Equal(fixture.resolvedRate, charge.State.ResolvedCostBasis.CostBasis)
		s.Equal(fixture.resolvedCurrencyCostBasisID, lo.FromPtr(charge.State.ResolvedCostBasis.CostBasisID))
		s.Equal(fixture.servicePeriod.From, charge.State.ResolvedCostBasis.ResolvedAt)

		customRoute := ledger.RouteFilter{
			Currency:          fixture.customCurrency.Reference(),
			CostBasisCurrency: mo.Some(&fixture.settlementCurrency),
			CostBasis:         mo.Some(&fixture.resolvedRate),
		}
		s.Equal(fixture.creditAmount, s.mustAccountBalance(fixture.customerAccounts.FBOAccount, ledger.RouteFilter{
			Currency:          customRoute.Currency,
			CostBasisCurrency: customRoute.CostBasisCurrency,
			CostBasis:         customRoute.CostBasis,
			CreditPriority:    lo.ToPtr(ledger.DefaultCustomerFBOPriority),
		}))
		s.Equal(fixture.creditAmount.Neg(), s.mustAccountBalance(fixture.customerAccounts.ReceivableAccount, ledger.RouteFilter{
			Currency:                       customRoute.Currency,
			CostBasisCurrency:              customRoute.CostBasisCurrency,
			CostBasis:                      customRoute.CostBasis,
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
		}))
	})

	s.Run("when the invoice payment is authorized", func() {
		// This phase verifies that authorization translates the custom-currency
		// receivable into the correctly valued fiat receivable.
		var err error
		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		invoice, err = s.BillingService.PaymentAuthorized(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, invoice.Status)

		charge, err := s.MustGetChargeByID(chargeID).AsCreditPurchaseCharge()
		s.Require().NoError(err)
		s.Equal(creditpurchase.StatusActivePaymentAuthorized, charge.Status)
		s.Require().NotNil(charge.Realizations.InvoiceSettlement)
		s.Equal(payment.StatusAuthorized, charge.Realizations.InvoiceSettlement.Status)
		s.Equal(fixture.fiatAmount, charge.Realizations.InvoiceSettlement.FiatAmount)

		s.Equal(alpacadecimal.Zero, s.mustAccountBalance(fixture.customerAccounts.ReceivableAccount, ledger.RouteFilter{
			Currency:                       fixture.customCurrency.Reference(),
			CostBasisCurrency:              mo.Some(&fixture.settlementCurrency),
			CostBasis:                      mo.Some(&fixture.resolvedRate),
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
		}))
		s.Equal(fixture.fiatAmount.Neg(), s.mustAccountBalance(fixture.customerAccounts.ReceivableAccount, ledger.RouteFilter{
			Currency:                       currencies.NewCurrencyReference(fixture.settlementCurrency),
			CostBasis:                      mo.Some(&fixture.resolvedRate),
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusAuthorized),
		}))
	})

	s.Run("when the invoice payment is settled", func() {
		// This phase verifies that settlement clears the fiat authorization while
		// preserving the granted custom credits at their pinned ledger route.
		var err error
		invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: invoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

		charge, err := s.MustGetChargeByID(chargeID).AsCreditPurchaseCharge()
		s.Require().NoError(err)
		s.Equal(creditpurchase.StatusFinal, charge.Status)
		s.Require().NotNil(charge.Realizations.InvoiceSettlement)
		s.Equal(payment.StatusSettled, charge.Realizations.InvoiceSettlement.Status)
		s.Equal(fixture.fiatAmount, charge.Realizations.InvoiceSettlement.FiatAmount)
		s.Require().NotNil(charge.Realizations.InvoiceSettlement.Authorized)
		s.Require().NotNil(charge.Realizations.InvoiceSettlement.Settled)

		s.Equal(alpacadecimal.Zero, s.mustAccountBalance(fixture.customerAccounts.ReceivableAccount, ledger.RouteFilter{
			Currency:                       currencies.NewCurrencyReference(fixture.settlementCurrency),
			CostBasis:                      mo.Some(&fixture.resolvedRate),
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusAuthorized),
		}))
		s.Equal(fixture.fiatAmount.Neg(), s.MustWashBalance(namespace, fixture.settlementCurrency, mo.Some(&fixture.resolvedRate)))
		s.Equal(fixture.creditAmount, s.mustAccountBalance(fixture.customerAccounts.FBOAccount, ledger.RouteFilter{
			Currency:          fixture.customCurrency.Reference(),
			CostBasisCurrency: mo.Some(&fixture.settlementCurrency),
			CostBasis:         mo.Some(&fixture.resolvedRate),
			CreditPriority:    lo.ToPtr(ledger.DefaultCustomerFBOPriority),
		}))
	})
}

func (s *CreditPurchaseCostBasisSuite) TestDynamicExternalSettlementLifecycle() {
	// given:
	// - a dynamic externally settled custom-currency credit purchase
	// - a cost basis effective at the service-period start and a newer cost basis
	// when:
	// - the external payment is authorized and settled
	// then:
	// - the state machine pins the threshold cost basis before granting credits
	// - every ledger realization uses that resolved cost basis
	t := s.T()
	ctx := t.Context()
	namespace := s.GetUniqueNamespace("dynamic-external-credit-purchase")
	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()

	var (
		fixture  dynamicCreditPurchaseFixture
		chargeID meta.ChargeID
	)

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	s.Run("given a dynamic external credit purchase with historical cost bases", func() {
		// This phase verifies that creation pins the cost basis effective at the
		// service-period threshold before booking the external credit grant.
		fixture = s.provisionDynamicCreditPurchaseFixture(namespace)
		created, err := s.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
			Namespace: namespace,
			Intent: fixture.newIntent(creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
				InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			})),
		})
		s.Require().NoError(err)
		charge := created.Charge
		chargeID = charge.GetChargeID()
		s.Equal(creditpurchase.StatusActivePaymentPending, charge.Status)
		s.Require().NotNil(charge.State.ResolvedCostBasis)
		s.Equal(fixture.resolvedRate, charge.State.ResolvedCostBasis.CostBasis)
		s.Equal(fixture.resolvedCurrencyCostBasisID, lo.FromPtr(charge.State.ResolvedCostBasis.CostBasisID))
		s.Equal(fixture.servicePeriod.From, charge.State.ResolvedCostBasis.ResolvedAt)
		s.Require().NotNil(charge.Realizations.CreditGrantRealization)
		s.Nil(charge.Realizations.ExternalPaymentSettlement)
		s.Nil(created.GatheringLineToCreate)

		persisted, err := s.MustGetChargeByID(chargeID).AsCreditPurchaseCharge()
		s.Require().NoError(err)
		s.Equal(charge.State.ResolvedCostBasis, persisted.State.ResolvedCostBasis)
		s.Equal(fixture.creditAmount, s.mustAccountBalance(fixture.customerAccounts.FBOAccount, ledger.RouteFilter{
			Currency:          fixture.customCurrency.Reference(),
			CostBasisCurrency: mo.Some(&fixture.settlementCurrency),
			CostBasis:         mo.Some(&fixture.resolvedRate),
			CreditPriority:    lo.ToPtr(ledger.DefaultCustomerFBOPriority),
		}))
		s.Equal(fixture.creditAmount.Neg(), s.mustAccountBalance(fixture.customerAccounts.ReceivableAccount, ledger.RouteFilter{
			Currency:                       fixture.customCurrency.Reference(),
			CostBasisCurrency:              mo.Some(&fixture.settlementCurrency),
			CostBasis:                      mo.Some(&fixture.resolvedRate),
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
		}))
	})

	s.Run("when the external payment is authorized", func() {
		// This phase verifies that authorization translates the custom-currency
		// receivable into the correctly valued fiat receivable.
		clock.FreezeTime(fixture.servicePeriod.From)
		charge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           chargeID,
			TargetPaymentState: payment.StatusAuthorized,
		})
		s.Require().NoError(err)
		s.Equal(creditpurchase.StatusActivePaymentAuthorized, charge.Status)
		s.Require().NotNil(charge.Realizations.ExternalPaymentSettlement)
		s.Equal(payment.StatusAuthorized, charge.Realizations.ExternalPaymentSettlement.Status)
		s.Equal(fixture.fiatAmount, charge.Realizations.ExternalPaymentSettlement.FiatAmount)

		s.Equal(alpacadecimal.Zero, s.mustAccountBalance(fixture.customerAccounts.ReceivableAccount, ledger.RouteFilter{
			Currency:                       fixture.customCurrency.Reference(),
			CostBasisCurrency:              mo.Some(&fixture.settlementCurrency),
			CostBasis:                      mo.Some(&fixture.resolvedRate),
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusOpen),
		}))
		s.Equal(fixture.fiatAmount.Neg(), s.mustAccountBalance(fixture.customerAccounts.ReceivableAccount, ledger.RouteFilter{
			Currency:                       currencies.NewCurrencyReference(fixture.settlementCurrency),
			CostBasis:                      mo.Some(&fixture.resolvedRate),
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusAuthorized),
		}))
	})

	s.Run("when the external payment is settled", func() {
		// This phase verifies that settlement clears the fiat authorization while
		// preserving the externally funded custom credits at their pinned route.
		charge, err := s.Charges.HandleCreditPurchaseExternalPaymentStateTransition(ctx, charges.HandleCreditPurchaseExternalPaymentStateTransitionInput{
			ChargeID:           chargeID,
			TargetPaymentState: payment.StatusSettled,
		})
		s.Require().NoError(err)
		s.Equal(creditpurchase.StatusFinal, charge.Status)
		s.Require().NotNil(charge.Realizations.ExternalPaymentSettlement)
		s.Equal(payment.StatusSettled, charge.Realizations.ExternalPaymentSettlement.Status)
		s.Equal(fixture.fiatAmount, charge.Realizations.ExternalPaymentSettlement.FiatAmount)
		s.Require().NotNil(charge.Realizations.ExternalPaymentSettlement.Authorized)
		s.Require().NotNil(charge.Realizations.ExternalPaymentSettlement.Settled)

		s.Equal(alpacadecimal.Zero, s.mustAccountBalance(fixture.customerAccounts.ReceivableAccount, ledger.RouteFilter{
			Currency:                       currencies.NewCurrencyReference(fixture.settlementCurrency),
			CostBasis:                      mo.Some(&fixture.resolvedRate),
			TransactionAuthorizationStatus: lo.ToPtr(ledger.TransactionAuthorizationStatusAuthorized),
		}))
		s.Equal(fixture.fiatAmount.Neg(), s.MustWashBalance(namespace, fixture.settlementCurrency, mo.Some(&fixture.resolvedRate)))
		s.Equal(fixture.creditAmount, s.mustAccountBalance(fixture.customerAccounts.FBOAccount, ledger.RouteFilter{
			Currency:          fixture.customCurrency.Reference(),
			CostBasisCurrency: mo.Some(&fixture.settlementCurrency),
			CostBasis:         mo.Some(&fixture.resolvedRate),
			CreditPriority:    lo.ToPtr(ledger.DefaultCustomerFBOPriority),
		}))
	})
}

type dynamicCreditPurchaseFixture struct {
	taxCodeID                   string
	servicePeriod               timeutil.ClosedPeriod
	settlementCurrency          currencyx.Code
	fiatCurrency                *currencyx.FiatCurrency
	resolvedRate                alpacadecimal.Decimal
	creditAmount                alpacadecimal.Decimal
	fiatAmount                  alpacadecimal.Decimal
	resolvedCurrencyCostBasisID string
	customerID                  customer.CustomerID
	customCurrency              currencies.Currency
	customerAccounts            ledger.CustomerAccounts
}

func (s *CreditPurchaseCostBasisSuite) provisionDynamicCreditPurchaseFixture(namespace string) dynamicCreditPurchaseFixture {
	s.T().Helper()

	ctx := s.T().Context()
	fixture := dynamicCreditPurchaseFixture{
		servicePeriod: timeutil.ClosedPeriod{
			From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
			To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
		},
		settlementCurrency: currencyx.Code(currency.USD),
		resolvedRate:       alpacadecimal.NewFromFloat(0.25),
		creditAmount:       alpacadecimal.NewFromInt(100),
		fiatAmount:         alpacadecimal.NewFromInt(25),
	}
	fixture.taxCodeID = s.ProvisionDefaultTaxCodes(ctx, namespace).CreditGrantTaxCodeID
	fixture.customerID = s.CreateLedgerBackedCustomer(namespace, namespace).GetID()

	var err error
	fixture.customCurrency, err = s.CurrencyService.CreateCurrency(ctx, currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "TOKENS",
			Name:               "Tokens",
			Symbol:             "T",
			Precision:          3,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	s.Require().NoError(err)

	thresholdCostBasis, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:     namespace,
		CurrencyID:    fixture.customCurrency.ID,
		FiatCode:      fixture.settlementCurrency,
		Rate:          fixture.resolvedRate,
		EffectiveFrom: lo.ToPtr(fixture.servicePeriod.From),
	})
	s.Require().NoError(err)
	fixture.resolvedCurrencyCostBasisID = thresholdCostBasis.ID

	_, err = s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:     namespace,
		CurrencyID:    fixture.customCurrency.ID,
		FiatCode:      fixture.settlementCurrency,
		Rate:          alpacadecimal.NewFromFloat(0.5),
		EffectiveFrom: lo.ToPtr(fixture.servicePeriod.From.Add(10 * 24 * time.Hour)),
	})
	s.Require().NoError(err)

	fixture.customerAccounts, err = s.LedgerResolver.GetCustomerAccounts(ctx, fixture.customerID)
	s.Require().NoError(err)
	fixture.fiatCurrency, err = currencyx.NewFiatCurrency(fixture.settlementCurrency)
	s.Require().NoError(err)

	return fixture
}

// newIntent keeps the cost-basis scenario identical while allowing each test
// to exercise its settlement-specific state machine.
func (f dynamicCreditPurchaseFixture) newIntent(settlement creditpurchase.Settlement) creditpurchase.Intent {
	return creditpurchase.Intent{
		Intent: meta.Intent{
			ManagedBy:  billing.ManuallyManagedLine,
			CustomerID: f.customerID.ID,
			Currency:   f.customCurrency,
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: f.taxCodeID,
			},
		},
		IntentMutableFields: creditpurchase.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "Dynamic Cost Basis Credit Purchase",
				ServicePeriod:     f.servicePeriod,
				BillingPeriod:     f.servicePeriod,
				FullServicePeriod: f.servicePeriod,
			},
			CreditAmount: f.creditAmount,
			Settlement:   settlement,
		},
		CostBasis: creditpurchase.NewCostBasis(costbasis.NewIntent(costbasis.DynamicIntent{
			FiatCurrency: f.fiatCurrency,
		})),
	}
}

func (s *CreditPurchaseCostBasisSuite) mustAccountBalance(account ledger.Account, route ledger.RouteFilter) alpacadecimal.Decimal {
	s.T().Helper()

	balance, err := s.BalanceQuerier.GetAccountBalance(s.T().Context(), account, route, ledger.BalanceQuery{})
	s.Require().NoError(err)

	return balance
}

type creditPurchaseCostBasisLineage struct {
	lineage.Service
}

func (creditPurchaseCostBasisLineage) BackfillAdvanceLineageSegments(context.Context, lineage.BackfillAdvanceLineageSegmentsInput) error {
	return nil
}
