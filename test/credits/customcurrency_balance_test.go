package credits

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *CustomCurrencyCreditsSuite) TestBalanceAndTransactionsPreserveCustomCurrencyIdentity() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("custom-currency-balance-history")
	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	customer := s.CreateLedgerBackedCustomer(ns, "custom-currency-balance-history")
	tokens := s.CreateCustomCurrency(ns, "TOKENS")
	points := s.CreateCustomCurrency(ns, "POINTS")

	createdAt := datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime()
	chargeAt := createdAt.Add(time.Hour)
	servicePeriod := timeutil.ClosedPeriod{
		From: chargeAt,
		To:   chargeAt.Add(30 * 24 * time.Hour),
	}
	clock.FreezeTime(createdAt)
	defer clock.UnFreeze()

	// given:
	// - 100 TOKENS are available now
	// - 50 POINTS are granted now but become available later
	s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace:  ns,
		Customer:   customer.GetID(),
		Currency:   tokens,
		Amount:     alpacadecimal.NewFromInt(100),
		At:         createdAt,
		Name:       "TOKENS funding",
		Settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})
	s.createCustomCurrencyCreditPurchase(ctx, customCurrencyCreditPurchaseInput{
		Namespace:  ns,
		Customer:   customer.GetID(),
		Currency:   points,
		Amount:     alpacadecimal.NewFromInt(50),
		At:         servicePeriod.To,
		Name:       "future POINTS funding",
		Settlement: creditpurchase.NewSettlement(creditpurchase.PromotionalSettlement{}),
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.CreditGrantTaxCodeID,
		},
	})

	// and:
	// - a 30 TOKENS in-advance fee consumes credit when its service starts
	charge := s.createCustomCurrencyFlatFeeCharge(ctx, customCurrencyFlatFeeChargeInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		ServicePeriod:  servicePeriod,
		InvoiceAt:      chargeAt,
		Amount:         alpacadecimal.NewFromInt(30),
		PaymentTerm:    productcatalog.InAdvancePaymentTerm,
		SettlementMode: productcatalog.CreditOnlySettlementMode,
		Name:           "TOKENS platform fee",
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.InvoicingTaxCodeID,
		},
	})
	s.Equal(flatfee.StatusCreated, charge.Status)

	clock.FreezeTime(chargeAt)
	advanced, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customer.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(advanced, 1)
	flatFee, err := advanced[0].AsFlatFeeCharge()
	s.Require().NoError(err)
	s.Equal(flatfee.StatusFinal, flatFee.Status)

	// when:
	// - the balance view is read without a currency filter
	facade, err := customerbalance.NewFacade(s.CustomerBalanceSvc)
	s.Require().NoError(err)
	balances, err := facade.GetBalances(ctx, customerbalance.GetBalancesInput{
		CustomerID: customer.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(balances, 2)

	// then:
	// - each custom currency exposes its managed ID and keeps settled, live, and
	//   pending amounts in its own identity
	balancesByID := lo.SliceToMap(balances, func(balance customerbalance.BalanceByCurrency) (string, customerbalance.BalanceByCurrency) {
		s.Require().NotNil(balance.CustomCurrencyID)
		return *balance.CustomCurrencyID, balance
	})
	s.Require().Contains(balancesByID, tokens.ID)
	s.Equal(float64(70), balancesByID[tokens.ID].Balance.Settled().InexactFloat64())
	s.Equal(float64(70), balancesByID[tokens.ID].Balance.Live().InexactFloat64())
	s.Equal(float64(0), balancesByID[tokens.ID].Balance.Pending().InexactFloat64())
	s.Require().Contains(balancesByID, points.ID)
	s.Equal(float64(0), balancesByID[points.ID].Balance.Settled().InexactFloat64())
	s.Equal(float64(0), balancesByID[points.ID].Balance.Live().InexactFloat64())
	s.Equal(float64(50), balancesByID[points.ID].Balance.Pending().InexactFloat64())

	// when:
	// - TOKENS history is listed by its display code
	tokenCode := tokens.GetCode()
	history, err := s.CustomerBalanceSvc.ListCreditTransactions(ctx, customerbalance.ListCreditTransactionsInput{
		CustomerID:    customer.GetID(),
		Limit:         10,
		Currency:      &tokenCode,
		AsOf:          &chargeAt,
		FeatureFilter: customerbalance.AllFeatureFilter(),
	})
	s.Require().NoError(err)
	s.Require().Len(history.Items, 2)

	// then:
	// - both rows identify TOKENS by managed ID and reconstruct one balance chain
	consumed := history.Items[0]
	s.Equal(customerbalance.CreditTransactionTypeConsumed, consumed.Type)
	s.Equal(float64(-30), consumed.Amount.InexactFloat64())
	s.Equal(float64(100), consumed.Balance.Before.InexactFloat64())
	s.Equal(float64(70), consumed.Balance.After.InexactFloat64())
	s.Require().NotNil(consumed.CustomCurrencyID)
	s.Equal(tokens.ID, *consumed.CustomCurrencyID)

	funded := history.Items[1]
	s.Equal(customerbalance.CreditTransactionTypeFunded, funded.Type)
	s.Equal(float64(100), funded.Amount.InexactFloat64())
	s.Equal(float64(0), funded.Balance.Before.InexactFloat64())
	s.Equal(float64(100), funded.Balance.After.InexactFloat64())
	s.Require().NotNil(funded.CustomCurrencyID)
	s.Equal(tokens.ID, *funded.CustomCurrencyID)

	// A code filter remains intentionally code-wide; it does not rewrite the
	// immutable identity carried by each returned row.
	s.Equal(tokens.GetCode(), consumed.Currency)
	s.Equal(tokens.GetCode(), funded.Currency)
}

func (s *CustomCurrencyCreditsSuite) TestBalanceDiscoversCustomCurrencyWithoutStartingCredits() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("custom-currency-balance-discovery")
	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	customer := s.CreateLedgerBackedCustomer(ns, "custom-currency-balance-discovery")
	tokens := s.CreateCustomCurrency(ns, "TOKENS")

	createdAt := datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime()
	chargeAt := createdAt.Add(time.Hour)
	servicePeriod := timeutil.ClosedPeriod{
		From: chargeAt,
		To:   chargeAt.Add(30 * 24 * time.Hour),
	}
	clock.FreezeTime(createdAt)
	defer clock.UnFreeze()

	// given:
	// - a 30 TOKENS credit-only charge exists without any starting credit
	charge := s.createCustomCurrencyFlatFeeCharge(ctx, customCurrencyFlatFeeChargeInput{
		Namespace:      ns,
		Customer:       customer.GetID(),
		Currency:       tokens,
		ServicePeriod:  servicePeriod,
		InvoiceAt:      chargeAt,
		Amount:         alpacadecimal.NewFromInt(30),
		PaymentTerm:    productcatalog.InAdvancePaymentTerm,
		SettlementMode: productcatalog.CreditOnlySettlementMode,
		Name:           "uncovered TOKENS platform fee",
		TaxConfig: productcatalog.TaxCodeConfig{
			TaxCodeID: defaults.InvoicingTaxCodeID,
		},
	})
	s.Equal(flatfee.StatusCreated, charge.Status)

	facade, err := customerbalance.NewFacade(s.CustomerBalanceSvc)
	s.Require().NoError(err)
	clock.FreezeTime(chargeAt)

	// when:
	// - balances are listed after the charge starts but before it is booked
	balances, err := facade.GetBalances(ctx, customerbalance.GetBalancesInput{
		CustomerID: customer.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(balances, 1)
	s.Require().NotNil(balances[0].CustomCurrencyID)
	s.Equal(tokens.ID, *balances[0].CustomCurrencyID)
	s.Equal(float64(0), balances[0].Balance.Settled().InexactFloat64())
	s.Equal(float64(-30), balances[0].Balance.Live().InexactFloat64())

	// then:
	// - after finalization, the receivable-backed advance keeps the currency
	//   discoverable even though no FBO credit has ever existed
	advanced, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customer.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(advanced, 1)

	balances, err = facade.GetBalances(ctx, customerbalance.GetBalancesInput{
		CustomerID: customer.GetID(),
	})
	s.Require().NoError(err)
	s.Require().Len(balances, 1)
	s.Require().NotNil(balances[0].CustomCurrencyID)
	s.Equal(tokens.ID, *balances[0].CustomCurrencyID)
	s.Equal(float64(-30), balances[0].Balance.Settled().InexactFloat64())
	s.Equal(float64(-30), balances[0].Balance.Live().InexactFloat64())
}
