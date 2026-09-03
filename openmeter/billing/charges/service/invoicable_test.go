package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeeservice "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	billingtotals "github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestInvoicableCharges(t *testing.T) {
	suite.Run(t, new(InvoicableChargesTestSuite))
}

type InvoicableChargesTestSuite struct {
	BaseSuite
}

type flatFeeFiatOverageCreditsHandler struct {
	available alpacadecimal.Decimal
}

func (h *flatFeeFiatOverageCreditsHandler) AddAvailable(amount float64) {
	h.available = h.available.Add(alpacadecimal.NewFromFloat(amount))
}

func (h *flatFeeFiatOverageCreditsHandler) Allocate(
	_ context.Context,
	input flatfee.AllocateFiatOverageCreditsInput,
) (creditrealization.CreateAllocationInputs, error) {
	amount := input.AmountToAllocate
	if amount.GreaterThan(h.available) {
		amount = h.available
	}

	if amount.IsZero() {
		return nil, nil
	}

	h.available = h.available.Sub(amount)

	return creditrealization.CreateAllocationInputs{
		{
			ServicePeriod: input.Run.ServicePeriod,
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: ulid.Make().String(),
			},
			Amount: amount,
		},
	}, nil
}

func (h *flatFeeFiatOverageCreditsHandler) Correct(
	_ context.Context,
	input flatfee.CorrectFiatOverageCreditAllocationsInput,
) (creditrealization.CreateCorrectionInputs, error) {
	for _, correction := range input.Corrections {
		h.available = h.available.Add(correction.Amount.Abs())
	}

	return newCreditCorrectionInputs(input.Corrections), nil
}

func (s *InvoicableChargesTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *InvoicableChargesTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
}

func (s *InvoicableChargesTestSuite) TestFlatFeeGatheringPreviewPopulatesTotalsWithoutRealizationRun() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-gathering-preview")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	creditAllocationCallback := newCountedCreditAllocationCallback[flatfee.OnAllocateCreditsInput]()
	s.FlatFeeTestHandler.onAllocateCredits = creditAllocationCallback.Handler(
		s.T(),
		func(flatfee.OnAllocateCreditsInput, ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs {
			return nil
		},
	)

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(100),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:              "flat-fee-gathering-preview",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "flat-fee-gathering-preview",
			}),
		},
	})
	s.NoError(err)
	s.Require().Len(created, 1)

	flatFeeCharge, err := created[0].AsFlatFeeCharge()
	s.NoError(err)
	flatFeeChargeID := flatFeeCharge.GetChargeID()

	s.assertGatheringPreview(assertGatheringPreviewInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ExpectedInvoiceTotals: billingtest.ExpectedTotals{
			Amount: 100,
			Total:  100,
		},
		ExpectedLines: 1,
		ExpectedLineTotals: billingtest.ExpectedTotals{
			Amount: 100,
			Total:  100,
		},
		ExpectedDetailedLines: 1,
		AssertLine: func(previewLine *billing.StandardLine) {
			s.Equal(flatFeeChargeID.ID, lo.FromPtr(previewLine.ChargeID))
			s.Empty(previewLine.CreditsApplied)
		},
	})

	chargeAfterPreview := mustGetFlatFeeChargeWithExpands(&s.BaseSuite, flatFeeChargeID, meta.Expands{meta.ExpandRealizations})
	s.Nil(chargeAfterPreview.Realizations.CurrentRun)
	s.Empty(chargeAfterPreview.Realizations.PriorRuns)
	s.Zero(creditAllocationCallback.nrInvocations)
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCustomCurrencyCreditThenInvoiceLifecycle() {
	type tc struct {
		name string

		// chargeAmount is the gross flat-fee amount in TOKENS.
		chargeAmount float64
		// creditsAllocated is the TOKENS allocation returned for the run.
		creditsAllocated float64
		// fiatOverageCreditsAvailable is the USD balance available for the converted overage.
		fiatOverageCreditsAvailable float64
		// disableFiatOverageCredits models the settlement handler rejecting FIAT credit use.
		disableFiatOverageCredits bool

		// expectRunTotals contains the realization run totals in TOKENS.
		expectRunTotals billingtest.ExpectedTotals
		// expectDraftInvoiceTotals contains the pre-finalization invoice totals in USD.
		expectDraftInvoiceTotals billingtest.ExpectedTotals
		// expectInvoiceTotals contains the standard invoice totals in USD.
		expectInvoiceTotals billingtest.ExpectedTotals

		expectPaymentSettled bool
		expectLineDeleted    bool
	}

	// setup:
	// - the charge is a flat fee settled through credit then invoice
	// - the charge and its credit allocations are denominated in TOKENS
	// - TOKENS use a manual USD cost basis of 0.5
	// - the invoice represents the post-allocation overage in USD
	tests := []tc{
		// given:
		// - a 10 TOKENS flat fee with no credit allocation
		// when:
		// - the charge is invoiced and paid
		// then:
		// - all 10 TOKENS are billed as 5 USD
		{
			name:                     "happy path",
			chargeAmount:             10,
			expectRunTotals:          billingtest.ExpectedTotals{Amount: 10, Total: 10},
			expectDraftInvoiceTotals: billingtest.ExpectedTotals{Amount: 5, Total: 5},
			expectInvoiceTotals:      billingtest.ExpectedTotals{Amount: 5, Total: 5},
			expectPaymentSettled:     true,
		},
		// given:
		// - a 5 USD overage and enough settlement-fiat credits to cover it
		// - the settlement handler does not allow those credits to be used
		// then:
		// - gross preparation still runs and the full 5 USD remains payable
		{
			name:                        "fiat overage credits disabled by handler",
			chargeAmount:                10,
			fiatOverageCreditsAvailable: 5,
			disableFiatOverageCredits:   true,
			expectRunTotals:             billingtest.ExpectedTotals{Amount: 10, Total: 10},
			expectDraftInvoiceTotals:    billingtest.ExpectedTotals{Amount: 5, Total: 5},
			expectInvoiceTotals:         billingtest.ExpectedTotals{Amount: 5, Total: 5},
			expectPaymentSettled:        true,
		},
		// given:
		// - a 10 TOKENS flat fee with 2 TOKENS allocated from credits
		// when:
		// - the charge is invoiced and paid
		// then:
		// - the remaining 8 TOKENS are billed as 4 USD
		{
			name:                     "happy path with credit allocation",
			chargeAmount:             10,
			creditsAllocated:         2,
			expectRunTotals:          billingtest.ExpectedTotals{Amount: 10, CreditsTotal: 2, Total: 8},
			expectDraftInvoiceTotals: billingtest.ExpectedTotals{Amount: 4, Total: 4},
			expectInvoiceTotals:      billingtest.ExpectedTotals{Amount: 4, Total: 4},
			expectPaymentSettled:     true,
		},
		// given:
		// - a 10 TOKENS flat fee produces a 5 USD overage
		// - 3 USD are allocated from settlement-fiat credits
		// when:
		// - the charge is invoiced and paid
		// then:
		// - the remaining 2 USD is settled through payment
		{
			name:                        "fiat overage partially covered by credits",
			chargeAmount:                10,
			fiatOverageCreditsAvailable: 3,
			expectRunTotals:             billingtest.ExpectedTotals{Amount: 10, Total: 10},
			expectDraftInvoiceTotals:    billingtest.ExpectedTotals{Amount: 5, Total: 5},
			expectInvoiceTotals:         billingtest.ExpectedTotals{Amount: 5, CreditsTotal: 3, Total: 2},
			expectPaymentSettled:        true,
		},
		// given:
		// - a 10 TOKENS flat fee produces a 5 USD overage
		// - the full 5 USD is allocated from settlement-fiat credits
		// when:
		// - the charge is invoiced
		// then:
		// - the credited line remains on the invoice and no payment is needed
		{
			name:                        "fiat overage fully covered by credits",
			chargeAmount:                10,
			fiatOverageCreditsAvailable: 5,
			expectRunTotals:             billingtest.ExpectedTotals{Amount: 10, Total: 10},
			expectDraftInvoiceTotals:    billingtest.ExpectedTotals{Amount: 5, Total: 5},
			expectInvoiceTotals:         billingtest.ExpectedTotals{Amount: 5, CreditsTotal: 5},
		},
		// given:
		// - a 10 TOKENS flat fee fully covered by allocated credits
		// when:
		// - the charge is invoiced
		// then:
		// - no fiat transaction or payment is needed
		{
			name:                     "fully covered by credits",
			chargeAmount:             10,
			creditsAllocated:         10,
			expectRunTotals:          billingtest.ExpectedTotals{Amount: 10, CreditsTotal: 10},
			expectDraftInvoiceTotals: billingtest.ExpectedTotals{},
			expectInvoiceTotals:      billingtest.ExpectedTotals{},
			expectLineDeleted:        true,
		},
		// given:
		// - a positive 0.001 TOKENS overage whose converted value is below USD precision
		// when:
		// - the charge is invoiced
		// then:
		// - the fiat amount rounds to zero and no fiat transaction or payment is needed
		{
			name:                     "fiat overage rounds to zero",
			chargeAmount:             0.001,
			expectRunTotals:          billingtest.ExpectedTotals{Amount: 0.001, Total: 0.001},
			expectDraftInvoiceTotals: billingtest.ExpectedTotals{},
			expectInvoiceTotals:      billingtest.ExpectedTotals{},
			expectLineDeleted:        true,
		},
	}

	s.enableFlatFeeCustomCurrenciesWithMockLineage()

	for _, test := range tests {
		s.Run(test.name, func() {
			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("charges-service-flatfee-custom-currency-lifecycle")

			s.FlatFeeTestHandler.Reset()
			defer s.FlatFeeTestHandler.Reset()
			clock.UnFreeze()
			defer clock.UnFreeze()

			defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
			sandboxApp := s.InstallSandboxApp(s.T(), ns)
			customer := s.CreateTestCustomer(ns, "customer-c1")
			_ = s.ProvisionBillingProfile(
				ctx,
				ns,
				sandboxApp.GetID(),
				billingtest.WithManualApproval(),
			)

			customCurrency := s.createTestCustomCurrency(ctx, ns)
			fiatCurrency, err := currencyx.NewFiatCurrency(USD)
			s.Require().NoError(err)

			createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
			servicePeriod := timeutil.ClosedPeriod{
				From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
				To:   datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime(),
			}
			chargeName := "flat-fee-custom-currency"
			overageName := chargeName + " (overage)"

			clock.FreezeTime(createAt)

			var (
				chargeID meta.ChargeID
				invoice  billing.StandardInvoice
				lineID   string
				runID    string

				allocationCallback                *countedCreditAllocationCallback[flatfee.OnAllocateCreditsInput]
				customCurrencyOverageAccruedCalls int
				accruedTransactionGroupID         string
				authorizedCallback                *countedLedgerTransactionCallback[flatfee.OnPaymentAuthorizedInput]
				settledCallback                   *countedLedgerTransactionCallback[flatfee.OnPaymentSettledInput]
				fiatOverageCreditsHandler         *flatFeeFiatOverageCreditsHandler
				fiatOverageAllocationInvocations  int
			)

			fiatOverageCreditsHandler = &flatFeeFiatOverageCreditsHandler{
				available: alpacadecimal.Zero,
			}
			s.FlatFeeTestHandler.onAllocateFiatOverageCredits = func(
				ctx context.Context,
				input flatfee.AllocateFiatOverageCreditsInput,
			) (creditrealization.CreateAllocationInputs, error) {
				fiatOverageAllocationInvocations++
				if test.disableFiatOverageCredits {
					return nil, nil
				}

				return fiatOverageCreditsHandler.Allocate(ctx, input)
			}
			s.FlatFeeTestHandler.onCorrectFiatOverageCreditAllocations = fiatOverageCreditsHandler.Correct

			s.Run("create charge and overage placeholder", func() {
				costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
					FiatCurrency: fiatCurrency,
					Rate:         alpacadecimal.NewFromFloat(0.5),
				})

				created, err := s.Charges.Create(ctx, charges.CreateInput{
					Namespace: ns,
					Intents: []charges.ChargeIntent{
						charges.NewChargeIntent(flatfee.Intent{
							Intent: meta.Intent{
								ManagedBy:         billing.SubscriptionManagedLine,
								UniqueReferenceID: lo.ToPtr("flat-fee-custom-currency-lifecycle"),
								CustomerID:        customer.ID,
								Currency:          customCurrency,
								TaxConfig: productcatalog.TaxCodeConfig{
									TaxCodeID: defaults.InvoicingTaxCodeID,
								},
							},
							IntentMutableFields: flatfee.IntentMutableFields{
								IntentMutableFields: meta.IntentMutableFields{
									Name:              chargeName,
									ServicePeriod:     servicePeriod,
									FullServicePeriod: servicePeriod,
									BillingPeriod:     servicePeriod,
								},
								InvoiceAt:             servicePeriod.To,
								PaymentTerm:           productcatalog.InArrearsPaymentTerm,
								AmountBeforeProration: alpacadecimal.NewFromFloat(test.chargeAmount),
							},
							SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
							CostBasis:      &costBasisIntent,
						}),
					},
				})
				s.Require().NoError(err)
				s.Require().Len(created, 1)

				charge, err := created[0].AsFlatFeeCharge()
				s.Require().NoError(err)
				chargeID = charge.GetChargeID()
				s.Require().NotNil(charge.State.ResolvedCostBasis)
				s.Equal(float64(0.5), charge.State.ResolvedCostBasis.CostBasis.InexactFloat64())

				lines := activeGatheringLinesForCharge(&s.BaseSuite, ns, customer.ID, charge.ID)
				s.Require().Len(lines, 1)

				line := lines[0]
				s.Equal(overageName, line.Name)
				s.Equal(currencyx.FiatCode(USD), line.Currency)
				s.Equal(billing.LineEngineTypeChargeFlatFee, line.Engine)
				s.Require().NotNil(line.ChargeID)
				s.Equal(charge.ID, *line.ChargeID)
				reason, ok := line.Annotations.GetString(billing.AnnotationKeyReason)
				s.True(ok)
				s.Equal(billing.AnnotationValueReasonOveragePlaceholder, reason)
				s.Empty(line.RateCardDiscounts)

				flatPrice, err := line.Price.AsFlat()
				s.Require().NoError(err)
				s.True(flatPrice.Amount.IsZero())
				s.Equal(productcatalog.InArrearsPaymentTerm, flatPrice.PaymentTerm)

				persistedCharge := mustGetFlatFeeChargeWithExpands(&s.BaseSuite, chargeID, meta.Expands{meta.ExpandRealizations})
				s.Nil(persistedCharge.Realizations.CurrentRun)
			})

			s.Run("invoice overage and create run", func() {
				allocationCallback = newCountedCreditAllocationCallback[flatfee.OnAllocateCreditsInput]()
				s.FlatFeeTestHandler.onAllocateCredits = allocationCallback.Handler(
					s.T(),
					func(input flatfee.OnAllocateCreditsInput, transactionGroup ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs {
						amount := input.PreTaxAmountToAllocate
						creditsAllocated := alpacadecimal.NewFromFloat(test.creditsAllocated)
						if amount.GreaterThan(creditsAllocated) {
							amount = creditsAllocated
						}
						if amount.IsZero() {
							return nil
						}

						return creditrealization.CreateAllocationInputs{
							{
								ServicePeriod:     input.ServicePeriod,
								Amount:            amount,
								LedgerTransaction: transactionGroup,
							},
						}
					},
					func(t *testing.T, input flatfee.OnAllocateCreditsInput) {
						assert.Equal(t, chargeID.ID, input.Charge.ID)
						assert.True(t, input.Charge.Intent.GetCurrency().IsCustom())
						assert.Equal(t, customCurrency.ID, input.Charge.Intent.GetCurrency().ID)
						assert.Equal(t, servicePeriod, input.ServicePeriod)
						assert.Equal(t, test.chargeAmount, input.PreTaxAmountToAllocate.InexactFloat64())
					},
				)

				clock.FreezeTime(servicePeriod.To)
				invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
					Customer: customer.GetID(),
					AsOf:     lo.ToPtr(servicePeriod.To),
				})
				s.Require().NoError(err)
				s.Require().Len(invoices, 1)
				invoice = invoices[0]
				s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
				s.Equal(currencyx.FiatCode(USD), invoice.Currency)
				s.RequireTotals(test.expectDraftInvoiceTotals, invoice.Totals)

				s.Require().Len(invoice.Lines.OrEmpty(), 1)
				line := invoice.Lines.OrEmpty()[0]
				lineID = line.ID
				if test.expectLineDeleted {
					s.requireDeletedCustomCurrencyOverageLine(requireDeletedCustomCurrencyOverageLineInput{
						line:             line,
						expectFiatTotals: test.expectDraftInvoiceTotals,
					})
				} else {
					s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
						line:               line,
						expectTokenOverage: test.expectRunTotals.Total,
						expectCostBasis:    0.5,
						expectFiatTotals:   test.expectDraftInvoiceTotals,
					})
				}
				s.Equal(overageName, line.Name)
				s.Empty(line.RateCardDiscounts)
				s.Empty(line.Discounts)
				s.Empty(line.CreditsApplied)

				charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
				var run *flatfee.RealizationRun
				if test.expectLineDeleted {
					s.Equal(flatfee.StatusFinal, charge.Status)
					s.Require().NotNil(charge.Realizations.CurrentRun)
					s.Empty(charge.Realizations.PriorRuns)
					run = charge.Realizations.CurrentRun
				} else {
					s.Equal(flatfee.StatusActiveRealizationProcessing, charge.Status)
					s.Require().NotNil(charge.Realizations.CurrentRun)
					run = charge.Realizations.CurrentRun
				}

				runID = run.ID.ID
				s.Equal(servicePeriod, run.ServicePeriod)
				s.Require().NotNil(run.LineID)
				s.Equal(lineID, *run.LineID)
				s.Require().NotNil(run.InvoiceID)
				s.Equal(invoice.ID, *run.InvoiceID)
				s.RequireTotals(test.expectRunTotals, run.Totals)
				s.Equal(test.creditsAllocated, run.CreditRealizations.Sum().InexactFloat64())
				s.Empty(run.FiatOverageCreditRealizations)
				s.True(run.DetailedLines.IsPresent())
				s.Require().Len(run.DetailedLines.OrEmpty(), 1)
				s.RequireTotals(test.expectRunTotals, run.DetailedLines.OrEmpty()[0].Totals)

				expectedCreditsApplied := 0
				if test.creditsAllocated > 0 {
					expectedCreditsApplied = 1
				}
				s.Len(run.CreditRealizations, expectedCreditsApplied)
				s.Len(run.DetailedLines.OrEmpty()[0].CreditsApplied, expectedCreditsApplied)
				s.Equal(test.expectLineDeleted, run.NoFiatTransactionRequired)
				s.False(run.Immutable)
				s.Equal(1, allocationCallback.nrInvocations)
				s.Zero(fiatOverageAllocationInvocations)
			})

			s.Run("approve invoice and settle overage", func() {
				fiatOverageCreditsHandler.AddAvailable(test.fiatOverageCreditsAvailable)

				if !test.expectLineDeleted {
					s.FlatFeeTestHandler.onCustomCurrencyOverageAccrued = func(_ context.Context, input flatfee.OnCustomCurrencyOverageAccruedInput) (flatfee.OnCustomCurrencyOverageAccruedResult, error) {
						customCurrencyOverageAccruedCalls++
						s.Equal(chargeID.ID, input.Charge.ID)
						s.Equal(runID, input.Run.ID.ID)
						s.Equal(test.expectRunTotals.Total, input.GetCustomCurrencyAmountAccrued().InexactFloat64())

						resolvedCostBasis, err := input.GetCostBasis()
						s.Require().NoError(err)
						s.Equal(float64(0.5), resolvedCostBasis.InexactFloat64())

						resolvedFiatCurrency, err := input.GetFiatCurrency()
						s.Require().NoError(err)
						s.Equal(USD, resolvedFiatCurrency.Details().Code)

						accruedTransactionGroupID = ulid.Make().String()

						return flatfee.OnCustomCurrencyOverageAccruedResult{
							TransactionGroup: ledgertransaction.GroupReference{
								TransactionGroupID: accruedTransactionGroupID,
							},
							TotalFiatAmount: alpacadecimal.NewFromFloat(test.expectInvoiceTotals.Amount),
						}, nil
					}
				}

				if test.expectPaymentSettled {
					authorizedCallback = newCountedLedgerTransactionCallback[flatfee.OnPaymentAuthorizedInput]()
					s.FlatFeeTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T(), func(t *testing.T, input flatfee.OnPaymentAuthorizedInput) {
						assert.Equal(t, chargeID.ID, input.Charge.ID)
						assert.Equal(t, runID, input.Run.ID.ID)
						assert.Equal(t, test.expectInvoiceTotals.Total, input.FiatAmount.InexactFloat64())
					})

					settledCallback = newCountedLedgerTransactionCallback[flatfee.OnPaymentSettledInput]()
					s.FlatFeeTestHandler.onPaymentSettled = settledCallback.Handler(s.T(), func(t *testing.T, input flatfee.OnPaymentSettledInput) {
						assert.Equal(t, chargeID.ID, input.Charge.ID)
						assert.Equal(t, runID, input.Run.ID.ID)
						assert.Equal(t, test.expectInvoiceTotals.Total, input.FiatAmount.InexactFloat64())
					})
				}

				var err error
				if test.expectLineDeleted {
					mockApp := s.SandboxApp.EnableMock(s.T())
					defer s.SandboxApp.DisableMock()
					mockApp.OnDeleteStandardInvoice(nil)

					invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
					s.Require().NoError(err)
					s.Equal(billing.StandardInvoiceStatusDeleted, invoice.Status)
					s.NotNil(invoice.DeletedAt)
					s.Equal(billing.ChangeSourceSystem, invoice.DeletionSource)
					s.Nil(invoice.IssuedAt)
					s.Equal(1, mockApp.DeleteInvoiceCallCount())
					s.Zero(mockApp.FinalizeInvoiceCallCount())
					s.Zero(fiatOverageAllocationInvocations)
					mockApp.AssertExpectations(s.T())

					return
				}

				invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)
				s.Equal(1, fiatOverageAllocationInvocations)
				s.RequireTotals(test.expectInvoiceTotals, invoice.Totals)
				s.Require().Len(invoice.Lines.OrEmpty(), 1)
				s.Equal(
					test.expectInvoiceTotals.CreditsTotal,
					invoice.Lines.OrEmpty()[0].CreditsApplied.SumAmount(fiatCurrency).InexactFloat64(),
				)
			})

			s.Run("reload persisted charge and invoice", func() {
				charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
				s.Equal(flatfee.StatusFinal, charge.Status)
				s.Require().NotNil(charge.Realizations.CurrentRun)
				s.Empty(charge.Realizations.PriorRuns)
				run := charge.Realizations.CurrentRun
				s.Equal(!test.expectLineDeleted, run.Immutable)

				s.Equal(runID, run.ID.ID)
				s.RequireTotals(test.expectRunTotals, run.Totals)
				s.Equal(!test.expectPaymentSettled, run.NoFiatTransactionRequired)
				s.Equal(test.creditsAllocated, run.CreditRealizations.Sum().InexactFloat64())
				expectedFiatCreditsAllocated := test.fiatOverageCreditsAvailable
				if test.disableFiatOverageCredits {
					expectedFiatCreditsAllocated = 0
				}
				s.Equal(expectedFiatCreditsAllocated, run.FiatOverageCreditRealizations.Sum().InexactFloat64())
				s.True(run.DetailedLines.IsPresent())
				s.Require().Len(run.DetailedLines.OrEmpty(), 1)
				s.RequireTotals(test.expectRunTotals, run.DetailedLines.OrEmpty()[0].Totals)
				expectedFiatRealizations := 0
				if expectedFiatCreditsAllocated > 0 {
					expectedFiatRealizations = 1
				}
				s.Len(run.FiatOverageCreditRealizations, expectedFiatRealizations)

				if !test.expectLineDeleted {
					s.Equal(1, customCurrencyOverageAccruedCalls)
					s.Require().NotNil(run.AccruedUsage)
					s.Equal(servicePeriod, run.AccruedUsage.ServicePeriod)
					s.Equal(accruedTransactionGroupID, run.AccruedUsage.LedgerTransaction.TransactionGroupID)
					s.RequireTotals(billingtest.ExpectedTotals{
						Amount: test.expectInvoiceTotals.Amount,
						Total:  test.expectInvoiceTotals.Amount,
					}, run.AccruedUsage.Totals)
				} else {
					s.Zero(customCurrencyOverageAccruedCalls)
					s.Nil(run.AccruedUsage)
				}

				if test.expectPaymentSettled {
					s.Require().NotNil(authorizedCallback)
					s.Equal(1, authorizedCallback.nrInvocations)
					s.Require().NotNil(settledCallback)
					s.Equal(1, settledCallback.nrInvocations)

					s.Require().NotNil(run.Payment)
					s.Equal(payment.StatusSettled, run.Payment.Status)
					s.Equal(test.expectInvoiceTotals.Total, run.Payment.FiatAmount.InexactFloat64())
					s.Require().NotNil(run.Payment.Authorized)
					s.Equal(authorizedCallback.id, run.Payment.Authorized.TransactionGroupID)
					s.Require().NotNil(run.Payment.Settled)
					s.Equal(settledCallback.id, run.Payment.Settled.TransactionGroupID)
				} else {
					s.Nil(run.Payment)
				}

				activeInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
					Invoice: invoice.GetInvoiceID(),
					Expand:  billing.StandardInvoiceExpandAll,
				})
				s.Require().NoError(err)
				expectedInvoiceStatus := billing.StandardInvoiceStatusPaid
				if test.expectLineDeleted {
					expectedInvoiceStatus = billing.StandardInvoiceStatusDeleted
				}
				s.Equal(expectedInvoiceStatus, activeInvoice.Status)
				s.Equal(currencyx.FiatCode(USD), activeInvoice.Currency)
				s.RequireTotals(test.expectInvoiceTotals, activeInvoice.Totals)
				if test.expectLineDeleted {
					s.NotNil(activeInvoice.DeletedAt)
					s.Equal(billing.ChangeSourceSystem, activeInvoice.DeletionSource)
					s.Nil(activeInvoice.IssuedAt)
					s.Empty(activeInvoice.Lines.OrEmpty())
				} else {
					s.Nil(activeInvoice.DeletedAt)
					s.Require().Len(activeInvoice.Lines.OrEmpty(), 1)
				}

				invoiceWithDeletedLines, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
					Invoice: invoice.GetInvoiceID(),
					Expand: billing.StandardInvoiceExpandAll.With(
						billing.StandardInvoiceExpandDeletedLines,
					),
				})
				s.Require().NoError(err)
				s.Require().Len(invoiceWithDeletedLines.Lines.OrEmpty(), 1)
				line := invoiceWithDeletedLines.Lines.OrEmpty()[0]
				s.Equal(lineID, line.ID)
				s.Equal(overageName, line.Name)

				if test.expectLineDeleted {
					s.requireDeletedCustomCurrencyOverageLine(requireDeletedCustomCurrencyOverageLineInput{
						line:             line,
						expectFiatTotals: test.expectInvoiceTotals,
					})
					s.NotNil(invoiceWithDeletedLines.DeletedAt)
					s.Equal(billing.StandardInvoiceStatusDeleted, invoiceWithDeletedLines.Status)
					s.Equal(billing.ChangeSourceSystem, invoiceWithDeletedLines.DeletionSource)
					s.Nil(invoiceWithDeletedLines.IssuedAt)
				} else {
					s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
						line:               line,
						expectTokenOverage: test.expectRunTotals.Total,
						expectCostBasis:    0.5,
						expectFiatTotals:   test.expectInvoiceTotals,
					})
					s.Nil(line.DeletedAt)
				}
			})

			if test.name == "happy path" {
				s.Run("delete issued charge without correcting immutable run", func() {
					deletePatch, err := meta.NewPatchDelete(meta.NewPatchDeleteInput{
						ChangeSource: billing.ChangeSourceSystem,
						Policy:       meta.RefundAsCreditsDeletePolicy,
					})
					s.Require().NoError(err)
					s.Require().NoError(s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
						CustomerID: customer.GetID(),
						PatchesByChargeID: map[string]charges.Patch{
							chargeID.ID: deletePatch,
						},
					}))

					charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
					s.Equal(flatfee.StatusDeleted, charge.Status)
					s.Nil(charge.Realizations.CurrentRun)
					s.Require().Len(charge.Realizations.PriorRuns, 1)
					run := charge.Realizations.PriorRuns[0]
					s.True(run.Immutable)
					s.Nil(run.DeletedAt)
					s.Require().NotNil(run.AccruedUsage)
				})
			}
		})
	}
}

type flatFeePreparedOverageDeleteMode uint8

const (
	flatFeePreparedOverageDeleteModeNone flatFeePreparedOverageDeleteMode = iota
	flatFeePreparedOverageDeleteModeInvoice
	flatFeePreparedOverageDeleteModeCharge
)

func (s *InvoicableChargesTestSuite) TestFlatFeeCustomCurrencyFiatOverageAllocationSurvivesInvoiceSyncRetry() {
	s.runFlatFeeCustomCurrencyFiatOverageAfterInvoiceSyncFailure(flatFeePreparedOverageDeleteModeNone)
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCustomCurrencyPreparedOverageCanBeDeletedAfterInvoiceSyncFailure() {
	s.runFlatFeeCustomCurrencyFiatOverageAfterInvoiceSyncFailure(flatFeePreparedOverageDeleteModeInvoice)
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCustomCurrencyPreparedOverageChargeDeleteReconcilesImmutableInvoice() {
	s.runFlatFeeCustomCurrencyFiatOverageAfterInvoiceSyncFailure(flatFeePreparedOverageDeleteModeCharge)
}

func (s *InvoicableChargesTestSuite) runFlatFeeCustomCurrencyFiatOverageAfterInvoiceSyncFailure(deleteMode flatFeePreparedOverageDeleteMode) {
	// given:
	// - a custom-currency flat-fee run has a 5 USD gross overage
	// - 5 USD of settlement credits are available at invoice finalization
	// when:
	// - invoice finalization persists the fiat allocation but external invoice sync fails
	// - issuing is retried after the invoicing app recovers
	// then:
	// - issuing reuses the persisted allocation without invoking allocation again

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-fiat-overage-sync-retry")

	s.FlatFeeTestHandler.Reset()
	s.T().Cleanup(s.FlatFeeTestHandler.Reset)
	s.enableFlatFeeCustomCurrenciesWithMockLineage()

	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	invoiceAt := datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   invoiceAt,
	}

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	customer := s.CreateTestCustomer(ns, "customer-c1")
	_ = s.ProvisionBillingProfile(
		ctx,
		ns,
		sandboxApp.GetID(),
		billingtest.WithManualApproval(),
	)

	customCurrency := s.createTestCustomCurrency(ctx, ns)
	fiatCurrency, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)

	fiatOverageCreditsHandler := &flatFeeFiatOverageCreditsHandler{
		available: alpacadecimal.Zero,
	}
	accountingEvents := make([]string, 0, 2)
	customCurrencyOverageAccruedInvocations := 0
	s.FlatFeeTestHandler.onCustomCurrencyOverageAccrued = func(_ context.Context, input flatfee.OnCustomCurrencyOverageAccruedInput) (flatfee.OnCustomCurrencyOverageAccruedResult, error) {
		customCurrencyOverageAccruedInvocations++
		accountingEvents = append(accountingEvents, "gross_overage")

		fiatCurrency, err := input.GetFiatCurrency()
		s.Require().NoError(err)
		costBasis, err := input.GetCostBasis()
		s.Require().NoError(err)

		return flatfee.OnCustomCurrencyOverageAccruedResult{
			TransactionGroup: ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()},
			TotalFiatAmount:  fiatCurrency.RoundToPrecision(input.GetCustomCurrencyAmountAccrued().Mul(costBasis)),
		}, nil
	}
	s.FlatFeeTestHandler.onAllocateCredits = func(
		_ context.Context,
		input flatfee.OnAllocateCreditsInput,
	) (creditrealization.CreateAllocationInputs, error) {
		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod: input.ServicePeriod,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
				Amount: alpacadecimal.NewFromInt(2),
			},
		}, nil
	}
	fiatOverageAllocationInvocations := 0
	returnFiatAllocationError := true
	s.FlatFeeTestHandler.onAllocateFiatOverageCredits = func(
		ctx context.Context,
		input flatfee.AllocateFiatOverageCreditsInput,
	) (creditrealization.CreateAllocationInputs, error) {
		fiatOverageAllocationInvocations++
		accountingEvents = append(accountingEvents, "fiat_coverage")
		if returnFiatAllocationError {
			return nil, errors.New("simulated fiat allocation failure")
		}

		return fiatOverageCreditsHandler.Allocate(ctx, input)
	}
	correctionEvents := make([]string, 0, 6)
	s.FlatFeeTestHandler.onCorrectFiatOverageCreditAllocations = func(
		ctx context.Context,
		input flatfee.CorrectFiatOverageCreditAllocationsInput,
	) (creditrealization.CreateCorrectionInputs, error) {
		correctionEvents = append(correctionEvents, "fiat_coverage_correction")

		return fiatOverageCreditsHandler.Correct(ctx, input)
	}
	s.FlatFeeTestHandler.onCustomCurrencyOverageAccruedCorrection = func(
		_ context.Context,
		_ flatfee.OnCustomCurrencyOverageAccruedCorrectionInput,
	) error {
		correctionEvents = append(correctionEvents, "gross_overage_correction")
		return nil
	}
	failChargeCurrencyCorrection := deleteMode == flatFeePreparedOverageDeleteModeInvoice
	s.FlatFeeTestHandler.onCorrectCreditAllocations = func(
		_ context.Context,
		input flatfee.CorrectCreditAllocationsInput,
	) (creditrealization.CreateCorrectionInputs, error) {
		correctionEvents = append(correctionEvents, "charge_currency_correction")
		if failChargeCurrencyCorrection {
			return nil, errors.New("simulated charge-currency correction failure")
		}

		return newCreditCorrectionInputs(input.Corrections), nil
	}

	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})

	var (
		chargeID meta.ChargeID
		invoice  billing.StandardInvoice
	)

	s.Run("given a draft invoice with gross fiat overage", func() {
		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				charges.NewChargeIntent(flatfee.Intent{
					Intent: meta.Intent{
						ManagedBy:         billing.SubscriptionManagedLine,
						UniqueReferenceID: lo.ToPtr("flat-fee-fiat-overage-sync-retry"),
						CustomerID:        customer.ID,
						Currency:          customCurrency,
						TaxConfig: productcatalog.TaxCodeConfig{
							TaxCodeID: defaults.InvoicingTaxCodeID,
						},
					},
					IntentMutableFields: flatfee.IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{
							Name:              "flat-fee-fiat-overage-sync-retry",
							ServicePeriod:     servicePeriod,
							FullServicePeriod: servicePeriod,
							BillingPeriod:     servicePeriod,
						},
						InvoiceAt:             invoiceAt,
						PaymentTerm:           productcatalog.InArrearsPaymentTerm,
						AmountBeforeProration: alpacadecimal.NewFromInt(12),
					},
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					CostBasis:      &costBasisIntent,
				}),
			},
		})
		s.Require().NoError(err)
		s.Require().Len(created, 1)
		charge, err := created[0].AsFlatFeeCharge()
		s.Require().NoError(err)
		chargeID = charge.GetChargeID()

		clock.FreezeTime(invoiceAt)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.GetID(),
			AsOf:     lo.ToPtr(invoiceAt),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		invoice = invoices[0]
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 5, Total: 5}, invoice.Totals)
		s.Zero(fiatOverageAllocationInvocations)
	})

	fiatOverageCreditsHandler.AddAvailable(5)
	mockApp := s.SandboxApp.EnableMock(s.T())
	defer s.SandboxApp.DisableMock()

	invoiceSyncAttempts := 0
	mockApp.OnUpsertStandardInvoice(func(_ billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
		invoiceSyncAttempts++
		if invoiceSyncAttempts == 1 {
			return nil, errors.New("simulated invoice sync failure")
		}

		return billing.NewUpsertStandardInvoiceResult(), nil
	})
	switch deleteMode {
	case flatFeePreparedOverageDeleteModeInvoice:
		mockApp.OnDeleteStandardInvoice(errors.New("simulated invoice delete sync failure"))
	case flatFeePreparedOverageDeleteModeNone:
		mockApp.OnFinalizeStandardInvoice(nil)
	case flatFeePreparedOverageDeleteModeCharge:
	}

	s.Run("when fiat allocation fails after gross preparation", func() {
		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusIssuingLineFinalizationFailed, invoice.Status)
		s.Require().NotNil(invoice.StatusDetails.AvailableActions.Retry)
		s.Nil(invoice.StatusDetails.AvailableActions.Delete)
		s.Zero(invoiceSyncAttempts)
		s.Equal(1, customCurrencyOverageAccruedInvocations)
		s.Equal(1, fiatOverageAllocationInvocations)
		s.Equal([]string{"gross_overage", "fiat_coverage"}, accountingEvents)

		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Equal(flatfee.StatusActiveRealizationProcessing, charge.Status)
		s.Require().NotNil(charge.Realizations.CurrentRun)
		run := charge.Realizations.CurrentRun
		s.Require().NotNil(run.AccruedUsage)
		s.False(run.Immutable)
		s.Require().NotNil(run.AccruedUsage.LedgerTransaction)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 5, Total: 5}, run.AccruedUsage.Totals)
		s.False(run.FiatOverageCreditAllocationCompleted)
		s.False(run.NoFiatTransactionRequired)
		s.Empty(run.FiatOverageCreditRealizations)

		_, err = s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
			Invoice:        invoice.GetInvoiceID(),
			DeletionSource: billing.ChangeSourceSystem,
		})
		s.ErrorContains(err, "invoice action not available")
	})

	returnFiatAllocationError = false
	accountingEvents = accountingEvents[:0]

	s.Run("when retry reaches sync after fiat allocation", func() {
		invoice, err = s.BillingService.RetryInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusIssuingSyncFailed, invoice.Status)
		s.Equal(1, invoiceSyncAttempts)
		s.Equal(1, customCurrencyOverageAccruedInvocations)
		s.Equal(2, fiatOverageAllocationInvocations)
		s.Equal([]string{"fiat_coverage"}, accountingEvents)
		s.Zero(mockApp.FinalizeInvoiceCallCount())
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 5, CreditsTotal: 5}, invoice.Totals)

		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Equal(flatfee.StatusActiveRealizationIssuing, charge.Status)
		s.Require().NotNil(charge.Realizations.CurrentRun)
		run := charge.Realizations.CurrentRun
		s.Require().NotNil(run.AccruedUsage)
		s.False(run.Immutable)
		s.Require().NotNil(run.AccruedUsage.LedgerTransaction)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 5, Total: 5}, run.AccruedUsage.Totals)
		s.True(run.FiatOverageCreditAllocationCompleted)
		s.True(run.NoFiatTransactionRequired)
		s.Require().Len(run.FiatOverageCreditRealizations, 1)
		s.Equal(creditrealization.TypeAllocation, run.FiatOverageCreditRealizations[0].Type)
		s.Equal(float64(5), run.FiatOverageCreditRealizations[0].Amount.InexactFloat64())

		s.Require().NotNil(invoice.StatusDetails.AvailableActions.Delete)

		invoice, err = s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusIssuingSyncFailed, invoice.Status)
	})

	if deleteMode == flatFeePreparedOverageDeleteModeCharge {
		s.Run("when the owning charge is deleted after invoice preparation", func() {
			// given the prepared run is still reversible
			// when the owning charge is deleted directly
			// then the state machine corrects it once and billing records the immutable invoice drift
			deletePatch, err := meta.NewPatchDelete(meta.NewPatchDeleteInput{
				ChangeSource: billing.ChangeSourceSystem,
				Policy:       meta.RefundAsCreditsDeletePolicy,
			})
			s.Require().NoError(err)
			s.Require().NoError(s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
				CustomerID: customer.GetID(),
				PatchesByChargeID: map[string]charges.Patch{
					chargeID.ID: deletePatch,
				},
			}))

			s.Equal([]string{
				"fiat_coverage_correction",
				"gross_overage_correction",
				"charge_currency_correction",
			}, correctionEvents)

			charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
			s.Equal(flatfee.StatusDeleted, charge.Status)
			s.Nil(charge.Realizations.CurrentRun)
			s.Require().Len(charge.Realizations.PriorRuns, 1)
			run := charge.Realizations.PriorRuns[0]
			s.Require().NotNil(run.DeletedAt)
			s.Nil(run.AccruedUsage)

			immutableInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: invoice.GetInvoiceID(),
				Expand:  billing.StandardInvoiceExpandAll,
			})
			s.Require().NoError(err)
			s.Equal(billing.StandardInvoiceStatusIssuingSyncFailed, immutableInvoice.Status)
			s.Require().Len(immutableInvoice.Lines.OrEmpty(), 1)
			s.Nil(immutableInvoice.Lines.OrEmpty()[0].DeletedAt)
			s.NotEmpty(immutableInvoice.ValidationIssues)
			mockApp.AssertExpectations(s.T())
		})

		return
	}

	if deleteMode == flatFeePreparedOverageDeleteModeInvoice {
		s.Run("when the first prepared cancellation fails", func() {
			// given the overage and FIAT corrections succeed
			// when the later charge-currency correction fails
			// then the whole cleanup transaction rolls back
			invoice, err = s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
				Invoice:        invoice.GetInvoiceID(),
				DeletionSource: billing.ChangeSourceAPIRequest,
			})
			s.Require().NoError(err)
			s.Equal(billing.StandardInvoiceStatusDeleteFailed, invoice.Status)
			s.Require().NotNil(invoice.StatusDetails.AvailableActions.Delete)
			s.Equal([]string{
				"fiat_coverage_correction",
				"gross_overage_correction",
				"charge_currency_correction",
			}, correctionEvents)

			charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
			s.Require().NotNil(charge.Realizations.CurrentRun)
			s.Require().NotNil(charge.Realizations.CurrentRun.AccruedUsage)
			s.True(charge.Realizations.CurrentRun.FiatOverageCreditAllocationCompleted)
			s.True(charge.Realizations.CurrentRun.NoFiatTransactionRequired)
			s.Require().Len(charge.Realizations.CurrentRun.FiatOverageCreditRealizations, 1)
			s.Equal(creditrealization.TypeAllocation, charge.Realizations.CurrentRun.FiatOverageCreditRealizations[0].Type)
			s.Require().Len(charge.Realizations.CurrentRun.CreditRealizations, 1)
			s.Equal(creditrealization.TypeAllocation, charge.Realizations.CurrentRun.CreditRealizations[0].Type)
			s.Nil(charge.Realizations.CurrentRun.DeletedAt)
		})

		failChargeCurrencyCorrection = false
		s.Run("when cleanup commits before invoice delete sync fails", func() {
			// given the correction failure is resolved
			// when cleanup commits but the invoicing app delete fails
			// then the prepared run stays deleted and only external sync remains retryable
			invoice, err = s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
				Invoice:        invoice.GetInvoiceID(),
				DeletionSource: billing.ChangeSourceAPIRequest,
			})
			s.Require().NoError(err)
			s.Equal(billing.StandardInvoiceStatusDeleteFailed, invoice.Status)
			s.Equal(billing.ChangeSourceAPIRequest, invoice.DeletionSource)
			s.Require().NotNil(invoice.DeletedAt)
			s.Equal([]string{
				"fiat_coverage_correction",
				"gross_overage_correction",
				"charge_currency_correction",
				"fiat_coverage_correction",
				"gross_overage_correction",
				"charge_currency_correction",
			}, correctionEvents)
			s.Equal(1, mockApp.DeleteInvoiceCallCount())

			charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
			s.Equal(flatfee.StatusDeleted, charge.Status)
			s.Nil(charge.Realizations.CurrentRun)
			s.Require().Len(charge.Realizations.PriorRuns, 1)
			run := charge.Realizations.PriorRuns[0]
			s.Require().NotNil(run.DeletedAt)
			s.Nil(run.AccruedUsage)
			s.False(run.FiatOverageCreditAllocationCompleted)
			s.True(run.NoFiatTransactionRequired)
			s.Require().Len(run.FiatOverageCreditRealizations, 2)
			s.Zero(run.FiatOverageCreditRealizations.Sum().InexactFloat64())
			s.Require().Len(run.CreditRealizations, 2)
			s.Zero(run.CreditRealizations.Sum().InexactFloat64())
		})

		mockApp.OnDeleteStandardInvoice(nil)
		s.Run("then invoice delete sync retries without repeating cleanup", func() {
			// given charge cleanup already committed
			// when invoice deletion is retried after the app recovers
			// then billing only retries external sync
			invoice, err = s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
				Invoice:        invoice.GetInvoiceID(),
				DeletionSource: billing.ChangeSourceAPIRequest,
			})
			s.Require().NoError(err)
			s.Equal(billing.StandardInvoiceStatusDeleted, invoice.Status)
			s.Equal([]string{
				"fiat_coverage_correction",
				"gross_overage_correction",
				"charge_currency_correction",
				"fiat_coverage_correction",
				"gross_overage_correction",
				"charge_currency_correction",
			}, correctionEvents)
			s.Equal(2, mockApp.DeleteInvoiceCallCount())
			charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
			s.Require().Len(charge.Realizations.PriorRuns, 1)
			s.Nil(charge.Realizations.PriorRuns[0].AccruedUsage)
			mockApp.AssertExpectations(s.T())
		})

		return
	}

	s.Run("then retry issuing without reallocating fiat credits", func() {
		invoice, err = s.BillingService.RetryInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		s.Equal(2, invoiceSyncAttempts)
		s.Equal(1, mockApp.FinalizeInvoiceCallCount())
		s.Equal(1, customCurrencyOverageAccruedInvocations)
		s.Equal(2, fiatOverageAllocationInvocations)
		s.Equal(flatfee.StatusFinal, s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID).Status)

		err = s.BillingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
			InvoiceTriggerInput: billing.InvoiceTriggerInput{
				Invoice: invoice.GetInvoiceID(),
				Trigger: billing.TriggerPaid,
			},
			AppType:    app.AppTypeSandbox,
			Capability: app.CapabilityTypeCollectPayments,
		})
		s.Require().NoError(err)

		invoice, err = s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 5, CreditsTotal: 5}, invoice.Totals)
		s.Equal(5.0, invoice.Lines.OrEmpty()[0].CreditsApplied.SumAmount(fiatCurrency).InexactFloat64())
		s.Equal(2, fiatOverageAllocationInvocations)
		mockApp.AssertExpectations(s.T())
	})
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCustomCurrencyCreditThenInvoiceReratingDefersFiatOverageAllocationUntilFinalization() {
	s.enableFlatFeeCustomCurrenciesWithMockLineage()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-custom-currency-rerating")

	s.FlatFeeTestHandler.Reset()
	s.T().Cleanup(s.FlatFeeTestHandler.Reset)
	clock.UnFreeze()
	s.T().Cleanup(clock.UnFreeze)

	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	customer := s.CreateTestCustomer(ns, "customer-c1")
	_ = s.ProvisionBillingProfile(
		ctx,
		ns,
		sandboxApp.GetID(),
		billingtest.WithManualApproval(),
	)

	customCurrency := s.createTestCustomCurrency(ctx, ns)
	fiatCurrency, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(createAt)

	chargeCurrencyCreditsAvailable := alpacadecimal.NewFromInt(12)
	s.FlatFeeTestHandler.onAllocateCredits = func(
		_ context.Context,
		input flatfee.OnAllocateCreditsInput,
	) (creditrealization.CreateAllocationInputs, error) {
		amount := input.PreTaxAmountToAllocate
		if amount.GreaterThan(chargeCurrencyCreditsAvailable) {
			amount = chargeCurrencyCreditsAvailable
		}
		if amount.IsZero() {
			return nil, nil
		}

		chargeCurrencyCreditsAvailable = chargeCurrencyCreditsAvailable.Sub(amount)

		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod: input.ServicePeriod,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
				Amount: amount,
			},
		}, nil
	}

	fiatOverageCreditsHandler := &flatFeeFiatOverageCreditsHandler{
		available: alpacadecimal.Zero,
	}
	customCurrencyOverageAccruedCalls := 0
	s.FlatFeeTestHandler.onCustomCurrencyOverageAccrued = func(_ context.Context, input flatfee.OnCustomCurrencyOverageAccruedInput) (flatfee.OnCustomCurrencyOverageAccruedResult, error) {
		customCurrencyOverageAccruedCalls++
		costBasis, err := input.GetCostBasis()
		s.Require().NoError(err)
		fiatCurrency, err := input.GetFiatCurrency()
		s.Require().NoError(err)

		return flatfee.OnCustomCurrencyOverageAccruedResult{
			TransactionGroup: ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()},
			TotalFiatAmount:  fiatCurrency.RoundToPrecision(input.GetCustomCurrencyAmountAccrued().Mul(costBasis)),
		}, nil
	}
	s.FlatFeeTestHandler.onAllocateFiatOverageCredits = fiatOverageCreditsHandler.Allocate
	s.FlatFeeTestHandler.onCorrectFiatOverageCreditAllocations = fiatOverageCreditsHandler.Correct

	mutableFields := flatfee.IntentMutableFields{
		IntentMutableFields: meta.IntentMutableFields{
			Name:              "flat-fee-custom-currency-rerating",
			ServicePeriod:     servicePeriod,
			FullServicePeriod: servicePeriod,
			BillingPeriod:     servicePeriod,
		},
		InvoiceAt:             servicePeriod.To,
		PaymentTerm:           productcatalog.InArrearsPaymentTerm,
		AmountBeforeProration: alpacadecimal.NewFromInt(26),
	}
	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			charges.NewChargeIntent(flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: lo.ToPtr("flat-fee-custom-currency-rerating"),
					CustomerID:        customer.ID,
					Currency:          customCurrency,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaults.InvoicingTaxCodeID,
					},
				},
				IntentMutableFields: mutableFields,
				SettlementMode:      productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:           &costBasisIntent,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)

	charge, err := created[0].AsFlatFeeCharge()
	s.Require().NoError(err)
	chargeID := charge.GetChargeID()

	var (
		invoice billing.StandardInvoice
		lineID  billing.LineID
		runID   flatfee.RealizationRunID
	)

	s.Run("create mutable run with charge-currency credits", func() {
		// given:
		// - the 26 TOKENS flat fee is covered by 12 TOKENS of charge-currency credits
		// when:
		// - billing creates the mutable run and draft invoice
		// then:
		// - the draft contains the gross 7 USD overage without allocating fiat credits
		clock.FreezeTime(servicePeriod.To)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		invoice = invoices[0]
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 7, Total: 7}, invoice.Totals)
		s.Require().Len(invoice.Lines.OrEmpty(), 1)
		line := invoice.Lines.OrEmpty()[0]
		lineID = line.GetLineID()
		s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
			line:               line,
			expectTokenOverage: 14,
			expectCostBasis:    0.5,
			expectFiatTotals:   billingtest.ExpectedTotals{Amount: 7, Total: 7},
		})
		s.Empty(line.CreditsApplied)

		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Require().NotNil(charge.Realizations.CurrentRun)
		run := charge.Realizations.CurrentRun
		runID = run.ID
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 26, CreditsTotal: 12, Total: 14}, run.Totals)
		s.Require().Len(run.CreditRealizations, 1)
		s.Equal(float64(12), run.CreditRealizations.Sum().InexactFloat64())
		s.Empty(run.FiatOverageCreditRealizations)
		s.False(run.NoFiatTransactionRequired)
	})

	s.Run("reconcile a retrospective charge-currency credit correction", func() {
		// given:
		// - a correction makes another 10 TOKENS available at the original booking time
		// when:
		// - the mutable run is rerated without changing the charge amount
		// then:
		// - the run allocates the 10 TOKENS and updates the gross fiat overage to 2 USD
		// - fiat credits remain unallocated until invoice finalization
		chargeCurrencyCreditsAvailable = chargeCurrencyCreditsAvailable.Add(alpacadecimal.NewFromInt(10))

		patch, err := meta.NewPatchSetOverride(flatfee.NewPatchSetOverrideInput{
			ChangeSource:        billing.ChangeSourceAPIRequest,
			IntentMutableFields: mutableFields,
		})
		s.Require().NoError(err)
		s.Require().NoError(s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
			CustomerID: customer.GetID(),
			PatchesByChargeID: map[string]charges.Patch{
				chargeID.ID: patch,
			},
		}))

		reloadedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 2, Total: 2}, reloadedInvoice.Totals)
		s.Require().Len(reloadedInvoice.Lines.OrEmpty(), 1)
		line := reloadedInvoice.Lines.GetByID(lineID.ID)
		s.Require().NotNil(line)
		s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
			line:               line,
			expectTokenOverage: 4,
			expectCostBasis:    0.5,
			expectFiatTotals:   billingtest.ExpectedTotals{Amount: 2, Total: 2},
		})
		s.Empty(line.CreditsApplied)

		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Require().NotNil(charge.Realizations.CurrentRun)
		run := charge.Realizations.CurrentRun
		s.Equal(runID, run.ID)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 26, CreditsTotal: 22, Total: 4}, run.Totals)
		s.Require().Len(run.CreditRealizations, 2)
		s.Equal(float64(22), run.CreditRealizations.Sum().InexactFloat64())
		s.Empty(run.FiatOverageCreditRealizations)
		s.False(run.NoFiatTransactionRequired)
	})

	s.Run("allocate only the final fiat overage", func() {
		// given:
		// - the rerated run has a final 2 USD gross overage
		// - 5 USD of settlement credits are available at finalization
		// when:
		// - the invoice is approved
		// then:
		// - only 2 USD is allocated once, with no provisional fiat correction
		fiatOverageCreditsHandler.AddAvailable(5)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 2, CreditsTotal: 2}, invoice.Totals)
		s.Require().Len(invoice.Lines.OrEmpty(), 1)
		line := invoice.Lines.OrEmpty()[0]
		s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
			line:               line,
			expectTokenOverage: 4,
			expectCostBasis:    0.5,
			expectFiatTotals:   billingtest.ExpectedTotals{Amount: 2, CreditsTotal: 2},
		})
		s.Equal(float64(2), line.CreditsApplied.SumAmount(fiatCurrency).InexactFloat64())

		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Require().NotNil(charge.Realizations.CurrentRun)
		run := charge.Realizations.CurrentRun
		s.Equal(runID, run.ID)
		s.Require().Len(run.FiatOverageCreditRealizations, 1)
		s.Equal(creditrealization.TypeAllocation, run.FiatOverageCreditRealizations[0].Type)
		s.Equal(float64(2), run.FiatOverageCreditRealizations[0].Amount.InexactFloat64())
		s.Equal(1, customCurrencyOverageAccruedCalls)
		s.True(run.NoFiatTransactionRequired)
	})
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCustomCurrencyCreditThenInvoiceMutableRunDeletionCorrectsChargeCurrencyRealizations() {
	s.enableFlatFeeCustomCurrenciesWithMockLineage()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-custom-currency-deletion")

	s.FlatFeeTestHandler.Reset()
	s.T().Cleanup(s.FlatFeeTestHandler.Reset)
	clock.UnFreeze()
	s.T().Cleanup(clock.UnFreeze)

	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	customer := s.CreateTestCustomer(ns, "customer-c1")
	_ = s.ProvisionBillingProfile(
		ctx,
		ns,
		sandboxApp.GetID(),
		billingtest.WithManualApproval(),
	)

	customCurrency := s.createTestCustomCurrency(ctx, ns)
	fiatCurrency, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(createAt)

	s.FlatFeeTestHandler.onAllocateCredits = func(
		_ context.Context,
		input flatfee.OnAllocateCreditsInput,
	) (creditrealization.CreateAllocationInputs, error) {
		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod: input.ServicePeriod,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
				Amount: alpacadecimal.NewFromInt(2),
			},
		}, nil
	}
	s.FlatFeeTestHandler.onCorrectCreditAllocations = func(
		_ context.Context,
		input flatfee.CorrectCreditAllocationsInput,
	) (creditrealization.CreateCorrectionInputs, error) {
		return newCreditCorrectionInputs(input.Corrections), nil
	}

	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			charges.NewChargeIntent(flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: lo.ToPtr("flat-fee-custom-currency-deletion"),
					CustomerID:        customer.ID,
					Currency:          customCurrency,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaults.InvoicingTaxCodeID,
					},
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "flat-fee-custom-currency-deletion",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:             servicePeriod.To,
					PaymentTerm:           productcatalog.InArrearsPaymentTerm,
					AmountBeforeProration: alpacadecimal.NewFromInt(10),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:      &costBasisIntent,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)

	charge, err := created[0].AsFlatFeeCharge()
	s.Require().NoError(err)
	chargeID := charge.GetChargeID()

	var (
		invoice billing.StandardInvoice
		runID   flatfee.RealizationRunID
	)

	s.Run("create mutable run before fiat allocation", func() {
		// given:
		// - 2 TOKENS of credits leave a 4 USD overage on the 10 TOKENS flat fee
		// when:
		// - billing creates the mutable run and draft invoice
		// then:
		// - only charge-currency credits are allocated before invoice finalization
		clock.FreezeTime(servicePeriod.To)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		invoice = invoices[0]
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 4, Total: 4}, invoice.Totals)
		s.Require().Len(invoice.Lines.OrEmpty(), 1)

		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Require().NotNil(charge.Realizations.CurrentRun)
		run := charge.Realizations.CurrentRun
		runID = run.ID
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 10, CreditsTotal: 2, Total: 8}, run.Totals)
		s.Require().Len(run.CreditRealizations, 1)
		s.Equal(float64(2), run.CreditRealizations.Sum().InexactFloat64())
		s.Empty(run.FiatOverageCreditRealizations)
	})

	s.Run("delete charge before issuing", func() {
		// given:
		// - the run and invoice line are still mutable
		// when:
		// - the owning charge is deleted with refund-as-credits semantics
		// then:
		// - billing deletes the line and the now-empty draft invoice
		mockApp := s.SandboxApp.EnableMock(s.T())
		s.T().Cleanup(s.SandboxApp.DisableMock)
		mockApp.OnValidateStandardInvoice(nil)
		mockApp.OnDeleteStandardInvoice(nil)

		deletePatch, err := meta.NewPatchDelete(meta.NewPatchDeleteInput{
			ChangeSource: billing.ChangeSourceSystem,
			Policy:       meta.RefundAsCreditsDeletePolicy,
		})
		s.Require().NoError(err)
		s.Require().NoError(s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
			CustomerID: customer.GetID(),
			PatchesByChargeID: map[string]charges.Patch{
				chargeID.ID: deletePatch,
			},
		}))
		s.Equal(1, mockApp.DeleteInvoiceCallCount())
		s.Zero(mockApp.FinalizeInvoiceCallCount())
		mockApp.AssertExpectations(s.T())
	})

	s.Run("reload deleted run audit history", func() {
		// given:
		// - deletion detached the mutable run from the current realization slot
		// when:
		// - the persisted run and its realization tables are reloaded
		// then:
		// - the charge-currency allocation has a correction and fiat history remains empty
		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Equal(flatfee.StatusDeleted, charge.Status)
		s.Nil(charge.Realizations.CurrentRun)
		s.Require().Len(charge.Realizations.PriorRuns, 1)
		s.Equal(runID, charge.Realizations.PriorRuns[0].ID)
		s.Require().NotNil(charge.Realizations.PriorRuns[0].DeletedAt)

		dbRun, err := s.DBClient.ChargeFlatFeeRun.Get(ctx, runID.ID)
		s.Require().NoError(err)
		s.Equal(runID.Namespace, dbRun.Namespace)
		s.Require().NotNil(dbRun.DeletedAt)

		dbChargeCurrencyRealizations, err := dbRun.QueryCreditAllocations().All(ctx)
		s.Require().NoError(err)
		chargeCurrencyRealizations := creditrealization.FromDBRealizations(dbChargeCurrencyRealizations)
		s.Require().Len(chargeCurrencyRealizations, 2)
		s.Equal(float64(0), chargeCurrencyRealizations.Sum().InexactFloat64())

		dbFiatOverageRealizations, err := dbRun.QueryFiatOverageCreditAllocations().All(ctx)
		s.Require().NoError(err)
		fiatOverageRealizations := creditrealization.FromDBRealizations(dbFiatOverageRealizations)
		s.Empty(fiatOverageRealizations)

		chargeCurrencyAllocation, ok := lo.Find(chargeCurrencyRealizations, func(realization creditrealization.Realization) bool {
			return realization.Type == creditrealization.TypeAllocation
		})
		s.Require().True(ok)
		chargeCurrencyCorrection, ok := lo.Find(chargeCurrencyRealizations, func(realization creditrealization.Realization) bool {
			return realization.Type == creditrealization.TypeCorrection
		})
		s.Require().True(ok)
		s.Equal(creditrealization.TypeAllocation, chargeCurrencyAllocation.Type)
		s.Equal(creditrealization.TypeCorrection, chargeCurrencyCorrection.Type)
		s.Require().NotNil(chargeCurrencyCorrection.CorrectsRealizationID)
		s.Equal(chargeCurrencyAllocation.ID, *chargeCurrencyCorrection.CorrectsRealizationID)

		hasInvoiceUsage, err := dbRun.QueryInvoicedUsage().Exist(ctx)
		s.Require().NoError(err)
		s.False(hasInvoiceUsage)
		hasPayment, err := dbRun.QueryPayment().Exist(ctx)
		s.Require().NoError(err)
		s.False(hasPayment)

		activeInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusDeleted, activeInvoice.Status)
		s.NotNil(activeInvoice.DeletedAt)
		s.Equal(billing.ChangeSourceSystem, activeInvoice.DeletionSource)
		s.Empty(activeInvoice.Lines.OrEmpty())

		invoiceWithDeletedLine, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand: billing.StandardInvoiceExpandAll.With(
				billing.StandardInvoiceExpandDeletedLines,
			),
		})
		s.Require().NoError(err)
		s.Require().Len(invoiceWithDeletedLine.Lines.OrEmpty(), 1)
		s.requireDeletedCustomCurrencyOverageLine(requireDeletedCustomCurrencyOverageLineInput{
			line:             invoiceWithDeletedLine.Lines.OrEmpty()[0],
			expectFiatTotals: billingtest.ExpectedTotals{Amount: 4, Total: 4},
		})
		s.Empty(activeGatheringLinesForCharge(&s.BaseSuite, ns, customer.ID, chargeID.ID))
	})
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCustomCurrencyGatheringPreviewAndAPILineMutation() {
	s.enableFlatFeeCustomCurrenciesWithMockLineage()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-custom-currency-managed-lines")

	s.FlatFeeTestHandler.Reset()
	defer s.FlatFeeTestHandler.Reset()
	clock.UnFreeze()
	defer clock.UnFreeze()

	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	customInvoicing := s.SetupCustomInvoicing(ns)
	customer := s.CreateTestCustomer(ns, "customer-c1")
	_ = s.ProvisionBillingProfile(
		ctx,
		ns,
		customInvoicing.App.GetID(),
		billingtest.WithManualApproval(),
	)

	customCurrency := s.createTestCustomCurrency(ctx, ns)
	fiatCurrency, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	const (
		chargeName  = "flat-fee-custom-currency"
		overageName = chargeName + " (overage)"
	)

	clock.FreezeTime(createAt)

	allocationCallback := newCountedCreditAllocationCallback[flatfee.OnAllocateCreditsInput]()
	s.FlatFeeTestHandler.onAllocateCredits = allocationCallback.Handler(
		s.T(),
		func(flatfee.OnAllocateCreditsInput, ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs {
			return nil
		},
	)

	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			charges.NewChargeIntent(flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: lo.ToPtr("flat-fee-custom-currency-managed-lines"),
					CustomerID:        customer.ID,
					Currency:          customCurrency,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaults.InvoicingTaxCodeID,
					},
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              chargeName,
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:             servicePeriod.To,
					PaymentTerm:           productcatalog.InArrearsPaymentTerm,
					AmountBeforeProration: alpacadecimal.NewFromInt(10),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:      &costBasisIntent,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)

	flatFeeCharge, err := created[0].AsFlatFeeCharge()
	s.Require().NoError(err)
	chargeID := flatFeeCharge.GetChargeID()

	gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespace: ns,
		Customers: []string{customer.ID},
		Expand: billing.GatheringInvoiceExpands{
			billing.GatheringInvoiceExpandLines,
		},
	})
	s.Require().NoError(err)
	s.Require().Len(gatheringInvoices.Items, 1)
	gatheringInvoice := gatheringInvoices.Items[0]
	s.Require().Len(gatheringInvoice.Lines.OrEmpty(), 1)

	originalGatheringLine, err := gatheringInvoice.Lines.OrEmpty()[0].Clone()
	s.Require().NoError(err)

	s.Run("preview omits the zero USD placeholder without realizing the charge", func() {
		// given:
		// - a custom-currency charge is represented by a zero-USD gathering placeholder
		// when:
		// - billing calculates the gathering invoice preview with live data
		// then:
		// - the placeholder is omitted and no run or credit allocation is created
		s.assertGatheringPreview(assertGatheringPreviewInput{
			Namespace:             ns,
			CustomerID:            customer.ID,
			ExpectedInvoiceTotals: billingtest.ExpectedTotals{},
			ExpectedLines:         0,
		})

		charge := mustGetFlatFeeChargeWithExpands(&s.BaseSuite, chargeID, meta.Expands{meta.ExpandRealizations})
		s.Equal(flatfee.StatusCreated, charge.Status)
		s.Nil(charge.Realizations.CurrentRun)
		s.Empty(charge.Realizations.PriorRuns)
		s.Zero(allocationCallback.nrInvocations)
	})

	for _, test := range []struct {
		name   string
		mutate func(*billing.GatheringLine)
	}{
		{
			name: "update",
			mutate: func(line *billing.GatheringLine) {
				line.Name = "manually updated"
			},
		},
		{
			name: "delete",
			mutate: func(line *billing.GatheringLine) {
				line.DeletedAt = lo.ToPtr(clock.Now())
			},
		},
	} {
		s.Run("reject gathering line "+test.name, func() {
			// given:
			// - the custom-currency charge still has only its gathering placeholder
			// when:
			// - an API-originated update attempts to mutate that managed line
			// then:
			// - billing rejects the edit without changing the line or realizing the charge
			_, err := s.BillingService.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
				Invoice:      gatheringInvoice.GetInvoiceID(),
				ChangeSource: billing.ChangeSourceAPIRequest,
				EditFn: func(invoice *billing.GatheringInvoice) error {
					lines := invoice.Lines.OrEmpty()
					s.Require().Len(lines, 1)
					test.mutate(&lines[0])
					return nil
				},
				IncludeDeletedLines: true,
			})
			s.ErrorIs(err, billing.ErrCannotUpdateChargeManagedLine)

			reloadedInvoice, err := s.BillingService.GetGatheringInvoiceById(ctx, billing.GetGatheringInvoiceByIdInput{
				Invoice: gatheringInvoice.GetInvoiceID(),
				Expand: billing.GatheringInvoiceExpands{
					billing.GatheringInvoiceExpandLines,
					billing.GatheringInvoiceExpandDeletedLines,
				},
			})
			s.Require().NoError(err)
			s.Require().Len(reloadedInvoice.Lines.OrEmpty(), 1)
			reloadedLine := reloadedInvoice.Lines.OrEmpty()[0]
			s.Equal(originalGatheringLine.ID, reloadedLine.ID)
			s.Equal(originalGatheringLine.Name, reloadedLine.Name)
			s.Equal(originalGatheringLine.ServicePeriod, reloadedLine.ServicePeriod)
			s.Equal(originalGatheringLine.InvoiceAt, reloadedLine.InvoiceAt)
			s.Equal(originalGatheringLine.Price, reloadedLine.Price)
			s.Nil(reloadedLine.DeletedAt)

			charge := mustGetFlatFeeChargeWithExpands(&s.BaseSuite, chargeID, meta.Expands{meta.ExpandRealizations})
			s.Equal(flatfee.StatusCreated, charge.Status)
			s.Nil(charge.Realizations.CurrentRun)
			s.Empty(charge.Realizations.PriorRuns)
			s.Zero(allocationCallback.nrInvocations)
		})
	}

	var (
		draftInvoice billing.StandardInvoice
		runID        flatfee.RealizationRunID
	)

	s.Run("collect the placeholder into a mutable standard line", func() {
		// given:
		// - the gathering placeholder and charge are unchanged by rejected API edits
		// when:
		// - billing collects the line into a draft invoice
		// then:
		// - the charge has one mutable run and one five-dollar overage line
		clock.FreezeTime(servicePeriod.To)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		draftInvoice = invoices[0]
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, draftInvoice.Status)
		s.Require().Len(draftInvoice.Lines.OrEmpty(), 1)
		s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
			line:               draftInvoice.Lines.OrEmpty()[0],
			expectTokenOverage: 10,
			expectCostBasis:    0.5,
			expectFiatTotals: billingtest.ExpectedTotals{
				Amount: 5,
				Total:  5,
			},
		})

		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Require().NotNil(charge.Realizations.CurrentRun)
		runID = charge.Realizations.CurrentRun.ID
		s.False(charge.Realizations.CurrentRun.Immutable)
		s.Nil(charge.Realizations.CurrentRun.AccruedUsage)
		s.Nil(charge.Realizations.CurrentRun.Payment)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 10, Total: 10}, charge.Realizations.CurrentRun.Totals)
		s.Equal(1, allocationCallback.nrInvocations)
	})

	originalStandardLine, err := draftInvoice.Lines.OrEmpty()[0].Clone()
	s.Require().NoError(err)

	for _, test := range []struct {
		name   string
		reject bool
		mutate func(*billing.StandardLine)
	}{
		{
			name:   "reject standard line update",
			reject: true,
			mutate: func(line *billing.StandardLine) {
				line.Name = "manually updated"
			},
		},
		{
			name: "delete standard line",
			mutate: func(line *billing.StandardLine) {
				line.DeletedAt = lo.ToPtr(clock.Now())
			},
		},
	} {
		s.Run(test.name, func() {
			// given:
			// - the custom-currency charge is attached to a mutable draft standard line
			// when:
			// - an API-originated update attempts to mutate that managed line
			// then:
			// - field updates are rejected, while deletion reconciles the mutable charge run
			_, err := s.BillingService.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
				Invoice:      draftInvoice.GetInvoiceID(),
				ChangeSource: billing.ChangeSourceAPIRequest,
				EditFn: func(invoice *billing.StandardInvoice) error {
					lines := invoice.Lines.OrEmpty()
					s.Require().Len(lines, 1)
					test.mutate(lines[0])
					return nil
				},
				IncludeDeletedLines: true,
			})
			if !test.reject {
				s.Require().NoError(err)

				reloadedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
					Invoice: draftInvoice.GetInvoiceID(),
					Expand: billing.StandardInvoiceExpandAll.With(
						billing.StandardInvoiceExpandDeletedLines,
					),
				})
				s.Require().NoError(err)
				s.Require().Len(reloadedInvoice.Lines.OrEmpty(), 1)
				s.NotNil(reloadedInvoice.Lines.OrEmpty()[0].DeletedAt)

				charge := mustGetFlatFeeChargeWithExpands(&s.BaseSuite, chargeID, meta.Expands{
					meta.ExpandRealizations,
					meta.ExpandDeletedRealizations,
				})
				s.Equal(flatfee.StatusDeleted, charge.Status)
				s.Nil(charge.Realizations.CurrentRun)
				s.Equal(1, allocationCallback.nrInvocations)
				return
			}

			s.ErrorIs(err, billing.ErrCannotUpdateChargeManagedLine)

			reloadedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: draftInvoice.GetInvoiceID(),
				Expand: billing.StandardInvoiceExpandAll.With(
					billing.StandardInvoiceExpandDeletedLines,
				),
			})
			s.Require().NoError(err)
			s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, reloadedInvoice.Status)
			s.Require().Len(reloadedInvoice.Lines.OrEmpty(), 1)
			reloadedLine := reloadedInvoice.Lines.OrEmpty()[0]
			s.Equal(originalStandardLine.ID, reloadedLine.ID)
			s.Equal(originalStandardLine.Name, reloadedLine.Name)
			s.Equal(originalStandardLine.Period, reloadedLine.Period)
			s.Equal(originalStandardLine.InvoiceAt, reloadedLine.InvoiceAt)
			s.Equal(originalStandardLine.Totals, reloadedLine.Totals)
			s.Nil(reloadedLine.DeletedAt)

			charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
			s.Require().NotNil(charge.Realizations.CurrentRun)
			s.Equal(runID, charge.Realizations.CurrentRun.ID)
			s.False(charge.Realizations.CurrentRun.Immutable)
			s.Nil(charge.Realizations.CurrentRun.AccruedUsage)
			s.Nil(charge.Realizations.CurrentRun.Payment)
			s.RequireTotals(billingtest.ExpectedTotals{Amount: 10, Total: 10}, charge.Realizations.CurrentRun.Totals)
			s.Equal(1, allocationCallback.nrInvocations)
		})
	}
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCustomCurrencyInvalidAccrualResultPersistsNoUsageOrPayment() {
	s.enableFlatFeeCustomCurrenciesWithMockLineage()

	tests := []struct {
		name               string
		accrualResult      flatfee.OnCustomCurrencyOverageAccruedResult
		expectIssueMessage string
	}{
		{
			name: "mismatched fiat total",
			accrualResult: flatfee.OnCustomCurrencyOverageAccruedResult{
				TransactionGroup: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
				TotalFiatAmount: alpacadecimal.NewFromInt(6),
			},
			expectIssueMessage: "custom currency overage booked fiat amount does not match line total",
		},
		{
			name: "missing transaction group",
			accrualResult: flatfee.OnCustomCurrencyOverageAccruedResult{
				TotalFiatAmount: alpacadecimal.NewFromInt(5),
			},
			expectIssueMessage: "transaction group ID is required",
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			// given:
			// - a custom-currency overage is ready for invoice accrual
			// - the mocked ledger returns an invalid accrual result
			// when:
			// - billing approves the invoice through the normal service lifecycle
			// then:
			// - invoice finalization becomes retryable without persisting accrued usage or payment
			ctx := s.T().Context()
			ns := s.GetUniqueNamespace("charges-service-flatfee-custom-currency-invalid-accrual")

			s.FlatFeeTestHandler.Reset()
			defer s.FlatFeeTestHandler.Reset()
			clock.UnFreeze()
			defer clock.UnFreeze()

			defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
			sandboxApp := s.InstallSandboxApp(s.T(), ns)
			customer := s.CreateTestCustomer(ns, "customer-c1")
			_ = s.ProvisionBillingProfile(
				ctx,
				ns,
				sandboxApp.GetID(),
				billingtest.WithManualApproval(),
			)

			customCurrency := s.createTestCustomCurrency(ctx, ns)
			fiatCurrency, err := currencyx.NewFiatCurrency(USD)
			s.Require().NoError(err)

			createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
			servicePeriod := timeutil.ClosedPeriod{
				From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
				To:   datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime(),
			}
			clock.FreezeTime(createAt)

			allocationCallback := newCountedCreditAllocationCallback[flatfee.OnAllocateCreditsInput]()
			s.FlatFeeTestHandler.onAllocateCredits = allocationCallback.Handler(
				s.T(),
				func(flatfee.OnAllocateCreditsInput, ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs {
					return nil
				},
			)

			costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
				FiatCurrency: fiatCurrency,
				Rate:         alpacadecimal.NewFromFloat(0.5),
			})
			created, err := s.Charges.Create(ctx, charges.CreateInput{
				Namespace: ns,
				Intents: []charges.ChargeIntent{
					charges.NewChargeIntent(flatfee.Intent{
						Intent: meta.Intent{
							ManagedBy:         billing.SubscriptionManagedLine,
							UniqueReferenceID: lo.ToPtr("flat-fee-custom-currency-invalid-accrual"),
							CustomerID:        customer.ID,
							Currency:          customCurrency,
							TaxConfig: productcatalog.TaxCodeConfig{
								TaxCodeID: defaults.InvoicingTaxCodeID,
							},
						},
						IntentMutableFields: flatfee.IntentMutableFields{
							IntentMutableFields: meta.IntentMutableFields{
								Name:              "flat-fee-custom-currency",
								ServicePeriod:     servicePeriod,
								FullServicePeriod: servicePeriod,
								BillingPeriod:     servicePeriod,
							},
							InvoiceAt:             servicePeriod.To,
							PaymentTerm:           productcatalog.InArrearsPaymentTerm,
							AmountBeforeProration: alpacadecimal.NewFromInt(10),
						},
						SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
						CostBasis:      &costBasisIntent,
					}),
				},
			})
			s.Require().NoError(err)
			s.Require().Len(created, 1)

			flatFeeCharge, err := created[0].AsFlatFeeCharge()
			s.Require().NoError(err)
			chargeID := flatFeeCharge.GetChargeID()

			clock.FreezeTime(servicePeriod.To)
			invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: customer.GetID(),
				AsOf:     lo.ToPtr(servicePeriod.To),
			})
			s.Require().NoError(err)
			s.Require().Len(invoices, 1)
			invoice := invoices[0]
			s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)

			chargeBeforeApproval := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
			s.Require().NotNil(chargeBeforeApproval.Realizations.CurrentRun)
			runID := chargeBeforeApproval.Realizations.CurrentRun.ID
			s.False(chargeBeforeApproval.Realizations.CurrentRun.Immutable)
			s.Nil(chargeBeforeApproval.Realizations.CurrentRun.AccruedUsage)
			s.Nil(chargeBeforeApproval.Realizations.CurrentRun.Payment)
			s.Equal(1, allocationCallback.nrInvocations)

			accrualCalls := 0
			s.FlatFeeTestHandler.onCustomCurrencyOverageAccrued = func(_ context.Context, input flatfee.OnCustomCurrencyOverageAccruedInput) (flatfee.OnCustomCurrencyOverageAccruedResult, error) {
				accrualCalls++
				s.Equal(chargeID.ID, input.Charge.ID)
				s.Equal(runID, input.Run.ID)
				return test.accrualResult, nil
			}

			authorizedCallback := newCountedLedgerTransactionCallback[flatfee.OnPaymentAuthorizedInput]()
			s.FlatFeeTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T())
			settledCallback := newCountedLedgerTransactionCallback[flatfee.OnPaymentSettledInput]()
			s.FlatFeeTestHandler.onPaymentSettled = settledCallback.Handler(s.T())

			invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
			s.Require().NoError(err)
			s.Equal(billing.StandardInvoiceStatusIssuingLineFinalizationFailed, invoice.Status)
			s.True(invoice.StatusDetails.Failed)
			s.Require().NotNil(invoice.StatusDetails.AvailableActions.Retry)
			s.Require().NotEmpty(invoice.ValidationIssues)
			s.Equal(billing.ValidationIssueSeverityCritical, invoice.ValidationIssues[0].Severity)
			s.Contains(invoice.ValidationIssues[0].Message, test.expectIssueMessage)

			persistedCharge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
			s.Equal(flatfee.StatusActiveRealizationProcessing, persistedCharge.Status)
			s.Require().NotNil(persistedCharge.Realizations.CurrentRun)
			s.Equal(runID, persistedCharge.Realizations.CurrentRun.ID)
			s.False(persistedCharge.Realizations.CurrentRun.Immutable)
			s.Nil(persistedCharge.Realizations.CurrentRun.AccruedUsage)
			s.Empty(persistedCharge.Realizations.CurrentRun.FiatOverageCreditRealizations)
			s.False(persistedCharge.Realizations.CurrentRun.FiatOverageCreditAllocationCompleted)
			s.Nil(persistedCharge.Realizations.CurrentRun.Payment)
			s.Equal(1, accrualCalls)
			s.Zero(authorizedCallback.nrInvocations)
			s.Zero(settledCallback.nrInvocations)

			s.Run("retry succeeds after the ledger recovers", func() {
				// given:
				// - invoice issuing failed without persisting charge-side state
				// - the ledger now returns a valid fiat booking result
				// when:
				// - billing retries the failed invoice action
				// then:
				// - the same run persists accrued usage and reaches settled payment
				accruedTransactionGroupID := ulid.Make().String()
				s.FlatFeeTestHandler.onCustomCurrencyOverageAccrued = func(_ context.Context, input flatfee.OnCustomCurrencyOverageAccruedInput) (flatfee.OnCustomCurrencyOverageAccruedResult, error) {
					accrualCalls++
					s.Equal(chargeID.ID, input.Charge.ID)
					s.Equal(runID, input.Run.ID)

					return flatfee.OnCustomCurrencyOverageAccruedResult{
						TransactionGroup: ledgertransaction.GroupReference{
							TransactionGroupID: accruedTransactionGroupID,
						},
						TotalFiatAmount: alpacadecimal.NewFromInt(5),
					}, nil
				}

				invoice, err = s.BillingService.RetryInvoice(ctx, invoice.GetInvoiceID())
				s.Require().NoError(err)
				s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

				persistedCharge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
				s.Equal(flatfee.StatusFinal, persistedCharge.Status)
				s.Require().NotNil(persistedCharge.Realizations.CurrentRun)
				s.Equal(runID, persistedCharge.Realizations.CurrentRun.ID)
				s.True(persistedCharge.Realizations.CurrentRun.Immutable)
				s.Require().NotNil(persistedCharge.Realizations.CurrentRun.AccruedUsage)
				s.Equal(accruedTransactionGroupID, persistedCharge.Realizations.CurrentRun.AccruedUsage.LedgerTransaction.TransactionGroupID)
				s.RequireTotals(
					billingtest.ExpectedTotals{Amount: 5, Total: 5},
					persistedCharge.Realizations.CurrentRun.AccruedUsage.Totals,
				)
				s.Require().NotNil(persistedCharge.Realizations.CurrentRun.Payment)
				s.Equal(payment.StatusSettled, persistedCharge.Realizations.CurrentRun.Payment.Status)
				s.Equal(float64(5), persistedCharge.Realizations.CurrentRun.Payment.FiatAmount.InexactFloat64())
				s.Require().NotNil(persistedCharge.Realizations.CurrentRun.Payment.Authorized)
				s.Equal(authorizedCallback.id, persistedCharge.Realizations.CurrentRun.Payment.Authorized.TransactionGroupID)
				s.Require().NotNil(persistedCharge.Realizations.CurrentRun.Payment.Settled)
				s.Equal(settledCallback.id, persistedCharge.Realizations.CurrentRun.Payment.Settled.TransactionGroupID)
				s.Equal(2, accrualCalls)
				s.Equal(1, authorizedCallback.nrInvocations)
				s.Equal(1, settledCallback.nrInvocations)
			})
		})
	}
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCustomCurrencyCreditThenInvoiceShrinkExtendMutableLine() {
	s.enableFlatFeeCustomCurrenciesWithMockLineage()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-custom-currency-shrink-extend")

	s.FlatFeeTestHandler.Reset()
	defer s.FlatFeeTestHandler.Reset()
	clock.UnFreeze()
	defer clock.UnFreeze()

	defaults := s.ProvisionDefaultTaxCodes(ctx, ns)
	customInvoicing := s.SetupCustomInvoicing(ns)
	customer := s.CreateTestCustomer(ns, "customer-c1")
	_ = s.ProvisionBillingProfile(
		ctx,
		ns,
		customInvoicing.App.GetID(),
		billingtest.WithManualApproval(),
	)

	customCurrency := s.createTestCustomCurrency(ctx, ns)
	fiatCurrency, err := currencyx.NewFiatCurrency(USD)
	s.Require().NoError(err)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2024-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2025-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	shrunkServicePeriodTo := datetime.MustParseTimeInLocation(s.T(), "2025-01-16T00:00:00Z", time.UTC).AsTime()
	zeroFiatServicePeriodTo := datetime.MustParseTimeInLocation(s.T(), "2025-01-01T00:02:00Z", time.UTC).AsTime()
	clock.FreezeTime(createAt)

	var allocationTargets []float64
	allocationCallback := newCountedCreditAllocationCallback[flatfee.OnAllocateCreditsInput]()
	s.FlatFeeTestHandler.onAllocateCredits = allocationCallback.Handler(
		s.T(),
		func(flatfee.OnAllocateCreditsInput, ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs {
			return nil
		},
		func(t *testing.T, input flatfee.OnAllocateCreditsInput) {
			allocationTargets = append(allocationTargets, input.PreTaxAmountToAllocate.InexactFloat64())
		},
	)

	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})
	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			charges.NewChargeIntent(flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: lo.ToPtr("flat-fee-custom-currency-shrink-extend"),
					CustomerID:        customer.ID,
					Currency:          customCurrency,
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: defaults.InvoicingTaxCodeID,
					},
				},
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "flat-fee-custom-currency",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:   servicePeriod.To,
					PaymentTerm: productcatalog.InArrearsPaymentTerm,
					ProRating: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
					AmountBeforeProration: alpacadecimal.NewFromInt(31),
				},
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				CostBasis:      &costBasisIntent,
			}),
		},
	})
	s.Require().NoError(err)
	s.Require().Len(created, 1)

	flatFeeCharge, err := created[0].AsFlatFeeCharge()
	s.Require().NoError(err)
	chargeID := flatFeeCharge.GetChargeID()

	var (
		invoice billing.StandardInvoice
		lineID  billing.LineID
		runID   flatfee.RealizationRunID
	)

	s.Run("create the mutable custom-currency realization", func() {
		// given:
		// - a prorated 31 TOKENS flat fee uses a 0.5 USD cost basis
		// when:
		// - billing collects its placeholder into a draft invoice
		// then:
		// - the mutable run remains in TOKENS while the standard line is 15.50 USD
		clock.FreezeTime(servicePeriod.To)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		invoice = invoices[0]
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 15.5, Total: 15.5}, invoice.Totals)
		s.Require().Len(invoice.Lines.OrEmpty(), 1)

		line := invoice.Lines.OrEmpty()[0]
		lineID = line.GetLineID()
		s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
			line:               line,
			expectTokenOverage: 31,
			expectCostBasis:    0.5,
			expectFiatTotals: billingtest.ExpectedTotals{
				Amount: 15.5,
				Total:  15.5,
			},
		})

		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Equal(flatfee.StatusActiveRealizationProcessing, charge.Status)
		s.Require().NotNil(charge.Realizations.CurrentRun)
		runID = charge.Realizations.CurrentRun.ID
		s.Require().NotNil(charge.Realizations.CurrentRun.LineID)
		s.Equal(lineID.ID, *charge.Realizations.CurrentRun.LineID)
		s.Equal(servicePeriod, charge.Realizations.CurrentRun.ServicePeriod)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 31, Total: 31}, charge.Realizations.CurrentRun.Totals)
		s.False(charge.Realizations.CurrentRun.NoFiatTransactionRequired)
		s.False(charge.Realizations.CurrentRun.Immutable)
		s.Nil(charge.Realizations.CurrentRun.AccruedUsage)
		s.Nil(charge.Realizations.CurrentRun.Payment)
		s.Equal([]float64{31}, allocationTargets)
		s.Equal(1, allocationCallback.nrInvocations)
	})

	s.Run("shrink the mutable realization", func() {
		// given:
		// - the draft line and realization run are still mutable
		// when:
		// - the charge is shrunk to January 16
		// then:
		// - the same run and standard line are rerated to 15 TOKENS and 7.50 USD
		patch, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
			ChangeSource:           billing.ChangeSourceSystem,
			NewServicePeriodTo:     shrunkServicePeriodTo,
			NewFullServicePeriodTo: servicePeriod.To,
			NewBillingPeriodTo:     shrunkServicePeriodTo,
			NewInvoiceAt:           servicePeriod.From,
		})
		s.Require().NoError(err)
		s.Require().NoError(s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
			CustomerID: customer.GetID(),
			PatchesByChargeID: map[string]charges.Patch{
				chargeID.ID: patch,
			},
		}))

		reloadedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 7.5, Total: 7.5}, reloadedInvoice.Totals)
		s.Require().Len(reloadedInvoice.Lines.OrEmpty(), 1)
		line := reloadedInvoice.Lines.GetByID(lineID.ID)
		s.Require().NotNil(line)
		s.Nil(line.DeletedAt)
		s.Equal(servicePeriod.From, line.Period.From)
		s.Equal(shrunkServicePeriodTo, line.Period.To)
		s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
			line:               line,
			expectTokenOverage: 15,
			expectCostBasis:    0.5,
			expectFiatTotals: billingtest.ExpectedTotals{
				Amount: 7.5,
				Total:  7.5,
			},
		})

		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Equal(flatfee.StatusActiveRealizationProcessing, charge.Status)
		s.Equal(float64(15), charge.State.AmountAfterProration.InexactFloat64())
		s.Require().NotNil(charge.Realizations.CurrentRun)
		s.Equal(runID, charge.Realizations.CurrentRun.ID)
		s.Require().NotNil(charge.Realizations.CurrentRun.LineID)
		s.Equal(lineID.ID, *charge.Realizations.CurrentRun.LineID)
		s.Equal(shrunkServicePeriodTo, charge.Realizations.CurrentRun.ServicePeriod.To)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 15, Total: 15}, charge.Realizations.CurrentRun.Totals)
		s.False(charge.Realizations.CurrentRun.NoFiatTransactionRequired)
		s.False(charge.Realizations.CurrentRun.Immutable)
		s.Nil(charge.Realizations.CurrentRun.AccruedUsage)
		s.Nil(charge.Realizations.CurrentRun.Payment)
		s.Empty(activeGatheringLinesForCharge(&s.BaseSuite, ns, customer.ID, chargeID.ID))
		s.Equal([]float64{31, 15}, allocationTargets)
		s.Equal(2, allocationCallback.nrInvocations)
	})

	s.Run("shrink the mutable realization to an overage that rounds to zero fiat", func() {
		// given:
		// - the mutable run and standard line have a positive fiat overage
		// when:
		// - the charge is shrunk to a positive TOKENS amount that rounds to zero USD
		// then:
		// - the same run and line remain, and the zero-fiat run is finalized without payment
		patch, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
			ChangeSource:           billing.ChangeSourceSystem,
			NewServicePeriodTo:     zeroFiatServicePeriodTo,
			NewFullServicePeriodTo: servicePeriod.To,
			NewBillingPeriodTo:     zeroFiatServicePeriodTo,
			NewInvoiceAt:           servicePeriod.From,
		})
		s.Require().NoError(err)
		s.Require().NoError(s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
			CustomerID: customer.GetID(),
			PatchesByChargeID: map[string]charges.Patch{
				chargeID.ID: patch,
			},
		}))

		reloadedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.RequireTotals(billingtest.ExpectedTotals{}, reloadedInvoice.Totals)
		s.Require().Len(reloadedInvoice.Lines.OrEmpty(), 1)
		line := reloadedInvoice.Lines.GetByID(lineID.ID)
		s.Require().NotNil(line)
		s.Nil(line.DeletedAt)
		s.Equal(servicePeriod.From, line.Period.From)
		s.Equal(zeroFiatServicePeriodTo, line.Period.To)
		s.requireCustomCurrencyOverageLine(requireCustomCurrencyOverageLineInput{
			line:               line,
			expectTokenOverage: 0.001,
			expectCostBasis:    0.5,
			expectFiatTotals:   billingtest.ExpectedTotals{},
		})

		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Equal(flatfee.StatusFinal, charge.Status)
		s.Equal(float64(0.001), charge.State.AmountAfterProration.InexactFloat64())
		s.Require().NotNil(charge.Realizations.CurrentRun)
		s.Equal(runID, charge.Realizations.CurrentRun.ID)
		s.Require().NotNil(charge.Realizations.CurrentRun.LineID)
		s.Equal(lineID.ID, *charge.Realizations.CurrentRun.LineID)
		s.Equal(zeroFiatServicePeriodTo, charge.Realizations.CurrentRun.ServicePeriod.To)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 0.001, Total: 0.001}, charge.Realizations.CurrentRun.Totals)
		s.True(charge.Realizations.CurrentRun.NoFiatTransactionRequired)
		s.False(charge.Realizations.CurrentRun.Immutable)
		s.Nil(charge.Realizations.CurrentRun.AccruedUsage)
		s.Nil(charge.Realizations.CurrentRun.Payment)
		s.Empty(activeGatheringLinesForCharge(&s.BaseSuite, ns, customer.ID, chargeID.ID))
		s.Equal([]float64{31, 15, 0.001}, allocationTargets)
		s.Equal(3, allocationCallback.nrInvocations)
	})

	s.Run("extend the finalized zero-fiat realization to the full period", func() {
		// given:
		// - the finalized run and standard line represent a positive TOKENS overage that rounds to zero USD
		// when:
		// - the charge is extended back to its full period
		// then:
		// - the zero-fiat line becomes history and replacement gathering work covers the full period
		patch, err := meta.NewPatchExtend(meta.NewPatchExtendInput{
			ChangeSource:           billing.ChangeSourceSystem,
			NewServicePeriodTo:     servicePeriod.To,
			NewFullServicePeriodTo: servicePeriod.To,
			NewBillingPeriodTo:     servicePeriod.To,
			NewInvoiceAt:           servicePeriod.To,
		})
		s.Require().NoError(err)
		s.Require().NoError(s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
			CustomerID: customer.GetID(),
			PatchesByChargeID: map[string]charges.Patch{
				chargeID.ID: patch,
			},
		}))

		reloadedInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.RequireTotals(billingtest.ExpectedTotals{}, reloadedInvoice.Totals)
		s.Empty(reloadedInvoice.Lines.OrEmpty())

		charge := s.mustGetFlatFeeChargeByIDWithDetailedLines(chargeID)
		s.Equal(flatfee.StatusActive, charge.Status)
		s.Equal(float64(31), charge.State.AmountAfterProration.InexactFloat64())
		s.Nil(charge.Realizations.CurrentRun)
		s.Require().Len(charge.Realizations.PriorRuns, 1)
		s.Equal(runID, charge.Realizations.PriorRuns[0].ID)
		s.True(charge.Realizations.PriorRuns[0].NoFiatTransactionRequired)
		s.False(charge.Realizations.PriorRuns[0].Immutable)

		gatheringLines := activeGatheringLinesForCharge(&s.BaseSuite, ns, customer.ID, chargeID.ID)
		s.Require().Len(gatheringLines, 1)
		s.Equal(servicePeriod, gatheringLines[0].ServicePeriod)
		s.Equal([]float64{31, 15, 0.001}, allocationTargets)
		s.Equal(3, allocationCallback.nrInvocations)
	})
}

func (s *InvoicableChargesTestSuite) TestUsageBasedGatheringPreviewPopulatesTotalsWithoutRealizationRun() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-gathering-preview")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				}),
				name:              "usage-based-gathering-preview",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "usage-based-gathering-preview",
				featureKey:        meterSlug,
			}),
		},
	})
	s.NoError(err)
	s.Require().Len(created, 1)

	usageBasedCharge, err := created[0].AsUsageBasedCharge()
	s.NoError(err)
	usageBasedChargeID := usageBasedCharge.GetChargeID()

	s.MockStreamingConnector.AddSimpleEvent(
		meterSlug,
		15,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)

	s.assertGatheringPreview(assertGatheringPreviewInput{
		Namespace:  ns,
		CustomerID: cust.ID,
		ExpectedInvoiceTotals: billingtest.ExpectedTotals{
			Amount: 30,
			Total:  30,
		},
		ExpectedLines: 1,
		ExpectedLineTotals: billingtest.ExpectedTotals{
			Amount: 30,
			Total:  30,
		},
		ExpectedDetailedLines: 1,
		AssertLine: func(previewLine *billing.StandardLine) {
			s.Require().NotNil(previewLine.UsageBased)
			s.Require().NotNil(previewLine.UsageBased.MeteredQuantity)
			s.Require().NotNil(previewLine.UsageBased.Quantity)
			s.Require().NotNil(previewLine.UsageBased.MeteredPreLinePeriodQuantity)
			s.Require().NotNil(previewLine.UsageBased.PreLinePeriodQuantity)
			s.Equal(float64(15), lo.FromPtr(previewLine.UsageBased.MeteredQuantity).InexactFloat64())
			s.Equal(float64(15), lo.FromPtr(previewLine.UsageBased.Quantity).InexactFloat64())
			s.Equal(float64(0), lo.FromPtr(previewLine.UsageBased.MeteredPreLinePeriodQuantity).InexactFloat64())
			s.Equal(float64(0), lo.FromPtr(previewLine.UsageBased.PreLinePeriodQuantity).InexactFloat64())
		},
	})

	chargeAfterPreview := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
	s.Nil(chargeAfterPreview.State.CurrentRealizationRunID)
	s.Empty(chargeAfterPreview.Realizations)
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditThenInvoiceImmutableProration() {
	for _, creditNotesAvailable := range []bool{true, false} {
		name := "credit notes unavailable"
		if creditNotesAvailable {
			name = "credit notes available"
		}

		s.Run(name, func() {
			flatFeeService := s.Charges.flatFeeService.(interface {
				SetCreditNotesSupportedByLineUpdater(*testing.T, bool) error
			})
			s.NoError(flatFeeService.SetCreditNotesSupportedByLineUpdater(s.T(), creditNotesAvailable))

			runFlatFeeCreditThenInvoiceImmutableProrationScenario(&s.BaseSuite, creditNotesAvailable)
		})
	}
}

func runFlatFeeCreditThenInvoiceImmutableProrationScenario(s *BaseSuite, expectReplacementGatheringLine bool) {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-then-invoice-immutable-proration")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	shrunkServicePeriodTo := datetime.MustParseTimeInLocation(s.T(), "2026-01-16T00:00:00Z", time.UTC).AsTime()

	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	var (
		flatFeeChargeID meta.ChargeID
		invoice         billing.StandardInvoice
		lineID          billing.LineID
	)

	s.Run("given a fully credited immutable flat fee invoice", func() {
		// given:
		// - a credit-then-invoice flat fee has a fully credited immutable invoice line
		s.FlatFeeTestHandler.onAllocateCredits = func(ctx context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			return creditrealization.CreateAllocationInputs{
				{
					ServicePeriod: input.ServicePeriod,
					Amount:        input.PreTaxAmountToAllocate,
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: ulid.Make().String(),
					},
				},
			}, nil
		}
		defer s.FlatFeeTestHandler.Reset()

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(31),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              "flat-fee-credit-then-invoice-immutable-proration",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "flat-fee-credit-then-invoice-immutable-proration",
					proRating: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				}),
			},
		})
		s.NoError(err)
		s.Len(created, 1)

		flatFeeCharge, err := created[0].AsFlatFeeCharge()
		s.NoError(err)
		flatFeeChargeID = flatFeeCharge.GetChargeID()

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.From),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]
		s.Require().Len(invoice.Lines.OrEmpty(), 1)
		lineID = invoice.Lines.OrEmpty()[0].GetLineID()

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.True(invoice.StatusDetails.Immutable)

		charge := mustGetFlatFeeChargeWithExpands(s, flatFeeChargeID, meta.Expands{meta.ExpandRealizations})
		s.Require().NotNil(charge.Realizations.CurrentRun)
		s.True(charge.Realizations.CurrentRun.Immutable)
		s.Equal(lineID.ID, lo.FromPtr(charge.Realizations.CurrentRun.LineID))
	})

	s.Run("when immutable invoice proration is requested", func() {
		// when:
		// - the charge is shrunk to a prorated amount
		patch, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
			ChangeSource:           billing.ChangeSourceSystem,
			NewServicePeriodTo:     shrunkServicePeriodTo,
			NewFullServicePeriodTo: servicePeriod.To,
			NewBillingPeriodTo:     shrunkServicePeriodTo,
			NewInvoiceAt:           servicePeriod.From,
		})
		s.NoError(err)

		s.NoError(s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
			CustomerID: cust.GetID(),
			PatchesByChargeID: map[string]charges.Patch{
				flatFeeChargeID.ID: patch,
			},
		}))

		// then:
		// - the immutable invoice is not rewritten and records a warning
		fetchedInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand: billing.InvoiceExpands{
				billing.InvoiceExpandLines,
			},
		})
		s.NoError(err)
		standardInvoice, err := fetchedInvoice.AsStandardInvoice()
		s.NoError(err)
		line := standardInvoice.Lines.GetByID(lineID.ID)
		s.Require().NotNil(line)
		s.Nil(line.DeletedAt)
		s.Require().Len(standardInvoice.ValidationIssues, 1)
		s.Equal(billing.ImmutableInvoiceHandlingNotSupportedErrorCode, standardInvoice.ValidationIssues[0].Code)
		s.Equal(billing.ComponentName("charges.invoiceupdater"), standardInvoice.ValidationIssues[0].Component)

		activeGatheringLines := activeGatheringLinesForCharge(s, ns, cust.ID, flatFeeChargeID.ID)

		if expectReplacementGatheringLine {
			charge := mustGetFlatFeeChargeWithExpands(s, flatFeeChargeID, meta.Expands{meta.ExpandRealizations})
			s.Equal(flatfee.StatusActive, charge.Status)
			s.Nil(charge.Realizations.CurrentRun)
			s.Require().Len(activeGatheringLines, 1)
			s.Equal(servicePeriod.From, activeGatheringLines[0].ServicePeriod.From)
			s.Equal(shrunkServicePeriodTo, activeGatheringLines[0].ServicePeriod.To)
			return
		}

		charge := mustGetFlatFeeChargeWithExpands(s, flatFeeChargeID, meta.Expands{meta.ExpandRealizations})
		s.Equal(flatfee.StatusFinal, charge.Status)
		s.Require().NotNil(charge.Realizations.CurrentRun)
		s.Equal(lineID.ID, lo.FromPtr(charge.Realizations.CurrentRun.LineID))
		s.Empty(activeGatheringLines)
	})
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditThenInvoiceZeroAmountCreatesNoGatheringLine() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-then-invoice-zero-amount")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(0),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:              "flat-fee-credit-then-invoice-zero-amount",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "flat-fee-credit-then-invoice-zero-amount",
			}),
		},
	})
	s.NoError(err)
	s.Require().Len(created, 1)
	s.Equal(meta.ChargeTypeFlatFee, created[0].Type())

	flatFeeCharge, err := created[0].AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatfee.StatusCreated, flatFeeCharge.Status)
	s.Equal(float64(0), flatFeeCharge.State.AmountAfterProration.InexactFloat64())
	s.Empty(activeGatheringLinesForCharge(&s.BaseSuite, ns, cust.ID, flatFeeCharge.ID))
}

func (s *InvoicableChargesTestSuite) TestFlatFeePartialCreditRealizations() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-partial-credit-realizations")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
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

	clock.SetTime(servicePeriod.From)

	flatFeeChargeID := meta.ChargeID{}

	s.Run("create new upcoming charge", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
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

		gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespace:  ns,
			Customers:  []string{cust.ID},
			Currencies: []currencyx.FiatCode{currencyx.FiatCode(currency.USD)},
			Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
		})
		s.NoError(err)
		s.Len(gatheringInvoices.Items, 1)
		gatheringInvoice := gatheringInvoices.Items[0]

		lines := gatheringInvoice.Lines.OrEmpty()
		s.Len(lines, 1)
		gatheringLine := lines[0]

		s.Equal(flatFeeCharge.ID, *gatheringLine.ChargeID)

		// TODO: validate periods, price, etc.

		flatFeeChargeID = flatFeeCharge.GetChargeID()
	})
	var stdInvoiceID billing.InvoiceID
	var stdLineID billing.LineID
	s.Run("invoice pending lines creates partial credit realizations", func() {
		defer s.FlatFeeTestHandler.Reset()

		testTrnsGroupID := ulid.Make().String()
		creditRealizationCallbackInvocations := 0
		s.FlatFeeTestHandler.onAllocateCredits = func(ctx context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			creditRealizationCallbackInvocations++

			return creditrealization.CreateAllocationInputs{
				{
					ServicePeriod: input.ServicePeriod,
					Amount:        input.PreTaxAmountToAllocate.Mul(alpacadecimal.NewFromFloat(0.3)), // 30% as credits
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: testTrnsGroupID,
					},
				},
			}, nil
		}

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

		s.Equal(1, creditRealizationCallbackInvocations)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		s.Equal(flatFeeChargeID.ID, updatedFlatFeeCharge.ID)

		// Validate the credit realizations
		// The charge should have $30 realized as credits
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.Len(updatedFlatFeeCharge.Realizations.CurrentRun.CreditRealizations, 1)
		creditRealization := updatedFlatFeeCharge.Realizations.CurrentRun.CreditRealizations[0]
		s.Equal(testTrnsGroupID, creditRealization.LedgerTransaction.TransactionGroupID)
		s.Equal(servicePeriod.From, creditRealization.ServicePeriod.From)
		s.Equal(servicePeriod.To, creditRealization.ServicePeriod.To)
		s.Equal(float64(30), creditRealization.Amount.InexactFloat64())

		// Validate the standard invoice's contents
		// Invoice totals should be $70
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       100,
			Total:        70,
			CreditsTotal: 30,
		}, invoice.Totals)

		// Validate the standard line's contents
		// Line totals should be $70
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       100,
			Total:        70,
			CreditsTotal: 30,
		}, stdLine.Totals)

		// The line should have a credit realization intent
		s.Len(stdLine.CreditsApplied, 1)
		creditRealizationIntent := stdLine.CreditsApplied[0]
		s.Equal(float64(30), creditRealizationIntent.Amount.InexactFloat64())
		s.Equal(creditRealization.ID, creditRealizationIntent.CreditRealizationID)

		// The line should have a single detailed line
		s.Len(stdLine.DetailedLines, 1)
		detailedLine := stdLine.DetailedLines[0]
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       100,
			Total:        70,
			CreditsTotal: 30,
		}, detailedLine.Totals)

		// The detailed line should have a credit realization intent
		s.Len(detailedLine.CreditsApplied, 1)
		creditRealizationDetail := detailedLine.CreditsApplied[0]
		s.Equal(float64(30), creditRealizationDetail.Amount.InexactFloat64())
		s.Equal(creditRealization.ID, creditRealizationDetail.CreditRealizationID)

		flatFeeWithDetailedLines := s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID)
		s.Require().NotNil(flatFeeWithDetailedLines.Realizations.CurrentRun)
		s.True(flatFeeWithDetailedLines.Realizations.CurrentRun.DetailedLines.IsPresent())
		s.Len(flatFeeWithDetailedLines.Realizations.CurrentRun.DetailedLines.OrEmpty(), 1)
		s.Equal(detailedLine.ChildUniqueReferenceID, flatFeeWithDetailedLines.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].ChildUniqueReferenceID)
		s.Equal(detailedLine.Totals.Total.String(), flatFeeWithDetailedLines.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].Totals.Total.String())
		s.Equal(detailedLine.Quantity.String(), flatFeeWithDetailedLines.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].Quantity.String())
		s.Len(flatFeeWithDetailedLines.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].CreditsApplied, 1)

		stdInvoiceID = invoice.GetInvoiceID()
		s.NotEmpty(stdInvoiceID)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	})
	s.Run("approve invoice accrues usage without authorizing payment", func() {
		defer s.FlatFeeTestHandler.Reset()

		authorizedCallback := newCountedLedgerTransactionCallback[flatfee.OnPaymentAuthorizedInput]()
		// Use non-fatal assertions inside handler callbacks so failures are reported
		// on the callback's testing context without aborting the parent test flow.
		s.FlatFeeTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T(), func(t *testing.T, input flatfee.OnPaymentAuthorizedInput) {
			assert.True(t, input.FiatAmount.IsPositive())
			assert.NotNil(t, input.Charge.Realizations.CurrentRun)
			assert.NotNil(t, input.Charge.Realizations.CurrentRun.AccruedUsage)
			assert.Nil(t, input.Charge.Realizations.CurrentRun.Payment)
			assert.Equal(t, flatfee.StatusActiveAwaitingPaymentSettlement, input.Charge.Status)
		})

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[flatfee.OnInvoiceUsageAccruedInput]()
		s.FlatFeeTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		invoice, err := s.BillingService.ApproveInvoice(ctx, stdInvoiceID)
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)
		s.Equal(0, authorizedCallback.nrInvocations)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		// Invoice usage accrued callback should have been invoked
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		accruedUsage := updatedFlatFeeCharge.Realizations.CurrentRun.AccruedUsage
		s.NotNil(accruedUsage)
		s.Equal(invoiceUsageAccruedCallback.id, accruedUsage.LedgerTransaction.TransactionGroupID, "ledger transaction gets recorded")
		s.Equal(servicePeriod, accruedUsage.ServicePeriod, "service period should be the same as the input")
		s.NotNil(updatedFlatFeeCharge.Realizations.CurrentRun.LineID, "run line ID should be set")
		s.Equal(stdLineID.ID, *updatedFlatFeeCharge.Realizations.CurrentRun.LineID, "run line ID should be the same as the standard line")
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       100,
			Total:        70,
			CreditsTotal: 30,
		}, accruedUsage.Totals)

		// Payment authorization should not be persisted until the payment flow advances past pending.
		s.Nil(updatedFlatFeeCharge.Realizations.CurrentRun.Payment)
		s.Equal(flatfee.StatusActiveAwaitingPaymentSettlement, updatedFlatFeeCharge.Status)
	})

	s.Run("trigger paid authorizes then settles payment", func() {
		defer s.FlatFeeTestHandler.Reset()

		authorizedCallback := newCountedLedgerTransactionCallback[flatfee.OnPaymentAuthorizedInput]()
		// Use non-fatal assertions inside handler callbacks so failures are reported
		// on the callback's testing context without aborting the parent test flow.
		s.FlatFeeTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T(), func(t *testing.T, input flatfee.OnPaymentAuthorizedInput) {
			assert.True(t, input.FiatAmount.IsPositive())
			assert.NotNil(t, input.Charge.Realizations.CurrentRun)
			assert.Nil(t, input.Charge.Realizations.CurrentRun.Payment)
			assert.NotNil(t, input.Charge.Realizations.CurrentRun.AccruedUsage)
			assert.Equal(t, flatfee.StatusActiveAwaitingPaymentSettlement, input.Charge.Status)
		})

		settledCallback := newCountedLedgerTransactionCallback[flatfee.OnPaymentSettledInput]()
		// Use non-fatal assertions inside handler callbacks so failures are reported
		// on the callback's testing context without aborting the parent test flow.
		s.FlatFeeTestHandler.onPaymentSettled = settledCallback.Handler(s.T(), func(t *testing.T, input flatfee.OnPaymentSettledInput) {
			assert.True(t, input.FiatAmount.IsPositive())
			assert.NotNil(t, input.Charge.Realizations.CurrentRun)
			assert.NotNil(t, input.Charge.Realizations.CurrentRun.Payment)
			assert.NotNil(t, input.Charge.Realizations.CurrentRun.Payment.Authorized)
			assert.Nil(t, input.Charge.Realizations.CurrentRun.Payment.Settled)
			assert.Equal(t, authorizedCallback.id, input.Charge.Realizations.CurrentRun.Payment.Authorized.TransactionGroupID)
			assert.Equal(t, payment.StatusAuthorized, input.Charge.Realizations.CurrentRun.Payment.Status)
			assert.Equal(t, flatfee.StatusActiveAwaitingPaymentSettlement, input.Charge.Status)
		})

		invoice, err := s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: stdInvoiceID,
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

		s.Equal(1, authorizedCallback.nrInvocations)
		s.Equal(1, settledCallback.nrInvocations)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.Equal(authorizedCallback.id, updatedFlatFeeCharge.Realizations.CurrentRun.Payment.Authorized.TransactionGroupID)
		s.Equal(settledCallback.id, updatedFlatFeeCharge.Realizations.CurrentRun.Payment.Settled.TransactionGroupID)
		s.Equal(flatfee.StatusFinal, updatedFlatFeeCharge.Status)
	})
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditThenInvoiceInAdvanceWithPromotionalCredits() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-then-invoice-in-advance-promotional")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	expectedTotals := billingtest.ExpectedTotals{
		Amount:       7,
		CreditsTotal: 5,
		Total:        2,
	}

	var (
		flatFeeChargeID meta.ChargeID
		invoice         billing.StandardInvoice
		stdLineID       billing.LineID
	)

	s.Run("given promotional credits and an in-advance flat fee", func() {
		// Given the customer has 5 promotional credits.
		promotionalCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = promotionalCallback.Handler(s.T())
		defer s.CreditPurchaseTestHandler.Reset()

		res := s.grantPromotionalCredits(ctx, cust.GetID(), 5)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())
		s.Equal(1, promotionalCallback.nrInvocations)

		// And a future in-advance flat fee is created for 7 USD.
		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(7),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              "flat-fee-credit-then-invoice-in-advance-promotional",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "flat-fee-credit-then-invoice-in-advance-promotional",
				}),
			},
		})
		s.NoError(err)
		s.Len(created, 1)

		flatFeeCharge, err := created[0].AsFlatFeeCharge()
		s.NoError(err)
		flatFeeChargeID = flatFeeCharge.GetChargeID()
	})

	s.Run("when the charge becomes active and draft invoice is created", func() {
		defer s.FlatFeeTestHandler.Reset()

		creditAllocationCallback := newCountedCreditAllocationCallback[flatfee.OnAllocateCreditsInput]()
		s.FlatFeeTestHandler.onAllocateCredits = creditAllocationCallback.Handler(
			s.T(),
			func(input flatfee.OnAllocateCreditsInput, ledgerTransaction ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs {
				return creditrealization.CreateAllocationInputs{
					{
						ServicePeriod:     input.ServicePeriod,
						Amount:            alpacadecimal.NewFromInt(5),
						LedgerTransaction: ledgerTransaction,
					},
				}
			},
			func(t *testing.T, input flatfee.OnAllocateCreditsInput) {
				assert.Equal(t, flatFeeChargeID.ID, input.Charge.ID)
				assert.Equal(t, servicePeriod, input.ServicePeriod)
				assert.Equal(t, float64(7), input.PreTaxAmountToAllocate.InexactFloat64())
			},
		)

		clock.FreezeTime(servicePeriod.From)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.From),
		})

		// Then a manually approved draft invoice contains the credited standard line and matching run details.
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		s.RequireTotals(expectedTotals, invoice.Totals)
		s.Equal(1, creditAllocationCallback.nrInvocations)
		s.Require().Len(invoice.Lines.OrEmpty(), 1)
		stdLineID = invoice.Lines.OrEmpty()[0].GetLineID()
		s.assertFlatFeeCreditThenInvoiceLineAndRun(assertFlatFeeCreditThenInvoiceLineAndRunInput{
			Invoice:                invoice,
			FlatFeeChargeID:        flatFeeChargeID,
			ServicePeriod:          servicePeriod,
			ExpectedTotals:         expectedTotals,
			ExpectedCreditsApplied: alpacadecimal.NewFromInt(5),
			ExpectAccruedUsage:     false,
		})
	})

	s.Run("when the draft invoice is approved into payment pending", func() {
		defer s.FlatFeeTestHandler.Reset()

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[flatfee.OnInvoiceUsageAccruedInput]()
		s.FlatFeeTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T(), func(t *testing.T, input flatfee.OnInvoiceUsageAccruedInput) {
			assert.Equal(t, flatFeeChargeID.ID, input.Charge.ID)
			assert.Equal(t, servicePeriod, input.ServicePeriod)
			billingtest.AssertTotals(t, expectedTotals, input.Totals)
		})

		var err error
		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())

		// Then the custom-invoicing invoice is payment-pending and preserves line/run details with accrued invoice usage.
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		s.RequireTotals(expectedTotals, invoice.Totals)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)
		paymentPendingLineID := s.assertFlatFeeCreditThenInvoiceLineAndRun(assertFlatFeeCreditThenInvoiceLineAndRunInput{
			Invoice:                       invoice,
			FlatFeeChargeID:               flatFeeChargeID,
			ServicePeriod:                 servicePeriod,
			ExpectedTotals:                expectedTotals,
			ExpectedCreditsApplied:        alpacadecimal.NewFromInt(5),
			ExpectAccruedUsage:            true,
			InvoiceUsageAccruedCallbackID: invoiceUsageAccruedCallback.id,
		})
		s.Equal(stdLineID, paymentPendingLineID)
	})
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditThenInvoiceInvoiceAtBeforeServicePeriodStart() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-then-invoice-before-service-period")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2026-05-28T18:00:00Z", time.UTC).AsTime()
	invoiceAt := datetime.MustParseTimeInLocation(s.T(), "2026-06-28T17:38:30Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-06-30T17:38:30Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-07-01T17:38:30Z", time.UTC).AsTime(),
	}
	billingPeriod := timeutil.ClosedPeriod{
		From: invoiceAt,
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-07-28T17:38:30Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	flatFeeChargeID := meta.ChargeID{}

	s.Run("given a flat fee invoiceable before service period start", func() {
		// given:
		// - a non-zero CTI flat fee has invoice_at before service period start
		// - invoice_at belongs to the billing period, while service period carries the charged usage window
		// when:
		// - the charge is created before invoice_at
		// then:
		// - the charge waits for invoice_at, not service period start
		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				charges.NewChargeIntent(flatfee.Intent{
					Intent: meta.Intent{
						ManagedBy:         billing.SubscriptionManagedLine,
						UniqueReferenceID: lo.ToPtr("flat-fee-invoice-at-before-service-period"),
						CustomerID:        cust.ID,
						Currency:          currenciestestutils.NewFiatCurrency(s.T(), USD),
					},
					IntentMutableFields: flatfee.IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{
							Name:              "flat-fee-invoice-at-before-service-period",
							ServicePeriod:     servicePeriod,
							FullServicePeriod: servicePeriod,
							BillingPeriod:     billingPeriod,
						},
						InvoiceAt:             invoiceAt,
						PaymentTerm:           productcatalog.InAdvancePaymentTerm,
						AmountBeforeProration: alpacadecimal.NewFromInt(100),
						ProRating: productcatalog.ProRatingConfig{
							Enabled: true,
							Mode:    productcatalog.ProRatingModeProratePrices,
						},
					},
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				}),
			},
		})
		s.NoError(err)
		s.Len(created, 1)

		flatFeeCharge, err := created[0].AsFlatFeeCharge()
		s.NoError(err)
		flatFeeChargeID = flatFeeCharge.GetChargeID()

		fetchedCharge := s.mustGetChargeByID(flatFeeChargeID)
		fetchedFF, err := fetchedCharge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatfee.StatusCreated, fetchedFF.Status)
		s.Require().NotNil(fetchedFF.State.AdvanceAfter)
		s.True(invoiceAt.Equal(*fetchedFF.State.AdvanceAfter))
		s.Nil(fetchedFF.Realizations.CurrentRun)
	})

	s.Run("when pending lines are invoiced at invoice_at", func() {
		defer s.FlatFeeTestHandler.Reset()
		defer clock.UnFreeze()

		creditAllocationCallback := newCountedCreditAllocationCallback[flatfee.OnAllocateCreditsInput]()
		s.FlatFeeTestHandler.onAllocateCredits = creditAllocationCallback.Handler(
			s.T(),
			func(flatfee.OnAllocateCreditsInput, ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs {
				return nil
			},
			func(t *testing.T, input flatfee.OnAllocateCreditsInput) {
				assert.Equal(t, flatFeeChargeID.ID, input.Charge.ID)
				assert.Equal(t, servicePeriod, input.ServicePeriod)
				assert.Equal(t, float64(100), input.PreTaxAmountToAllocate.InexactFloat64())
			},
		)

		// given:
		// - wall clock is still before invoice_at
		// when:
		// - pending flat-fee lines are invoiced
		// then:
		// - no invoice is created yet
		beforeInvoiceAt := invoiceAt.Add(-time.Nanosecond)
		clock.FreezeTime(beforeInvoiceAt)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(beforeInvoiceAt),
		})
		s.ErrorIs(err, billing.ErrInvoiceCreateNoLines)
		s.Empty(invoices)
		s.Equal(0, creditAllocationCallback.nrInvocations)

		// given:
		// - wall clock has reached invoice_at but is still before service period start
		// when:
		// - pending flat-fee lines are invoiced
		// then:
		// - the CTI lifecycle accepts invoice_created and creates the run
		clock.FreezeTime(invoiceAt)
		invoices, err = s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(invoiceAt),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		s.Equal(1, creditAllocationCallback.nrInvocations)

		invoice := invoices[0]
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		s.Require().Len(invoice.Lines.OrEmpty(), 1)

		stdLine := invoice.Lines.OrEmpty()[0]
		s.Equal(flatFeeChargeID.ID, lo.FromPtr(stdLine.ChargeID))
		s.Equal(servicePeriod, stdLine.Period)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount: 100,
			Total:  100,
		}, stdLine.Totals)

		flatFeeWithDetailedLines := s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID)
		s.Equal(flatfee.StatusActiveRealizationProcessing, flatFeeWithDetailedLines.Status)
		s.Require().NotNil(flatFeeWithDetailedLines.Realizations.CurrentRun)

		currentRun := flatFeeWithDetailedLines.Realizations.CurrentRun
		s.Equal(servicePeriod, currentRun.ServicePeriod)
		s.Require().NotNil(currentRun.LineID)
		s.Equal(stdLine.ID, *currentRun.LineID)
		s.Require().NotNil(currentRun.InvoiceID)
		s.Equal(invoice.ID, *currentRun.InvoiceID)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount: 100,
			Total:  100,
		}, currentRun.Totals)
	})
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditThenInvoiceFullyCreditedDoesNotAccrueInvoiceUsage() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-then-invoice-fully-credited")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(servicePeriod.From)

	flatFeeChargeID := meta.ChargeID{}

	s.Run("create charge", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              "flat-fee-fully-credited",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "flat-fee-fully-credited",
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		flatFeeCharge, err := res[0].AsFlatFeeCharge()
		s.NoError(err)
		flatFeeChargeID = flatFeeCharge.GetChargeID()
	})

	var invoice billing.StandardInvoice

	s.Run("invoice pending lines fully settled by credits", func() {
		defer s.FlatFeeTestHandler.Reset()

		s.FlatFeeTestHandler.onAllocateCredits = func(ctx context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			return creditrealization.CreateAllocationInputs{
				{
					ServicePeriod: input.ServicePeriod,
					Amount:        input.PreTaxAmountToAllocate,
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: ulid.Make().String(),
					},
				},
			}, nil
		}

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.From),
		})
		s.NoError(err)
		s.Len(invoices, 1)

		invoice = invoices[0]
		s.Len(invoice.Lines.OrEmpty(), 1)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       100,
			CreditsTotal: 100,
		}, invoice.Totals)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.Len(updatedFlatFeeCharge.Realizations.CurrentRun.CreditRealizations, 1)
		s.Nil(updatedFlatFeeCharge.Realizations.CurrentRun.AccruedUsage)

		flatFeeWithDetailedLines := s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID)
		s.Require().NotNil(flatFeeWithDetailedLines.Realizations.CurrentRun)
		s.True(flatFeeWithDetailedLines.Realizations.CurrentRun.DetailedLines.IsPresent())
		s.Len(flatFeeWithDetailedLines.Realizations.CurrentRun.DetailedLines.OrEmpty(), len(invoice.Lines.OrEmpty()[0].DetailedLines))
	})

	s.Run("finalize invoice without invoice usage accrual", func() {
		defer s.FlatFeeTestHandler.Reset()

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[flatfee.OnInvoiceUsageAccruedInput]()
		s.FlatFeeTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		lineEngine := s.Charges.flatFeeService.GetLineEngine()
		lines, err := lineEngine.OnCollectionCompleted(ctx, billing.OnCollectionCompletedInput{
			Invoice: invoice,
			Lines:   invoice.Lines.OrEmpty(),
		})
		s.NoError(err)
		invoice.Lines = billing.NewStandardInvoiceLines(lines)

		lines, err = lineEngine.OnInvoiceFinalizing(ctx, billing.OnInvoiceFinalizingInput{
			Invoice: invoice,
			Lines:   invoice.Lines.OrEmpty(),
		})
		s.NoError(err)
		invoice.Lines = billing.NewStandardInvoiceLines(lines)
		s.Equal(0, invoiceUsageAccruedCallback.nrInvocations)

		updatedFlatFeeCharge := s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID)
		s.Equal(flatfee.StatusActiveRealizationIssuing, updatedFlatFeeCharge.Status)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.Nil(updatedFlatFeeCharge.Realizations.CurrentRun.AccruedUsage)
		s.False(updatedFlatFeeCharge.Realizations.CurrentRun.Immutable)
		s.True(updatedFlatFeeCharge.Realizations.CurrentRun.DetailedLines.IsPresent())
		s.Len(updatedFlatFeeCharge.Realizations.CurrentRun.DetailedLines.OrEmpty(), len(invoice.Lines.OrEmpty()[0].DetailedLines))

		mismatchedLine, err := invoice.Lines.OrEmpty()[0].Clone()
		s.NoError(err)
		mismatchedLine.ID = "mismatched-issued-line"
		err = lineEngine.OnInvoiceIssued(ctx, billing.OnInvoiceIssuedInput{
			Invoice: invoice,
			Lines:   billing.StandardLines{mismatchedLine},
		})
		s.ErrorContains(err, "does not match issued line")
		s.Equal(flatfee.StatusActiveRealizationIssuing, s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID).Status)

		err = lineEngine.OnInvoiceIssued(ctx, billing.OnInvoiceIssuedInput{
			Invoice: invoice,
			Lines:   invoice.Lines.OrEmpty(),
		})
		s.NoError(err)
		s.Equal(0, invoiceUsageAccruedCallback.nrInvocations)
		updatedFlatFeeCharge = s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID)
		s.Equal(flatfee.StatusFinal, updatedFlatFeeCharge.Status)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.True(updatedFlatFeeCharge.Realizations.CurrentRun.Immutable)
	})
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditThenInvoiceZeroAmountNonZeroChargesAccruesInvoiceUsage() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-then-invoice-zero-amount-charges")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.SetTime(servicePeriod.From)

	flatFeeChargeID := meta.ChargeID{}
	var invoice billing.StandardInvoice

	s.Run("create charge and draft invoice", func() {
		// given:
		// - a credit-then-invoice flat fee charge exists for the customer
		// when:
		// - billing invoices pending lines at the service period start
		// then:
		// - the draft invoice has one standard line and the charge has a mutable run
		s.FlatFeeTestHandler.onAllocateCredits = func(context.Context, flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			return nil, nil
		}

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
						Amount:      alpacadecimal.NewFromInt(100),
						PaymentTerm: productcatalog.InAdvancePaymentTerm,
					}),
					name:              "flat-fee-zero-amount-non-zero-charges",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "flat-fee-zero-amount-non-zero-charges",
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		flatFeeCharge, err := res[0].AsFlatFeeCharge()
		s.NoError(err)
		flatFeeChargeID = flatFeeCharge.GetChargeID()

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.From),
		})
		s.NoError(err)
		s.Len(invoices, 1)

		invoice = invoices[0]
		s.Len(invoice.Lines.OrEmpty(), 1)
		fetchedFlatFeeCharge := s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID)
		s.Equal(flatfee.StatusActiveRealizationProcessing, fetchedFlatFeeCharge.Status)
	})

	s.Run("issue invoice with zero amount and non-zero charges", func() {
		// given:
		// - the standard line has zero Amount but non-zero ChargesTotal and Total
		// when:
		// - the flat-fee line engine finalizes and then issues the invoice
		// then:
		// - invoice usage accrual runs only at issuance because the payable total is non-zero
		defer s.FlatFeeTestHandler.Reset()

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[flatfee.OnInvoiceUsageAccruedInput]()
		s.FlatFeeTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T(), func(t *testing.T, input flatfee.OnInvoiceUsageAccruedInput) {
			billingtest.AssertTotals(t, billingtest.ExpectedTotals{
				Amount:       0,
				ChargesTotal: 100,
				Total:        100,
			}, input.Totals)
		})

		lines := invoice.Lines.OrEmpty()
		lines[0].Totals = billingtotals.Totals{
			ChargesTotal: alpacadecimal.NewFromInt(100),
			Total:        alpacadecimal.NewFromInt(100),
		}
		for idx := range lines[0].DetailedLines {
			lines[0].DetailedLines[idx].Totals = lines[0].Totals
		}
		invoice.Lines = billing.NewStandardInvoiceLines(lines)

		lineEngine := s.Charges.flatFeeService.GetLineEngine()
		updatedLines, err := lineEngine.OnCollectionCompleted(ctx, billing.OnCollectionCompletedInput{
			Invoice: invoice,
			Lines:   invoice.Lines.OrEmpty(),
		})
		s.NoError(err)
		invoice.Lines = billing.NewStandardInvoiceLines(updatedLines)
		finalizingLine := invoice.Lines.OrEmpty()[0]
		expectedFinalizingLine, err := finalizingLine.Clone()
		s.NoError(err)

		updatedLines, err = lineEngine.OnInvoiceFinalizing(ctx, billing.OnInvoiceFinalizingInput{
			Invoice: invoice,
			Lines:   invoice.Lines.OrEmpty(),
		})
		s.NoError(err)
		s.Equal(expectedFinalizingLine, finalizingLine)
		s.Same(finalizingLine, updatedLines[0])
		invoice.Lines = billing.NewStandardInvoiceLines(updatedLines)
		s.Equal(0, invoiceUsageAccruedCallback.nrInvocations)

		updatedFlatFeeCharge := s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID)
		s.Equal(flatfee.StatusActiveRealizationIssuing, updatedFlatFeeCharge.Status)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.False(updatedFlatFeeCharge.Realizations.CurrentRun.Immutable)
		s.Nil(updatedFlatFeeCharge.Realizations.CurrentRun.AccruedUsage)

		err = lineEngine.OnInvoiceIssued(ctx, billing.OnInvoiceIssuedInput{
			Invoice: invoice,
			Lines:   invoice.Lines.OrEmpty(),
		})
		s.NoError(err)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)
		updatedFlatFeeCharge = s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID)
		s.Equal(flatfee.StatusActiveAwaitingPaymentSettlement, updatedFlatFeeCharge.Status)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.True(updatedFlatFeeCharge.Realizations.CurrentRun.Immutable)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun.AccruedUsage)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun.AccruedUsage.LedgerTransaction)
		s.Equal(invoiceUsageAccruedCallback.id, updatedFlatFeeCharge.Realizations.CurrentRun.AccruedUsage.LedgerTransaction.TransactionGroupID)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       0,
			ChargesTotal: 100,
			Total:        100,
		}, updatedFlatFeeCharge.Realizations.CurrentRun.AccruedUsage.Totals)
	})
}

func (s *InvoicableChargesTestSuite) TestUsageBasedCreditOnlyLifecycle() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-credit-only-lifecycle")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	profile := s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)
	s.True(profile.Default)

	defaultProfile, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
		Namespace: ns,
	})
	s.NoError(err)
	s.NotNil(defaultProfile)
	s.Equal(profile.ID, defaultProfile.ID)

	const (
		usageBasedName = "usage-based"
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	firstCollectionAdvanceAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-01T12:00:00Z", time.UTC).AsTime()
	waitingAdvanceAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:00:00Z", time.UTC).AsTime()
	finalAdvanceAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:01:00Z", time.UTC).AsTime()
	// These are explicit cutoff timestamps rather than computed values so the test asserts the
	// one-minute internal collection period boundary directly.
	finalStoredAtLT := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:00:00Z", time.UTC).AsTime()
	expectedCollectionEnd := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:00:00Z", time.UTC).AsTime()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	usageBasedChargeID := meta.ChargeID{}

	s.Run("#1 create before service period start", func() {
		// Given current wall clock is 2025-12-01T00:00:00Z.
		clock.FreezeTime(createAt)

		// When creating a credit-only usage-based charge for 2026-01-01T00:00:00Z...2026-02-01T00:00:00Z at $1/unit.
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:       cust.GetID(),
					currency:       USD,
					servicePeriod:  servicePeriod,
					settlementMode: productcatalog.CreditOnlySettlementMode,
					price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromFloat(1),
					}),
					name:              usageBasedName,
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: usageBasedName,
					featureKey:        meterSlug,
				}),
			},
		})
		s.NoError(err)

		s.Len(res, 1)
		s.Equal(res[0].Type(), meta.ChargeTypeUsageBased)
		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)

		gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespace:  ns,
			Customers:  []string{cust.ID},
			Currencies: []currencyx.FiatCode{currencyx.FiatCode(currency.USD)},
			Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
		})
		s.NoError(err)
		s.Len(gatheringInvoices.Items, 0)

		fetchedCharge := s.mustGetChargeByID(usageBasedCharge.GetChargeID())
		fetchedUsageBasedCharge, err := fetchedCharge.AsUsageBasedCharge()
		s.NoError(err)

		usageBasedChargeID = usageBasedCharge.GetChargeID()

		// Then the created charge stays in created state, no realization is done, and advancing it is a noop.
		s.Equal(usageBasedCharge.ID, fetchedUsageBasedCharge.ID)
		s.Equal(meta.ChargeStatusCreated, meta.ChargeStatus(fetchedUsageBasedCharge.Status))
		s.Empty(fetchedUsageBasedCharge.Realizations)
		s.Nil(fetchedUsageBasedCharge.State.CurrentRealizationRunID)
		s.Nil(fetchedUsageBasedCharge.State.AdvanceAfter)

		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		s.Nil(advancedCharge)
		s.Equal(meta.ChargeStatusCreated, meta.ChargeStatus(usageBasedFromDB.Status))
		s.Empty(usageBasedFromDB.Realizations)
	})

	s.NotEmpty(usageBasedChargeID)

	s.Run("#2.1 advance into active state", func() {
		// Given the wall clock advances to 2026-01-01T00:00:00Z.
		clock.FreezeTime(servicePeriod.From)

		// When advancing the usage-based charge.
		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		// Then the charge becomes active and no collection is run.
		s.Require().NotNil(advancedCharge)
		s.Equal(usageBasedFromDB.Status, advancedCharge.Status)
		s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(usageBasedFromDB.Status))
		s.Empty(usageBasedFromDB.Realizations)
		s.Nil(usageBasedFromDB.State.CurrentRealizationRunID)
		s.NotNil(usageBasedFromDB.State.AdvanceAfter)
		s.True(servicePeriod.To.Equal(*usageBasedFromDB.State.AdvanceAfter))
	})

	s.Run("#2.2 second advance is noop", func() {
		// Given the charge is already active.
		// When advancing the usage-based charge again.
		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		// Then the advancing does not happen.
		s.Nil(advancedCharge)
		s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(usageBasedFromDB.Status))
		s.Empty(usageBasedFromDB.Realizations)
	})

	s.Run("#3.1 start final realization with stored_at filtering", func() {
		defer s.UsageBasedTestHandler.Reset()

		type callbackInvocation struct {
			Input usagebased.CreditsOnlyUsageAccruedInput
		}

		var startedCallbacks []callbackInvocation

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
			startedCallbacks = append(startedCallbacks, callbackInvocation{Input: input})

			return creditrealization.CreateAllocationInputs{
				{
					ServicePeriod: input.Charge.Intent.GetEffectiveServicePeriod(),
					Amount:        input.AmountToAllocate,
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: ulid.Make().String(),
					},
				},
			}, nil
		}

		// Given the current customer's billing profile makes the collection window end at 2026-02-03T00:00:00Z
		// and the wall clock advances to 2026-02-01T12:00:00Z.
		clock.FreezeTime(firstCollectionAdvanceAt)
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			1,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			2,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T01:00:00Z", time.UTC).AsTime(),
			streamingtestutils.WithStoredAt(datetime.MustParseTimeInLocation(s.T(), "2026-02-01T11:00:00Z", time.UTC).AsTime()),
		)
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			3,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T02:00:00Z", time.UTC).AsTime(),
			streamingtestutils.WithStoredAt(finalStoredAtLT),
		)

		// When advancing the usage-based charge.
		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		// Then a new run is added, only events before the exclusive stored_at cutoff are considered,
		// totals are persisted, and the start callback receives $3.
		s.Require().NotNil(advancedCharge)
		s.Equal(usageBasedFromDB.Status, advancedCharge.Status)
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, usageBasedFromDB.Status)
		s.Len(usageBasedFromDB.Realizations, 1)
		s.NotNil(usageBasedFromDB.State.CurrentRealizationRunID)
		s.NotNil(usageBasedFromDB.State.AdvanceAfter)
		s.True(finalAdvanceAt.Equal(*usageBasedFromDB.State.AdvanceAfter))

		currentRun, err := usageBasedFromDB.Realizations.GetByID(*usageBasedFromDB.State.CurrentRealizationRunID)
		s.NoError(err)
		s.True(finalStoredAtLT.Equal(currentRun.StoredAtLT))
		s.False(currentRun.StoredAtLT.IsZero())
		s.True(expectedCollectionEnd.Equal(currentRun.StoredAtLT.UTC()))
		s.Equal(float64(3), currentRun.MeteredQuantity.InexactFloat64())
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       3,
			CreditsTotal: 3,
		}, currentRun.Totals)
		s.Len(currentRun.CreditsAllocated, 1)
		s.Equal(float64(3), currentRun.CreditsAllocated[0].Amount.InexactFloat64())

		s.Len(startedCallbacks, 1)
		s.Equal(float64(3), startedCallbacks[0].Input.AmountToAllocate.InexactFloat64())
		s.Equal(usagebased.RealizationRunTypeFinalRealization, startedCallbacks[0].Input.Run.Type)
		s.True(servicePeriod.To.Equal(startedCallbacks[0].Input.BookedAt))
	})

	s.Run("#3.2 second realization advance is noop", func() {
		// Given the charge is waiting for collection.
		// When advancing the usage-based charge again.
		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		// Then nothing happens.
		s.Nil(advancedCharge)
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, usageBasedFromDB.Status)
		s.Len(usageBasedFromDB.Realizations, 1)
	})

	s.Run("#4.1 still waiting for the stored_at window", func() {
		// Given time advances to 2026-02-03T00:00:00Z.
		clock.FreezeTime(waitingAdvanceAt)

		// When advancing the usage-based charge.
		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		// Then advancing does nothing because the stored_at cutoff is not ready until 2026-02-03T00:01:00Z.
		s.Nil(advancedCharge)
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, usageBasedFromDB.Status)
		s.Len(usageBasedFromDB.Realizations, 1)
	})

	s.Run("#4.2 finalize realization with incremental credits", func() {
		defer s.UsageBasedTestHandler.Reset()

		type callbackInvocation struct {
			Input usagebased.CreditsOnlyUsageAccruedInput
		}

		var finalizedCallbacks []callbackInvocation

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
			finalizedCallbacks = append(finalizedCallbacks, callbackInvocation{Input: input})

			return creditrealization.CreateAllocationInputs{
				{
					ServicePeriod: input.Charge.Intent.GetEffectiveServicePeriod(),
					Amount:        input.AmountToAllocate,
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: ulid.Make().String(),
					},
				},
			}, nil
		}

		// Given time advances to 2026-02-03T00:01:00Z and new events arrive.
		clock.FreezeTime(finalAdvanceAt)
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			5,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T03:00:00Z", time.UTC).AsTime(),
			streamingtestutils.WithStoredAt(datetime.MustParseTimeInLocation(s.T(), "2026-02-01T23:59:00Z", time.UTC).AsTime()),
		)
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			7,
			servicePeriod.To,
			streamingtestutils.WithStoredAt(datetime.MustParseTimeInLocation(s.T(), "2026-02-02T00:00:00Z", time.UTC).AsTime()),
		)

		// When advancing the usage-based charge.
		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		// Then the new $5 event is included,
		// the finalization callback receives incremental $5, totals are updated to $8,
		// and the charge becomes final.
		s.Require().NotNil(advancedCharge)
		s.Equal(usageBasedFromDB.Status, advancedCharge.Status)
		s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(usageBasedFromDB.Status))
		s.Len(usageBasedFromDB.Realizations, 1)
		s.Nil(usageBasedFromDB.State.CurrentRealizationRunID)
		s.Nil(usageBasedFromDB.State.AdvanceAfter)

		finalRun := usageBasedFromDB.Realizations[0]
		s.True(finalStoredAtLT.Equal(finalRun.StoredAtLT))
		s.False(finalRun.StoredAtLT.IsZero())
		s.True(expectedCollectionEnd.Equal(finalRun.StoredAtLT.UTC()))
		s.Equal(float64(8), finalRun.MeteredQuantity.InexactFloat64())
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       8,
			CreditsTotal: 8,
		}, finalRun.Totals)
		s.Len(finalRun.CreditsAllocated, 2)
		s.Equal(float64(3), finalRun.CreditsAllocated[0].Amount.InexactFloat64())
		s.Equal(float64(5), finalRun.CreditsAllocated[1].Amount.InexactFloat64())

		s.Len(finalizedCallbacks, 1)
		s.Equal(float64(5), finalizedCallbacks[0].Input.AmountToAllocate.InexactFloat64())
		s.Equal(usagebased.RealizationRunTypeFinalRealization, finalizedCallbacks[0].Input.Run.Type)
		s.True(servicePeriod.To.Equal(finalizedCallbacks[0].Input.BookedAt))
	})

	s.Run("#5 final charge advance is noop", func() {
		// Given the charge is already final.
		// When advancing the usage-based charge.
		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		// Then no further allocation occurs.
		s.Nil(advancedCharge)
		s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(usageBasedFromDB.Status))
	})
}

func (s *InvoicableChargesTestSuite) TestUsageBasedCreditOnlyLifecycleVolumeTieredCorrection() {
	defer s.UsageBasedTestHandler.Reset()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-credit-only-lifecycle-volume-tiered-correction")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	profile := s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)
	s.True(profile.Default)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	firstCollectionAdvanceAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-01T12:00:00Z", time.UTC).AsTime()
	finalAdvanceAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:01:00Z", time.UTC).AsTime()
	finalStoredAtLT := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:00:00Z", time.UTC).AsTime()
	expectedCollectionEnd := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:00:00Z", time.UTC).AsTime()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	price := productcatalog.NewPriceFrom(productcatalog.TieredPrice{
		Mode: productcatalog.VolumeTieredPrice,
		Tiers: []productcatalog.PriceTier{
			{
				UpToAmount: lo.ToPtr(alpacadecimal.NewFromInt(10)),
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(2),
				},
			},
			{
				UpToAmount: nil,
				UnitPrice: &productcatalog.PriceTierUnitPrice{
					Amount: alpacadecimal.NewFromInt(1),
				},
			},
		},
	})

	type startedInvocation struct {
		Input usagebased.CreditsOnlyUsageAccruedInput
	}
	type correctedInvocation struct {
		Input usagebased.CreditsOnlyUsageAccruedCorrectionInput
	}

	var usageBasedChargeID meta.ChargeID
	var startedCallbacks []startedInvocation
	var correctedCallbacks []correctedInvocation

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	s.Run("#1 create before service period start", func() {
		clock.FreezeTime(createAt)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          cust.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditOnlySettlementMode,
					price:             price,
					name:              "usage-based-volume-tiered",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "usage-based-volume-tiered",
					featureKey:        meterSlug,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		fetched := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(meta.ChargeStatusCreated, meta.ChargeStatus(fetched.Status))
		s.Equal(usagebased.RatingEngineDelta, fetched.State.RatingEngine)
		s.Empty(fetched.Realizations)
	})

	s.Run("#2 advance into active state", func() {
		clock.FreezeTime(servicePeriod.From)

		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		s.Require().NotNil(advancedCharge)
		s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(usageBasedFromDB.Status))
		s.Empty(usageBasedFromDB.Realizations)
	})

	s.Run("#3 start final realization at quantity 10 and $20", func() {
		defer s.UsageBasedTestHandler.Reset()

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
			startedCallbacks = append(startedCallbacks, startedInvocation{Input: input})

			s.Equal(usageBasedChargeID.ID, input.Charge.ID)
			s.Equal(productcatalog.CreditOnlySettlementMode, input.Charge.Intent.GetSettlementMode())
			s.Equal(usagebased.RealizationRunTypeFinalRealization, input.Run.Type)
			s.True(servicePeriod.To.Equal(input.BookedAt))
			s.Equal(float64(20), input.AmountToAllocate.InexactFloat64())
			s.Equal(float64(10), input.Run.MeteredQuantity.InexactFloat64())
			s.RequireTotals(billingtest.ExpectedTotals{
				Amount: 20,
				Total:  20,
			}, input.Run.Totals)
			s.Empty(input.Run.CreditsAllocated)

			return creditrealization.CreateAllocationInputs{
				{
					ServicePeriod: input.Charge.Intent.GetEffectiveServicePeriod(),
					Amount:        input.AmountToAllocate,
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: ulid.Make().String(),
					},
				},
			}, nil
		}

		clock.FreezeTime(firstCollectionAdvanceAt)

		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			10,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		s.Require().NotNil(advancedCharge)
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, usageBasedFromDB.Status)
		s.Len(usageBasedFromDB.Realizations, 1)
		s.Len(startedCallbacks, 1)
		s.Equal(float64(20), startedCallbacks[0].Input.AmountToAllocate.InexactFloat64())

		currentRun, err := usageBasedFromDB.Realizations.GetByID(*usageBasedFromDB.State.CurrentRealizationRunID)
		s.NoError(err)
		s.True(finalStoredAtLT.Equal(currentRun.StoredAtLT))
		s.True(expectedCollectionEnd.Equal(currentRun.StoredAtLT.UTC()))
		s.Equal(float64(10), currentRun.MeteredQuantity.InexactFloat64())
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       20,
			CreditsTotal: 20,
		}, currentRun.Totals)
		s.Len(currentRun.CreditsAllocated, 1)
		s.Equal(creditrealization.TypeAllocation, currentRun.CreditsAllocated[0].Type)
		s.Equal(float64(20), currentRun.CreditsAllocated[0].Amount.InexactFloat64())

		expandedCharge := s.mustGetUsageBasedChargeByIDWithDetailedLines(usageBasedChargeID)
		expandedRun, err := expandedCharge.Realizations.GetByID(*expandedCharge.State.CurrentRealizationRunID)
		s.NoError(err)
		s.True(expandedRun.DetailedLines.IsPresent())
		s.Len(expandedRun.DetailedLines.OrEmpty(), 1)
		s.Equal("volume-tiered-price", expandedRun.DetailedLines.OrEmpty()[0].ChildUniqueReferenceID)
		s.Equal(float64(10), expandedRun.DetailedLines.OrEmpty()[0].Quantity.InexactFloat64())
		s.Equal(float64(2), expandedRun.DetailedLines.OrEmpty()[0].PerUnitAmount.InexactFloat64())
	})

	s.Run("#4 finalize with persisted negative correction", func() {
		defer s.UsageBasedTestHandler.Reset()

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccruedCorrection = func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedCorrectionInput) (creditrealization.CreateCorrectionInputs, error) {
			correctedCallbacks = append(correctedCallbacks, correctedInvocation{Input: input})

			s.Equal(usageBasedChargeID.ID, input.Charge.ID)
			s.Equal(productcatalog.CreditOnlySettlementMode, input.Charge.Intent.GetSettlementMode())
			s.Equal(usagebased.RealizationRunTypeFinalRealization, input.Run.Type)
			s.True(servicePeriod.To.Equal(input.BookedAt))
			s.Equal(float64(10), input.Run.MeteredQuantity.InexactFloat64())
			s.RequireTotals(billingtest.ExpectedTotals{
				Amount:       20,
				CreditsTotal: 20,
			}, input.Run.Totals)
			s.Len(input.Run.CreditsAllocated, 1)
			s.Equal(creditrealization.TypeAllocation, input.Run.CreditsAllocated[0].Type)
			s.Equal(float64(20), input.Run.CreditsAllocated[0].Amount.InexactFloat64())

			s.Require().Len(input.Corrections, 1)
			s.Equal(input.Run.CreditsAllocated[0].ID, input.Corrections[0].Allocation.ID)
			s.Equal(creditrealization.TypeAllocation, input.Corrections[0].Allocation.Type)
			s.Equal(float64(20), input.Corrections[0].Allocation.Amount.InexactFloat64())
			s.Equal(float64(-9), input.Corrections[0].Amount.InexactFloat64())

			return creditrealization.CreateCorrectionInputs{
				{
					Amount:                input.Corrections[0].Amount,
					CorrectsRealizationID: input.Corrections[0].Allocation.ID,
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: ulid.Make().String(),
					},
				},
			}, nil
		}

		clock.FreezeTime(finalAdvanceAt)

		// Two additional usages happen during collection, but only one is stored before the final cutoff.
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			1,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-20T00:00:00Z", time.UTC).AsTime(),
			streamingtestutils.WithStoredAt(datetime.MustParseTimeInLocation(s.T(), "2026-02-02T00:00:00Z", time.UTC).AsTime()),
		)
		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			1,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-21T00:00:00Z", time.UTC).AsTime(),
			streamingtestutils.WithStoredAt(datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:00:30Z", time.UTC).AsTime()),
		)

		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		s.Require().NotNil(advancedCharge)
		s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(usageBasedFromDB.Status))
		s.Len(correctedCallbacks, 1)
		s.True(servicePeriod.To.Equal(correctedCallbacks[0].Input.BookedAt))
		s.Len(correctedCallbacks[0].Input.Corrections, 1)
		s.Equal(float64(-9), correctedCallbacks[0].Input.Corrections[0].Amount.InexactFloat64())

		s.Len(usageBasedFromDB.Realizations, 1)
		s.Nil(usageBasedFromDB.State.CurrentRealizationRunID)
		s.Nil(usageBasedFromDB.State.AdvanceAfter)

		finalRun := usageBasedFromDB.Realizations[0]
		s.True(finalStoredAtLT.Equal(finalRun.StoredAtLT))
		s.True(expectedCollectionEnd.Equal(finalRun.StoredAtLT.UTC()))
		s.Equal(float64(11), finalRun.MeteredQuantity.InexactFloat64())
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       11,
			CreditsTotal: 11,
		}, finalRun.Totals)
		s.Len(finalRun.CreditsAllocated, 2)

		s.Equal(creditrealization.TypeAllocation, finalRun.CreditsAllocated[0].Type)
		s.Equal(float64(20), finalRun.CreditsAllocated[0].Amount.InexactFloat64())
		s.Equal(creditrealization.TypeCorrection, finalRun.CreditsAllocated[1].Type)
		s.Equal(float64(-9), finalRun.CreditsAllocated[1].Amount.InexactFloat64())
		s.Equal(finalRun.CreditsAllocated[0].ID, lo.FromPtr(finalRun.CreditsAllocated[1].CorrectsRealizationID))

		expandedCharge := s.mustGetUsageBasedChargeByIDWithDetailedLines(usageBasedChargeID)
		s.Len(expandedCharge.Realizations, 1)
		s.True(expandedCharge.Realizations[0].DetailedLines.IsPresent())
		s.Len(expandedCharge.Realizations[0].DetailedLines.OrEmpty(), 1)
		s.Equal("volume-tiered-price", expandedCharge.Realizations[0].DetailedLines.OrEmpty()[0].ChildUniqueReferenceID)
		s.Equal(float64(11), expandedCharge.Realizations[0].DetailedLines.OrEmpty()[0].Quantity.InexactFloat64())
		s.Equal(float64(1), expandedCharge.Realizations[0].DetailedLines.OrEmpty()[0].PerUnitAmount.InexactFloat64())
	})
}

func (s *InvoicableChargesTestSuite) TestUsageBasedCreditThenInvoiceLifecycle() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-credit-then-invoice")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
		stdLineID          billing.LineID
		remainingCredits   *alpacadecimal.Decimal
	)

	s.Run("#1 grant promotional credits", func() {
		promotionalCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = promotionalCallback.Handler(s.T())

		res := s.grantPromotionalCredits(ctx, cust.GetID(), 5)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())
		s.Equal(1, promotionalCallback.nrInvocations)
	})

	s.Run("#2 create future credit-then-invoice usage-based charge", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          cust.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
					price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(0.1)}),
					name:              "usage-based-credit-then-invoice",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "usage-based-credit-then-invoice",
					featureKey:        meterSlug,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		fetched := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(meta.ChargeStatusCreated, meta.ChargeStatus(fetched.Status))
		s.Equal(usagebased.RatingEngineDelta, fetched.State.RatingEngine)
		s.Empty(fetched.Realizations)
		s.Nil(fetched.State.AdvanceAfter)
	})

	s.Run("#4 invoice pending lines at service period end", func() {
		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, remainingCredits = newCappedCreditAllocator(5)

		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			100,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		clock.FreezeTime(servicePeriod.To)

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)

		invoice = invoices[0]
		s.Len(invoice.Lines.OrEmpty(), 1)

		stdLine := invoice.Lines.OrEmpty()[0]
		stdLineID = stdLine.GetLineID()
		s.NotNil(stdLine.UsageBased)
		s.NotNil(stdLine.UsageBased.Quantity)
		s.NotNil(stdLine.UsageBased.MeteredQuantity)
		s.Equal(float64(100), lo.FromPtr(stdLine.UsageBased.Quantity).InexactFloat64())
		s.Equal(float64(100), lo.FromPtr(stdLine.UsageBased.MeteredQuantity).InexactFloat64())
		s.Len(stdLine.CreditsApplied, 1)
		s.Equal(float64(5), stdLine.CreditsApplied[0].Amount.InexactFloat64())
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       10,
			Total:        5,
			CreditsTotal: 5,
		}, stdLine.Totals)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       10,
			Total:        5,
			CreditsTotal: 5,
		}, invoice.Totals)
		s.Equal(usageBasedChargeID.ID, lo.FromPtr(stdLine.ChargeID))

		usageBasedCharge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, usageBasedCharge.Status)
		s.NotNil(usageBasedCharge.State.CurrentRealizationRunID)
		s.Len(usageBasedCharge.Realizations, 1)

		currentRun, err := usageBasedCharge.GetCurrentRealizationRun()
		s.NoError(err)
		s.Equal(float64(100), currentRun.MeteredQuantity.InexactFloat64())
		s.Len(currentRun.CreditsAllocated, 1)
		s.Equal(float64(5), currentRun.CreditsAllocated[0].Amount.InexactFloat64())
		s.True((*remainingCredits).IsZero())

		expandedCharge := s.mustGetUsageBasedChargeByIDWithDetailedLines(usageBasedChargeID)
		expandedRun, err := expandedCharge.GetCurrentRealizationRun()
		s.NoError(err)
		s.True(expandedRun.DetailedLines.IsPresent())
		s.Len(expandedRun.DetailedLines.OrEmpty(), 1)
		s.Equal("unit-price-usage", expandedRun.DetailedLines.OrEmpty()[0].ChildUniqueReferenceID)
		s.Equal(float64(100), expandedRun.DetailedLines.OrEmpty()[0].Quantity.InexactFloat64())
		s.Equal(float64(0.1), expandedRun.DetailedLines.OrEmpty()[0].PerUnitAmount.InexactFloat64())
	})

	s.Run("#5 advance invoice at collection period end", func() {
		*remainingCredits = (*remainingCredits).Add(alpacadecimal.NewFromFloat(3))

		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			25,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-20T00:00:00Z", time.UTC).AsTime(),
			streamingtestutils.WithStoredAt(datetime.MustParseTimeInLocation(s.T(), "2026-02-02T12:00:00Z", time.UTC).AsTime()),
		)
		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Len(invoice.Lines.OrEmpty(), 1)

		stdLine := invoice.Lines.OrEmpty()[0]
		s.Len(stdLine.CreditsApplied, 2)
		s.Equal(float64(5), stdLine.CreditsApplied[0].Amount.InexactFloat64())
		s.Equal(float64(3), stdLine.CreditsApplied[1].Amount.InexactFloat64())
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       12.5,
			Total:        4.5,
			CreditsTotal: 8,
		}, stdLine.Totals)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       12.5,
			Total:        4.5,
			CreditsTotal: 8,
		}, invoice.Totals)

		usageBasedCharge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationProcessing, usageBasedCharge.Status)
		s.NotNil(usageBasedCharge.State.CurrentRealizationRunID)
		s.Len(usageBasedCharge.Realizations, 1)

		currentRun, err := usageBasedCharge.GetCurrentRealizationRun()
		s.NoError(err)
		s.Equal(float64(125), currentRun.MeteredQuantity.InexactFloat64())
		s.True(currentRun.StoredAtLT.Add(usagebased.InternalCollectionPeriod).Equal(invoice.DefaultCollectionAtForStandardInvoice()))
		s.NotNil(currentRun.LineID)
		s.Equal(stdLineID.ID, *currentRun.LineID)
		s.Len(currentRun.CreditsAllocated, 2)
		s.Equal(float64(5), currentRun.CreditsAllocated[0].Amount.InexactFloat64())
		s.Equal(float64(3), currentRun.CreditsAllocated[1].Amount.InexactFloat64())
		s.True((*remainingCredits).IsZero())

		expandedCharge := s.mustGetUsageBasedChargeByIDWithDetailedLines(usageBasedChargeID)
		expandedRun, err := expandedCharge.GetCurrentRealizationRun()
		s.NoError(err)
		s.True(expandedRun.DetailedLines.IsPresent())
		s.Len(expandedRun.DetailedLines.OrEmpty(), 1)
		s.Equal("unit-price-usage", expandedRun.DetailedLines.OrEmpty()[0].ChildUniqueReferenceID)
		s.Equal(float64(125), expandedRun.DetailedLines.OrEmpty()[0].Quantity.InexactFloat64())
		s.Equal(float64(0.1), expandedRun.DetailedLines.OrEmpty()[0].PerUnitAmount.InexactFloat64())
	})

	s.Run("#6 approve invoice and finalize the realization run at issuance", func() {
		defer s.UsageBasedTestHandler.Reset()

		expectedLine := invoice.Lines.OrEmpty()[0]
		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[usagebased.OnInvoiceUsageAccruedInput]()
		s.UsageBasedTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T(), func(t *testing.T, input usagebased.OnInvoiceUsageAccruedInput) {
			s.Equal(usageBasedChargeID.ID, input.Charge.ID)
			s.Equal(expectedLine.Period, input.ServicePeriod)
			s.Equal(float64(4.5), input.Amount.InexactFloat64())
			s.Equal(float64(125), input.Run.MeteredQuantity.InexactFloat64())
			s.NotNil(input.Run.LineID)
			s.Equal(stdLineID.ID, *input.Run.LineID)
		})

		invoice, err := s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)

		usageBasedCharge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, usageBasedCharge.Status)
		s.Nil(usageBasedCharge.State.CurrentRealizationRunID)
		s.Nil(usageBasedCharge.State.AdvanceAfter)
		s.Len(usageBasedCharge.Realizations, 1)

		finalRun := usageBasedCharge.Realizations[0]
		s.True(finalRun.Immutable)
		s.Equal(float64(125), finalRun.MeteredQuantity.InexactFloat64())
		s.NotNil(finalRun.LineID)
		s.Equal(stdLineID.ID, *finalRun.LineID)
		s.NotNil(finalRun.InvoiceUsage)
		s.Equal(invoice.Lines.OrEmpty()[0].Period, finalRun.InvoiceUsage.ServicePeriod)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       12.5,
			Total:        4.5,
			CreditsTotal: 8,
		}, finalRun.InvoiceUsage.Totals)
		s.NotNil(finalRun.InvoiceUsage.LedgerTransaction)
		s.Equal(invoiceUsageAccruedCallback.id, finalRun.InvoiceUsage.LedgerTransaction.TransactionGroupID)
	})

	s.Run("#7 payment authorization keeps charge awaiting settlement", func() {
		defer s.UsageBasedTestHandler.Reset()

		authorizedCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentAuthorizedInput]()
		s.UsageBasedTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T(), func(t *testing.T, input usagebased.OnPaymentAuthorizedInput) {
			assert.Equal(t, usageBasedChargeID.ID, input.Charge.ID)
			assert.NotNil(t, input.Run.InvoiceUsage)
			assert.Nil(t, input.Run.Payment)
			assert.NotNil(t, input.Run.LineID)
			assert.Equal(t, stdLineID.ID, *input.Run.LineID)
		})

		updatedInvoice, err := s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: invoice.GetInvoiceID(),
			Trigger:   billing.TriggerAuthorized,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingAuthorized, updatedInvoice.Status)
		s.Equal(1, authorizedCallback.nrInvocations)

		usageBasedCharge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, usageBasedCharge.Status)
		s.Len(usageBasedCharge.Realizations, 1)

		finalRun := usageBasedCharge.Realizations[0]
		s.NotNil(finalRun.Payment)
		s.NotNil(finalRun.Payment.Authorized)
		s.Nil(finalRun.Payment.Settled)
		s.Equal(authorizedCallback.id, finalRun.Payment.Authorized.TransactionGroupID)
	})

	s.Run("#8 payment settlement finalizes charge", func() {
		defer s.UsageBasedTestHandler.Reset()

		settledCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentSettledInput]()
		s.UsageBasedTestHandler.onPaymentSettled = settledCallback.Handler(s.T(), func(t *testing.T, input usagebased.OnPaymentSettledInput) {
			assert.Equal(t, usageBasedChargeID.ID, input.Charge.ID)
			assert.NotNil(t, input.Run.Payment)
			assert.NotNil(t, input.Run.Payment.Authorized)
			assert.Nil(t, input.Run.Payment.Settled)
			assert.Equal(t, payment.StatusAuthorized, input.Run.Payment.Status)
		})

		updatedInvoice, err := s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: invoice.GetInvoiceID(),
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, updatedInvoice.Status)
		s.Equal(1, settledCallback.nrInvocations)

		usageBasedCharge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusFinal, usageBasedCharge.Status)
		s.Len(usageBasedCharge.Realizations, 1)

		finalRun := usageBasedCharge.Realizations[0]
		s.NotNil(finalRun.Payment)
		s.NotNil(finalRun.Payment.Settled)
		s.Equal(settledCallback.id, finalRun.Payment.Settled.TransactionGroupID)
		s.Equal(payment.StatusSettled, finalRun.Payment.Status)
	})
}

func (s *InvoicableChargesTestSuite) TestUsageBasedCreditThenInvoiceFullyCreditedDoesNotAccrueInvoiceUsage() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-credit-then-invoice-fully-credited")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
		stdLineID          billing.LineID
	)

	s.Run("#1 grant promotional credits", func() {
		promotionalCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
		s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = promotionalCallback.Handler(s.T())

		res := s.grantPromotionalCredits(ctx, cust.GetID(), 20)
		s.Equal(meta.ChargeTypeCreditPurchase, res[0].Type())
		s.Equal(1, promotionalCallback.nrInvocations)
	})

	s.Run("#2 create future credit-then-invoice usage-based charge", func() {
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          cust.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
					price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromFloat(0.1)}),
					name:              "usage-based-credit-then-invoice-fully-credited",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "usage-based-credit-then-invoice-fully-credited",
					featureKey:        meterSlug,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		fetched := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(meta.ChargeStatusCreated, meta.ChargeStatus(fetched.Status))
		s.Empty(fetched.Realizations)
		s.Nil(fetched.State.AdvanceAfter)
	})

	s.Run("#3 invoice pending lines fully settled by credits", func() {
		defer s.UsageBasedTestHandler.Reset()

		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(20)

		s.MockStreamingConnector.AddSimpleEvent(
			meterSlug,
			100,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		clock.FreezeTime(servicePeriod.To)

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)

		invoice = invoices[0]
		s.Len(invoice.Lines.OrEmpty(), 1)

		stdLine := invoice.Lines.OrEmpty()[0]
		stdLineID = stdLine.GetLineID()
		s.NotNil(stdLine.UsageBased)
		s.NotNil(stdLine.UsageBased.Quantity)
		s.Equal(float64(100), lo.FromPtr(stdLine.UsageBased.Quantity).InexactFloat64())
		s.Len(stdLine.CreditsApplied, 1)
		s.Equal(float64(10), stdLine.CreditsApplied[0].Amount.InexactFloat64())
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       10,
			CreditsTotal: 10,
		}, stdLine.Totals)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       10,
			CreditsTotal: 10,
		}, invoice.Totals)
		s.Equal(usageBasedChargeID.ID, lo.FromPtr(stdLine.ChargeID))

		usageBasedCharge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationWaitingForCollection, usageBasedCharge.Status)
		s.NotNil(usageBasedCharge.State.CurrentRealizationRunID)
		s.Len(usageBasedCharge.Realizations, 1)

		currentRun, err := usageBasedCharge.GetCurrentRealizationRun()
		s.NoError(err)
		s.Equal(float64(100), currentRun.MeteredQuantity.InexactFloat64())
		s.Len(currentRun.CreditsAllocated, 1)
		s.Equal(float64(10), currentRun.CreditsAllocated[0].Amount.InexactFloat64())
	})

	s.Run("#4 advance invoice at collection period end", func() {
		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())

		var err error
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Len(invoice.Lines.OrEmpty(), 1)

		stdLine := invoice.Lines.OrEmpty()[0]
		s.Len(stdLine.CreditsApplied, 1)
		s.Equal(float64(10), stdLine.CreditsApplied[0].Amount.InexactFloat64())
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       10,
			CreditsTotal: 10,
		}, stdLine.Totals)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       10,
			CreditsTotal: 10,
		}, invoice.Totals)

		usageBasedCharge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationProcessing, usageBasedCharge.Status)
		s.NotNil(usageBasedCharge.State.CurrentRealizationRunID)
		s.Len(usageBasedCharge.Realizations, 1)

		currentRun, err := usageBasedCharge.GetCurrentRealizationRun()
		s.NoError(err)
		s.Equal(float64(100), currentRun.MeteredQuantity.InexactFloat64())
		s.True(currentRun.StoredAtLT.Add(usagebased.InternalCollectionPeriod).Equal(invoice.DefaultCollectionAtForStandardInvoice()))
		s.NotNil(currentRun.LineID)
		s.Equal(stdLineID.ID, *currentRun.LineID)
		s.Len(currentRun.CreditsAllocated, 1)
		s.Equal(float64(10), currentRun.CreditsAllocated[0].Amount.InexactFloat64())
	})

	s.Run("#5 finalize and issue invoice with no fiat invoice usage accrual", func() {
		defer s.UsageBasedTestHandler.Reset()

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[usagebased.OnInvoiceUsageAccruedInput]()
		s.UsageBasedTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		lineEngine := s.Charges.usageBasedService.GetLineEngine()
		finalizingLine := invoice.Lines.OrEmpty()[0]
		expectedFinalizingLine, err := finalizingLine.Clone()
		s.NoError(err)

		finalizedLines, err := lineEngine.OnInvoiceFinalizing(ctx, billing.OnInvoiceFinalizingInput{
			Invoice: invoice,
			Lines:   invoice.Lines.OrEmpty(),
		})
		s.NoError(err)
		s.Equal(expectedFinalizingLine, finalizingLine)
		s.Same(finalizingLine, finalizedLines[0])
		invoice.Lines = billing.NewStandardInvoiceLines(finalizedLines)
		s.Equal(0, invoiceUsageAccruedCallback.nrInvocations)

		usageBasedCharge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationIssuing, usageBasedCharge.Status)
		s.NotNil(usageBasedCharge.State.CurrentRealizationRunID)
		s.Nil(usageBasedCharge.State.AdvanceAfter)
		s.Len(usageBasedCharge.Realizations, 1)

		preparedRun := usageBasedCharge.Realizations[0]
		s.False(preparedRun.Immutable)
		s.Equal(float64(100), preparedRun.MeteredQuantity.InexactFloat64())
		s.NotNil(preparedRun.LineID)
		s.Equal(stdLineID.ID, *preparedRun.LineID)
		s.True(preparedRun.NoFiatTransactionRequired)
		s.Nil(preparedRun.InvoiceUsage)

		mismatchedLine, err := finalizingLine.Clone()
		s.NoError(err)
		mismatchedLine.ID = "mismatched-issued-line"
		err = lineEngine.OnInvoiceIssued(ctx, billing.OnInvoiceIssuedInput{
			Invoice: invoice,
			Lines:   billing.StandardLines{mismatchedLine},
		})
		s.ErrorContains(err, "does not match issued line")
		usageBasedCharge = s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveRealizationIssuing, usageBasedCharge.Status)
		s.Nil(usageBasedCharge.Realizations[0].InvoiceUsage)

		err = lineEngine.OnInvoiceIssued(ctx, billing.OnInvoiceIssuedInput{
			Invoice: invoice,
			Lines:   invoice.Lines.OrEmpty(),
		})
		s.NoError(err)
		s.Equal(0, invoiceUsageAccruedCallback.nrInvocations)

		usageBasedCharge = s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusFinal, usageBasedCharge.Status)
		s.Nil(usageBasedCharge.State.CurrentRealizationRunID)
		s.Nil(usageBasedCharge.State.AdvanceAfter)

		finalRun := usageBasedCharge.Realizations[0]
		s.True(finalRun.Immutable)
		s.Equal(float64(100), finalRun.MeteredQuantity.InexactFloat64())
		s.NotNil(finalRun.LineID)
		s.Equal(stdLineID.ID, *finalRun.LineID)
		s.True(finalRun.NoFiatTransactionRequired)
		s.NotNil(finalRun.InvoiceUsage)
		s.Nil(finalRun.InvoiceUsage.LedgerTransaction)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       10,
			CreditsTotal: 10,
		}, finalRun.InvoiceUsage.Totals)
	})
}

func (s *InvoicableChargesTestSuite) TestUsageBasedCreateImmediatelyActive() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-create-immediately-active")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	// Given clock is frozen at the service period start.
	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	// When creating a credit-only usage-based charge at service period start.
	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(1),
				}),
				name:              "usage-based",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "usage-based",
				featureKey:        meterSlug,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	// Then the returned charge is already active.
	s.Equal(meta.ChargeTypeUsageBased, res[0].Type())
	returnedCharge, err := res[0].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(returnedCharge.Status))
	s.NotNil(returnedCharge.State.AdvanceAfter)
	s.True(servicePeriod.To.Equal(*returnedCharge.State.AdvanceAfter))
	s.Empty(returnedCharge.Realizations)
	s.Nil(returnedCharge.State.CurrentRealizationRunID)

	// And the DB state matches the returned charge.
	dbCharge := s.mustGetUsageBasedChargeByID(returnedCharge.GetChargeID())
	s.Equal(returnedCharge.Status, dbCharge.Status)
	s.Equal(meta.ChargeStatusActive, meta.ChargeStatus(dbCharge.Status))
	s.NotNil(dbCharge.State.AdvanceAfter)
	s.True(servicePeriod.To.Equal(*dbCharge.State.AdvanceAfter))
	s.Empty(dbCharge.Realizations)
	s.Nil(dbCharge.State.CurrentRealizationRunID)
}

func (s *InvoicableChargesTestSuite) TestUsageBasedCreditThenInvoiceDirectPaidFlow() {
	// Given
	// - a credit-then-invoice usage-based charge with metered usage in the service period,
	// When
	// - the invoice is issued and the payment app emits a direct paid trigger,
	// Then
	// - billing should run the usage-based payment authorization and settlement hooks in order
	//   and persist the finalized payment state on the realization run.

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-credit-then-invoice-direct-paid")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	promotionalCallback := newCountedLedgerTransactionCallback[creditpurchase.Charge]()
	s.CreditPurchaseTestHandler.onPromotionalCreditPurchase = promotionalCallback.Handler(s.T())
	s.grantPromotionalCredits(ctx, cust.GetID(), 5)

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(0.1),
				}),
				name:              "usage-based-direct-paid",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "usage-based-direct-paid",
				featureKey:        apiRequestsTotal.Feature.Key,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	usageBasedChargeID, err := res[0].GetChargeID()
	s.NoError(err)

	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued, _ = newCappedCreditAllocator(5)

	s.MockStreamingConnector.AddSimpleEvent(
		apiRequestsTotal.Feature.Key,
		100,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)

	clock.FreezeTime(servicePeriod.To.Add(time.Second))

	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(servicePeriod.To),
	})
	s.NoError(err)
	s.Len(invoices, 1)

	invoice := invoices[0]
	s.Len(invoice.Lines.OrEmpty(), 1)
	stdLine := invoice.Lines.OrEmpty()[0]
	stdLineID := stdLine.GetLineID()

	s.MockStreamingConnector.AddSimpleEvent(
		apiRequestsTotal.Feature.Key,
		25,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-20T00:00:00Z", time.UTC).AsTime(),
		streamingtestutils.WithStoredAt(datetime.MustParseTimeInLocation(s.T(), "2026-02-02T12:00:00Z", time.UTC).AsTime()),
	)

	clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
	invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
	s.NoError(err)

	defer s.UsageBasedTestHandler.Reset()

	invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[usagebased.OnInvoiceUsageAccruedInput]()
	s.UsageBasedTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

	invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
	s.NoError(err)
	s.Equalf(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status, "validation issues: %v", invoice.ValidationIssues.AsError())
	s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)

	authorizedCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentAuthorizedInput]()
	s.UsageBasedTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T(), func(t *testing.T, input usagebased.OnPaymentAuthorizedInput) {
		assert.Equal(t, usageBasedChargeID.ID, input.Charge.ID)
		assert.NotNil(t, input.Run.InvoiceUsage)
		assert.Nil(t, input.Run.Payment)
		assert.NotNil(t, input.Run.LineID)
		assert.Equal(t, stdLineID.ID, *input.Run.LineID)
	})

	settledCallback := newCountedLedgerTransactionCallback[usagebased.OnPaymentSettledInput]()
	s.UsageBasedTestHandler.onPaymentSettled = settledCallback.Handler(s.T(), func(t *testing.T, input usagebased.OnPaymentSettledInput) {
		assert.Equal(t, usageBasedChargeID.ID, input.Charge.ID)
		assert.NotNil(t, input.Run.Payment)
		assert.NotNil(t, input.Run.Payment.Authorized)
		assert.Equal(t, authorizedCallback.id, input.Run.Payment.Authorized.TransactionGroupID)
		assert.Nil(t, input.Run.Payment.Settled)
		assert.Equal(t, payment.StatusAuthorized, input.Run.Payment.Status)
	})

	invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
		InvoiceID: invoice.GetInvoiceID(),
		Trigger:   billing.TriggerPaid,
	})
	s.NoError(err)
	s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)
	s.Equal(1, authorizedCallback.nrInvocations)
	s.Equal(1, settledCallback.nrInvocations)

	usageBasedCharge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
	s.Equal(usagebased.StatusFinal, usageBasedCharge.Status)
	s.Len(usageBasedCharge.Realizations, 1)

	finalRun := usageBasedCharge.Realizations[0]
	s.NotNil(finalRun.Payment)
	s.NotNil(finalRun.Payment.Authorized)
	s.NotNil(finalRun.Payment.Settled)
	s.Equal(authorizedCallback.id, finalRun.Payment.Authorized.TransactionGroupID)
	s.Equal(settledCallback.id, finalRun.Payment.Settled.TransactionGroupID)
	s.Equal(payment.StatusSettled, finalRun.Payment.Status)
}

func (s *InvoicableChargesTestSuite) TestUsageBasedCreateImmediatelyFinal() {
	defer s.UsageBasedTestHandler.Reset()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-create-immediately-final")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	// collectionEnd = servicePeriod.To + P2D = 2026-02-03T00:00:00Z
	// finalAdvanceAt = collectionEnd + InternalCollectionPeriod (1 minute) = 2026-02-03T00:01:00Z
	// storedAtLT = clock.Now() - InternalCollectionPeriod = finalAdvanceAt - 1min = collectionEnd
	finalAdvanceAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:01:00Z", time.UTC).AsTime()
	expectedCollectionEnd := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:00:00Z", time.UTC).AsTime()
	expectedStoredAtLT := finalAdvanceAt.Add(-usagebased.InternalCollectionPeriod) // == expectedCollectionEnd

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	meterSlug := apiRequestsTotal.Feature.Key

	// Two events inside the service period; default StoredAt == event time so both are well below
	// storedAtLT (2026-02-03T00:00:00Z) and will be included in the rating.
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 3,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
	)
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 5,
		datetime.MustParseTimeInLocation(s.T(), "2026-01-20T00:00:00Z", time.UTC).AsTime(),
	)

	const expectedUsage = float64(8) // 3 + 5

	// OnCollectionStarted is called during StartFinalRealizationRun because usage > 0.
	// OnCollectionFinalized is not called because the finalize rating is identical to the start
	// rating (frozen clock) so additionalAmount == 0.
	s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = func(ctx context.Context, input usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateAllocationInputs, error) {
		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod: input.Charge.Intent.GetEffectiveServicePeriod(),
				Amount:        input.AmountToAllocate,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			},
		}, nil
	}

	// Given clock is frozen past the collection period end.
	clock.FreezeTime(finalAdvanceAt)
	defer clock.UnFreeze()

	// When creating a credit-only usage-based charge well after the service period.
	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(1),
				}),
				name:              "usage-based",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "usage-based",
				featureKey:        meterSlug,
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	// Then the returned charge is already final.
	s.Equal(meta.ChargeTypeUsageBased, res[0].Type())
	returnedCharge, err := res[0].AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(returnedCharge.Status))
	s.Nil(returnedCharge.State.AdvanceAfter)
	s.Nil(returnedCharge.State.CurrentRealizationRunID)
	s.Len(returnedCharge.Realizations, 1)

	finalRun := returnedCharge.Realizations[0]
	s.True(expectedStoredAtLT.Equal(finalRun.StoredAtLT))
	s.False(finalRun.StoredAtLT.IsZero())
	s.True(expectedCollectionEnd.Equal(finalRun.StoredAtLT.UTC()))
	s.Equal(expectedUsage, finalRun.MeteredQuantity.InexactFloat64())
	s.RequireTotals(billingtest.ExpectedTotals{
		Amount:       expectedUsage,
		CreditsTotal: expectedUsage,
	}, finalRun.Totals)
	s.Len(finalRun.CreditsAllocated, 1)
	s.Equal(expectedUsage, finalRun.CreditsAllocated[0].Amount.InexactFloat64())

	// And the DB state matches the returned charge.
	dbCharge := s.mustGetUsageBasedChargeByID(returnedCharge.GetChargeID())
	s.Equal(meta.ChargeStatusFinal, meta.ChargeStatus(dbCharge.Status))
	s.Nil(dbCharge.State.AdvanceAfter)
	s.Nil(dbCharge.State.CurrentRealizationRunID)
	s.Len(dbCharge.Realizations, 1)
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditOnlyLifecycle() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-only-lifecycle")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	const flatFeeName = "flat-fee-credit-only"

	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	// InAdvance payment term means InvoiceAt = ServicePeriod.From
	invoiceAt := servicePeriod.From

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	flatFeeChargeID := meta.ChargeID{}

	s.Run("#1 create before invoice_at", func() {
		// Given current wall clock is 2025-12-01T00:00:00Z (before InvoiceAt).
		clock.FreezeTime(createAt)

		// When creating a credit-only flat fee charge.
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: []charges.ChargeIntent{
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
					percentageDiscounts: &billing.PercentageDiscount{
						PercentageDiscount: productcatalog.PercentageDiscount{
							Percentage: models.NewPercentage(10),
						},
						CorrelationID: "flat-fee-credit-only-discount",
					},
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)
		s.Equal(meta.ChargeTypeFlatFee, res[0].Type())

		flatFeeCharge, err := res[0].AsFlatFeeCharge()
		s.NoError(err)

		// Then no gathering invoice is created (credit-only skips invoicing).
		gatheringInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
			Namespace:  ns,
			Customers:  []string{cust.ID},
			Currencies: []currencyx.FiatCode{currencyx.FiatCode(currency.USD)},
			Expand:     []billing.GatheringInvoiceExpand{billing.GatheringInvoiceExpandLines},
		})
		s.NoError(err)
		s.Len(gatheringInvoices.Items, 0)

		// The charge starts in Created status (not Active).
		fetchedCharge := s.mustGetChargeByID(flatFeeCharge.GetChargeID())
		fetchedFF, err := fetchedCharge.AsFlatFeeCharge()
		s.NoError(err)

		flatFeeChargeID = flatFeeCharge.GetChargeID()

		s.Equal(flatfee.StatusCreated, fetchedFF.Status)
		s.Nil(fetchedFF.Realizations.CurrentRun)
		s.NotNil(fetchedFF.State.AdvanceAfter)
		s.True(servicePeriod.From.Equal(*fetchedFF.State.AdvanceAfter))

		// Advancing is a noop (clock is before InvoiceAt).
		advancedCharges := s.mustAdvanceFlatFeeCharges(ctx, cust.GetID())
		s.Empty(advancedCharges)

		// Status unchanged after advance attempt.
		fetchedCharge = s.mustGetChargeByID(flatFeeChargeID)
		fetchedFF, err = fetchedCharge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatfee.StatusCreated, fetchedFF.Status)
	})

	s.NotEmpty(flatFeeChargeID)

	s.Run("#2 advance at invoice_at persists detailed-line amount discounts", func() {
		defer s.FlatFeeTestHandler.Reset()

		type callbackInvocation struct {
			Input flatfee.OnAllocateCreditsInput
		}

		var callbacks []callbackInvocation

		s.FlatFeeTestHandler.onAllocateCredits = func(ctx context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			callbacks = append(callbacks, callbackInvocation{Input: input})

			return creditrealization.CreateAllocationInputs{
				{
					ServicePeriod: input.Charge.Intent.GetEffectiveServicePeriod(),
					Amount:        input.PreTaxAmountToAllocate,
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: ulid.Make().String(),
					},
				},
			}, nil
		}

		// Given the wall clock advances to InvoiceAt (2026-01-01T00:00:00Z).
		clock.FreezeTime(invoiceAt)

		// When advancing the flat fee charge.
		advancedCharges := s.mustAdvanceFlatFeeCharges(ctx, cust.GetID())

		// Then the charge transitions Created → Active → Final in one advance call.
		s.Len(advancedCharges, 1)
		advancedFF, err := advancedCharges[0].AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatfee.StatusFinal, advancedFF.Status)

		// Verify DB state matches.
		fetchedCharge := s.mustGetChargeByID(flatFeeChargeID)
		fetchedFF, err := fetchedCharge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatfee.StatusFinal, fetchedFF.Status)
		s.Nil(fetchedFF.State.AdvanceAfter)

		// The handler was called exactly once with the correct amount.
		s.Len(callbacks, 1)
		s.Equal(float64(90), callbacks[0].Input.PreTaxAmountToAllocate.InexactFloat64())

		// Credit realizations and gross rated details were persisted separately.
		fetchedFF = s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID)
		s.Require().NotNil(fetchedFF.Realizations.CurrentRun)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:         100,
			DiscountsTotal: 10,
			CreditsTotal:   90,
		}, fetchedFF.Realizations.CurrentRun.Totals)
		s.Len(fetchedFF.Realizations.CurrentRun.CreditRealizations, 1)
		s.Equal(float64(90), fetchedFF.Realizations.CurrentRun.CreditRealizations[0].Amount.InexactFloat64())
		s.True(fetchedFF.Realizations.CurrentRun.DetailedLines.IsPresent())
		s.Require().Len(fetchedFF.Realizations.CurrentRun.DetailedLines.OrEmpty(), 1)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:         100,
			DiscountsTotal: 10,
			Total:          90,
		}, fetchedFF.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].Totals)
		detailedLine := fetchedFF.Realizations.CurrentRun.DetailedLines.OrEmpty()[0]
		s.Empty(detailedLine.CreditsApplied)

		amountDiscounts := detailedLine.AmountDiscounts
		s.Require().Len(amountDiscounts, 1)
		amountDiscount := amountDiscounts[0]
		s.Equal("rateCardDiscount/correlationID=flat-fee-credit-only-discount", amountDiscount.ChildUniqueReferenceID)
		s.Nil(amountDiscount.Description)
		s.Require().Equal(float64(10), amountDiscount.Amount.InexactFloat64())
		s.True(amountDiscount.RoundingAmount.IsZero())
		s.Equal(billing.RatecardPercentageDiscountReason, amountDiscount.Reason.Type())
		percentageDiscount, err := amountDiscount.Reason.AsRatecardPercentage()
		s.Require().NoError(err)
		s.Equal("flat-fee-credit-only-discount", percentageDiscount.CorrelationID)
		s.Require().Equal(float64(10), percentageDiscount.Percentage.InexactFloat64())
	})

	s.Run("#3 final charge advance is noop", func() {
		// Given the charge is already final.
		// When advancing the flat fee charge.
		advancedCharges := s.mustAdvanceFlatFeeCharges(ctx, cust.GetID())

		// Then no further allocation occurs.
		s.Empty(advancedCharges)

		fetchedCharge := s.mustGetChargeByID(flatFeeChargeID)
		fetchedFF, err := fetchedCharge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(flatfee.StatusFinal, fetchedFF.Status)
	})
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditOnlyCreateImmediatelyFinal() {
	defer s.FlatFeeTestHandler.Reset()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-only-create-immediately-final")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	s.FlatFeeTestHandler.onAllocateCredits = func(ctx context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod: input.Charge.Intent.GetEffectiveServicePeriod(),
				Amount:        input.PreTaxAmountToAllocate,
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: ulid.Make().String(),
				},
			},
		}, nil
	}

	// Given clock is frozen at the service period start (== InvoiceAt for InAdvance).
	clock.FreezeTime(servicePeriod.From)
	defer clock.UnFreeze()

	// When creating a credit-only flat fee charge at InvoiceAt.
	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(50),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
				name:              "flat-fee-immediate",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "flat-fee-immediate",
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	// Then the returned charge is already final (auto-advanced on create).
	s.Equal(meta.ChargeTypeFlatFee, res[0].Type())
	returnedCharge, err := res[0].AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatfee.StatusFinal, returnedCharge.Status)
	s.Nil(returnedCharge.State.AdvanceAfter)
	s.Require().NotNil(returnedCharge.Realizations.CurrentRun)
	s.Len(returnedCharge.Realizations.CurrentRun.CreditRealizations, 1)
	s.Equal(float64(50), returnedCharge.Realizations.CurrentRun.CreditRealizations[0].Amount.InexactFloat64())

	// And the DB state matches.
	dbCharge := s.mustGetChargeByID(returnedCharge.GetChargeID())
	dbFF, err := dbCharge.AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatfee.StatusFinal, dbFF.Status)
	s.Nil(dbFF.State.AdvanceAfter)
	s.Require().NotNil(dbFF.Realizations.CurrentRun)
	s.Len(dbFF.Realizations.CurrentRun.CreditRealizations, 1)
	s.Equal(float64(50), dbFF.Realizations.CurrentRun.CreditRealizations[0].Amount.InexactFloat64())
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditOnlyWithCustomCurrency() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-only-custom-currency")

	var customCurrency currencies.Currency
	var customerID string
	var createdCharge flatfee.Charge
	var allocationTransactionGroupID string

	s.Run("#1 setup customer and custom currency", func() {
		// given:
		// - a customer and a persisted custom currency
		s.ProvisionDefaultTaxCodes(ctx, ns)

		cust := s.CreateTestCustomer(ns, "test-subject")
		s.NotEmpty(cust.ID)
		customerID = cust.ID
		customCurrency = s.createTestCustomCurrency(ctx, ns)
	})

	s.Run("#2 reject credit-only custom currency through the customer charge API", func() {
		// given:
		// - the same customer and custom currency used by the supported root charge flow below
		servicePeriod := timeutil.ClosedPeriod{
			From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
			To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
		}

		fiatCurrency, err := currencyx.NewFiatCurrency(USD)
		s.Require().NoError(err)
		costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
			FiatCurrency: fiatCurrency,
			Rate:         alpacadecimal.NewFromFloat(0.5),
		})

		// when:
		// - a manual customer charge tries to use that custom currency with credit-only settlement
		_, err = s.Charges.CreateCustomerCharge(ctx, charges.CreateCustomerChargeInput{
			Namespace:    ns,
			CustomerID:   customerID,
			CurrencyCode: customCurrency.GetCode(),
			CostBasis:    &costBasisIntent,
			FlatFee: &charges.CreateCustomerChargeFlatFeeInput{
				IntentMutableFields: flatfee.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "Unsupported custom currency charge",
						ServicePeriod:     servicePeriod,
						FullServicePeriod: servicePeriod,
						BillingPeriod:     servicePeriod,
					},
					InvoiceAt:             servicePeriod.From,
					PaymentTerm:           productcatalog.InAdvancePaymentTerm,
					AmountBeforeProration: alpacadecimal.NewFromInt(10),
				},
				SettlementMode: productcatalog.CreditOnlySettlementMode,
			},
		})

		// then:
		// - the API boundary rejects the settlement mode before the supported root creation path is reached
		s.Require().Error(err)
		s.ErrorIs(err, meta.ErrCustomCurrencyNotSupported)
		s.True(models.IsGenericValidationError(err))

		persisted, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
			Namespace:   ns,
			CustomerIDs: []string{customerID},
		})
		s.Require().NoError(err)
		s.Empty(persisted.Items)
	})

	s.Run("#3 create credits-only flat fee through the root service", func() {
		// given:
		// - an immediately due flat fee in the custom currency
		// - mocked ledger allocation and lineage callbacks
		// when:
		// - the credits-only charge is created through the root charges service
		// then:
		// - the callbacks run once and the charge reaches final with a persisted allocation
		servicePeriod := timeutil.ClosedPeriod{
			From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
			To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
		}
		clock.FreezeTime(servicePeriod.From)
		defer clock.UnFreeze()

		allocationCallback := newCountedCreditAllocationCallback[flatfee.OnAllocateCreditsInput]()
		allocationTransactionGroupID = allocationCallback.id
		s.FlatFeeTestHandler.onAllocateCredits = allocationCallback.Handler(
			s.T(),
			func(input flatfee.OnAllocateCreditsInput, transactionGroup ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs {
				return creditrealization.CreateAllocationInputs{
					{
						ServicePeriod:     input.ServicePeriod,
						Amount:            input.PreTaxAmountToAllocate,
						LedgerTransaction: transactionGroup,
					},
				}
			},
			func(t *testing.T, input flatfee.OnAllocateCreditsInput) {
				assert.True(t, input.Charge.Intent.GetCurrency().IsCustom())
				assert.Equal(t, customCurrency.ID, input.Charge.Intent.GetCurrency().ID)
			},
		)

		lineageMock := &mockLineageService{Service: s.LineageService}
		lineageMock.On("CreateInitialLineages", mock.Anything, mock.Anything).
			Return(nil).
			Once()
		lineageMock.On("PersistCorrectionLineageSegments", mock.Anything, mock.Anything).
			Return(nil).
			Once()

		customCurrencyFlatFeeService, err := flatfeeservice.New(flatfeeservice.Config{
			Adapter:       s.FlatFeeAdapter,
			Handler:       s.FlatFeeTestHandler,
			Lineage:       lineageMock,
			MetaAdapter:   s.MetaAdapter,
			Locker:        s.Locker,
			RatingService: billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: s.UnitConfigEnabled}),
			Currencies:    s.CurrencyService,
		})
		s.Require().NoError(err)

		originalFlatFeeService := s.Charges.flatFeeService
		s.Charges.flatFeeService = customCurrencyFlatFeeService
		defer func() {
			s.Charges.flatFeeService = originalFlatFeeService
		}()

		intent := charges.NewChargeIntent(flatfee.Intent{
			Intent: meta.Intent{
				ManagedBy:  billing.ManuallyManagedLine,
				CustomerID: customerID,
				Currency:   customCurrency,
			},
			IntentMutableFields: flatfee.IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              "Custom Currency Flat Fee",
					ServicePeriod:     servicePeriod,
					BillingPeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
				},
				InvoiceAt:             servicePeriod.From,
				PaymentTerm:           productcatalog.InAdvancePaymentTerm,
				AmountBeforeProration: alpacadecimal.NewFromFloat(50.1234),
			},
			SettlementMode: productcatalog.CreditOnlySettlementMode,
		})

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents:   charges.ChargeIntents{intent},
		})
		s.Require().NoError(err)
		s.Require().Len(created, 1)
		s.Equal(1, allocationCallback.nrInvocations)
		lineageMock.AssertExpectations(s.T())

		createdCharge, err = created[0].AsFlatFeeCharge()
		s.Require().NoError(err)
		s.Equal(flatfee.StatusFinal, createdCharge.Status)
		s.True(createdCharge.Intent.GetCurrency().IsCustom())
		s.Equal(customCurrency.ID, createdCharge.Intent.GetCurrency().ID)
		s.Require().NotNil(createdCharge.Realizations.CurrentRun)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       50.123,
			CreditsTotal: 50.123,
		}, createdCharge.Realizations.CurrentRun.Totals)
		s.Require().Len(createdCharge.Realizations.CurrentRun.CreditRealizations, 1)
		s.Equal(float64(50.123), createdCharge.Realizations.CurrentRun.CreditRealizations[0].Amount.InexactFloat64())
		s.Equal(allocationTransactionGroupID, createdCharge.Realizations.CurrentRun.CreditRealizations[0].LedgerTransaction.TransactionGroupID)
		s.True(createdCharge.Realizations.CurrentRun.DetailedLines.IsPresent())
		s.Require().Len(createdCharge.Realizations.CurrentRun.DetailedLines.OrEmpty(), 1)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount: 50.123,
			Total:  50.123,
		}, createdCharge.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].Totals)
		s.Empty(createdCharge.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].CreditsApplied)
	})

	s.Run("#4 reload persisted charge", func() {
		// when:
		// - the flat-fee charge is loaded again from Postgres
		// then:
		// - its final state, custom currency, totals, and allocation are preserved
		persisted := s.mustGetFlatFeeChargeByIDWithDetailedLines(createdCharge.GetChargeID())
		s.Equal(flatfee.StatusFinal, persisted.Status)
		s.True(persisted.Intent.GetCurrency().IsCustom())
		s.Equal(customCurrency.ID, persisted.Intent.GetCurrency().ID)
		s.Require().NotNil(persisted.Realizations.CurrentRun)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       50.123,
			CreditsTotal: 50.123,
		}, persisted.Realizations.CurrentRun.Totals)
		s.Require().Len(persisted.Realizations.CurrentRun.CreditRealizations, 1)
		s.Equal(allocationTransactionGroupID, persisted.Realizations.CurrentRun.CreditRealizations[0].LedgerTransaction.TransactionGroupID)
		s.True(persisted.Realizations.CurrentRun.DetailedLines.IsPresent())
		s.Require().Len(persisted.Realizations.CurrentRun.DetailedLines.OrEmpty(), 1)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount: 50.123,
			Total:  50.123,
		}, persisted.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].Totals)
		s.Empty(persisted.Realizations.CurrentRun.DetailedLines.OrEmpty()[0].CreditsApplied)
	})
}

func (s *InvoicableChargesTestSuite) TestUsageBasedCreditOnlyWithCustomCurrency() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-usage-based-credit-only-custom-currency")

	var customCurrency currencies.Currency
	var customerID string
	var featureKey string
	var createdCharge usagebased.Charge
	var allocationTransactionGroupID string

	s.Run("#1 setup metered customer and custom currency", func() {
		// given:
		// - a metered customer, a persisted custom currency, and usage events
		s.ProvisionDefaultTaxCodes(ctx, ns)

		customInvoicing := s.SetupCustomInvoicing(ns)
		cust := s.CreateTestCustomer(ns, "test-subject")
		s.NotEmpty(cust.ID)
		customerID = cust.ID
		customCurrency = s.createTestCustomCurrency(ctx, ns)

		_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
			billingtest.WithProgressiveBilling(),
			billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
			billingtest.WithManualApproval(),
		)

		feature := s.SetupApiRequestsTotalFeature(ctx, ns)
		featureKey = feature.Feature.Key
		s.MockStreamingConnector.AddSimpleEvent(featureKey, 3,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)
		s.MockStreamingConnector.AddSimpleEvent(featureKey, 5,
			datetime.MustParseTimeInLocation(s.T(), "2026-01-20T00:00:00Z", time.UTC).AsTime(),
		)
	})

	s.Run("#2 create credits-only usage charge", func() {
		// given:
		// - a usage-based intent whose collection period has ended
		// - mocked ledger allocation and lineage callbacks
		// when:
		// - the credits-only charge is created through the root charges service
		// then:
		// - usage is rated and the charge reaches final with a persisted allocation
		servicePeriod := timeutil.ClosedPeriod{
			From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
			To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
		}
		finalAdvanceAt := datetime.MustParseTimeInLocation(s.T(), "2026-02-03T00:01:00Z", time.UTC).AsTime()
		clock.FreezeTime(finalAdvanceAt)
		defer clock.UnFreeze()

		allocationCallback := newCountedCreditAllocationCallback[usagebased.CreditsOnlyUsageAccruedInput]()
		allocationTransactionGroupID = allocationCallback.id
		s.UsageBasedTestHandler.onCreditsOnlyUsageAccrued = allocationCallback.Handler(
			s.T(),
			func(input usagebased.CreditsOnlyUsageAccruedInput, transactionGroup ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs {
				return creditrealization.CreateAllocationInputs{
					{
						ServicePeriod:     input.Charge.Intent.GetEffectiveServicePeriod(),
						Amount:            input.AmountToAllocate,
						LedgerTransaction: transactionGroup,
					},
				}
			},
			func(t *testing.T, input usagebased.CreditsOnlyUsageAccruedInput) {
				assert.True(t, input.Charge.Intent.GetCurrency().IsCustom())
				assert.Equal(t, customCurrency.ID, input.Charge.Intent.GetCurrency().ID)
			},
		)

		lineageMock := &mockLineageService{Service: s.LineageService}
		lineageMock.On("CreateInitialLineages", mock.Anything, mock.Anything).
			Return(nil).
			Once()
		lineageMock.On("PersistCorrectionLineageSegments", mock.Anything, mock.Anything).
			Return(nil).
			Once()
		customCurrencyUsageBasedService, err := usagebasedservice.New(usagebasedservice.Config{
			Adapter:                 s.UsageBasedAdapter,
			Handler:                 s.UsageBasedTestHandler,
			Lineage:                 lineageMock,
			Locker:                  s.Locker,
			MetaAdapter:             s.MetaAdapter,
			InvoiceUpdater:          s.InvoiceUpdater,
			CustomerOverrideService: s.BillingService,
			FeatureMeterResolver:    s.FeatureMeterResolver,
			RatingService:           billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: s.UnitConfigEnabled}),
			Currencies:              s.CurrencyService,
			StreamingConnector:      s.MockStreamingConnector,
		})
		s.Require().NoError(err)

		originalUsageBasedService := s.Charges.usageBasedService
		s.Charges.usageBasedService = customCurrencyUsageBasedService
		defer func() {
			s.Charges.usageBasedService = originalUsageBasedService
		}()

		intent := charges.NewChargeIntent(usagebased.Intent{
			Intent: meta.Intent{
				ManagedBy:  billing.ManuallyManagedLine,
				CustomerID: customerID,
				Currency:   customCurrency,
			},
			FeatureKey: featureKey,
			IntentMutableFields: usagebased.IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              "Custom Currency Usage",
					ServicePeriod:     servicePeriod,
					BillingPeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
				},
				InvoiceAt: servicePeriod.To,
				Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(1.234),
				})),
			},
			SettlementMode: productcatalog.CreditOnlySettlementMode,
		})

		created, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents:   charges.ChargeIntents{intent},
		})
		s.Require().NoError(err)
		s.Require().Len(created, 1)
		s.Equal(1, allocationCallback.nrInvocations)
		lineageMock.AssertExpectations(s.T())

		createdCharge, err = created[0].AsUsageBasedCharge()
		s.Require().NoError(err)
		s.Equal(usagebased.StatusFinal, createdCharge.Status)
		s.True(createdCharge.Intent.GetCurrency().IsCustom())
		s.Equal(customCurrency.ID, createdCharge.Intent.GetCurrency().ID)
		s.Require().Len(createdCharge.Realizations, 1)
		finalRun := createdCharge.Realizations[0]
		s.Equal(float64(8), finalRun.MeteredQuantity.InexactFloat64())
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       9.872,
			CreditsTotal: 9.872,
		}, finalRun.Totals)
		s.Require().Len(finalRun.CreditsAllocated, 1)
		s.Equal(float64(9.872), finalRun.CreditsAllocated[0].Amount.InexactFloat64())
		s.Equal(allocationTransactionGroupID, finalRun.CreditsAllocated[0].LedgerTransaction.TransactionGroupID)
	})

	s.Run("#3 reload persisted charge", func() {
		// when:
		// - the usage-based charge is loaded again from Postgres
		// then:
		// - its final state, custom currency, rated totals, and allocation are preserved
		persisted := s.mustGetUsageBasedChargeByID(createdCharge.GetChargeID())
		s.Equal(usagebased.StatusFinal, persisted.Status)
		s.True(persisted.Intent.GetCurrency().IsCustom())
		s.Equal(customCurrency.ID, persisted.Intent.GetCurrency().ID)
		s.Require().Len(persisted.Realizations, 1)
		s.Equal(float64(8), persisted.Realizations[0].MeteredQuantity.InexactFloat64())
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       9.872,
			CreditsTotal: 9.872,
		}, persisted.Realizations[0].Totals)
		s.Require().Len(persisted.Realizations[0].CreditsAllocated, 1)
		s.Equal(allocationTransactionGroupID, persisted.Realizations[0].CreditsAllocated[0].LedgerTransaction.TransactionGroupID)
	})
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditOnlyInArrearsAllocatesAtInvoiceAt() {
	defer s.FlatFeeTestHandler.Reset()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-only-in-arrears")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithProgressiveBilling(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
		billingtest.WithManualApproval(),
	)

	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	createAt := datetime.MustParseTimeInLocation(s.T(), "2025-12-01T00:00:00Z", time.UTC).AsTime()

	allocateCreditsCallback := newCountedCreditAllocationCallback[flatfee.OnAllocateCreditsInput]()
	s.FlatFeeTestHandler.onAllocateCredits = allocateCreditsCallback.Handler(s.T(), func(input flatfee.OnAllocateCreditsInput, ledgerTransaction ledgertransaction.GroupReference) creditrealization.CreateAllocationInputs {
		return creditrealization.CreateAllocationInputs{
			{
				ServicePeriod:     input.Charge.Intent.GetEffectiveServicePeriod(),
				Amount:            input.PreTaxAmountToAllocate,
				LedgerTransaction: ledgerTransaction,
			},
		}
	})

	clock.FreezeTime(createAt)
	defer clock.UnFreeze()

	res, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: ns,
		Intents: []charges.ChargeIntent{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:       cust.GetID(),
				currency:       USD,
				servicePeriod:  servicePeriod,
				settlementMode: productcatalog.CreditOnlySettlementMode,
				price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(75),
					PaymentTerm: productcatalog.InArrearsPaymentTerm,
				}),
				name:              "flat-fee-credit-only-in-arrears",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "flat-fee-credit-only-in-arrears",
			}),
		},
	})
	s.NoError(err)
	s.Len(res, 1)

	createdCharge, err := res[0].AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatfee.StatusCreated, createdCharge.Status)
	s.NotNil(createdCharge.State.AdvanceAfter)
	s.True(servicePeriod.To.Equal(*createdCharge.State.AdvanceAfter))
	s.Nil(createdCharge.Realizations.CurrentRun)
	s.Zero(allocateCreditsCallback.nrInvocations)

	clock.FreezeTime(servicePeriod.From)
	advancedCharges := s.mustAdvanceFlatFeeCharges(ctx, cust.GetID())
	s.Empty(advancedCharges)
	s.Zero(allocateCreditsCallback.nrInvocations)

	clock.FreezeTime(servicePeriod.To)
	advancedCharges = s.mustAdvanceFlatFeeCharges(ctx, cust.GetID())
	s.Len(advancedCharges, 1)
	finalCharge, err := advancedCharges[0].AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatfee.StatusFinal, finalCharge.Status)
	s.Nil(finalCharge.State.AdvanceAfter)
	s.Require().NotNil(finalCharge.Realizations.CurrentRun)
	s.Len(finalCharge.Realizations.CurrentRun.CreditRealizations, 1)
	s.Equal(float64(75), finalCharge.Realizations.CurrentRun.CreditRealizations[0].Amount.InexactFloat64())
	s.Equal(1, allocateCreditsCallback.nrInvocations)
}

func (s *InvoicableChargesTestSuite) mustAdvanceFlatFeeCharges(ctx context.Context, customerID customer.CustomerID) charges.Charges {
	s.T().Helper()

	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customerID,
	})
	s.NoError(err)

	// Filter to only flat fee charges
	var flatFeeCharges charges.Charges
	for _, c := range advancedCharges {
		if c.Type() == meta.ChargeTypeFlatFee {
			flatFeeCharges = append(flatFeeCharges, c)
		}
	}

	return flatFeeCharges
}

func (s *InvoicableChargesTestSuite) mustAdvanceSingleUsageBasedCharge(ctx context.Context, customerID customer.CustomerID) *usagebased.Charge {
	s.T().Helper()

	advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
		Customer: customerID,
	})
	s.NoError(err)

	if len(advancedCharges) == 0 {
		return nil
	}

	s.Len(advancedCharges, 1)
	s.Equal(meta.ChargeTypeUsageBased, advancedCharges[0].Type())

	advancedCharge, err := advancedCharges[0].AsUsageBasedCharge()
	s.NoError(err)

	return &advancedCharge
}

func (s *InvoicableChargesTestSuite) mustGetUsageBasedChargeByID(chargeID meta.ChargeID) usagebased.Charge {
	s.T().Helper()

	charge := s.mustGetChargeByID(chargeID)
	usageBasedCharge, err := charge.AsUsageBasedCharge()
	s.NoError(err)

	return usageBasedCharge
}

func (s *InvoicableChargesTestSuite) mustGetUsageBasedChargeByIDWithDetailedLines(chargeID meta.ChargeID) usagebased.Charge {
	s.T().Helper()

	charge, err := s.Charges.GetByID(s.T().Context(), charges.GetByIDInput{
		ChargeID: chargeID,
		Expands: meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDetailedLines,
		},
	})
	s.NoError(err)

	usageBasedCharge, err := charge.AsUsageBasedCharge()
	s.NoError(err)

	return usageBasedCharge
}

func (s *InvoicableChargesTestSuite) mustGetFlatFeeChargeByIDWithDetailedLines(chargeID meta.ChargeID) flatfee.Charge {
	s.T().Helper()

	charge, err := s.Charges.GetByID(s.T().Context(), charges.GetByIDInput{
		ChargeID: chargeID,
		Expands: meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDetailedLines,
		},
	})
	s.NoError(err)

	flatFeeCharge, err := charge.AsFlatFeeCharge()
	s.NoError(err)

	return flatFeeCharge
}

func (s *InvoicableChargesTestSuite) enableFlatFeeCustomCurrenciesWithMockLineage() {
	s.T().Helper()

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

	customCurrencyFlatFeeService, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:       s.FlatFeeAdapter,
		Handler:       s.FlatFeeTestHandler,
		Lineage:       lineageMock,
		MetaAdapter:   s.MetaAdapter,
		Locker:        s.Locker,
		RatingService: billingratingservice.New(billingratingservice.Config{UnitConfigEnabled: s.UnitConfigEnabled}),
		Currencies:    s.CurrencyService,
	})
	s.Require().NoError(err)

	originalFlatFeeService := s.Charges.flatFeeService
	s.Charges.flatFeeService = customCurrencyFlatFeeService
	s.Require().NoError(s.BillingService.DeregisterLineEngine(billing.LineEngineTypeChargeFlatFee))
	s.Require().NoError(s.BillingService.RegisterLineEngine(customCurrencyFlatFeeService.GetLineEngine()))
	s.T().Cleanup(func() {
		s.Charges.flatFeeService = originalFlatFeeService
		s.Require().NoError(s.BillingService.DeregisterLineEngine(billing.LineEngineTypeChargeFlatFee))
		s.Require().NoError(s.BillingService.RegisterLineEngine(originalFlatFeeService.GetLineEngine()))
	})
}

func mustGetFlatFeeChargeWithExpands(s *BaseSuite, chargeID meta.ChargeID, expands meta.Expands) flatfee.Charge {
	s.T().Helper()

	charge, err := s.Charges.GetByID(s.T().Context(), charges.GetByIDInput{
		ChargeID: chargeID,
		Expands:  expands,
	})
	s.NoError(err)

	flatFeeCharge, err := charge.AsFlatFeeCharge()
	s.NoError(err)

	return flatFeeCharge
}

func activeGatheringLinesForCharge(s *BaseSuite, namespace, customerID, chargeID string) []billing.GatheringLine {
	s.T().Helper()

	gatheringInvoices, err := s.BillingService.ListGatheringInvoices(s.T().Context(), billing.ListGatheringInvoicesInput{
		Namespace: namespace,
		Customers: []string{customerID},
		Expand: billing.GatheringInvoiceExpands{
			billing.GatheringInvoiceExpandLines,
		},
	})
	s.NoError(err)

	var lines []billing.GatheringLine
	for _, invoice := range gatheringInvoices.Items {
		for _, line := range invoice.Lines.OrEmpty() {
			if line.DeletedAt != nil || line.ChargeID == nil || *line.ChargeID != chargeID {
				continue
			}

			lines = append(lines, line)
		}
	}

	return lines
}

type assertFlatFeeCreditThenInvoiceLineAndRunInput struct {
	Invoice                       billing.StandardInvoice
	FlatFeeChargeID               meta.ChargeID
	ServicePeriod                 timeutil.ClosedPeriod
	ExpectedTotals                billingtest.ExpectedTotals
	ExpectedCreditsApplied        alpacadecimal.Decimal
	ExpectAccruedUsage            bool
	InvoiceUsageAccruedCallbackID string
}

func (s *InvoicableChargesTestSuite) assertFlatFeeCreditThenInvoiceLineAndRun(input assertFlatFeeCreditThenInvoiceLineAndRunInput) billing.LineID {
	s.T().Helper()

	lines := input.Invoice.Lines.OrEmpty()
	s.Len(lines, 1)
	stdLine := lines[0]
	s.Equal(input.FlatFeeChargeID.ID, lo.FromPtr(stdLine.ChargeID))
	s.RequireTotals(input.ExpectedTotals, stdLine.Totals)
	s.Len(stdLine.CreditsApplied, 1)
	s.True(input.ExpectedCreditsApplied.Equal(stdLine.CreditsApplied[0].Amount), "standard line credits applied amount should match")
	s.Len(stdLine.DetailedLines, 1)

	detailedLine := stdLine.DetailedLines[0]
	s.True(detailedLine.Totals.Equal(stdLine.Totals), "standard line detailed line totals should match standard line totals")
	s.RequireTotals(input.ExpectedTotals, detailedLine.Totals)
	s.Len(detailedLine.CreditsApplied, 1)
	s.True(input.ExpectedCreditsApplied.Equal(detailedLine.CreditsApplied[0].Amount), "standard line detailed credits applied amount should match")
	s.Equal(stdLine.CreditsApplied[0].CreditRealizationID, detailedLine.CreditsApplied[0].CreditRealizationID)

	flatFeeWithDetailedLines := s.mustGetFlatFeeChargeByIDWithDetailedLines(input.FlatFeeChargeID)
	s.Require().NotNil(flatFeeWithDetailedLines.Realizations.CurrentRun)
	currentRun := flatFeeWithDetailedLines.Realizations.CurrentRun
	s.NotNil(currentRun.LineID)
	s.Equal(stdLine.ID, *currentRun.LineID)
	s.NotNil(currentRun.InvoiceID)
	s.Equal(input.Invoice.ID, *currentRun.InvoiceID)
	s.Len(currentRun.CreditRealizations, 1)
	s.True(input.ExpectedCreditsApplied.Equal(currentRun.CreditRealizations[0].Amount), "run credit realization amount should match")
	s.Equal(stdLine.CreditsApplied[0].CreditRealizationID, currentRun.CreditRealizations[0].ID)
	s.RequireTotals(input.ExpectedTotals, currentRun.Totals)
	s.True(currentRun.DetailedLines.IsPresent())
	runDetailedLines := currentRun.DetailedLines.OrEmpty()
	s.Len(runDetailedLines, len(stdLine.DetailedLines))
	runDetailedLine := runDetailedLines[0]
	s.Equal(detailedLine.ChildUniqueReferenceID, runDetailedLine.ChildUniqueReferenceID)
	s.Equal(detailedLine.Category, runDetailedLine.Category)
	s.Equal(detailedLine.PaymentTerm, runDetailedLine.PaymentTerm)
	s.Equal(detailedLine.ServicePeriod, runDetailedLine.ServicePeriod)
	s.True(detailedLine.PerUnitAmount.Equal(runDetailedLine.PerUnitAmount), "persisted run detailed line per-unit amount should match standard detailed line")
	s.Equal(detailedLine.Quantity.String(), runDetailedLine.Quantity.String())
	s.True(runDetailedLine.Totals.Equal(detailedLine.Totals), "persisted run detailed line totals should match standard detailed line totals")
	s.True(runDetailedLine.Totals.Equal(stdLine.Totals), "persisted run detailed line totals should match standard line totals")
	s.RequireTotals(input.ExpectedTotals, runDetailedLine.Totals)
	s.Len(runDetailedLine.CreditsApplied, 1)
	s.True(input.ExpectedCreditsApplied.Equal(runDetailedLine.CreditsApplied[0].Amount), "run detailed line credits applied amount should match")
	s.Equal(detailedLine.CreditsApplied[0].CreditRealizationID, runDetailedLine.CreditsApplied[0].CreditRealizationID)

	if input.ExpectAccruedUsage {
		s.Require().NotNil(currentRun.AccruedUsage)
		s.Require().NotNil(currentRun.AccruedUsage.LedgerTransaction)
		s.Equal(input.InvoiceUsageAccruedCallbackID, currentRun.AccruedUsage.LedgerTransaction.TransactionGroupID)
		s.Equal(input.ServicePeriod, currentRun.AccruedUsage.ServicePeriod)
		s.RequireTotals(input.ExpectedTotals, currentRun.AccruedUsage.Totals)
	} else {
		s.Nil(currentRun.AccruedUsage)
	}

	return stdLine.GetLineID()
}
