package service

import (
	"context"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	dbchargecreditpurchasecostbasis "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchasecostbasis"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *CreditPurchaseTestSuite) TestInvoiceCreditPurchaseResolvesPinnedCostBasis() {
	// given:
	// - a custom-currency credit purchase pinned to a durable USD cost basis
	// when:
	// - the invoice-settled purchase is created
	// then:
	// - the pinned rate and source are persisted before the gathering line is priced
	ctx := s.T().Context()
	now := time.Date(2026, time.January, 15, 12, 0, 0, 123456789, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	namespace := s.GetUniqueNamespace("credit-purchase-pinned-cost-basis")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "credit-purchase-pinned-cost-basis")
	customCurrency := s.createTestCustomCurrency(ctx, namespace)
	pinnedCostBasis, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: customCurrency.ID,
		FiatCode:   currencyx.Code(currency.USD),
		Rate:       alpacadecimal.NewFromFloat(0.25),
	})
	s.Require().NoError(err)

	fiatCurrency, err := currencyx.NewFiatCurrency(currency.USD)
	s.Require().NoError(err)
	intent := newCustomCurrencyInvoiceCreditPurchaseIntent(
		customer.ID,
		customCurrency,
		defaults.CreditGrantTaxCodeID,
		costbasis.NewIntent(costbasis.PinnedIntent{
			FiatCurrency:        fiatCurrency,
			CurrencyCostBasisID: pinnedCostBasis.ID,
		}),
	)

	created, err := s.Charges.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
		Namespace: namespace,
		Intent:    intent,
	})
	s.Require().NoError(err)
	s.Require().NotNil(created.Charge.State.ResolvedCostBasis)
	s.Equal(float64(0.25), created.Charge.State.ResolvedCostBasis.CostBasis.InexactFloat64())
	s.Equal(pinnedCostBasis.ID, lo.FromPtr(created.Charge.State.ResolvedCostBasis.CostBasisID))
	s.Equal(now.Truncate(time.Microsecond), created.Charge.State.ResolvedCostBasis.ResolvedAt.Truncate(time.Microsecond))
	s.Equal(created.Charge.CreatedAt.Truncate(time.Microsecond), created.Charge.State.ResolvedCostBasis.ResolvedAt.Truncate(time.Microsecond))

	s.Require().NotNil(created.GatheringLineToCreate)
	s.Equal(currencyx.FiatCode(currency.USD), created.GatheringLineToCreate.Currency)
	price, err := created.GatheringLineToCreate.Price.AsFlat()
	s.Require().NoError(err)
	s.Equal(float64(25), price.Amount.InexactFloat64())

	persisted, err := s.Charges.creditPurchaseService.GetByIDs(ctx, creditpurchase.GetByIDsInput{
		Namespace: namespace,
		IDs:       []string{created.Charge.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(persisted, 1)
	s.Require().NotNil(persisted[0].State.ResolvedCostBasis)
	s.Equal(created.Charge.State.ResolvedCostBasis.CostBasis, persisted[0].State.ResolvedCostBasis.CostBasis)
	s.Equal(created.Charge.State.ResolvedCostBasis.CostBasisID, persisted[0].State.ResolvedCostBasis.CostBasisID)
	s.Equal(
		created.Charge.State.ResolvedCostBasis.ResolvedAt.Truncate(time.Microsecond),
		persisted[0].State.ResolvedCostBasis.ResolvedAt,
	)
}

func (s *CreditPurchaseTestSuite) TestPinnedCreditPurchaseCostBasisMustMatchCurrencyAndFiatCurrency() {
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("credit-purchase-pinned-cost-basis-mismatch")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "credit-purchase-pinned-cost-basis-mismatch")
	customCurrency := s.createTestCustomCurrency(ctx, namespace)
	otherCurrency, err := s.CurrencyService.CreateCurrency(ctx, currencies.CreateCurrencyInput{
		Namespace: namespace,
		CurrencyDetails: currencyx.CurrencyDetails{
			Code:               "OTHER",
			Name:               "Other",
			Symbol:             "O",
			Precision:          3,
			DecimalMark:        ".",
			ThousandsSeparator: ",",
		},
	})
	s.Require().NoError(err)

	otherCurrencyCostBasis, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: otherCurrency.ID,
		FiatCode:   currencyx.Code(currency.USD),
		Rate:       alpacadecimal.NewFromInt(1),
	})
	s.Require().NoError(err)
	eurCostBasis, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
		Namespace:  namespace,
		CurrencyID: customCurrency.ID,
		FiatCode:   currencyx.Code(currency.EUR),
		Rate:       alpacadecimal.NewFromInt(1),
	})
	s.Require().NoError(err)

	usd, err := currencyx.NewFiatCurrency(currency.USD)
	s.Require().NoError(err)
	tests := []struct {
		name          string
		fiatCurrency  *currencyx.FiatCurrency
		costBasisID   string
		expectedError string
	}{
		{
			name:          "custom currency mismatch",
			fiatCurrency:  usd,
			costBasisID:   otherCurrencyCostBasis.ID,
			expectedError: "currency cost basis currency mismatch",
		},
		{
			name:          "fiat currency mismatch",
			fiatCurrency:  usd,
			costBasisID:   eurCostBasis.ID,
			expectedError: "currency cost basis fiat currency mismatch",
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			// given:
			// - a pinned source that disagrees with the purchased or settlement currency
			// when:
			// - the credit purchase is created
			// then:
			// - resolution fails before either charge row is persisted
			intent := newCustomCurrencyInvoiceCreditPurchaseIntent(
				customer.ID,
				customCurrency,
				defaults.CreditGrantTaxCodeID,
				costbasis.NewIntent(costbasis.PinnedIntent{
					FiatCurrency:        test.fiatCurrency,
					CurrencyCostBasisID: test.costBasisID,
				}),
			)

			_, err := s.Charges.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
				Namespace: namespace,
				Intent:    intent,
			})
			s.Require().ErrorContains(err, test.expectedError)
		})
	}

	s.requireNoPersistedCreditPurchases(ctx, namespace)
}

func (s *CreditPurchaseTestSuite) TestCreditPurchaseRejectsDynamicCostBasisBeforePersistence() {
	// given:
	// - a custom-currency credit purchase with dynamic cost-basis intent
	// when:
	// - the purchase is created before dynamic lifecycle resolution exists
	// then:
	// - creation is rejected without persisting a charge or cost-basis row
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("credit-purchase-dynamic-cost-basis")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "credit-purchase-dynamic-cost-basis")
	customCurrency := s.createTestCustomCurrency(ctx, namespace)
	fiatCurrency, err := currencyx.NewFiatCurrency(currency.USD)
	s.Require().NoError(err)
	intent := newCustomCurrencyInvoiceCreditPurchaseIntent(
		customer.ID,
		customCurrency,
		defaults.CreditGrantTaxCodeID,
		costbasis.NewIntent(costbasis.DynamicIntent{FiatCurrency: fiatCurrency}),
	)

	_, err = s.Charges.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
		Namespace: namespace,
		Intent:    intent,
	})
	s.Require().ErrorContains(err, "dynamic cost basis is not supported for credit purchases")

	s.requireNoPersistedCreditPurchases(ctx, namespace)
}

func newCustomCurrencyInvoiceCreditPurchaseIntent(
	customerID string,
	currency currencies.Currency,
	taxCodeID string,
	costBasisIntent costbasis.Intent,
) creditpurchase.Intent {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	}

	return creditpurchase.Intent{
		Intent: meta.Intent{
			ManagedBy:  billing.ManuallyManagedLine,
			CustomerID: customerID,
			Currency:   currency,
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: taxCodeID,
			},
		},
		IntentMutableFields: creditpurchase.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "Custom Currency Credit Purchase",
				ServicePeriod:     servicePeriod,
				BillingPeriod:     servicePeriod,
				FullServicePeriod: servicePeriod,
			},
			CreditAmount: alpacadecimal.NewFromInt(100),
			Settlement:   creditpurchase.NewInvoiceSettlement(),
		},
		CostBasis: lo.ToPtr(creditpurchase.NewCostBasis(costBasisIntent)),
	}
}

func (s *CreditPurchaseTestSuite) requireNoPersistedCreditPurchases(ctx context.Context, namespace string) {
	s.T().Helper()

	chargeCount, err := s.DBClient.ChargeCreditPurchase.Query().
		Where(dbchargecreditpurchase.NamespaceEQ(namespace)).
		Count(ctx)
	s.Require().NoError(err)
	s.Zero(chargeCount)

	costBasisCount, err := s.DBClient.ChargeCreditPurchaseCostBasis.Query().
		Where(dbchargecreditpurchasecostbasis.NamespaceEQ(namespace)).
		Count(ctx)
	s.Require().NoError(err)
	s.Zero(costBasisCount)
}
