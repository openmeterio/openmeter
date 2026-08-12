package service

import (
	"context"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	creditpurchaseservice "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	dbchargecreditpurchase "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchase"
	dbchargecreditpurchasecostbasis "github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchasecostbasis"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *CreditPurchaseTestSuite) TestInvoiceCreditPurchaseResolvesInitialCostBasis() {
	// given:
	// - manual and pinned custom-currency cost-basis intents
	// when:
	// - invoice-settled purchases are created
	// then:
	// - each initial resolution, gathering line, and durable charge state agree
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

	tests := []struct {
		name                      string
		intent                    costbasis.Intent
		expectedRate              alpacadecimal.Decimal
		expectedCurrencyCostBasis *string
		expectedFiatAmount        alpacadecimal.Decimal
	}{
		{
			name: "manual",
			intent: costbasis.NewIntent(costbasis.ManualIntent{
				FiatCurrency: fiatCurrency,
				Rate:         alpacadecimal.NewFromFloat(0.5),
			}),
			expectedRate:       alpacadecimal.NewFromFloat(0.5),
			expectedFiatAmount: alpacadecimal.NewFromInt(50),
		},
		{
			name: "pinned",
			intent: costbasis.NewIntent(costbasis.PinnedIntent{
				FiatCurrency:        fiatCurrency,
				CurrencyCostBasisID: pinnedCostBasis.ID,
			}),
			expectedRate:              pinnedCostBasis.Rate,
			expectedCurrencyCostBasis: lo.ToPtr(pinnedCostBasis.ID),
			expectedFiatAmount:        alpacadecimal.NewFromInt(25),
		},
	}

	var (
		referencedChargeID        string
		referencedChargeCostBasis string
	)

	for _, test := range tests {
		s.Run(test.name, func() {
			// This subtest verifies the mode-specific initial resolution while
			// sharing the invoice-line and persistence contract across both modes.
			intent := newCustomCurrencyInvoiceCreditPurchaseIntent(
				customer.ID,
				customCurrency,
				defaults.CreditGrantTaxCodeID,
				test.intent,
			)

			created, err := s.Charges.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
				Namespace: namespace,
				Intent:    intent,
			})
			s.Require().NoError(err)
			s.Require().NotNil(created.Charge.State.ChargeCostBasisID)
			s.Require().NotNil(created.Charge.State.ResolvedCostBasis)
			s.Require().Equal(
				test.expectedRate.InexactFloat64(),
				created.Charge.State.ResolvedCostBasis.CostBasis.InexactFloat64(),
			)
			s.Equal(test.expectedCurrencyCostBasis, created.Charge.State.ResolvedCostBasis.CostBasisID)
			s.Equal(now.Truncate(time.Microsecond), created.Charge.State.ResolvedCostBasis.ResolvedAt.Truncate(time.Microsecond))

			s.Require().NotNil(created.GatheringLineToCreate)
			s.Equal(currencyx.FiatCode(currency.USD), created.GatheringLineToCreate.Currency)
			price, err := created.GatheringLineToCreate.Price.AsFlat()
			s.Require().NoError(err)
			s.Require().Equal(test.expectedFiatAmount.InexactFloat64(), price.Amount.InexactFloat64())

			persisted, err := s.Charges.creditPurchaseService.GetByIDs(ctx, creditpurchase.GetByIDsInput{
				Namespace: namespace,
				IDs:       []string{created.Charge.ID},
			})
			s.Require().NoError(err)
			s.Require().Len(persisted, 1)
			s.Equal(created.Charge.State.ChargeCostBasisID, persisted[0].State.ChargeCostBasisID)
			s.Require().NotNil(persisted[0].State.ResolvedCostBasis)
			s.Require().Equal(
				created.Charge.State.ResolvedCostBasis.CostBasis.InexactFloat64(),
				persisted[0].State.ResolvedCostBasis.CostBasis.InexactFloat64(),
			)
			s.Equal(created.Charge.State.ResolvedCostBasis.CostBasisID, persisted[0].State.ResolvedCostBasis.CostBasisID)
			s.Equal(
				created.Charge.State.ResolvedCostBasis.ResolvedAt.Truncate(time.Microsecond),
				persisted[0].State.ResolvedCostBasis.ResolvedAt,
			)

			if test.intent.Kind() == costbasis.ModePinned {
				referencedChargeID = created.Charge.ID
				referencedChargeCostBasis = *created.Charge.State.ChargeCostBasisID
			}
		})
	}

	s.Run("referenced charge cost basis cannot be deleted", func() {
		// This subtest verifies that a charge-owned cost-basis record cannot be
		// removed while the financial charge still references it.
		s.Require().NotEmpty(referencedChargeID)
		s.Require().NotEmpty(referencedChargeCostBasis)

		err := s.DBClient.ChargeCreditPurchaseCostBasis.DeleteOneID(referencedChargeCostBasis).Exec(ctx)
		s.Require().Error(err)

		persisted, err := s.Charges.creditPurchaseService.GetByIDs(ctx, creditpurchase.GetByIDsInput{
			Namespace: namespace,
			IDs:       []string{referencedChargeID},
		})
		s.Require().NoError(err)
		s.Require().Len(persisted, 1)
	})
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

func (s *CreditPurchaseTestSuite) TestSetResolvedDynamicCostBasisOverwritesExistingState() {
	// given:
	// - a dynamically resolved credit-purchase cost basis and its owning charge
	// when:
	// - the service persists a different resolution and another charge attempts an update
	// then:
	// - the latest owner resolution persists and the unrelated charge is rejected
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("credit-purchase-dynamic-cost-basis-retry")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "credit-purchase-dynamic-cost-basis-retry")
	customCurrency := s.createTestCustomCurrency(ctx, namespace)
	servicePeriodFrom := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(servicePeriodFrom)
	defer clock.UnFreeze()

	currencyCostBasis, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
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
		costbasis.NewIntent(costbasis.DynamicIntent{FiatCurrency: fiatCurrency}),
	)
	intent.Settlement = creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{
		InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
	})

	defer s.CreditPurchaseTestHandler.Reset()
	initiatedCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
	s.CreditPurchaseTestHandler.onCreditPurchaseInitiated = initiatedCallback.Handler(s.T())
	lineageMock := &mockLineageService{Service: s.LineageService}
	lineageMock.On("BackfillAdvanceLineageSegments", mock.Anything, mock.Anything).
		Return(nil).
		Once()
	creditPurchaseService, err := creditpurchaseservice.New(creditpurchaseservice.Config{
		Adapter:     s.CreditPurchaseAdapter,
		Handler:     s.CreditPurchaseTestHandler,
		Lineage:     lineageMock,
		MetaAdapter: s.MetaAdapter,
		Currencies:  s.CurrencyService,
	})
	s.Require().NoError(err)

	created, err := creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
		Namespace: namespace,
		Intent:    intent,
	})
	s.Require().NoError(err)
	s.Require().NotNil(created.Charge.State.ChargeCostBasisID)
	s.Require().NotNil(created.Charge.State.ResolvedCostBasis)
	lineageMock.AssertExpectations(s.T())

	replacement := costbasis.State{
		CostBasis:   alpacadecimal.NewFromInt(9),
		CostBasisID: lo.ToPtr(currencyCostBasis.ID),
		ResolvedAt:  servicePeriodFrom.Add(time.Hour),
	}
	updated, err := s.CreditPurchaseAdapter.SetResolvedCostBasis(ctx, creditpurchase.SetResolvedCostBasisAdapterInput{
		ChargeID:          created.Charge.GetChargeID(),
		ChargeCostBasisID: *created.Charge.State.ChargeCostBasisID,
		State:             replacement,
	})
	s.Require().NoError(err)
	s.Require().Equal(replacement.CostBasis.InexactFloat64(), updated.CostBasis.InexactFloat64())
	s.Equal(replacement.CostBasisID, updated.CostBasisID)
	s.Equal(replacement.ResolvedAt, updated.ResolvedAt)

	wrongChargeID := created.Charge.GetChargeID()
	wrongChargeID.ID = "another-charge"
	_, err = s.CreditPurchaseAdapter.SetResolvedCostBasis(ctx, creditpurchase.SetResolvedCostBasisAdapterInput{
		ChargeID:          wrongChargeID,
		ChargeCostBasisID: *created.Charge.State.ChargeCostBasisID,
		State: costbasis.State{
			CostBasis:   alpacadecimal.NewFromInt(10),
			CostBasisID: lo.ToPtr(currencyCostBasis.ID),
			ResolvedAt:  servicePeriodFrom.Add(2 * time.Hour),
		},
	})
	s.Require().ErrorContains(err, "not found")

	persisted, err := creditPurchaseService.GetByIDs(ctx, creditpurchase.GetByIDsInput{
		Namespace: namespace,
		IDs:       []string{created.Charge.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(persisted, 1)
	s.Require().NotNil(persisted[0].State.ResolvedCostBasis)
	s.Require().Equal(
		replacement.CostBasis.InexactFloat64(),
		persisted[0].State.ResolvedCostBasis.CostBasis.InexactFloat64(),
	)
	s.Equal(replacement.CostBasisID, persisted[0].State.ResolvedCostBasis.CostBasisID)
	s.Equal(replacement.ResolvedAt, persisted[0].State.ResolvedCostBasis.ResolvedAt)
}

func (s *CreditPurchaseTestSuite) TestInvoiceCreditPurchaseLineEnginePreviewsUnresolvedCostBasisAsZero() {
	// given: a dynamic invoice credit purchase that has not entered its invoice lifecycle
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("invoice-credit-purchase-dynamic-cost-basis")
	defaults := s.ProvisionDefaultTaxCodes(ctx, namespace)
	customer := s.CreateTestCustomer(namespace, "invoice-credit-purchase-dynamic-cost-basis")
	customCurrency := s.createTestCustomCurrency(ctx, namespace)
	servicePeriodFrom := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(servicePeriodFrom)
	defer clock.UnFreeze()

	_, err := s.CurrencyService.CreateCostBasis(ctx, currencies.CreateCostBasisInput{
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
		costbasis.NewIntent(costbasis.DynamicIntent{FiatCurrency: fiatCurrency}),
	)

	created, err := s.Charges.creditPurchaseService.Create(ctx, creditpurchase.CreateInput{
		Namespace: namespace,
		Intent:    intent,
	})
	s.Require().NoError(err)
	s.Require().NotNil(created.Charge.State.ChargeCostBasisID)
	s.Nil(created.Charge.State.ResolvedCostBasis)
	s.Require().NotNil(created.GatheringLineToCreate)

	gatheringLine := *created.GatheringLineToCreate
	gatheringLine.ManagedResource = models.NewManagedResource(models.ManagedResourceInput{
		ID:        "gathering-line-1",
		Namespace: namespace,
		CreatedAt: servicePeriodFrom,
		UpdatedAt: servicePeriodFrom,
		Name:      gatheringLine.Name,
	})
	invoice := billing.StandardInvoice{}
	invoice.ID = "invoice-1"
	invoice.Namespace = namespace
	lineInput := billing.BuildStandardInvoiceLinesInput{
		Invoice:        invoice,
		GatheringLines: billing.GatheringLines{gatheringLine},
	}
	lineEngine := s.Charges.creditPurchaseService.GetLineEngine()

	// when: the side-effect-free line preview materializes the unresolved gathering line
	lines, err := lineEngine.BuildStandardLinesForGatheringPreview(ctx, lineInput)
	s.Require().NoError(err)

	// then: it preserves the temporary zero value without resolving the charge
	s.Require().Len(lines, 1)
	price, err := lines[0].UsageBased.Price.AsFlat()
	s.Require().NoError(err)
	s.Require().Equal(float64(0), price.Amount.InexactFloat64())
	s.Empty(lines[0].DetailedLines)
	persisted, err := s.Charges.creditPurchaseService.GetByIDs(ctx, creditpurchase.GetByIDsInput{
		Namespace: namespace,
		IDs:       []string{created.Charge.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(persisted, 1)
	s.Nil(persisted[0].State.ResolvedCostBasis)
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
		CostBasis: creditpurchase.NewCostBasis(costBasisIntent),
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
