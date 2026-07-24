package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestUsageBasedCharges(t *testing.T) {
	suite.Run(t, new(UsageBasedChargesTestSuite))
}

type UsageBasedChargesTestSuite struct {
	BaseSuite
}

func (s *UsageBasedChargesTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *UsageBasedChargesTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *UsageBasedChargesTestSuite) TestUsageBasedCustomCurrencyCreditThenInvoiceCreatesFiatOveragePlaceholder() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-custom-currency")
	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	customInvoicing := s.SetupCustomInvoicing(ns)
	customer := s.CreateTestCustomer(ns, "test-subject")
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID())

	feature := s.SetupApiRequestsTotalFeature(ctx, ns)
	customCurrency := s.createTestCustomCurrency(ctx, ns)
	fiatCurrency, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
	}
	price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(2),
	})
	costBasisIntent := costbasis.NewIntent(costbasis.DynamicIntent{
		FiatCurrency: fiatCurrency,
	})

	s.setUsageBasedCustomCurrencyEnabled(true)
	defer s.setUsageBasedCustomCurrencyEnabled(false)

	var charge usagebased.Charge

	s.Run("create custom currency credit then invoice charge", func() {
		// given:
		// - a custom-currency usage-based charge settled through a fiat invoice
		// when:
		// - the charge is created through the charge service
		// then:
		// - the usage-based charge is returned
		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				charges.NewChargeIntent(usagebased.Intent{
					Intent: meta.Intent{
						ManagedBy:         billing.SubscriptionManagedLine,
						UniqueReferenceID: lo.ToPtr("usage-based-custom-currency"),
						CustomerID:        customer.ID,
						Currency:          customCurrency,
						TaxConfig: productcatalog.TaxCodeConfig{
							TaxCodeID: defaults.InvoicingTaxCodeID,
						},
					},
					IntentMutableFields: usagebased.IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{
							Name:              "usage-based-custom-currency",
							ServicePeriod:     servicePeriod,
							FullServicePeriod: servicePeriod,
							BillingPeriod:     servicePeriod,
						},
						InvoiceAt: servicePeriod.To,
						Price:     *price,
					},
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					FeatureKey:     feature.Feature.Key,
					CostBasis:      &costBasisIntent,
				}),
			},
		})
		s.Require().NoError(err)
		s.Require().Len(created, 1)

		charge, err = created[0].AsUsageBasedCharge()
		s.Require().NoError(err)
	})

	s.Run("persist fiat overage gathering line placeholder", func() {
		// given:
		// - a custom-currency credit-then-invoice usage-based charge
		// when:
		// - its active gathering lines are loaded
		// then:
		// - billing has persisted a fiat overage placeholder owned by the usage-based charge engine
		lines := activeGatheringLinesForCharge(&s.BaseSuite, ns, customer.ID, charge.ID)
		s.Require().Len(lines, 1)

		line := lines[0]
		s.Equal(currencyx.FiatCode("USD"), line.Currency)
		s.True(line.Price.Equal(price))
		s.Equal(billing.LineEngineTypeChargeUsageBased, line.Engine)
		s.Require().NotNil(line.ChargeID)
		s.Equal(charge.ID, *line.ChargeID)
		s.Equal(
			billing.AnnotationValueReasonOveragePlaceholder,
			line.Annotations[billing.AnnotationKeyReason],
		)
	})
}

type customCurrencyCreditThenInvoiceRealizationVariant struct {
	invoiceAt                        time.Time
	expectedCollectionEnd            time.Time
	enableProgressiveBilling         bool
	expectedRunType                  usagebased.RealizationRunType
	expectedChargeStatusAfterPayment usagebased.Status
	expectRemainingGatheringLine     bool
}

func (s *UsageBasedChargesTestSuite) TestUsageBasedCustomCurrencyCreditThenInvoiceLifecycle() {
	s.runUsageBasedCustomCurrencyCreditThenInvoiceLifecycle(customCurrencyCreditThenInvoiceRealizationVariant{
		invoiceAt:                        datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime(),
		expectedCollectionEnd:            datetime.MustParseTimeInLocation(s.T(), "2025-02-02T00:01:00Z", time.UTC).AsTime(),
		expectedRunType:                  usagebased.RealizationRunTypeFinalRealization,
		expectedChargeStatusAfterPayment: usagebased.StatusFinal,
	})
}

func (s *UsageBasedChargesTestSuite) TestUsageBasedCustomCurrencyCreditThenInvoiceProgressiveLifecycle() {
	s.runUsageBasedCustomCurrencyCreditThenInvoiceLifecycle(customCurrencyCreditThenInvoiceRealizationVariant{
		invoiceAt:                        datetime.MustParseTimeInLocation(s.T(), "2025-01-16T00:00:00Z", time.UTC).AsTime(),
		expectedCollectionEnd:            datetime.MustParseTimeInLocation(s.T(), "2025-01-16T00:01:00Z", time.UTC).AsTime(),
		enableProgressiveBilling:         true,
		expectedRunType:                  usagebased.RealizationRunTypePartialInvoice,
		expectedChargeStatusAfterPayment: usagebased.StatusActive,
		expectRemainingGatheringLine:     true,
	})
}

func (s *UsageBasedChargesTestSuite) runUsageBasedCustomCurrencyCreditThenInvoiceLifecycle(
	realizationVariant customCurrencyCreditThenInvoiceRealizationVariant,
) {
	type runPhase struct {
		// usageAdded is the usage that becomes visible during this pass.
		usageAdded float64
		// creditsAllocated is the additional TOKENS allocation returned during this pass.
		creditsAllocated float64
		// expectRunTotals contains the realization run totals in TOKENS.
		expectRunTotals billingtest.ExpectedTotals
		// expectInvoiceTotals contains the invoice line totals in USD.
		expectInvoiceTotals billingtest.ExpectedTotals
	}

	type tc struct {
		name string
		skip string

		onRunCreated         runPhase
		onCollectionComplete runPhase

		expectLineDeleted    bool
		expectPaymentSettled bool
	}

	// setup:
	// - F1 is a metered feature owned by customer C1
	// - the billing profile has a 1 day collection period
	// - the charge covers [2025-01-01T00:00:00Z, 2025-02-01T00:00:00Z)
	// - usage is priced at 2 TOKENS per unit and settled with credit then invoice
	// - TOKENS use a manual USD cost basis of 0.5
	tests := []tc{
		// given:
		// - 5 metered units produce 10 TOKENS and no credits are allocated
		// when:
		// - the realization run is collected and invoiced
		// then:
		// - the full 10 TOKENS overage is settled as 5 USD
		{
			name: "happy path",
			onRunCreated: runPhase{
				usageAdded:          5,
				creditsAllocated:    0,
				expectRunTotals:     billingtest.ExpectedTotals{Amount: 10, Total: 10},
				expectInvoiceTotals: billingtest.ExpectedTotals{Amount: 5, Total: 5},
			},
			onCollectionComplete: runPhase{
				usageAdded:          0,
				creditsAllocated:    0,
				expectRunTotals:     billingtest.ExpectedTotals{Amount: 10, Total: 10},
				expectInvoiceTotals: billingtest.ExpectedTotals{Amount: 5, Total: 5},
			},
			expectPaymentSettled: true,
		},
		// given:
		// - 5 metered units produce 10 TOKENS and 2 TOKENS are allocated when the run is created
		// when:
		// - the realization run is collected and invoiced
		// then:
		// - the remaining 8 TOKENS overage is settled as 4 USD
		{
			name: "happy path with credit allocation",
			onRunCreated: runPhase{
				usageAdded:          5,
				creditsAllocated:    2,
				expectRunTotals:     billingtest.ExpectedTotals{Amount: 10, CreditsTotal: 2, Total: 8},
				expectInvoiceTotals: billingtest.ExpectedTotals{Amount: 4, Total: 4},
			},
			onCollectionComplete: runPhase{
				usageAdded:          0,
				creditsAllocated:    0,
				expectRunTotals:     billingtest.ExpectedTotals{Amount: 10, CreditsTotal: 2, Total: 8},
				expectInvoiceTotals: billingtest.ExpectedTotals{Amount: 4, Total: 4},
			},
			expectPaymentSettled: true,
		},
		// given:
		// - 5 metered units produce 10 TOKENS and 10 TOKENS are allocated when the run is created
		// when:
		// - the zero-overage run is collected and invoiced
		// then:
		// - the empty overage line is removed and no payment is booked
		{
			name: "fully covered by credits",
			skip: "TODO: delete the custom-currency overage line when credits reduce its fiat total to zero",
			onRunCreated: runPhase{
				usageAdded:          5,
				creditsAllocated:    10,
				expectRunTotals:     billingtest.ExpectedTotals{Amount: 10, CreditsTotal: 10},
				expectInvoiceTotals: billingtest.ExpectedTotals{},
			},
			onCollectionComplete: runPhase{
				usageAdded:          0,
				creditsAllocated:    0,
				expectRunTotals:     billingtest.ExpectedTotals{Amount: 10, CreditsTotal: 10},
				expectInvoiceTotals: billingtest.ExpectedTotals{},
			},
			expectLineDeleted: true,
		},
		// given:
		// - 5 metered units are initially covered by 10 TOKENS of allocated credits
		// when:
		// - 1 late metered unit becomes visible during collection
		// then:
		// - the resulting 2 TOKENS overage is settled as 1 USD
		{
			name: "fully covered by credits with late overage",
			onRunCreated: runPhase{
				usageAdded:          5,
				creditsAllocated:    10,
				expectRunTotals:     billingtest.ExpectedTotals{Amount: 10, CreditsTotal: 10},
				expectInvoiceTotals: billingtest.ExpectedTotals{},
			},
			onCollectionComplete: runPhase{
				usageAdded:          1,
				creditsAllocated:    0,
				expectRunTotals:     billingtest.ExpectedTotals{Amount: 12, CreditsTotal: 10, Total: 2},
				expectInvoiceTotals: billingtest.ExpectedTotals{Amount: 1, Total: 1},
			},
			expectPaymentSettled: true,
		},
		// given:
		// - no usage is visible when the run is created
		// when:
		// - 2 metered units become visible during collection and no credits are allocated
		// then:
		// - the resulting 4 TOKENS overage is settled as 2 USD
		{
			name: "collection usage is billed without initial usage",
			onRunCreated: runPhase{
				usageAdded:          0,
				creditsAllocated:    0,
				expectRunTotals:     billingtest.ExpectedTotals{},
				expectInvoiceTotals: billingtest.ExpectedTotals{},
			},
			onCollectionComplete: runPhase{
				usageAdded:          2,
				creditsAllocated:    0,
				expectRunTotals:     billingtest.ExpectedTotals{Amount: 4, Total: 4},
				expectInvoiceTotals: billingtest.ExpectedTotals{Amount: 2, Total: 2},
			},
			expectPaymentSettled: true,
		},
		// given:
		// - no usage is visible when the run is created
		// when:
		// - 2 metered units become visible during collection and 4 TOKENS are allocated
		// then:
		// - credits cover the full amount and the empty overage line is removed
		{
			name: "collection usage is fully covered by credits without initial usage",
			skip: "TODO: delete the custom-currency overage line when collection-time credits cover the full amount",
			onRunCreated: runPhase{
				usageAdded:          0,
				creditsAllocated:    0,
				expectRunTotals:     billingtest.ExpectedTotals{},
				expectInvoiceTotals: billingtest.ExpectedTotals{},
			},
			onCollectionComplete: runPhase{
				usageAdded:          2,
				creditsAllocated:    4,
				expectRunTotals:     billingtest.ExpectedTotals{Amount: 4, CreditsTotal: 4},
				expectInvoiceTotals: billingtest.ExpectedTotals{},
			},
			expectLineDeleted: true,
		},
		// given:
		// - no usage is visible when the run is created
		// when:
		// - collection completes without any usage becoming visible
		// then:
		// - the empty overage line is removed
		{
			name: "no usage deletes the overage line",
			skip: "TODO: delete the custom-currency overage line when no usage is realized",
			onRunCreated: runPhase{
				usageAdded:          0,
				creditsAllocated:    0,
				expectRunTotals:     billingtest.ExpectedTotals{},
				expectInvoiceTotals: billingtest.ExpectedTotals{},
			},
			onCollectionComplete: runPhase{
				usageAdded:          0,
				creditsAllocated:    0,
				expectRunTotals:     billingtest.ExpectedTotals{},
				expectInvoiceTotals: billingtest.ExpectedTotals{},
			},
			expectLineDeleted: true,
		},
	}

	// TODO: use the real lineage service once it supports custom currencies.
	lineageMock := &mockLineageService{Service: s.LineageService}
	lineageMock.On("CreateInitialLineages", mock.Anything, mock.Anything).
		Return(nil).
		Maybe()
	lineageMock.On("PersistCorrectionLineageSegments", mock.Anything, mock.Anything).
		Return(nil).
		Maybe()
	lineageMock.On("BackfillAdvanceLineageSegments", mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	customCurrencyUsageBasedService, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter:                 s.UsageBasedAdapter,
		Handler:                 s.UsageBasedTestHandler,
		Lineage:                 lineageMock,
		Locker:                  s.Locker,
		MetaAdapter:             s.MetaAdapter,
		InvoiceUpdater:          s.InvoiceUpdater,
		CustomerOverrideService: s.BillingService,
		FeatureService:          s.FeatureService,
		RatingService:           billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: s.UnitConfigEnabled}),
		Currencies:              s.CurrencyService,
		StreamingConnector:      s.MockStreamingConnector,
	})
	s.Require().NoError(err)

	originalUsageBasedService := s.Charges.usageBasedService
	s.Charges.usageBasedService = customCurrencyUsageBasedService
	s.Require().NoError(s.BillingService.DeregisterLineEngine(billing.LineEngineTypeChargeUsageBased))
	s.Require().NoError(s.BillingService.RegisterLineEngine(customCurrencyUsageBasedService.GetLineEngine()))
	defer func() {
		s.Charges.usageBasedService = originalUsageBasedService
		s.Require().NoError(s.BillingService.DeregisterLineEngine(billing.LineEngineTypeChargeUsageBased))
		s.Require().NoError(s.BillingService.RegisterLineEngine(originalUsageBasedService.GetLineEngine()))
	}()

	for _, test := range tests {
		s.Run(test.name, func() {
			if test.skip != "" {
				s.T().Skip(test.skip)
			}

			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("charges-service-usage-based-custom-currency-lifecycle")

			s.UsageBasedTestHandler.Reset()
			defer s.UsageBasedTestHandler.Reset()
			s.MockStreamingConnector.Reset()
			defer s.MockStreamingConnector.Reset()
			clock.UnFreeze()
			defer clock.UnFreeze()

			defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
			sandboxApp := s.InstallSandboxApp(s.T(), ns)
			customer := s.CreateTestCustomer(ns, "customer-c1")
			profileOptions := []billingtest.BillingProfileProvisionOption{
				billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P1D")),
				billingtest.WithManualApproval(),
			}
			if realizationVariant.enableProgressiveBilling {
				profileOptions = append(profileOptions, billingtest.WithProgressiveBilling())
			}
			_ = s.ProvisionBillingProfile(
				ctx,
				ns,
				sandboxApp.GetID(),
				profileOptions...,
			)

			feature := s.SetupApiRequestsTotalFeature(ctx, ns)
			defer feature.Cleanup()
			customCurrency := s.createTestCustomCurrency(ctx, ns)
			fiatCurrency, err := currencyx.NewFiatCurrency(USD)
			s.Require().NoError(err)

			createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
			servicePeriod := timeutil.ClosedPeriod{
				From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
				To:   datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime(),
			}
			usageAt := datetime.MustParseTimeInLocation(s.T(), "2025-01-15T00:00:00Z", time.UTC).AsTime()
			lateUsageAt := usageAt.Add(12 * time.Hour)
			lateUsageStoredAt := realizationVariant.invoiceAt.Add(-time.Second)

			s.setUsageBasedCustomCurrencyEnabled(true)
			defer s.setUsageBasedCustomCurrencyEnabled(false)
			clock.FreezeTime(createAt)

			var (
				chargeID meta.ChargeID
				invoice  billing.StandardInvoice

				customCurrencyOverageAccruedInvocations int
				authorizedCallback                      *countedLedgerTransactionCallback[usagebased.OnPaymentAuthorizedInput]
				settledCallback                         *countedLedgerTransactionCallback[usagebased.OnPaymentSettledInput]
			)

			s.Run("create realization run and fiat overage line", func() {
				costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
					FiatCurrency: fiatCurrency,
					Rate:         alpacadecimal.NewFromFloat(0.5),
				})
				price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				})

				created, err := s.Charges.Create(ctx, charges.CreateInput{
					Namespace: ns,
					Intents: []charges.ChargeIntent{
						charges.NewChargeIntent(usagebased.Intent{
							Intent: meta.Intent{
								ManagedBy:         billing.SubscriptionManagedLine,
								UniqueReferenceID: lo.ToPtr("usage-based-custom-currency-lifecycle"),
								CustomerID:        customer.ID,
								Currency:          customCurrency,
								TaxConfig: productcatalog.TaxCodeConfig{
									TaxCodeID: defaults.InvoicingTaxCodeID,
								},
							},
							IntentMutableFields: usagebased.IntentMutableFields{
								IntentMutableFields: meta.IntentMutableFields{
									Name:              "usage-based-custom-currency",
									ServicePeriod:     servicePeriod,
									FullServicePeriod: servicePeriod,
									BillingPeriod:     servicePeriod,
								},
								InvoiceAt: servicePeriod.To,
								Price:     *price,
							},
							SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
							FeatureKey:     feature.Feature.Key,
							CostBasis:      &costBasisIntent,
						}),
					},
				})
				s.Require().NoError(err)
				s.Require().Len(created, 1)

				charge, err := created[0].AsUsageBasedCharge()
				s.Require().NoError(err)
				chargeID = charge.GetChargeID()

				s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(test.onRunCreated.creditsAllocated)
				if test.onRunCreated.usageAdded > 0 {
					s.MockStreamingConnector.AddSimpleEvent(
						feature.Feature.Key,
						test.onRunCreated.usageAdded,
						usageAt,
						streamingtestutils.WithStoredAt(usageAt),
					)
				}

				clock.FreezeTime(realizationVariant.invoiceAt)
				invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
					Customer: customer.GetID(),
					AsOf:     lo.ToPtr(realizationVariant.invoiceAt),
				})
				s.Require().NoError(err)
				s.Require().Len(invoices, 1)
				invoice = invoices[0]
				s.Require().NotNil(invoice.CollectionAt)
				s.True(realizationVariant.expectedCollectionEnd.Equal(*invoice.CollectionAt))
				s.Require().Len(invoice.Lines.OrEmpty(), 1)
				s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
					line:               invoice.Lines.OrEmpty()[0],
					expectTokenOverage: test.onRunCreated.expectRunTotals.Total,
					expectCostBasis:    0.5,
					expectFiatTotals:   test.onRunCreated.expectInvoiceTotals,
				})

				charge = s.mustGetUsageBasedChargeByID(chargeID)
				s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, charge.Status)
				initialRun, err := charge.GetCurrentRealizationRun()
				s.Require().NoError(err)
				s.Equal(realizationVariant.expectedRunType, initialRun.Type)
				s.True(realizationVariant.invoiceAt.Equal(initialRun.ServicePeriodTo))
				s.Equal(test.onRunCreated.usageAdded, initialRun.MeteredQuantity.InexactFloat64())
				s.RequireTotals(test.onRunCreated.expectRunTotals, initialRun.Totals)
			})

			s.Run("collect realization run and settle fiat overage", func() {
				s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(test.onCollectionComplete.creditsAllocated)
				if test.onCollectionComplete.usageAdded > 0 {
					s.MockStreamingConnector.AddSimpleEvent(
						feature.Feature.Key,
						test.onCollectionComplete.usageAdded,
						lateUsageAt,
						streamingtestutils.WithStoredAt(lateUsageStoredAt),
					)
				}

				clock.FreezeTime(realizationVariant.expectedCollectionEnd)
				invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
				s.Require().NotNil(invoice.CollectionAt)
				s.True(realizationVariant.expectedCollectionEnd.Equal(*invoice.CollectionAt))

				charge := s.mustGetUsageBasedChargeByID(chargeID)
				s.Equal(usagebased.StatusActiveRealizationProcessing, charge.Status)
				collectedRun, err := charge.GetCurrentRealizationRun()
				s.Require().NoError(err)
				s.Equal(realizationVariant.expectedRunType, collectedRun.Type)
				s.Equal(test.onRunCreated.usageAdded+test.onCollectionComplete.usageAdded, collectedRun.MeteredQuantity.InexactFloat64())
				s.RequireTotals(test.onCollectionComplete.expectRunTotals, collectedRun.Totals)

				if test.expectLineDeleted {
					s.Empty(invoice.Lines.OrEmpty())
				} else {
					s.Require().Len(invoice.Lines.OrEmpty(), 1)
					s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
						line:               invoice.Lines.OrEmpty()[0],
						expectTokenOverage: test.onCollectionComplete.expectRunTotals.Total,
						expectCostBasis:    0.5,
						expectFiatTotals:   test.onCollectionComplete.expectInvoiceTotals,
					})
				}

				if test.expectPaymentSettled {
					s.UsageBasedTestHandler.onCustomCurrencyOverageAccrued = func(_ context.Context, input usagebased.OnCustomCurrencyOverageAccruedInput) (usagebased.OnCustomCurrencyOverageAccruedResult, error) {
						customCurrencyOverageAccruedInvocations++
						s.Equal(chargeID.ID, input.Charge.ID)
						s.Equal(test.onCollectionComplete.expectRunTotals.Total, input.GetCustomCurrencyAmountAccrued().InexactFloat64())

						resolvedCostBasis, err := input.GetCostBasis()
						s.Require().NoError(err)
						s.Equal(float64(0.5), resolvedCostBasis.InexactFloat64())

						resolvedFiatCurrency, err := input.GetFiatCurrency()
						s.Require().NoError(err)
						s.Equal(USD, resolvedFiatCurrency.Details().Code)

						return usagebased.OnCustomCurrencyOverageAccruedResult{
							TransactionGroup: ledgertransaction.GroupReference{
								TransactionGroupID: ulid.Make().String(),
							},
							TotalFiatAmount: alpacadecimal.NewFromFloat(test.onCollectionComplete.expectInvoiceTotals.Total),
						}, nil
					}

					authorizedCallback = newCountedLedgerTransactionCallback[usagebased.OnPaymentAuthorizedInput]()
					s.UsageBasedTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T(), func(_ *testing.T, input usagebased.OnPaymentAuthorizedInput) {
						s.Equal(chargeID.ID, input.Charge.ID)
						s.Equal(test.onCollectionComplete.expectInvoiceTotals.Total, input.FiatAmount.InexactFloat64())
					})

					settledCallback = newCountedLedgerTransactionCallback[usagebased.OnPaymentSettledInput]()
					s.UsageBasedTestHandler.onPaymentSettled = settledCallback.Handler(s.T(), func(_ *testing.T, input usagebased.OnPaymentSettledInput) {
						s.Equal(chargeID.ID, input.Charge.ID)
						s.Equal(test.onCollectionComplete.expectInvoiceTotals.Total, input.FiatAmount.InexactFloat64())
					})
				}

				invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)
				s.Require().NotNil(invoice.CollectionAt)
				s.True(realizationVariant.expectedCollectionEnd.Equal(*invoice.CollectionAt))
			})

			s.Run("reload realized charge and invoice state", func() {
				charge := s.mustGetUsageBasedChargeByID(chargeID)
				s.Equal(realizationVariant.expectedChargeStatusAfterPayment, charge.Status)
				s.Nil(charge.State.CurrentRealizationRunID)
				s.Require().Len(charge.Realizations, 1)
				realizedRun, ok := charge.Realizations.Latest()
				s.Require().True(ok)
				s.Equal(realizationVariant.expectedRunType, realizedRun.Type)
				s.Require().NotNil(realizedRun.InvoiceUsage)

				if test.expectPaymentSettled {
					s.Equal(1, customCurrencyOverageAccruedInvocations)
					s.Require().NotNil(authorizedCallback)
					s.Equal(1, authorizedCallback.nrInvocations)
					s.Require().NotNil(settledCallback)
					s.Equal(1, settledCallback.nrInvocations)
					s.Require().NotNil(realizedRun.Payment)
					s.Equal(payment.StatusSettled, realizedRun.Payment.Status)
					s.Equal(test.onCollectionComplete.expectInvoiceTotals.Total, realizedRun.Payment.FiatAmount.InexactFloat64())
					s.False(realizedRun.NoFiatTransactionRequired)
				} else {
					s.Zero(customCurrencyOverageAccruedInvocations)
					s.Nil(realizedRun.Payment)
					s.True(realizedRun.NoFiatTransactionRequired)
				}

				activeInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
					Invoice: invoice.GetInvoiceID(),
					Expand:  billing.StandardInvoiceExpandAll,
				})
				s.Require().NoError(err)
				s.Require().NotNil(activeInvoice.CollectionAt)
				s.True(realizationVariant.expectedCollectionEnd.Equal(*activeInvoice.CollectionAt))

				if test.expectLineDeleted {
					s.Empty(activeInvoice.Lines.OrEmpty())

					invoiceWithDeletedLine, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
						Invoice: invoice.GetInvoiceID(),
						Expand: billing.StandardInvoiceExpandAll.With(
							billing.StandardInvoiceExpandDeletedLines,
						),
					})
					s.Require().NoError(err)
					s.Require().NotNil(invoiceWithDeletedLine.CollectionAt)
					s.True(realizationVariant.expectedCollectionEnd.Equal(*invoiceWithDeletedLine.CollectionAt))
					s.Require().Len(invoiceWithDeletedLine.Lines.OrEmpty(), 1)
					deletedLine := invoiceWithDeletedLine.Lines.OrEmpty()[0]
					s.Require().NotNil(deletedLine.DeletedAt)
					s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
						line:               deletedLine,
						expectTokenOverage: test.onCollectionComplete.expectRunTotals.Total,
						expectCostBasis:    0.5,
						expectFiatTotals:   test.onCollectionComplete.expectInvoiceTotals,
					})

					// TODO: delete the standard invoice when zero overage removes its only line.
					s.Nil(invoiceWithDeletedLine.DeletedAt)
				} else {
					s.Require().Len(activeInvoice.Lines.OrEmpty(), 1)
					s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
						line:               activeInvoice.Lines.OrEmpty()[0],
						expectTokenOverage: test.onCollectionComplete.expectRunTotals.Total,
						expectCostBasis:    0.5,
						expectFiatTotals:   test.onCollectionComplete.expectInvoiceTotals,
					})
				}

				if realizationVariant.expectRemainingGatheringLine {
					gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
						Namespaces: []string{ns},
						Customers:  []string{customer.ID},
						Currencies: []currencyx.FiatCode{currencyx.FiatCode(USD)},
						Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
					})
					s.Require().NoError(err)
					s.Require().Len(gatheringInvoices.Items, 1)

					remainingLines := gatheringInvoices.Items[0].Lines.OrEmpty()
					s.Require().Len(remainingLines, 1)
					remainingLine := remainingLines[0]
					s.Require().NotNil(remainingLine.ChargeID)
					s.Equal(chargeID.ID, *remainingLine.ChargeID)
					s.True(realizationVariant.invoiceAt.Equal(remainingLine.ServicePeriod.From))
					s.True(servicePeriod.To.Equal(remainingLine.ServicePeriod.To))
				}
			})
		})
	}
}

func (s *UsageBasedChargesTestSuite) TestUsageBasedCreditThenInvoicePartialInvoiceLifecycle() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-partial-invoice-lifecycle")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	midPeriodInvoiceAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-16T00:00:00Z", time.UTC).AsTime()
	secondPartialAttemptAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-21T00:00:00Z", time.UTC).AsTime()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	graduatedTieredPrice := productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.GraduatedTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(1),
				},
			},
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(20)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(0.5),
				},
			},
			{
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromFloat(0.25),
				},
			},
		},
	})

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()
	defer s.UsageBasedTestHandler.Reset()

	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(0)

	var (
		usageBasedChargeID meta.ChargeID
		partialInvoice     billing.StandardInvoice
		finalInvoice       billing.StandardInvoice
		partialRunID       string
	)

	s.Run("given a graduated tiered usage-based charge", func() {
		// given:
		// - a credit-then-invoice usage-based charge with graduated tiered pricing
		// when:
		// - the charge is created
		// then:
		// - it starts in created status without realization runs
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          cust.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
					price:             graduatedTieredPrice,
					name:              "usage-based-partial-invoice",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "usage-based-partial-invoice",
					featureKey:        meterSlug,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		// then
		fetched := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(meta.ChargeStatusCreated, meta.ChargeStatus(fetched.Status))
		s.Empty(fetched.Realizations)
	})

	s.Run("when partially invoiced at service period start", func() {
		// given:
		// - the usage-based charge exists at the exact service period start
		// when:
		// - billing tries to invoice pending lines immediately
		// then:
		// - no invoice is created and the charge remains uninvoiced
		clock.FreezeTime(servicePeriod.From)

		// when
		_, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.From),
		})

		// then
		s.Error(err)
		s.ErrorAs(err, &billing.ValidationError{})
		s.ErrorIs(err, billing.ErrInvoiceCreateNoLines)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Empty(charge.Realizations)
		s.Equal(usagebased.StatusCreated, charge.Status)
	})

	s.Run("when partially invoiced mid period", func() {
		// given:
		// - mid-period usage exists for the charge
		// when:
		// - billing invoices pending lines mid period
		// then:
		// - a partial invoice is created and a partial realization run becomes active
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			15,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)
		clock.FreezeTime(midPeriodInvoiceAt)

		// when
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(midPeriodInvoiceAt),
		})

		// then
		s.Require().NoError(err)
		s.Len(invoices, 1)

		partialInvoice = invoices[0]
		s.Require().Len(partialInvoice.Lines.OrEmpty(), 1)

		stdLine := partialInvoice.Lines.OrEmpty()[0]
		expectedPartialCollectionEnd := midPeriodInvoiceAt.Add(usagebased.InternalCollectionPeriod)
		s.Require().NotNil(stdLine.OverrideCollectionPeriodEnd)
		s.True(expectedPartialCollectionEnd.Equal(*stdLine.OverrideCollectionPeriodEnd))
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount: 12.5,
			Total:  12.5,
		}, stdLine.Totals)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount: 12.5,
			Total:  12.5,
		}, partialInvoice.Totals)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, charge.Status)
		s.Len(charge.Realizations, 1)

		currentRun, err := charge.GetCurrentRealizationRun()
		s.Require().NoError(err)
		s.Equal(usagebased.RealizationRunTypePartialInvoice, currentRun.Type)
		s.Require().NotNil(currentRun.LineID)
		s.Equal(stdLine.ID, *currentRun.LineID)
		s.Require().NotNil(currentRun.InvoiceID)
		s.Equal(partialInvoice.ID, *currentRun.InvoiceID)
		s.True(midPeriodInvoiceAt.Equal(currentRun.ServicePeriodTo))
		s.True(midPeriodInvoiceAt.Equal(currentRun.StoredAtLT))
		s.Require().NotNil(partialInvoice.CollectionAt)
		s.True(expectedPartialCollectionEnd.Equal(*partialInvoice.CollectionAt))

		partialRunID = currentRun.ID.ID

		invoicesResult, err := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
			Namespaces: []string{ns},
		})
		s.NoError(err)
		s.Len(invoicesResult.Items, 1)
	})

	s.Run("when partially invoiced again before the first mid period invoice is issued", func() {
		// given:
		// - a partial realization run is already active for the charge
		// when:
		// - billing tries to invoice pending lines again before the first invoice is issued
		// then:
		// - the request is rejected and no additional run or invoice is created
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			5,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-20T00:00:00Z", time.UTC).AsTime(),
		)
		clock.FreezeTime(secondPartialAttemptAt)

		// when
		_, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(secondPartialAttemptAt),
		})

		// then
		s.Error(err)
		s.ErrorAs(err, &billing.ValidationError{})
		s.ErrorIs(err, usagebased.ErrActiveRealizationRunAlreadyExists)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, charge.Status)
		s.Len(charge.Realizations, 1)

		currentRun, runErr := charge.GetCurrentRealizationRun()
		s.NoError(runErr)
		s.Equal(partialRunID, currentRun.ID.ID)

		invoicesResult, listErr := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
			Namespaces: []string{ns},
		})
		s.NoError(listErr)
		s.Len(invoicesResult.Items, 1)
		s.Equal(partialInvoice.ID, invoicesResult.Items[0].ID)
	})

	s.Run("when gathering invoice is previewed with an active partial run", func() {
		// given:
		// - the partial realization run is still active
		// - a remaining gathering line exists for the rest of the service period
		// when:
		// - billing lists the gathering invoice with live preview expansion
		// then:
		// - preview uses the active run as prior billing history and does not create a new run
		s.assertGatheringPreview(assertGatheringPreviewInput{
			Namespace:  ns,
			CustomerID: cust.ID,
			ExpectedInvoiceTotals: billingtest.ExpectedTotals{
				Amount: 2.5,
				Total:  2.5,
			},
			ExpectedLineTotals: billingtest.ExpectedTotals{
				Amount: 2.5,
				Total:  2.5,
			},
			AssertLine: func(previewLine *billing.StandardLine) {
				s.Require().NotNil(previewLine.UsageBased)
				s.Require().NotNil(previewLine.UsageBased.MeteredQuantity)
				s.Require().NotNil(previewLine.UsageBased.Quantity)
				s.Require().NotNil(previewLine.UsageBased.MeteredPreLinePeriodQuantity)
				s.Require().NotNil(previewLine.UsageBased.PreLinePeriodQuantity)
				s.Equal(float64(5), lo.FromPtr(previewLine.UsageBased.MeteredQuantity).InexactFloat64())
				s.Equal(float64(5), lo.FromPtr(previewLine.UsageBased.Quantity).InexactFloat64())
				s.Equal(float64(15), lo.FromPtr(previewLine.UsageBased.MeteredPreLinePeriodQuantity).InexactFloat64())
				s.Equal(float64(15), lo.FromPtr(previewLine.UsageBased.PreLinePeriodQuantity).InexactFloat64())
			},
		})

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, charge.Status)
		s.Len(charge.Realizations, 1)
		currentRun, runErr := charge.GetCurrentRealizationRun()
		s.NoError(runErr)
		s.Equal(partialRunID, currentRun.ID.ID)
	})

	s.Run("when the first partial invoice is advanced and approved", func() {
		// given:
		// - the first partial invoice is ready to be collected and manually approved
		// when:
		// - the invoice is advanced and then approved
		// then:
		// - the run waits in processing until issuance, then accrues invoice usage and returns to active
		defer s.UsageBasedTestHandler.Reset()

		clock.FreezeTime(partialInvoice.DefaultCollectionAtForStandardInvoice())

		// when
		invoice, err := s.BillingService.AdvanceInvoice(ctx, partialInvoice.GetInvoiceID())

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationProcessing, charge.Status)

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[usagebased.OnInvoiceUsageAccruedInput]()
		s.UsageBasedTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		// when
		invoice, err = s.BillingService.ApproveInvoice(ctx, partialInvoice.GetInvoiceID())

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)

		charge = s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActive, charge.Status)
	})

	s.Run("when the final invoice is created and the final realization completes after the service period", func() {
		// given:
		// - more usage arrives and the earlier partial invoice is already approved
		// when:
		// - billing invoices again after the service period and later advances collection
		// then:
		// - a final realization run is created and reaches processing before issuance
		defer s.UsageBasedTestHandler.Reset()

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(0)
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			10,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-25T00:00:00Z", time.UTC).AsTime(),
		)
		clock.FreezeTime(servicePeriod.To)

		// when
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})

		// then
		s.NoError(err)
		s.Len(invoices, 1)
		finalInvoice = invoices[0]
		// TODO[rating]: totals are off due to rating not yet supporting progressive billing via charges

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, charge.Status)
		s.Len(charge.Realizations, 2)

		currentRun, runErr := charge.GetCurrentRealizationRun()
		s.NoError(runErr)
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
		s.Require().NotNil(currentRun.InvoiceID)
		s.Equal(finalInvoice.ID, *currentRun.InvoiceID)

		// given
		clock.FreezeTime(finalInvoice.DefaultCollectionAtForStandardInvoice())

		// when
		finalInvoice, err = s.BillingService.AdvanceInvoice(ctx, finalInvoice.GetInvoiceID())

		// then
		s.NoError(err)

		charge = s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationProcessing, charge.Status)

		currentRun, runErr = charge.GetCurrentRealizationRun()
		s.NoError(runErr)
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
	})

	s.Run("when the final invoice is approved while the partial invoice is still unpaid", func() {
		// given:
		// - the final realization run is processing and the earlier partial invoice payment is still unsettled
		// when:
		// - the final invoice is approved
		// then:
		// - the final run accrues invoice usage and the charge waits for payment settlement
		defer s.UsageBasedTestHandler.Reset()

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[usagebased.OnInvoiceUsageAccruedInput]()
		s.UsageBasedTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		// when
		approvedInvoice, err := s.BillingService.ApproveInvoice(ctx, finalInvoice.GetInvoiceID())

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, approvedInvoice.Status)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)

		finalInvoice = approvedInvoice

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, charge.Status)
		s.Nil(charge.State.CurrentRealizationRunID)
		s.Len(charge.Realizations, 2)
	})

	s.Run("when the final invoice is paid before the partial invoice is settled", func() {
		// given:
		// - the charge is awaiting payment settlement with the partial invoice still unpaid
		// when:
		// - the final invoice is paid
		// then:
		// - the charge keeps waiting because not all invoiced runs are settled yet
		defer s.UsageBasedTestHandler.Reset()

		authorizedCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentAuthorizedInput]()
		s.UsageBasedTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T())
		settledCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentSettledInput]()
		s.UsageBasedTestHandler.onPaymentSettled = settledCallback.Handler(s.T())

		// when
		paidInvoice, err := s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: finalInvoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, paidInvoice.Status)
		s.Equal(1, authorizedCallback.nrInvocations)
		s.Equal(1, settledCallback.nrInvocations)

		finalInvoice = paidInvoice

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, charge.Status)
		s.Len(charge.Realizations, 2)
	})

	s.Run("when the outstanding partial invoice is finally paid", func() {
		// given:
		// - the final invoice is already settled but the earlier partial invoice is still unpaid
		// when:
		// - the partial invoice is paid
		// then:
		// - all invoiced runs are settled and the charge reaches final
		defer s.UsageBasedTestHandler.Reset()

		authorizedCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentAuthorizedInput]()
		s.UsageBasedTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T())
		settledCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentSettledInput]()
		s.UsageBasedTestHandler.onPaymentSettled = settledCallback.Handler(s.T())

		// when
		paidInvoice, err := s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: partialInvoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, paidInvoice.Status)
		s.Equal(1, authorizedCallback.nrInvocations)
		s.Equal(1, settledCallback.nrInvocations)

		partialInvoice = paidInvoice

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusFinal, charge.Status)
		s.Len(charge.Realizations, 2)
		s.Nil(charge.State.CurrentRealizationRunID)
	})
}

func (s *UsageBasedChargesTestSuite) TestUsageBasedCreditThenInvoicePendingPartialInvoiceBlocksFinalRealizationUntilApproval() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-pending-partial-invoice-blocks-final")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	midPeriodInvoiceAt := datetime.MustParseTimeInLocation(s.T(), "2026-01-16T00:00:00Z", time.UTC).AsTime()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()
	defer s.UsageBasedTestHandler.Reset()

	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(0)

	var (
		usageBasedChargeID meta.ChargeID
		partialInvoice     billing.StandardInvoice
	)

	s.Run("given a credit-then-invoice usage-based charge", func() {
		// given:
		// - a credit-then-invoice usage-based charge with unit pricing
		// when:
		// - the charge is created
		// then:
		// - it starts in created status
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(1),
					}),
					name:              "usage-based-partial-invoice-pending-blocks-final",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "usage-based-partial-invoice-pending-blocks-final",
					featureKey:        meterSlug,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		// then
		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusCreated, charge.Status)
	})

	s.Run("when a partial invoice is created mid period and left waiting for manual approval", func() {
		// given:
		// - mid-period usage exists for the charge
		// when:
		// - billing creates a partial invoice and the invoice is advanced but not approved
		// then:
		// - the charge remains on the processing partial-invoice branch while waiting for approval
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			10,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)
		clock.FreezeTime(midPeriodInvoiceAt)

		// when
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(midPeriodInvoiceAt),
		})

		// then
		s.NoError(err)
		s.Len(invoices, 1)
		partialInvoice = invoices[0]

		// given
		clock.FreezeTime(partialInvoice.DefaultCollectionAtForStandardInvoice())

		// when
		partialInvoice, err = s.BillingService.AdvanceInvoice(ctx, partialInvoice.GetInvoiceID())

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, partialInvoice.Status)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationProcessing, charge.Status)
	})

	s.Run("when the service period ends before the partial invoice is approved", func() {
		// given:
		// - the partial invoice still owns the active realization run while the branch is processing
		// when:
		// - billing tries to invoice pending lines for the final period
		// then:
		// - final realization is blocked by the active-run invariant
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			5,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-25T00:00:00Z", time.UTC).AsTime(),
		)
		clock.FreezeTime(servicePeriod.To)

		// when
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})

		// then
		s.Error(err)
		s.ErrorAs(err, &billing.ValidationError{})
		s.ErrorIs(err, usagebased.ErrActiveRealizationRunAlreadyExists)
		s.Nil(invoices)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationProcessing, charge.Status)

		invoicesResult, listErr := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
			Namespaces: []string{ns},
		})
		s.NoError(listErr)
		s.Len(invoicesResult.Items, 1)
		s.Equal(partialInvoice.ID, invoicesResult.Items[0].ID)
	})

	s.Run("when the pending partial invoice is approved after the service period end", func() {
		// given:
		// - the partial invoice is still pending manual approval after the service period end
		// when:
		// - the invoice is approved
		// then:
		// - invoice usage is accrued and the charge returns to active
		defer s.UsageBasedTestHandler.Reset()

		clock.FreezeTime(servicePeriod.To)
		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[usagebased.OnInvoiceUsageAccruedInput]()
		s.UsageBasedTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		// when
		partialInvoice, err := s.BillingService.ApproveInvoice(ctx, partialInvoice.GetInvoiceID())

		// then
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, partialInvoice.Status)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActive, charge.Status)
	})

	s.Run("when invoice pending lines is retried after the partial invoice approval", func() {
		// given:
		// - the previously blocking partial invoice has been approved
		// when:
		// - billing retries invoice pending lines after the service period
		// then:
		// - final realization can start successfully
		defer s.UsageBasedTestHandler.Reset()

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(0)
		clock.FreezeTime(servicePeriod.To)

		// when
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})

		// then
		s.NoError(err)
		s.Len(invoices, 1)

		charge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, charge.Status)

		currentRun, runErr := charge.GetCurrentRealizationRun()
		s.NoError(runErr)
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
	})
}

func (s *UsageBasedChargesTestSuite) mustGetUsageBasedChargeByID(chargeID meta.ChargeID) usagebased.Charge {
	s.T().Helper()

	charge := s.mustGetChargeByID(chargeID)
	usageBasedCharge, err := charge.AsUsageBasedCharge()
	s.NoError(err)

	return usageBasedCharge
}

func (s *UsageBasedChargesTestSuite) setUsageBasedCustomCurrencyEnabled(enabled bool) {
	s.T().Helper()

	enabler, ok := s.Charges.usageBasedService.(customCurrencyEnabler)
	s.Require().True(ok)
	s.Require().NoError(enabler.SetEnableCustomCurrency(s.T(), enabled))
}

type requireCustomCurrencyOverageLineInput struct {
	line               *billing.StandardLine
	expectTokenOverage float64
	expectCostBasis    float64
	expectFiatTotals   billingtest.ExpectedTotals
}

func (s *UsageBasedChargesTestSuite) requireCustomCurrencyOverageLine(in requireCustomCurrencyOverageLineInput) {
	s.T().Helper()

	s.Equal(currencyx.FiatCode(USD), in.line.Currency)
	switch reason := in.line.Annotations[billing.AnnotationKeyReason].(type) {
	case string:
		s.Equal(billing.AnnotationValueReasonOverage, reason)
	case *string:
		s.Require().NotNil(reason)
		s.Equal(billing.AnnotationValueReasonOverage, *reason)
	default:
		s.Fail("overage reason annotation has an unexpected type")
	}

	s.Require().NotNil(in.line.UsageBased)
	s.Require().NotNil(in.line.UsageBased.Price)
	flatPrice, err := in.line.UsageBased.Price.AsFlat()
	s.Require().NoError(err)
	s.Equal(in.expectFiatTotals.Amount, flatPrice.Amount.InexactFloat64())

	s.Require().Len(in.line.DetailedLines, 1)
	detailedLine := in.line.DetailedLines[0]
	s.Equal(in.expectTokenOverage, detailedLine.Quantity.InexactFloat64())
	s.Equal(in.expectCostBasis, detailedLine.PerUnitAmount.InexactFloat64())
	s.RequireTotals(in.expectFiatTotals, detailedLine.Totals)
	s.RequireTotals(in.expectFiatTotals, in.line.Totals)
}
