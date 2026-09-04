package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	creditgrantservice "github.com/openmeterio/openmeter/openmeter/billing/creditgrant/service"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

const TOKENS = currencyx.Code("TOKENS")

func (s *CreditGrantTestSuite) createTokensCurrency(ns string) currencies.Currency {
	s.T().Helper()

	currency, err := s.CurrencyService.CreateCurrency(s.T().Context(), currencies.CreateCurrencyInput{
		Namespace: ns,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               TOKENS,
			Name:               "Tokens",
			Symbol:             "T",
			Precision:          3,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	s.Require().NoError(err)

	return currency
}

func (s *CreditGrantTestSuite) TestCreateCustomCurrencyExternalGrantWithManualCostBasisAndSettle() {
	// given:
	// - a custom currency and an externally funded grant priced manually in USD
	// when:
	// - the grant is created and the external payment is settled
	// then:
	// - the grant carries the shared custom-currency cost basis resolved on creation
	// - the credits land on the customer's FBO account in the custom currency
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("creditgrant-custom-currency-external")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	s.createTokensCurrency(ns)

	now := datetime.MustParseTimeInLocation(s.T(), "2026-04-17T11:23:53Z", time.UTC).AsTime()
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	usd, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)

	rate := alpacadecimal.NewFromFloat(0.25)
	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "100 TOKENS for $25.00",
		Currency:      TOKENS,
		Amount:        alpacadecimal.NewFromInt(100),
		FundingMethod: creditgrant.FundingMethodExternal,
		Purchase: &creditgrant.PurchaseTerms{
			Currency: USD,
			CostBasis: lo.ToPtr(creditpurchase.NewCostBasis(costbasis.NewIntent(costbasis.ManualIntent{
				FiatCurrency: usd,
				Rate:         rate,
			}))),
			AvailabilityPolicy: lo.ToPtr(creditpurchase.CreatedInitialPaymentSettlementStatus),
		},
	})
	s.Require().NoError(err)

	s.True(grant.Intent.Currency.IsCustom())
	s.Equal(creditpurchase.CostBasisTypeCustomCurrency, grant.Intent.CostBasis.Type())
	s.NotNil(grant.State.ChargeCostBasisID)
	s.Require().NotNil(grant.State.ResolvedCostBasis)
	s.Equal(rate.InexactFloat64(), grant.State.ResolvedCostBasis.CostBasis.InexactFloat64())
	s.Nil(grant.State.ResolvedCostBasis.CostBasisID)

	settlementCurrency, err := grant.Intent.GetSettlementFiatCurrency()
	s.Require().NoError(err)
	s.Equal(USD, settlementCurrency.Details().Code)

	fiatAmount, err := grant.GetFiatSettlementAmount()
	s.Require().NoError(err)
	s.Equal(float64(25), fiatAmount.InexactFloat64())

	s.Equal(creditpurchase.StatusActivePaymentPending, grant.Status)
	s.Require().NotNil(grant.Realizations.CreditGrantRealization)
	s.Equal(float64(100), s.MustCustomerFBOBalance(cust.GetID(), TOKENS, mo.None[*alpacadecimal.Decimal]()).InexactFloat64())

	grant, err = s.CreditGrantService.UpdateExternalSettlement(ctx, creditgrant.UpdateExternalSettlementInput{
		Namespace:    ns,
		CustomerID:   cust.ID,
		ChargeID:     grant.ID,
		TargetStatus: payment.StatusSettled,
	})
	s.Require().NoError(err)
	s.Equal(creditpurchase.StatusFinal, grant.Status)
	s.Equal(payment.StatusSettled, grant.Realizations.ExternalPaymentSettlement.Status)
	s.Equal(float64(100), s.MustCustomerFBOBalance(cust.GetID(), TOKENS, mo.None[*alpacadecimal.Decimal]()).InexactFloat64())
}

func (s *CreditGrantTestSuite) TestCreateCustomCurrencyInvoiceGrantWithDynamicCostBasis() {
	// given:
	// - a custom currency with a USD cost basis effective before the grant
	// - an invoice funded grant with a dynamic USD cost basis
	// when:
	// - the grant is created
	// then:
	// - the cost basis resolves to the effective currency cost basis
	// - the standard invoice line is priced in USD from the resolved rate
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("creditgrant-custom-currency-invoice")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)
	tokens := s.createTokensCurrency(ns)

	rate := alpacadecimal.NewFromFloat(0.5)
	effectiveFrom := clock.Now().Add(time.Hour)
	currencyCostBasis, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:     ns,
		CurrencyID:    tokens.ID,
		FiatCode:      USD,
		Rate:          rate,
		EffectiveFrom: lo.ToPtr(effectiveFrom),
	})
	s.Require().NoError(err)

	clock.FreezeTime(effectiveFrom.Add(24 * time.Hour))
	defer clock.UnFreeze()

	usd, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)

	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "40 TOKENS at the current rate",
		Currency:      TOKENS,
		Amount:        alpacadecimal.NewFromInt(40),
		FundingMethod: creditgrant.FundingMethodInvoice,
		Purchase: &creditgrant.PurchaseTerms{
			Currency:  USD,
			CostBasis: lo.ToPtr(creditpurchase.NewCostBasis(costbasis.NewIntent(costbasis.DynamicIntent{FiatCurrency: usd}))),
		},
	})
	s.Require().NoError(err)

	s.Equal(creditpurchase.SettlementTypeInvoice, grant.Intent.Settlement.Type())
	s.Equal(creditpurchase.StatusActivePaymentPending, grant.Status)
	s.Require().NotNil(grant.State.ResolvedCostBasis)
	s.Equal(rate.InexactFloat64(), grant.State.ResolvedCostBasis.CostBasis.InexactFloat64())
	s.Equal(currencyCostBasis.ID, lo.FromPtr(grant.State.ResolvedCostBasis.CostBasisID))

	fiatAmount, err := grant.GetFiatSettlementAmount()
	s.Require().NoError(err)
	s.Equal(float64(20), fiatAmount.InexactFloat64())

	standardInvoices, err := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
		Namespace: ns,
		Expand:    billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Require().Len(standardInvoices.Items, 1)

	invoice := standardInvoices.Items[0]
	s.Equal(currencyx.Code(invoice.Currency), USD)
	s.Require().Len(invoice.Lines.OrEmpty(), 1)
	s.Equal(grant.ID, *invoice.Lines.OrEmpty()[0].ChargeID)
	s.Equal(float64(20), invoice.Lines.OrEmpty()[0].Totals.Total.InexactFloat64())
}

func (s *CreditGrantTestSuite) TestCreateFiatGrantWithManualCostBasis() {
	// given:
	// - a USD grant priced through a fiat cost basis rate
	// when:
	// - the grant is created
	// then:
	// - the rate is stored as the scalar fiat cost basis of the purchase
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("creditgrant-fiat-manual-cost-basis")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	grant, err := s.CreditGrantService.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "$100.00 for $50.00",
		Currency:      USD,
		Amount:        alpacadecimal.NewFromInt(100),
		FundingMethod: creditgrant.FundingMethodExternal,
		Purchase: &creditgrant.PurchaseTerms{
			Currency:           USD,
			CostBasis:          lo.ToPtr(creditpurchase.NewCostBasis(creditpurchase.FiatCostBasis{Rate: alpacadecimal.NewFromFloat(0.5)})),
			AvailabilityPolicy: lo.ToPtr(creditpurchase.CreatedInitialPaymentSettlementStatus),
		},
	})
	s.Require().NoError(err)

	s.Equal(creditpurchase.CostBasisTypeFiat, grant.Intent.CostBasis.Type())
	s.Nil(grant.State.ChargeCostBasisID)
	s.Require().NotNil(grant.State.ResolvedCostBasis)
	s.Equal(0.5, grant.State.ResolvedCostBasis.CostBasis.InexactFloat64())

	fiatAmount, err := grant.GetFiatSettlementAmount()
	s.Require().NoError(err)
	s.Equal(float64(50), fiatAmount.InexactFloat64())
}

func (s *CreditGrantTestSuite) TestCreateCustomCurrencyGrantRejectedWhenDisabled() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("creditgrant-custom-currency-disabled")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	_ = s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())
	s.createTokensCurrency(ns)

	svc, err := creditgrantservice.New(creditgrantservice.Config{
		CreditPurchaseService: s.CreditPurchaseService,
		ChargesService:        s.Charges,
		BillingService:        s.BillingService,
		CustomerService:       s.CustomerService,
		CreditVoidService:     creditvoid.NewNoopService(),
		TransactionManager:    enttx.NewCreator(s.DBClient),
		CurrencyResolver:      s.CurrencyResolver,
		CreditsConfig:         config.CreditsConfiguration{EnableCustomCurrencyCharge: false},
	})
	s.Require().NoError(err)

	_, err = svc.Create(ctx, creditgrant.CreateInput{
		Namespace:     ns,
		CustomerID:    cust.ID,
		Name:          "Promotional TOKENS",
		Currency:      TOKENS,
		Amount:        alpacadecimal.NewFromInt(10),
		FundingMethod: creditgrant.FundingMethodNone,
	})
	s.Require().ErrorIs(err, meta.ErrCustomCurrencyNotSupported)
}
