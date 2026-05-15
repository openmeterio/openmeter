package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingtotals "github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestInvoicableCharges(t *testing.T) {
	suite.Run(t, new(InvoicableChargesTestSuite))
}

type InvoicableChargesTestSuite struct {
	BaseSuite
}

func (s *InvoicableChargesTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *InvoicableChargesTestSuite) TearDownTest() {
	s.BaseSuite.TearDownTest()
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
			s.Equal(flatfee.StatusCreated, charge.Status)
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
			Namespaces: []string{ns},
			Customers:  []string{cust.ID},
			Currencies: []currencyx.Code{currencyx.Code(currency.USD)},
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
			assert.True(t, input.Amount.IsPositive())
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
			assert.True(t, input.Amount.IsPositive())
			assert.NotNil(t, input.Charge.Realizations.CurrentRun)
			assert.Nil(t, input.Charge.Realizations.CurrentRun.Payment)
			assert.NotNil(t, input.Charge.Realizations.CurrentRun.AccruedUsage)
			assert.Equal(t, flatfee.StatusActiveAwaitingPaymentSettlement, input.Charge.Status)
		})

		settledCallback := newCountedLedgerTransactionCallback[flatfee.OnPaymentSettledInput]()
		// Use non-fatal assertions inside handler callbacks so failures are reported
		// on the callback's testing context without aborting the parent test flow.
		s.FlatFeeTestHandler.onPaymentSettled = settledCallback.Handler(s.T(), func(t *testing.T, input flatfee.OnPaymentSettledInput) {
			assert.True(t, input.Amount.IsPositive())
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

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditThenInvoiceFullyCreditedDoesNotAccrueInvoiceUsage() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-then-invoice-fully-credited")

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

	s.Run("post invoice issued without invoice usage accrual", func() {
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

		err = lineEngine.OnInvoiceIssued(ctx, billing.OnInvoiceIssuedInput{
			Invoice: invoice,
			Lines:   invoice.Lines.OrEmpty(),
		})
		s.NoError(err)
		s.Equal(0, invoiceUsageAccruedCallback.nrInvocations)

		updatedFlatFeeCharge := s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
		s.Nil(updatedFlatFeeCharge.Realizations.CurrentRun.AccruedUsage)
		s.True(updatedFlatFeeCharge.Realizations.CurrentRun.DetailedLines.IsPresent())
		s.Len(updatedFlatFeeCharge.Realizations.CurrentRun.DetailedLines.OrEmpty(), len(invoice.Lines.OrEmpty()[0].DetailedLines))
	})
}

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditThenInvoiceZeroAmountNonZeroChargesAccruesInvoiceUsage() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-then-invoice-zero-amount-charges")

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
		// - the flat-fee line engine receives the invoice-issued callback
		// then:
		// - invoice usage accrual still runs because the payable total is non-zero
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

		err = lineEngine.OnInvoiceIssued(ctx, billing.OnInvoiceIssuedInput{
			Invoice: invoice,
			Lines:   invoice.Lines.OrEmpty(),
		})
		s.NoError(err)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)

		updatedFlatFeeCharge := s.mustGetFlatFeeChargeByIDWithDetailedLines(flatFeeChargeID)
		s.Require().NotNil(updatedFlatFeeCharge.Realizations.CurrentRun)
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
			Namespaces: []string{ns},
			Customers:  []string{cust.ID},
			Currencies: []currencyx.Code{currencyx.Code(currency.USD)},
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
					ServicePeriod: input.Charge.Intent.ServicePeriod,
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
		s.Equal(usagebased.StatusActiveFinalRealizationWaitingForCollection, usageBasedFromDB.Status)
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
		s.True(finalStoredAtLT.Equal(startedCallbacks[0].Input.AllocateAt))
	})

	s.Run("#3.2 second realization advance is noop", func() {
		// Given the charge is waiting for collection.
		// When advancing the usage-based charge again.
		advancedCharge := s.mustAdvanceSingleUsageBasedCharge(ctx, cust.GetID())
		usageBasedFromDB := s.mustGetUsageBasedChargeByID(usageBasedChargeID)

		// Then nothing happens.
		s.Nil(advancedCharge)
		s.Equal(usagebased.StatusActiveFinalRealizationWaitingForCollection, usageBasedFromDB.Status)
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
		s.Equal(usagebased.StatusActiveFinalRealizationWaitingForCollection, usageBasedFromDB.Status)
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
					ServicePeriod: input.Charge.Intent.ServicePeriod,
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
		s.True(finalStoredAtLT.Equal(finalizedCallbacks[0].Input.AllocateAt))
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
			s.Equal(productcatalog.CreditOnlySettlementMode, input.Charge.Intent.SettlementMode)
			s.Equal(usagebased.RealizationRunTypeFinalRealization, input.Run.Type)
			s.True(finalStoredAtLT.Equal(input.AllocateAt))
			s.Equal(float64(20), input.AmountToAllocate.InexactFloat64())
			s.Equal(float64(10), input.Run.MeteredQuantity.InexactFloat64())
			s.RequireTotals(billingtest.ExpectedTotals{
				Amount: 20,
				Total:  20,
			}, input.Run.Totals)
			s.Empty(input.Run.CreditsAllocated)

			return creditrealization.CreateAllocationInputs{
				{
					ServicePeriod: input.Charge.Intent.ServicePeriod,
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
		s.Equal(usagebased.StatusActiveFinalRealizationWaitingForCollection, usageBasedFromDB.Status)
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
			s.Equal(productcatalog.CreditOnlySettlementMode, input.Charge.Intent.SettlementMode)
			s.Equal(usagebased.RealizationRunTypeFinalRealization, input.Run.Type)
			s.True(finalStoredAtLT.Equal(input.AllocateAt))
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
		s.True(finalStoredAtLT.Equal(correctedCallbacks[0].Input.AllocateAt))
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
		s.Equal(usagebased.StatusActiveFinalRealizationWaitingForCollection, usageBasedCharge.Status)
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
		s.Equal(usagebased.StatusActiveFinalRealizationProcessing, usageBasedCharge.Status)
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
		s.Equal(usagebased.StatusActiveFinalRealizationWaitingForCollection, usageBasedCharge.Status)
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
		s.Equal(usagebased.StatusActiveFinalRealizationProcessing, usageBasedCharge.Status)
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

	s.Run("#5 approve invoice with no fiat invoice usage accrual", func() {
		defer s.UsageBasedTestHandler.Reset()

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[usagebased.OnInvoiceUsageAccruedInput]()
		s.UsageBasedTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		var err error
		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(0, invoiceUsageAccruedCallback.nrInvocations)

		usageBasedCharge := s.mustGetUsageBasedChargeByID(usageBasedChargeID)
		s.Equal(usagebased.StatusActiveAwaitingPaymentSettlement, usageBasedCharge.Status)
		s.Nil(usageBasedCharge.State.CurrentRealizationRunID)
		s.Nil(usageBasedCharge.State.AdvanceAfter)
		s.Len(usageBasedCharge.Realizations, 1)

		finalRun := usageBasedCharge.Realizations[0]
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
				ServicePeriod: input.Charge.Intent.ServicePeriod,
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
			Namespaces: []string{ns},
			Customers:  []string{cust.ID},
			Currencies: []currencyx.Code{currencyx.Code(currency.USD)},
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

	s.Run("#2 advance at invoice_at goes to final", func() {
		defer s.FlatFeeTestHandler.Reset()

		type callbackInvocation struct {
			Input flatfee.OnAllocateCreditsInput
		}

		var callbacks []callbackInvocation

		s.FlatFeeTestHandler.onAllocateCredits = func(ctx context.Context, input flatfee.OnAllocateCreditsInput) (creditrealization.CreateAllocationInputs, error) {
			callbacks = append(callbacks, callbackInvocation{Input: input})

			return creditrealization.CreateAllocationInputs{
				{
					ServicePeriod: input.Charge.Intent.ServicePeriod,
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
		s.Equal(float64(100), callbacks[0].Input.PreTaxAmountToAllocate.InexactFloat64())

		// Credit realizations were persisted.
		s.Require().NotNil(fetchedFF.Realizations.CurrentRun)
		s.Len(fetchedFF.Realizations.CurrentRun.CreditRealizations, 1)
		s.Equal(float64(100), fetchedFF.Realizations.CurrentRun.CreditRealizations[0].Amount.InexactFloat64())
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
				ServicePeriod: input.Charge.Intent.ServicePeriod,
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

func (s *InvoicableChargesTestSuite) TestFlatFeeCreditOnlyInArrearsActivatesAtServiceStartAndAllocatesAtInvoiceAt() {
	defer s.FlatFeeTestHandler.Reset()

	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-flatfee-credit-only-in-arrears")

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
				ServicePeriod:     input.Charge.Intent.ServicePeriod,
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
	s.True(servicePeriod.From.Equal(*createdCharge.State.AdvanceAfter))
	s.Nil(createdCharge.Realizations.CurrentRun)
	s.Zero(allocateCreditsCallback.nrInvocations)

	clock.FreezeTime(servicePeriod.From)
	advancedCharges := s.mustAdvanceFlatFeeCharges(ctx, cust.GetID())
	s.Len(advancedCharges, 1)
	activeCharge, err := advancedCharges[0].AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(flatfee.StatusActive, activeCharge.Status)
	s.NotNil(activeCharge.State.AdvanceAfter)
	s.True(servicePeriod.To.Equal(*activeCharge.State.AdvanceAfter))
	s.Nil(activeCharge.Realizations.CurrentRun)
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
		Namespaces: []string{namespace},
		Customers:  []string{customerID},
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
	s.Equal(detailedLine.Currency, runDetailedLine.Currency)
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
