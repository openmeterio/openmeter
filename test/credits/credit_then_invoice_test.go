package credits

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func TestCreditThenInvoiceTestSuite(t *testing.T) {
	suite.Run(t, new(CreditThenInvoiceTestSuite))
}

type CreditThenInvoiceTestSuite struct {
	BaseSuite
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceDeletePatchDeletesPendingGatheringLine() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-delete-gathering")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var usageBasedChargeID meta.ChargeID
	ledgerSnapshotInput := LedgerSnapshotInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Currency:  USD,
		CostBasis: mo.None[*alpacadecimal.Decimal](),
	}
	startLedger := s.CreateLedgerSnapshot(ledgerSnapshotInput)

	s.Run("given a credit-then-invoice usage charge with a pending gathering line", func() {
		// given:
		// - a ledger-backed customer has no credit allocations or invoice bookings
		// when:
		// - a credit-then-invoice usage charge is created for a future service period
		// then:
		// - billing has one active gathering line for the charge and the ledger remains unchanged
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-delete-gathering",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-delete-gathering",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusCreated)

		activeLines := s.mustGatheringLinesForCharge(ns, cust.ID, usageBasedChargeID.ID, false)
		s.Len(activeLines, 1)
		s.Nil(activeLines[0].DeletedAt)

		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, startLedger)
	})

	s.Run("when the charge delete patch is applied", func() {
		// given:
		// - the only billing artifact is a mutable gathering line
		// when:
		// - the charge is deleted through the patch flow
		// then:
		// - the gathering line is soft-deleted
		s.MustRefundCharge(ctx, cust.GetID(), usageBasedChargeID)

		activeLines := s.mustGatheringLinesForCharge(ns, cust.ID, usageBasedChargeID.ID, false)
		s.Empty(activeLines)

		allLines := s.mustGatheringLinesForCharge(ns, cust.ID, usageBasedChargeID.ID, true)
		s.Len(allLines, 1)
		s.NotNil(allLines[0].DeletedAt)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusDeleted)
	})

	s.Run("then no ledger transaction was reversed or created for the gathering-line-only delete", func() {
		// given:
		// - gathering lines do not have credit allocations, invoice accrual, or payment bookings
		// when:
		// - the deleted gathering line is inspected after the patch
		// then:
		// - every ledger balance is still identical to the pre-delete snapshot
		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusDeleted)
		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, startLedger)
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceDeletePatchDeletesMutableStandardLineAndCorrectsCredits() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-delete-standard")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
		lineID             billing.LineID
		runID              usagebased.RealizationRunID
		startLedger        LedgerSnapshot
	)
	ledgerSnapshotInput := LedgerSnapshotInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Currency:  USD,
		CostBasis: mo.Some(&zeroCostBasis),
	}

	s.Run("given prepaid credits and a credit-then-invoice usage charge", func() {
		// given:
		// - a ledger-backed customer receives 5 USD promotional credits
		// - 5 usage units are visible inside the service period
		// when:
		// - a unit-priced credit-then-invoice usage charge is created
		// then:
		// - credits are available in FBO and the charge has no invoice-backed run yet
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(5),
			At:        setupAt,
			CostBasis: zeroCostBasis,
		})

		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-delete-standard",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-delete-standard",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusCreated)

		startLedger = s.CreateLedgerSnapshot(ledgerSnapshotInput)
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), startLedger.FBO, "prepaid credits should be available before invoicing")
		s.AssertDecimalEqual(alpacadecimal.Zero, startLedger.Accrued, "no usage should be accrued before invoicing")
	})

	s.Run("when the pending line is collected into a mutable draft invoice", func() {
		// given:
		// - usage is fully covered by available credits
		// when:
		// - billing creates and collects the standard invoice but does not approve it
		// then:
		// - the mutable standard line has credit allocations but no invoice accrued usage or payment booking
		clock.FreezeTime(servicePeriod.To.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]

		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		s.Len(invoice.Lines.OrEmpty(), 1)

		line := invoice.Lines.OrEmpty()[0]
		lineID = line.GetLineID()
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, line.Totals)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveFinalRealizationProcessing)
		s.Len(charge.Realizations, 1)
		currentRun, err := charge.GetCurrentRealizationRun()
		s.NoError(err)
		runID = currentRun.ID
		s.Equal(lineID.ID, lo.FromPtr(currentRun.LineID))
		s.Equal(invoice.ID, lo.FromPtr(currentRun.InvoiceID))
		s.Equal(alpacadecimal.NewFromInt(5), currentRun.CreditsAllocated.Sum())
		s.Nil(currentRun.InvoiceUsage)
		s.Nil(currentRun.Payment)

		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&zeroCostBasis)), "draft line credit allocation should consume FBO")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&zeroCostBasis)), "draft line credit allocation should accrue credits")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen), "fully credited draft should not create open receivable")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized), "fully credited draft should not create authorized receivable")
	})

	s.Run("when the charge delete patch removes the mutable standard line", func() {
		// given:
		// - the standard invoice is still mutable and has no payment or invoice accrued allocation
		// when:
		// - the charge is deleted through the patch flow
		// then:
		// - the standard line is soft-deleted and the realization run is marked deleted
		s.MustRefundCharge(ctx, cust.GetID(), usageBasedChargeID)

		fetchedInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand: billing.InvoiceExpands{
				billing.InvoiceExpandLines,
				billing.InvoiceExpandDeletedLines,
			},
		})
		s.NoError(err)

		standardInvoice, err := fetchedInvoice.AsStandardInvoice()
		s.NoError(err)
		deletedLine := standardInvoice.Lines.GetByID(lineID.ID)
		s.Require().NotNil(deletedLine)
		s.NotNil(deletedLine.DeletedAt)
		s.Zero(standardInvoice.Lines.NonDeletedLineCount())

		charge := s.mustGetUsageBasedChargeByIDWithExpands(usageBasedChargeID, meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDeletedRealizations,
		})
		run, err := charge.Realizations.GetByID(runID.ID)
		s.NoError(err)
		s.NotNil(run.DeletedAt)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusDeleted)
	})

	s.Run("then deleting the mutable line reverses only the credit allocation ledger transactions", func() {
		// given:
		// - the deleted draft line had only credit allocations
		// when:
		// - the line-engine cleanup has completed
		// then:
		// - credits are returned to FBO, accrued is cleared, and receivables stay unchanged
		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusDeleted)
		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, startLedger)
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen), "aggregate open receivable should stay empty")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized), "aggregate authorized receivable should stay empty")
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceDeletePatchKeepsImmutableStandardLineAndLedgerBookings() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-delete-immutable")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
		lineID             billing.LineID
		runID              usagebased.RealizationRunID
		immutableLedger    LedgerSnapshot
	)
	ledgerSnapshotInput := LedgerSnapshotInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Currency:  USD,
		CostBasis: mo.Some(&zeroCostBasis),
	}

	s.Run("given a credit-then-invoice usage charge with an immutable invoice", func() {
		// given:
		// - a ledger-backed customer has enough credits to cover the invoice line
		// - usage is visible inside the service period
		// when:
		// - the standard invoice is collected and approved
		// then:
		// - the invoice line is immutable and its credit/invoice-usage ledger bookings exist
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(5),
			At:        setupAt,
			CostBasis: zeroCostBasis,
		})

		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-delete-immutable",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-delete-immutable",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusCreated)

		clock.FreezeTime(servicePeriod.To.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]

		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		s.Len(invoice.Lines.OrEmpty(), 1)

		line := invoice.Lines.OrEmpty()[0]
		lineID = line.GetLineID()
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, line.Totals)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		s.True(invoice.StatusDetails.Immutable)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveAwaitingPaymentSettlement)
		s.Len(charge.Realizations, 1)
		currentRun := charge.Realizations[0]
		runID = currentRun.ID
		s.Equal(lineID.ID, lo.FromPtr(currentRun.LineID))
		s.Equal(invoice.ID, lo.FromPtr(currentRun.InvoiceID))
		s.Equal(alpacadecimal.NewFromInt(5), currentRun.CreditsAllocated.Sum())
		s.NotNil(currentRun.InvoiceUsage)
		s.Nil(currentRun.Payment)

		immutableLedger = s.CreateLedgerSnapshot(ledgerSnapshotInput)
		s.AssertDecimalEqual(alpacadecimal.Zero, immutableLedger.FBO, "immutable invoice credit allocation should keep FBO consumed")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), immutableLedger.Accrued, "immutable invoice should keep accrued credit booking")
		s.AssertDecimalEqual(alpacadecimal.Zero, immutableLedger.OpenReceivable, "fully credited immutable invoice should not create open receivable")
	})

	s.Run("when the charge delete patch targets the immutable standard line", func() {
		// given:
		// - the invoice line is immutable and cannot be deleted without prorating support
		// when:
		// - the charge is deleted through the patch flow
		// then:
		// - the invoice line and realization run remain active, and the invoice records a warning
		s.MustRefundCharge(ctx, cust.GetID(), usageBasedChargeID)

		fetchedInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand: billing.InvoiceExpands{
				billing.InvoiceExpandLines,
			},
		})
		s.NoError(err)

		standardInvoice, err := fetchedInvoice.AsStandardInvoice()
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, standardInvoice.Status)

		line := standardInvoice.Lines.GetByID(lineID.ID)
		s.Require().NotNil(line)
		s.Nil(line.DeletedAt)
		s.Equal(1, standardInvoice.Lines.NonDeletedLineCount())

		s.Require().Len(standardInvoice.ValidationIssues, 1)
		issue := standardInvoice.ValidationIssues[0]
		s.Equal(billing.ValidationIssueSeverityWarning, issue.Severity)
		s.Equal(billing.ImmutableInvoiceHandlingNotSupportedErrorCode, issue.Code)
		s.Equal(billing.ComponentName("charges.invoiceupdater"), issue.Component)
		s.Equal("line should be deleted, but the invoice is immutable", issue.Message)
		s.Equal("lines/"+lineID.ID, issue.Path)

		charge := s.mustGetUsageBasedChargeByIDWithExpands(usageBasedChargeID, meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDeletedRealizations,
		})
		run, err := charge.Realizations.GetByID(runID.ID)
		s.NoError(err)
		s.Nil(run.DeletedAt)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusDeleted)
	})

	s.Run("then immutable invoice deletion does not reverse ledger bookings", func() {
		// given:
		// - the delete request only produced an immutable-invoice warning
		// when:
		// - the ledger is inspected after the patch
		// then:
		// - the already-issued invoice credit and accrual bookings remain unchanged
		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusDeleted)
		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, immutableLedger)
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen), "aggregate open receivable should stay empty")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized), "aggregate authorized receivable should stay empty")
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceExtendPatchUpdatesPendingGatheringLine() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-extend-gathering")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	extendedServicePeriodTo := datetime.MustParseTimeInLocation(t, "2026-03-01T00:00:00Z", time.UTC).AsTime()
	extendedInvoiceAt := datetime.MustParseTimeInLocation(t, "2026-03-03T00:00:00Z", time.UTC).AsTime()

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID  meta.ChargeID
		gatheringLineID     string
		ledgerSnapshotInput = LedgerSnapshotInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Currency:  USD,
			CostBasis: mo.None[*alpacadecimal.Decimal](),
		}
		startLedger = s.CreateLedgerSnapshot(ledgerSnapshotInput)
	)

	s.Run("given a credit-then-invoice usage charge with a pending gathering line", func() {
		// given:
		// - a ledger-backed customer has no credit allocations or invoice bookings
		// when:
		// - a credit-then-invoice usage charge is created for a future service period
		// then:
		// - billing has one active gathering line for the charge and the ledger remains unchanged
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-extend-gathering",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-extend-gathering",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusCreated)

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		gatheringLineID = activeLine.ID
		s.Equal(servicePeriod, activeLine.ServicePeriod)
		s.Equal(servicePeriod.To, activeLine.InvoiceAt)

		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, startLedger)
	})

	s.Run("when the charge extend patch is applied before collection", func() {
		// given:
		// - the charge is still represented only by a pending gathering line
		// when:
		// - the charge is extended to the later service-period end
		// then:
		// - the same charge and gathering line are kept, with the gathering line extended in place
		s.mustExtendChargeWithInvoiceAt(ctx, cust.GetID(), usageBasedChargeID, extendedServicePeriodTo, extendedInvoiceAt)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusCreated)
		s.Equal(usageBasedChargeID.ID, charge.ID)
		s.Equal(extendedServicePeriodTo, charge.Intent.ServicePeriod.To)
		s.Equal(extendedServicePeriodTo, charge.Intent.FullServicePeriod.To)
		s.Equal(extendedServicePeriodTo, charge.Intent.BillingPeriod.To)
		s.Equal(extendedInvoiceAt, charge.Intent.InvoiceAt)

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		s.Equal(gatheringLineID, activeLine.ID)
		s.Equal(servicePeriod.From, activeLine.ServicePeriod.From)
		s.Equal(extendedServicePeriodTo, activeLine.ServicePeriod.To)
		s.Equal(extendedInvoiceAt, activeLine.InvoiceAt)
	})

	s.Run("then extending a gathering-line-only charge does not change ledger balances", func() {
		// given:
		// - gathering lines do not have credit allocations, invoice accrual, or payment bookings
		// when:
		// - the ledger is inspected after the extend patch
		// then:
		// - every ledger balance is still identical to the pre-extend snapshot
		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusCreated)
		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, startLedger)
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceShrinkPatchUpdatesPendingGatheringLine() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-shrink-gathering")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	shrunkServicePeriodTo := datetime.MustParseTimeInLocation(t, "2026-01-20T00:00:00Z", time.UTC).AsTime()

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID  meta.ChargeID
		gatheringLineID     string
		ledgerSnapshotInput = LedgerSnapshotInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Currency:  USD,
			CostBasis: mo.None[*alpacadecimal.Decimal](),
		}
		startLedger = s.CreateLedgerSnapshot(ledgerSnapshotInput)
	)

	s.Run("given a credit-then-invoice usage charge with a pending gathering line", func() {
		// given:
		// - a ledger-backed customer has no credit allocations or invoice bookings
		// when:
		// - a credit-then-invoice usage charge is created for a future service period
		// then:
		// - billing has one active gathering line for the charge and the ledger remains unchanged
		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-shrink-gathering",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-shrink-gathering",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusCreated)

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		gatheringLineID = activeLine.ID
		s.Equal(servicePeriod, activeLine.ServicePeriod)
		s.Equal(servicePeriod.To, activeLine.InvoiceAt)

		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, startLedger)
	})

	s.Run("when the charge shrink patch is applied before collection", func() {
		// given:
		// - the charge is still represented only by a pending gathering line
		// when:
		// - the charge is shrunk to the earlier service-period end
		// then:
		// - the same charge and gathering line are kept, with the gathering line shrunk in place
		s.mustShrinkCharge(ctx, cust.GetID(), usageBasedChargeID, shrunkServicePeriodTo)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusCreated)
		s.Equal(usageBasedChargeID.ID, charge.ID)
		s.Equal(shrunkServicePeriodTo, charge.Intent.ServicePeriod.To)
		s.Equal(shrunkServicePeriodTo, charge.Intent.FullServicePeriod.To)
		s.Equal(shrunkServicePeriodTo, charge.Intent.BillingPeriod.To)

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		s.Equal(gatheringLineID, activeLine.ID)
		s.Equal(servicePeriod.From, activeLine.ServicePeriod.From)
		s.Equal(shrunkServicePeriodTo, activeLine.ServicePeriod.To)
		s.Equal(shrunkServicePeriodTo, activeLine.InvoiceAt)
	})

	s.Run("then shrinking a gathering-line-only charge does not change ledger balances", func() {
		// given:
		// - gathering lines do not have credit allocations, invoice accrual, or payment bookings
		// when:
		// - the ledger is inspected after the shrink patch
		// then:
		// - every ledger balance is still identical to the pre-shrink snapshot
		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusCreated)
		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, startLedger)
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceExtendPatchDeletesMutableStandardLineAndCorrectsCredits() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-extend-mutable-standard")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	extendedServicePeriodTo := datetime.MustParseTimeInLocation(t, "2026-03-01T00:00:00Z", time.UTC).AsTime()
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
		lineID             billing.LineID
		runID              usagebased.RealizationRunID
		startLedger        LedgerSnapshot
	)
	ledgerSnapshotInput := LedgerSnapshotInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Currency:  USD,
		CostBasis: mo.Some(&zeroCostBasis),
	}

	s.Run("given a mutable standard invoice line with credit allocations", func() {
		// given:
		// - a ledger-backed customer has enough promotional credits to cover usage
		// - usage is visible inside the original service period
		// when:
		// - the pending line is collected into a draft standard invoice
		// then:
		// - the current final run is backed by a mutable standard line with credit allocations only
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(5),
			At:        setupAt,
			CostBasis: zeroCostBasis,
		})

		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-extend-mutable-standard",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-extend-mutable-standard",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		startLedger = s.CreateLedgerSnapshot(ledgerSnapshotInput)
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), startLedger.FBO, "prepaid credits should be available before invoicing")

		clock.FreezeTime(servicePeriod.To.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]

		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		s.Len(invoice.Lines.OrEmpty(), 1)

		line := invoice.Lines.OrEmpty()[0]
		lineID = line.GetLineID()
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, line.Totals)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveFinalRealizationProcessing)
		currentRun, err := charge.GetCurrentRealizationRun()
		s.NoError(err)
		runID = currentRun.ID
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
		s.Equal(lineID.ID, lo.FromPtr(currentRun.LineID))
		s.Equal(invoice.ID, lo.FromPtr(currentRun.InvoiceID))
		s.Equal(alpacadecimal.NewFromInt(5), currentRun.CreditsAllocated.Sum())
		s.Nil(currentRun.InvoiceUsage)
		s.Nil(currentRun.Payment)
	})

	s.Run("when the charge is extended while the final run is backed by a mutable line", func() {
		// given:
		// - the current final realization run is backed by a mutable standard invoice line
		// when:
		// - the charge is extended to a later service-period end
		// then:
		// - the mutable standard line is soft-deleted and the run is marked deleted
		s.mustExtendCharge(ctx, cust.GetID(), usageBasedChargeID, extendedServicePeriodTo)

		fetchedInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand: billing.InvoiceExpands{
				billing.InvoiceExpandLines,
				billing.InvoiceExpandDeletedLines,
			},
		})
		s.NoError(err)

		standardInvoice, err := fetchedInvoice.AsStandardInvoice()
		s.NoError(err)
		deletedLine := standardInvoice.Lines.GetByID(lineID.ID)
		s.Require().NotNil(deletedLine)
		s.NotNil(deletedLine.DeletedAt)
		s.Zero(standardInvoice.Lines.NonDeletedLineCount())

		charge := s.mustGetUsageBasedChargeByIDWithExpands(usageBasedChargeID, meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDeletedRealizations,
		})
		run, err := charge.Realizations.GetByID(runID.ID)
		s.NoError(err)
		s.NotNil(run.DeletedAt)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusActive)

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		s.Equal(servicePeriod.From, activeLine.ServicePeriod.From)
		s.Equal(extendedServicePeriodTo, activeLine.ServicePeriod.To)
		s.Equal(extendedServicePeriodTo, activeLine.InvoiceAt)
	})

	s.Run("then the charge returns to active and only credit allocations are reversed", func() {
		// given:
		// - the deleted draft line had only credit allocations
		// when:
		// - the line-engine cleanup has completed
		// then:
		// - credits are returned to FBO, accrued is cleared, and the charge waits for the extended end
		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActive)
		s.Nil(charge.State.CurrentRealizationRunID)
		s.Require().NotNil(charge.State.AdvanceAfter)
		s.True(charge.State.AdvanceAfter.Equal(extendedServicePeriodTo), "advance after should match the extended service-period end")
		s.True(charge.Intent.ServicePeriod.To.Equal(extendedServicePeriodTo), "service-period end should match the extension")

		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, startLedger)
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen), "aggregate open receivable should stay empty")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized), "aggregate authorized receivable should stay empty")
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceShrinkPatchDeletesMutableStandardLineAndCorrectsCredits() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-shrink-mutable-standard")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	shrunkServicePeriodTo := datetime.MustParseTimeInLocation(t, "2026-01-20T00:00:00Z", time.UTC).AsTime()
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
		lineID             billing.LineID
		runID              usagebased.RealizationRunID
		startLedger        LedgerSnapshot
	)
	ledgerSnapshotInput := LedgerSnapshotInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Currency:  USD,
		CostBasis: mo.Some(&zeroCostBasis),
	}

	s.Run("given a mutable standard invoice line with credit allocations", func() {
		// given:
		// - a ledger-backed customer has enough promotional credits to cover usage
		// - usage is visible before the future shrink boundary
		// when:
		// - the original pending line is collected into a draft standard invoice
		// then:
		// - the current final run is backed by a mutable standard line with credit allocations only
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(5),
			At:        setupAt,
			CostBasis: zeroCostBasis,
		})

		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-shrink-mutable-standard",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-shrink-mutable-standard",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		startLedger = s.CreateLedgerSnapshot(ledgerSnapshotInput)
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), startLedger.FBO, "prepaid credits should be available before invoicing")

		clock.FreezeTime(servicePeriod.To.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]

		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		s.Len(invoice.Lines.OrEmpty(), 1)

		line := invoice.Lines.OrEmpty()[0]
		lineID = line.GetLineID()
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, line.Totals)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveFinalRealizationProcessing)
		currentRun, err := charge.GetCurrentRealizationRun()
		s.NoError(err)
		runID = currentRun.ID
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
		s.Equal(lineID.ID, lo.FromPtr(currentRun.LineID))
		s.Equal(invoice.ID, lo.FromPtr(currentRun.InvoiceID))
		s.Equal(servicePeriod.To, currentRun.ServicePeriodTo)
		s.Equal(alpacadecimal.NewFromInt(5), currentRun.CreditsAllocated.Sum())
		s.Nil(currentRun.InvoiceUsage)
		s.Nil(currentRun.Payment)
	})

	s.Run("when the charge is shrunk while the final run is backed by a mutable line", func() {
		// given:
		// - the current final realization run extends beyond the new charge end
		// when:
		// - the charge is shrunk to an earlier service-period end
		// then:
		// - the mutable standard line is soft-deleted, the run is marked deleted, and a replacement gathering line is created
		s.mustShrinkCharge(ctx, cust.GetID(), usageBasedChargeID, shrunkServicePeriodTo)

		fetchedInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand: billing.InvoiceExpands{
				billing.InvoiceExpandLines,
				billing.InvoiceExpandDeletedLines,
			},
		})
		s.NoError(err)

		standardInvoice, err := fetchedInvoice.AsStandardInvoice()
		s.NoError(err)
		deletedLine := standardInvoice.Lines.GetByID(lineID.ID)
		s.Require().NotNil(deletedLine)
		s.NotNil(deletedLine.DeletedAt)
		s.Zero(standardInvoice.Lines.NonDeletedLineCount())

		charge := s.mustGetUsageBasedChargeByIDWithExpands(usageBasedChargeID, meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDeletedRealizations,
		})
		run, err := charge.Realizations.GetByID(runID.ID)
		s.NoError(err)
		s.NotNil(run.DeletedAt)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusActive)

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		s.Equal(servicePeriod.From, activeLine.ServicePeriod.From)
		s.Equal(shrunkServicePeriodTo, activeLine.ServicePeriod.To)
		s.Equal(shrunkServicePeriodTo, activeLine.InvoiceAt)
	})

	s.Run("then the charge returns to active and only credit allocations are reversed", func() {
		// given:
		// - the deleted draft line had only credit allocations
		// when:
		// - the line-engine cleanup has completed
		// then:
		// - credits are returned to FBO, accrued is cleared, and the charge waits for the shrunk end
		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActive)
		s.Nil(charge.State.CurrentRealizationRunID)
		s.Require().NotNil(charge.State.AdvanceAfter)
		s.True(charge.State.AdvanceAfter.Equal(shrunkServicePeriodTo), "advance after should match the shrunk service-period end")
		s.True(charge.Intent.ServicePeriod.To.Equal(shrunkServicePeriodTo), "service-period end should match the shrink")

		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, startLedger)
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen), "aggregate open receivable should stay empty")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized), "aggregate authorized receivable should stay empty")
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceShrinkPatchDuringAwaitingPaymentSettlementCreatesReplacementFinalRun() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-shrink-immutable")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	shrunkServicePeriodTo := datetime.MustParseTimeInLocation(t, "2026-01-20T00:00:00Z", time.UTC).AsTime()
	shrunkInvoiceAt := datetime.MustParseTimeInLocation(t, "2026-01-22T00:00:00Z", time.UTC).AsTime()
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
		lineID             billing.LineID
		runID              usagebased.RealizationRunID
		immutableLedger    LedgerSnapshot
		replacementInvoice billing.StandardInvoice
		replacementLineID  billing.LineID
	)
	ledgerSnapshotInput := LedgerSnapshotInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Currency:  USD,
		CostBasis: mo.Some(&zeroCostBasis),
	}

	s.Run("given a credit-then-invoice usage charge with an immutable invoice", func() {
		// given:
		// - a ledger-backed customer has enough credits to cover both the old immutable invoice and the replacement invoice
		// - usage is visible before the future shrink boundary
		// when:
		// - the standard invoice is collected and approved
		// then:
		// - the invoice line is immutable and its credit/invoice-usage ledger bookings exist
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(10),
			At:        setupAt,
			CostBasis: zeroCostBasis,
		})

		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-shrink-immutable",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-shrink-immutable",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		clock.FreezeTime(servicePeriod.To.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]

		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		s.Len(invoice.Lines.OrEmpty(), 1)

		line := invoice.Lines.OrEmpty()[0]
		lineID = line.GetLineID()
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, line.Totals)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		s.True(invoice.StatusDetails.Immutable)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveAwaitingPaymentSettlement)
		s.Len(charge.Realizations, 1)
		currentRun := charge.Realizations[0]
		runID = currentRun.ID
		s.Equal(lineID.ID, lo.FromPtr(currentRun.LineID))
		s.Equal(invoice.ID, lo.FromPtr(currentRun.InvoiceID))
		s.Equal(alpacadecimal.NewFromInt(5), currentRun.CreditsAllocated.Sum())
		s.NotNil(currentRun.InvoiceUsage)
		s.Nil(currentRun.Payment)

		immutableLedger = s.CreateLedgerSnapshot(ledgerSnapshotInput)
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), immutableLedger.FBO, "half of the credits should remain available for the replacement invoice")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), immutableLedger.Accrued, "old immutable invoice should keep accrued credit booking")
	})

	s.Run("when the charge shrink patch targets the immutable invoice period", func() {
		// given:
		// - the invoice line is immutable and the charge is awaiting payment settlement
		// when:
		// - the charge is shrunk to a date before the immutable run end
		// then:
		// - the immutable invoice receives the prorating warning and a replacement gathering line is created
		err := s.shrinkCharge(ctx, cust.GetID(), usageBasedChargeID, shrunkServicePeriodTo, shrunkInvoiceAt)
		s.NoError(err)

		fetchedInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand: billing.InvoiceExpands{
				billing.InvoiceExpandLines,
			},
		})
		s.NoError(err)

		standardInvoice, err := fetchedInvoice.AsStandardInvoice()
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, standardInvoice.Status)
		line := standardInvoice.Lines.GetByID(lineID.ID)
		s.Require().NotNil(line)
		s.Nil(line.DeletedAt)
		s.Equal(1, standardInvoice.Lines.NonDeletedLineCount())

		s.Require().Len(standardInvoice.ValidationIssues, 1)
		issue := standardInvoice.ValidationIssues[0]
		s.Equal(billing.ValidationIssueSeverityWarning, issue.Severity)
		s.Equal(billing.ImmutableInvoiceHandlingNotSupportedErrorCode, issue.Code)
		s.Equal(billing.ComponentName("charges.invoiceupdater"), issue.Component)
		s.Equal("line should be deleted, but the invoice is immutable", issue.Message)
		s.Equal("lines/"+lineID.ID, issue.Path)

		charge := s.mustGetUsageBasedChargeByIDWithExpands(usageBasedChargeID, meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDeletedRealizations,
		})
		run, err := charge.Realizations.GetByID(runID.ID)
		s.NoError(err)
		s.Nil(run.DeletedAt)
		s.Equal(usagebased.RealizationRunTypeInvalidDueToUnsupportedCreditNote, run.Type)
		s.Equal(usagebased.RealizationRunTypeFinalRealization, run.InitialType)
		s.True(run.IsVoidedBillingHistory())

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusActive)

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		s.Equal(servicePeriod.From, activeLine.ServicePeriod.From)
		s.Equal(shrunkServicePeriodTo, activeLine.ServicePeriod.To)
		s.Equal(shrunkInvoiceAt, activeLine.InvoiceAt)
	})

	s.Run("then immutable invoice shrink does not immediately reverse ledger bookings", func() {
		// given:
		// - the shrink request only produced an immutable-invoice warning for the old invoice
		// when:
		// - the ledger is inspected after the shrink patch
		// then:
		// - the already-issued invoice credit and accrual bookings remain unchanged
		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusActive)
		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, immutableLedger)
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen), "aggregate open receivable should stay empty")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized), "aggregate authorized receivable should stay empty")
	})

	s.Run("then billing creates a new final realization run for the shrunk period", func() {
		// given:
		// - the replacement gathering line is due at the invoice-at from the shrink patch
		// when:
		// - billing invoices and collects the replacement gathering line
		// then:
		// - a new final run is started for the shrunk service-period end
		clock.FreezeTime(shrunkInvoiceAt)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(shrunkInvoiceAt),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		replacementInvoice = invoices[0]
		s.NotEqual(invoice.ID, replacementInvoice.ID)
		s.Len(replacementInvoice.Lines.OrEmpty(), 1)

		replacementLine := replacementInvoice.Lines.OrEmpty()[0]
		replacementLineID = replacementLine.GetLineID()
		s.Equal(servicePeriod.From, replacementLine.Period.From)
		s.Equal(shrunkServicePeriodTo, replacementLine.Period.To)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveFinalRealizationWaitingForCollection)
		currentRun, err := charge.GetCurrentRealizationRun()
		s.NoError(err)
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
		s.Equal(shrunkServicePeriodTo, currentRun.ServicePeriodTo)
		s.Equal(replacementLineID.ID, lo.FromPtr(currentRun.LineID))
		s.Equal(replacementInvoice.ID, lo.FromPtr(currentRun.InvoiceID))
		s.NotEqual(runID.ID, currentRun.ID.ID)
	})

	s.Run("then collecting the replacement invoice rates the shrunk period", func() {
		// given:
		// - the replacement final run is waiting for collection
		// when:
		// - the replacement invoice reaches collection
		// then:
		// - usage before the shrunk end is rated on the new invoice
		clock.FreezeTime(replacementInvoice.DefaultCollectionAtForStandardInvoice())
		collectedInvoice, err := s.BillingService.AdvanceInvoice(ctx, replacementInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, collectedInvoice.Status)
		s.Len(collectedInvoice.Lines.OrEmpty(), 1)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, collectedInvoice.Lines.OrEmpty()[0].Totals)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveFinalRealizationProcessing)
		currentRun, err := charge.GetCurrentRealizationRun()
		s.NoError(err)
		s.Equal(shrunkServicePeriodTo, currentRun.ServicePeriodTo)
		s.Equal(alpacadecimal.NewFromInt(5), currentRun.MeteredQuantity)
		s.Equal(alpacadecimal.NewFromInt(5), currentRun.CreditsAllocated.Sum())
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceExtendPatchDuringFinalRunCollectionKeepsAdvanceNoop() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-extend-final-collection")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	collectionInterval := datetime.MustParseDuration(t, "P2D")
	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(collectionInterval),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	extendAt := datetime.MustParseTimeInLocation(t, "2026-02-02T00:00:00Z", time.UTC).AsTime()
	extendedServicePeriodTo := datetime.MustParseTimeInLocation(t, "2026-03-01T00:00:00Z", time.UTC).AsTime()
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
		lineID             billing.LineID
		deletedRunID       usagebased.RealizationRunID
	)

	s.Run("given a final run is ongoing inside the invoice collection interval", func() {
		// given:
		// - a ledger-backed customer has enough credits for original and tail usage
		// - usage is visible in both the original service period and the extension tail
		// when:
		// - billing creates the final standard invoice and enters the collection interval
		// then:
		// - the charge has an ongoing final realization run driven by the invoice lifecycle
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(8),
			At:        setupAt,
			CostBasis: zeroCostBasis,
		})

		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			3,
			datetime.MustParseTimeInLocation(t, "2026-02-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-extend-final-collection",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-extend-final-collection",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		clock.FreezeTime(servicePeriod.To)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]
		s.Len(invoice.Lines.OrEmpty(), 1)
		lineID = invoice.Lines.OrEmpty()[0].GetLineID()

		clock.FreezeTime(extendAt)
		s.True(clock.Now().After(servicePeriod.To), "test must run after original service-period end")
		s.True(clock.Now().Before(invoice.DefaultCollectionAtForStandardInvoice()), "test must run inside the invoice collection interval")

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveFinalRealizationWaitingForCollection)
		currentRun, err := charge.GetCurrentRealizationRun()
		s.NoError(err)
		deletedRunID = currentRun.ID
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
		s.Equal(lineID.ID, lo.FromPtr(currentRun.LineID))
		s.Equal(invoice.ID, lo.FromPtr(currentRun.InvoiceID))
	})

	s.Run("when the charge is extended and charge advancement runs before the extended end", func() {
		// given:
		// - the original final realization is still controlled by the mutable invoice
		// when:
		// - the charge is extended and the charge advancer runs inside the old collection interval
		// then:
		// - the original line/run cleanup happens, but charge advancement itself is a no-op
		s.mustExtendCharge(ctx, cust.GetID(), usageBasedChargeID, extendedServicePeriodTo)

		advancedCharges, err := s.Charges.AdvanceCharges(ctx, charges.AdvanceChargesInput{
			Customer: cust.GetID(),
		})
		s.NoError(err)
		s.Empty(advancedCharges)

		fetchedInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand: billing.InvoiceExpands{
				billing.InvoiceExpandLines,
				billing.InvoiceExpandDeletedLines,
			},
		})
		s.NoError(err)

		standardInvoice, err := fetchedInvoice.AsStandardInvoice()
		s.NoError(err)
		deletedLine := standardInvoice.Lines.GetByID(lineID.ID)
		s.Require().NotNil(deletedLine)
		s.NotNil(deletedLine.DeletedAt)
		s.Zero(standardInvoice.Lines.NonDeletedLineCount())

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusActive)

		charge := s.mustGetUsageBasedChargeByIDWithExpands(usageBasedChargeID, meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDeletedRealizations,
		})
		deletedRun, err := charge.Realizations.GetByID(deletedRunID.ID)
		s.NoError(err)
		s.NotNil(deletedRun.DeletedAt)
		s.Nil(charge.State.CurrentRealizationRunID)
		s.Require().NotNil(charge.State.AdvanceAfter)
		s.True(charge.State.AdvanceAfter.Equal(extendedServicePeriodTo), "advance after should match the extended service-period end")

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		s.Equal(servicePeriod.From, activeLine.ServicePeriod.From)
		s.Equal(extendedServicePeriodTo, activeLine.ServicePeriod.To)
		s.Equal(extendedServicePeriodTo, activeLine.InvoiceAt)
	})

	s.Run("then billing can create and collect one replacement final run for the extended period", func() {
		// given:
		// - charge advancement did not create a run while the replacement line was not due
		// when:
		// - billing invoices the replacement gathering line and reaches collection completion
		// then:
		// - one new final realization run is created for the full extended service period
		clock.FreezeTime(extendedServicePeriodTo)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(extendedServicePeriodTo),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		replacementInvoice := invoices[0]
		s.Len(replacementInvoice.Lines.OrEmpty(), 1)
		s.Equal(servicePeriod.From, replacementInvoice.Lines.OrEmpty()[0].Period.From)
		s.Equal(extendedServicePeriodTo, replacementInvoice.Lines.OrEmpty()[0].Period.To)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusActiveFinalRealizationWaitingForCollection)

		charge := s.mustGetUsageBasedChargeByIDWithExpands(usageBasedChargeID, meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDeletedRealizations,
		})
		currentRun, err := charge.GetCurrentRealizationRun()
		s.NoError(err)
		s.Equal(usagebased.RealizationRunTypeFinalRealization, currentRun.Type)
		s.Nil(currentRun.DeletedAt)
		s.Equal(extendedServicePeriodTo, currentRun.ServicePeriodTo)

		clock.FreezeTime(replacementInvoice.DefaultCollectionAtForStandardInvoice())
		replacementInvoice, err = s.BillingService.AdvanceInvoice(ctx, replacementInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, replacementInvoice.Status)
		s.Len(replacementInvoice.Lines.OrEmpty(), 1)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       8,
			CreditsTotal: 8,
			Total:        0,
		}, replacementInvoice.Lines.OrEmpty()[0].Totals)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusActiveFinalRealizationProcessing)

		charge = s.mustGetUsageBasedChargeByIDWithExpands(usageBasedChargeID, meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDeletedRealizations,
		})
		currentRun, err = charge.GetCurrentRealizationRun()
		s.NoError(err)
		s.Equal(alpacadecimal.NewFromInt(8), currentRun.MeteredQuantity)
		s.Equal(alpacadecimal.NewFromInt(8), currentRun.CreditsAllocated.Sum())

		nonDeletedFinalRuns := lo.Filter(charge.Realizations, func(run usagebased.RealizationRun, _ int) bool {
			return run.DeletedAt == nil && run.Type == usagebased.RealizationRunTypeFinalRealization
		})
		s.Len(nonDeletedFinalRuns, 1)
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceExtendPatchDuringAwaitingPaymentSettlementReclassifiesFinalRunAndKeepsLedgerBookings() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-extend-immutable")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	extendedServicePeriodTo := datetime.MustParseTimeInLocation(t, "2026-03-01T00:00:00Z", time.UTC).AsTime()
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
		lineID             billing.LineID
		runID              usagebased.RealizationRunID
		immutableLedger    LedgerSnapshot
	)
	ledgerSnapshotInput := LedgerSnapshotInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Currency:  USD,
		CostBasis: mo.Some(&zeroCostBasis),
	}

	s.Run("given a final standard invoice line awaiting payment settlement", func() {
		// given:
		// - a ledger-backed customer has enough credits to cover usage
		// - usage is visible inside the original service period
		// when:
		// - the standard invoice is collected and approved
		// then:
		// - the final run has invoice usage booked and the charge is waiting for payment settlement
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(5),
			At:        setupAt,
			CostBasis: zeroCostBasis,
		})

		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-extend-immutable",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-extend-immutable",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		clock.FreezeTime(servicePeriod.To.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(servicePeriod.To),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]

		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		s.Len(invoice.Lines.OrEmpty(), 1)

		line := invoice.Lines.OrEmpty()[0]
		lineID = line.GetLineID()
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, line.Totals)

		invoice, err = s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
		s.True(invoice.StatusDetails.Immutable)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveAwaitingPaymentSettlement)
		s.Len(charge.Realizations, 1)
		run := charge.Realizations[0]
		runID = run.ID
		s.Equal(usagebased.RealizationRunTypeFinalRealization, run.Type)
		s.Equal(lineID.ID, lo.FromPtr(run.LineID))
		s.Equal(invoice.ID, lo.FromPtr(run.InvoiceID))
		s.Equal(alpacadecimal.NewFromInt(5), run.CreditsAllocated.Sum())
		s.NotNil(run.InvoiceUsage)
		s.Nil(run.Payment)

		immutableLedger = s.CreateLedgerSnapshot(ledgerSnapshotInput)
		s.AssertDecimalEqual(alpacadecimal.Zero, immutableLedger.FBO, "immutable invoice credit allocation should keep FBO consumed")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), immutableLedger.Accrued, "immutable invoice should keep accrued credit booking")
	})

	s.Run("when the charge is extended during active.awaiting_payment_settlement", func() {
		// given:
		// - the original final run is no longer current and the invoice lifecycle callbacks have completed
		// when:
		// - the charge is extended to a later service-period end
		// then:
		// - the standard invoice line is left untouched and the final run is reclassified as partial
		s.mustExtendCharge(ctx, cust.GetID(), usageBasedChargeID, extendedServicePeriodTo)

		fetchedInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoice.GetInvoiceID(),
			Expand: billing.InvoiceExpands{
				billing.InvoiceExpandLines,
			},
		})
		s.NoError(err)

		standardInvoice, err := fetchedInvoice.AsStandardInvoice()
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, standardInvoice.Status)
		line := standardInvoice.Lines.GetByID(lineID.ID)
		s.Require().NotNil(line)
		s.Nil(line.DeletedAt)
		s.Equal(1, standardInvoice.Lines.NonDeletedLineCount())

		charge := s.mustGetUsageBasedChargeByIDWithExpands(usageBasedChargeID, meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDeletedRealizations,
		})
		run, err := charge.Realizations.GetByID(runID.ID)
		s.NoError(err)
		s.Nil(run.DeletedAt)
		s.Equal(usagebased.RealizationRunTypePartialInvoice, run.Type)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusActive)

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		s.Equal(servicePeriod.To, activeLine.ServicePeriod.From)
		s.Equal(extendedServicePeriodTo, activeLine.ServicePeriod.To)
		s.Equal(extendedServicePeriodTo, activeLine.InvoiceAt)
	})

	s.Run("then immutable invoice extension keeps ledger bookings and waits for the extended tail", func() {
		// given:
		// - the immutable invoice line was preserved
		// when:
		// - the charge and ledger are inspected after the extend patch
		// then:
		// - the old invoice ledger bookings remain unchanged and the charge waits for the new end
		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActive)
		s.Nil(charge.State.CurrentRealizationRunID)
		s.Require().NotNil(charge.State.AdvanceAfter)
		s.True(charge.State.AdvanceAfter.Equal(extendedServicePeriodTo), "advance after should match the extended service-period end")
		s.True(charge.Intent.ServicePeriod.To.Equal(extendedServicePeriodTo), "service-period end should match the extension")
		s.Len(charge.Realizations, 1)
		s.Equal(usagebased.RealizationRunTypePartialInvoice, charge.Realizations[0].Type)

		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, immutableLedger)
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen), "aggregate open receivable should stay empty")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized), "aggregate authorized receivable should stay empty")
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceShrinkExtendShrinkPreservesImmutableRunsAndLedger() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-shrink-extend-shrink")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-04-01T00:00:00Z", time.UTC).AsTime(),
	}
	firstShrinkTo := datetime.MustParseTimeInLocation(t, "2026-03-01T00:00:00Z", time.UTC).AsTime()
	extendedServicePeriodTo := datetime.MustParseTimeInLocation(t, "2026-05-01T00:00:00Z", time.UTC).AsTime()
	secondShrinkTo := datetime.MustParseTimeInLocation(t, "2026-04-01T00:00:00Z", time.UTC).AsTime()
	secondShrinkInvoiceAt := datetime.MustParseTimeInLocation(t, "2026-04-02T00:00:00Z", time.UTC).AsTime()
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID  meta.ChargeID
		firstInvoice        billing.StandardInvoice
		firstLineID         billing.LineID
		firstRunID          usagebased.RealizationRunID
		secondInvoice       billing.StandardInvoice
		secondLineID        billing.LineID
		secondRunID         usagebased.RealizationRunID
		thirdInvoice        billing.StandardInvoice
		thirdLineID         billing.LineID
		firstLedgerSnapshot LedgerSnapshot
		secondLedger        LedgerSnapshot
		secondShrinkLedger  LedgerSnapshot
		thirdLedger         LedgerSnapshot
	)
	ledgerSnapshotInput := LedgerSnapshotInput{
		Namespace: ns,
		Customer:  cust.GetID(),
		Currency:  USD,
		CostBasis: mo.Some(&zeroCostBasis),
	}

	s.Run("given a credit-then-invoice charge that is shrunk before the first invoice", func() {
		// given:
		// - a ledger-backed customer has enough credits for the whole scenario
		// - usage exists before the first shrink, after the extend, and inside the second shrink replacement period
		// when:
		// - the charge is created and shrunk before billing invoices it
		// then:
		// - the active gathering line covers the first shrunk period
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(15),
			At:        setupAt,
			CostBasis: zeroCostBasis,
		})

		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			3,
			datetime.MustParseTimeInLocation(t, "2026-03-15T00:00:00Z", time.UTC).AsTime(),
		)
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			4,
			datetime.MustParseTimeInLocation(t, "2026-04-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-shrink-extend-shrink",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-shrink-extend-shrink",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		s.mustShrinkCharge(ctx, cust.GetID(), usageBasedChargeID, firstShrinkTo)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusCreated)
		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		s.Equal(servicePeriod.From, activeLine.ServicePeriod.From)
		s.Equal(firstShrinkTo, activeLine.ServicePeriod.To)
		s.Equal(firstShrinkTo, activeLine.InvoiceAt)

		startLedger := s.CreateLedgerSnapshot(ledgerSnapshotInput)
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(15), startLedger.FBO, "all credits should be available before invoicing")
		s.AssertDecimalEqual(alpacadecimal.Zero, startLedger.Accrued, "no usage should be accrued before invoicing")
	})

	s.Run("when billing finalizes the first shrunk period", func() {
		// given:
		// - the charge was shrunk to the first invoice boundary
		// when:
		// - billing invoices, collects, and approves that period
		// then:
		// - immutable final realization invoice #1 is created and ledger bookings are posted once
		clock.FreezeTime(firstShrinkTo.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(firstShrinkTo),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		firstInvoice = invoices[0]
		s.Len(firstInvoice.Lines.OrEmpty(), 1)

		clock.FreezeTime(firstInvoice.DefaultCollectionAtForStandardInvoice())
		firstInvoice, err = s.BillingService.AdvanceInvoice(ctx, firstInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, firstInvoice.Status)
		s.Len(firstInvoice.Lines.OrEmpty(), 1)

		firstLine := firstInvoice.Lines.OrEmpty()[0]
		firstLineID = firstLine.GetLineID()
		s.Equal(servicePeriod.From, firstLine.Period.From)
		s.Equal(firstShrinkTo, firstLine.Period.To)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       5,
			CreditsTotal: 5,
			Total:        0,
		}, firstLine.Totals)

		firstInvoice, err = s.BillingService.ApproveInvoice(ctx, firstInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, firstInvoice.Status)
		s.True(firstInvoice.StatusDetails.Immutable)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveAwaitingPaymentSettlement)
		s.Len(charge.Realizations, 1)
		firstRun := charge.Realizations[0]
		firstRunID = firstRun.ID
		s.Equal(usagebased.RealizationRunTypeFinalRealization, firstRun.Type)
		s.Equal(firstShrinkTo, firstRun.ServicePeriodTo)
		s.Equal(firstLineID.ID, lo.FromPtr(firstRun.LineID))
		s.Equal(firstInvoice.ID, lo.FromPtr(firstRun.InvoiceID))
		s.Nil(firstRun.DeletedAt)
		s.NotNil(firstRun.InvoiceUsage)
		s.RequireAllRunsNonDeleted(charge.Realizations)

		firstLedgerSnapshot = s.CreateLedgerSnapshot(ledgerSnapshotInput)
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(10), firstLedgerSnapshot.FBO, "unused credits should remain after the first immutable invoice")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(5), firstLedgerSnapshot.Accrued, "first immutable invoice should keep accrued credit booking")
		s.AssertDecimalEqual(alpacadecimal.Zero, firstLedgerSnapshot.OpenReceivable, "fully credited invoice should not create open receivable")
		s.AssertDecimalEqual(alpacadecimal.Zero, firstLedgerSnapshot.AuthorizedReceivable, "fully credited invoice should not create authorized receivable")
	})

	s.Run("when the charge is extended after the first immutable final invoice", func() {
		// given:
		// - immutable final realization invoice #1 exists
		// when:
		// - the charge is extended to a later service-period end
		// then:
		// - invoice #1 remains immutable history and the open tail starts at the first final run end
		s.mustExtendCharge(ctx, cust.GetID(), usageBasedChargeID, extendedServicePeriodTo)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActive)
		s.Len(charge.Realizations, 1)
		s.Equal(firstRunID.ID, charge.Realizations[0].ID.ID)
		s.Nil(charge.Realizations[0].DeletedAt)
		s.Equal(usagebased.RealizationRunTypePartialInvoice, charge.Realizations[0].Type)
		s.RequireAllRunsNonDeleted(charge.Realizations)

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		s.Equal(firstShrinkTo, activeLine.ServicePeriod.From)
		s.Equal(extendedServicePeriodTo, activeLine.ServicePeriod.To)
		s.Equal(extendedServicePeriodTo, activeLine.InvoiceAt)

		s.AssertLedgerSnapshotUnchanged(ledgerSnapshotInput, firstLedgerSnapshot)
	})

	s.Run("when billing finalizes the extended tail", func() {
		// given:
		// - the extended tail gathering line is due
		// when:
		// - billing invoices, collects, and approves the tail
		// then:
		// - immutable final realization invoice #2 is created and ledger bookings include invoice #1 and #2
		clock.FreezeTime(extendedServicePeriodTo.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(extendedServicePeriodTo),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		secondInvoice = invoices[0]
		s.NotEqual(firstInvoice.ID, secondInvoice.ID)
		s.Len(secondInvoice.Lines.OrEmpty(), 1)

		clock.FreezeTime(secondInvoice.DefaultCollectionAtForStandardInvoice())
		secondInvoice, err = s.BillingService.AdvanceInvoice(ctx, secondInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, secondInvoice.Status)
		s.Len(secondInvoice.Lines.OrEmpty(), 1)

		secondLine := secondInvoice.Lines.OrEmpty()[0]
		secondLineID = secondLine.GetLineID()
		s.Equal(firstShrinkTo, secondLine.Period.From)
		s.Equal(extendedServicePeriodTo, secondLine.Period.To)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       7,
			CreditsTotal: 7,
			Total:        0,
		}, secondLine.Totals)

		secondInvoice, err = s.BillingService.ApproveInvoice(ctx, secondInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, secondInvoice.Status)
		s.True(secondInvoice.StatusDetails.Immutable)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveAwaitingPaymentSettlement)
		s.Len(charge.Realizations, 2)
		secondRun := lo.MaxBy(charge.Realizations, func(run usagebased.RealizationRun, latest usagebased.RealizationRun) bool {
			return run.ServicePeriodTo.After(latest.ServicePeriodTo)
		})
		secondRunID = secondRun.ID
		s.Equal(usagebased.RealizationRunTypeFinalRealization, secondRun.Type)
		s.Equal(extendedServicePeriodTo, secondRun.ServicePeriodTo)
		s.Equal(secondLineID.ID, lo.FromPtr(secondRun.LineID))
		s.Equal(secondInvoice.ID, lo.FromPtr(secondRun.InvoiceID))
		s.Nil(secondRun.DeletedAt)
		s.NotNil(secondRun.InvoiceUsage)
		s.RequireAllRunsNonDeleted(charge.Realizations)

		secondLedger = s.CreateLedgerSnapshot(ledgerSnapshotInput)
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(3), secondLedger.FBO, "remaining credits should exclude immutable invoices #1 and #2")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(12), secondLedger.Accrued, "immutable invoices #1 and #2 should keep accrued credit bookings")
		s.AssertDecimalEqual(alpacadecimal.Zero, secondLedger.OpenReceivable, "fully credited invoices should not create open receivable")
		s.AssertDecimalEqual(alpacadecimal.Zero, secondLedger.AuthorizedReceivable, "fully credited invoices should not create authorized receivable")
	})

	s.Run("when the charge is shrunk across immutable final realization invoice #2", func() {
		// given:
		// - immutable invoice #2 overlaps the new shrink boundary
		// when:
		// - the charge is shrunk back into invoice #2's period
		// then:
		// - invoice #2 receives a warning, invoice and ledger history stay unchanged, and a replacement gathering line is created
		err := s.shrinkCharge(ctx, cust.GetID(), usageBasedChargeID, secondShrinkTo, secondShrinkInvoiceAt)
		s.NoError(err)

		charge := s.mustGetUsageBasedChargeByIDWithExpands(usageBasedChargeID, meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDeletedRealizations,
		})
		s.Equal(usagebased.StatusActive, charge.Status)
		s.Equal(secondShrinkTo, charge.Intent.ServicePeriod.To)
		s.Len(charge.Realizations, 2)

		firstRun, err := charge.Realizations.GetByID(firstRunID.ID)
		s.NoError(err)
		s.Nil(firstRun.DeletedAt)
		s.Equal(firstShrinkTo, firstRun.ServicePeriodTo)

		secondRun, err := charge.Realizations.GetByID(secondRunID.ID)
		s.NoError(err)
		s.Nil(secondRun.DeletedAt)
		s.Equal(extendedServicePeriodTo, secondRun.ServicePeriodTo)
		s.RequireAllRunsNonDeleted(charge.Realizations)

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		s.Equal(firstShrinkTo, activeLine.ServicePeriod.From)
		s.Equal(secondShrinkTo, activeLine.ServicePeriod.To)
		s.Equal(secondShrinkInvoiceAt, activeLine.InvoiceAt)

		fetchedSecondInvoice, err := s.BillingService.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
			Invoice: secondInvoice.GetInvoiceID(),
			Expand: billing.InvoiceExpands{
				billing.InvoiceExpandLines,
			},
		})
		s.NoError(err)

		standardSecondInvoice, err := fetchedSecondInvoice.AsStandardInvoice()
		s.NoError(err)
		secondLine := standardSecondInvoice.Lines.GetByID(secondLineID.ID)
		s.Require().NotNil(secondLine)
		s.Nil(secondLine.DeletedAt)
		s.Equal(1, standardSecondInvoice.Lines.NonDeletedLineCount())
		s.Require().Len(standardSecondInvoice.ValidationIssues, 1)
		issue := standardSecondInvoice.ValidationIssues[0]
		s.Equal(billing.ValidationIssueSeverityWarning, issue.Severity)
		s.Equal(billing.ImmutableInvoiceHandlingNotSupportedErrorCode, issue.Code)
		s.Equal(billing.ComponentName("charges.invoiceupdater"), issue.Component)
		s.Equal("line should be deleted, but the invoice is immutable", issue.Message)
		s.Equal("lines/"+secondLineID.ID, issue.Path)

		secondShrinkLedger = s.CreateLedgerSnapshot(ledgerSnapshotInput)
		s.AssertLedgerSnapshotEqual(secondLedger, secondShrinkLedger)
	})

	s.Run("when billing finalizes the replacement period after the second shrink", func() {
		// given:
		// - the replacement gathering line covers the shrunk period inside immutable invoice #2
		// when:
		// - billing invoices, collects, and approves the replacement
		// then:
		// - immutable final realization invoice #3 is created and ledger bookings include all three invoices
		clock.FreezeTime(secondShrinkInvoiceAt)
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(secondShrinkInvoiceAt),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		thirdInvoice = invoices[0]
		s.NotEqual(firstInvoice.ID, thirdInvoice.ID)
		s.NotEqual(secondInvoice.ID, thirdInvoice.ID)
		s.Len(thirdInvoice.Lines.OrEmpty(), 1)

		clock.FreezeTime(thirdInvoice.DefaultCollectionAtForStandardInvoice())
		thirdInvoice, err = s.BillingService.AdvanceInvoice(ctx, thirdInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, thirdInvoice.Status)
		s.Len(thirdInvoice.Lines.OrEmpty(), 1)

		thirdLine := thirdInvoice.Lines.OrEmpty()[0]
		thirdLineID = thirdLine.GetLineID()
		s.Equal(firstShrinkTo, thirdLine.Period.From)
		s.Equal(secondShrinkTo, thirdLine.Period.To)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       3,
			CreditsTotal: 3,
			Total:        0,
		}, thirdLine.Totals)

		thirdInvoice, err = s.BillingService.ApproveInvoice(ctx, thirdInvoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, thirdInvoice.Status)
		s.True(thirdInvoice.StatusDetails.Immutable)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveAwaitingPaymentSettlement)
		s.Len(charge.Realizations, 3)
		thirdRun, err := charge.Realizations.GetByLineID(thirdLineID.ID)
		s.NoError(err)
		s.Equal(usagebased.RealizationRunTypeFinalRealization, thirdRun.Type)
		s.Equal(secondShrinkTo, thirdRun.ServicePeriodTo)
		s.Equal(thirdLineID.ID, lo.FromPtr(thirdRun.LineID))
		s.Equal(thirdInvoice.ID, lo.FromPtr(thirdRun.InvoiceID))
		s.Nil(thirdRun.DeletedAt)
		s.NotNil(thirdRun.InvoiceUsage)
		s.RequireAllRunsNonDeleted(charge.Realizations)

		thirdLedger = s.CreateLedgerSnapshot(ledgerSnapshotInput)
		s.AssertDecimalEqual(alpacadecimal.Zero, thirdLedger.FBO, "all credits should be consumed by the three immutable invoices")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(15), thirdLedger.Accrued, "all three immutable invoices should keep accrued credit bookings")
		s.AssertDecimalEqual(alpacadecimal.Zero, thirdLedger.OpenReceivable, "fully credited invoices should not create open receivable")
		s.AssertDecimalEqual(alpacadecimal.Zero, thirdLedger.AuthorizedReceivable, "fully credited invoices should not create authorized receivable")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen), "aggregate open receivable should stay empty")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized), "aggregate authorized receivable should stay empty")
	})
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedCreditThenInvoiceExtendPatchFinalizesExtendedPeriodWithTailUsage() {
	t := s.T()
	ctx := t.Context()
	ns := s.GetUniqueNamespace("charges-credits-usagebased-credit-then-invoice-extend-tail")

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(t, "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	setupAt := datetime.MustParseTimeInLocation(t, "2025-12-01T00:00:00Z", time.UTC).AsTime()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime(),
	}
	extendedServicePeriodTo := datetime.MustParseTimeInLocation(t, "2026-03-01T00:00:00Z", time.UTC).AsTime()
	zeroCostBasis := alpacadecimal.Zero

	clock.FreezeTime(setupAt)
	defer clock.UnFreeze()

	var (
		usageBasedChargeID meta.ChargeID
		invoice            billing.StandardInvoice
	)

	s.Run("given a charge extended before collection with usage in the original period and tail", func() {
		// given:
		// - a ledger-backed customer has enough promotional credits to cover all usage
		// - usage exists in both the original period and the extended tail
		// when:
		// - the charge is extended before billing collection
		// then:
		// - the pending gathering line covers the full extended service period
		s.CreatePromotionalCreditFunding(ctx, CreatePromotionalCreditFundingInput{
			Namespace: ns,
			Customer:  cust.GetID(),
			Amount:    alpacadecimal.NewFromInt(8),
			At:        setupAt,
			CostBasis: zeroCostBasis,
		})

		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			5,
			datetime.MustParseTimeInLocation(t, "2026-01-15T00:00:00Z", time.UTC).AsTime(),
		)
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotal.Feature.Key,
			3,
			datetime.MustParseTimeInLocation(t, "2026-02-15T00:00:00Z", time.UTC).AsTime(),
		)

		res, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: ns,
			Intents: charges.ChargeIntents{
				s.CreateMockChargeIntent(CreateMockChargeIntentInput{
					Customer:       cust.GetID(),
					Currency:       USD,
					ServicePeriod:  servicePeriod,
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(1),
					}),
					Name:              "usage-based-credit-then-invoice-extend-tail",
					ManagedBy:         billing.SubscriptionManagedLine,
					UniqueReferenceID: "usage-based-credit-then-invoice-extend-tail",
					FeatureKey:        apiRequestsTotal.Feature.Key,
				}),
			},
		})
		s.NoError(err)
		s.Len(res, 1)

		usageBasedCharge, err := res[0].AsUsageBasedCharge()
		s.NoError(err)
		usageBasedChargeID = usageBasedCharge.GetChargeID()

		s.mustExtendCharge(ctx, cust.GetID(), usageBasedChargeID, extendedServicePeriodTo)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusCreated)

		activeLine := s.mustSingleActiveGatheringLineForCharge(ns, cust.ID, usageBasedChargeID.ID)
		s.Equal(servicePeriod.From, activeLine.ServicePeriod.From)
		s.Equal(extendedServicePeriodTo, activeLine.ServicePeriod.To)
		s.Equal(extendedServicePeriodTo, activeLine.InvoiceAt)
	})

	s.Run("when the extended period is collected into a standard invoice", func() {
		// given:
		// - the pending line covers the original period plus the extended tail
		// when:
		// - billing creates and collects the standard invoice at the extended end
		// then:
		// - the single standard line includes both original-period and tail usage exactly once
		clock.FreezeTime(extendedServicePeriodTo.Add(time.Second))
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: cust.GetID(),
			AsOf:     lo.ToPtr(extendedServicePeriodTo),
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice = invoices[0]

		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
		s.Len(invoice.Lines.OrEmpty(), 1)

		line := invoice.Lines.OrEmpty()[0]
		s.Equal(servicePeriod.From, line.Period.From)
		s.Equal(extendedServicePeriodTo, line.Period.To)
		s.RequireTotals(billingtest.ExpectedTotals{
			Amount:       8,
			CreditsTotal: 8,
			Total:        0,
		}, line.Totals)

		s.RequireChargeStatus(usageBasedChargeID, usagebased.StatusActiveFinalRealizationProcessing)
	})

	s.Run("then finalizing the extended period records one final run for all visible usage", func() {
		// given:
		// - the extended standard line is fully covered by credits
		// when:
		// - the invoice is approved and issued
		// then:
		// - the charge has one final run for the extended period and ledger balances reflect one booking
		invoice, err := s.BillingService.ApproveInvoice(ctx, invoice.GetInvoiceID())
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		charge := s.RequireUsageBasedChargeStatus(usageBasedChargeID, usagebased.StatusActiveAwaitingPaymentSettlement)
		s.Len(charge.Realizations, 1)

		run := charge.Realizations[0]
		s.Equal(usagebased.RealizationRunTypeFinalRealization, run.Type)
		s.Equal(extendedServicePeriodTo, run.ServicePeriodTo)
		s.Equal(alpacadecimal.NewFromInt(8), run.CreditsAllocated.Sum())
		s.NotNil(run.InvoiceUsage)
		s.Nil(run.Payment)

		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerFBOBalance(cust.GetID(), USD, mo.Some(&zeroCostBasis)), "all credits should be consumed once")
		s.AssertDecimalEqual(alpacadecimal.NewFromInt(8), s.MustCustomerAccruedBalance(cust.GetID(), USD, mo.Some(&zeroCostBasis)), "all credited usage should be accrued once")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusOpen), "fully credited invoice should not create open receivable")
		s.AssertDecimalEqual(alpacadecimal.Zero, s.MustCustomerReceivableBalance(cust.GetID(), USD, mo.None[*alpacadecimal.Decimal](), ledger.TransactionAuthorizationStatusAuthorized), "fully credited invoice should not create authorized receivable")
	})
}

func (s *CreditThenInvoiceTestSuite) mustGatheringLinesForCharge(namespace, customerID, chargeID string, includeDeletedLines bool) []billing.GatheringLine {
	s.T().Helper()

	expand := billing.GatheringInvoiceExpands{billing.GatheringInvoiceExpandLines}
	if includeDeletedLines {
		expand = append(expand, billing.GatheringInvoiceExpandDeletedLines)
	}

	gatheringInvoices, err := s.BillingService.ListGatheringInvoices(s.T().Context(), billing.ListGatheringInvoicesInput{
		Namespaces:     []string{namespace},
		Customers:      []string{customerID},
		Currencies:     []currencyx.Code{USD},
		IncludeDeleted: includeDeletedLines,
		Expand:         expand,
	})
	s.NoError(err)

	var lines []billing.GatheringLine
	for _, invoice := range gatheringInvoices.Items {
		for _, line := range invoice.Lines.OrEmpty() {
			if line.ChargeID == nil || *line.ChargeID != chargeID {
				continue
			}

			lines = append(lines, line)
		}
	}

	return lines
}

func (s *CreditThenInvoiceTestSuite) mustSingleActiveGatheringLineForCharge(namespace, customerID, chargeID string) billing.GatheringLine {
	s.T().Helper()

	activeLines := s.mustGatheringLinesForCharge(namespace, customerID, chargeID, false)
	s.Len(activeLines, 1)

	return activeLines[0]
}

func (s *CreditThenInvoiceTestSuite) RequireAllRunsNonDeleted(runs usagebased.RealizationRuns) {
	s.T().Helper()

	for _, run := range runs {
		s.Nil(run.DeletedAt, "run %s should not be deleted", run.ID.ID)
	}
}

func (s *CreditThenInvoiceTestSuite) mustExtendCharge(ctx context.Context, customerID customer.CustomerID, chargeID meta.ChargeID, servicePeriodTo time.Time) {
	s.T().Helper()

	s.mustExtendChargeWithInvoiceAt(ctx, customerID, chargeID, servicePeriodTo, servicePeriodTo)
}

func (s *CreditThenInvoiceTestSuite) mustExtendChargeWithInvoiceAt(ctx context.Context, customerID customer.CustomerID, chargeID meta.ChargeID, servicePeriodTo time.Time, invoiceAt time.Time) {
	s.T().Helper()

	patch, err := meta.NewPatchExtend(meta.NewPatchExtendInput{
		NewServicePeriodTo:     servicePeriodTo,
		NewFullServicePeriodTo: servicePeriodTo,
		NewBillingPeriodTo:     servicePeriodTo,
		NewInvoiceAt:           invoiceAt,
	})
	s.NoError(err)

	err = s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: customerID,
		PatchesByChargeID: map[string]charges.Patch{
			chargeID.ID: patch,
		},
	})
	s.NoError(err)
}

func (s *CreditThenInvoiceTestSuite) mustShrinkCharge(ctx context.Context, customerID customer.CustomerID, chargeID meta.ChargeID, servicePeriodTo time.Time) {
	s.T().Helper()

	s.NoError(s.shrinkCharge(ctx, customerID, chargeID, servicePeriodTo, servicePeriodTo))
}

func (s *CreditThenInvoiceTestSuite) shrinkCharge(ctx context.Context, customerID customer.CustomerID, chargeID meta.ChargeID, servicePeriodTo time.Time, invoiceAt time.Time) error {
	s.T().Helper()

	patch, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
		NewServicePeriodTo:     servicePeriodTo,
		NewFullServicePeriodTo: servicePeriodTo,
		NewBillingPeriodTo:     servicePeriodTo,
		NewInvoiceAt:           invoiceAt,
	})
	s.NoError(err)

	return s.Charges.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: customerID,
		PatchesByChargeID: map[string]charges.Patch{
			chargeID.ID: patch,
		},
	})
}

func (s *CreditThenInvoiceTestSuite) RequireChargeStatus(chargeID meta.ChargeID, status any) charges.Charge {
	s.T().Helper()

	charge := s.MustGetChargeByID(chargeID)

	var actualStatus string
	switch charge.Type() {
	case meta.ChargeTypeUsageBased:
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		s.NoError(err)
		actualStatus = string(usageBasedCharge.Status)
	case meta.ChargeTypeFlatFee:
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		actualStatus = string(flatFeeCharge.Status)
	case meta.ChargeTypeCreditPurchase:
		creditPurchaseCharge, err := charge.AsCreditPurchaseCharge()
		s.NoError(err)
		actualStatus = string(creditPurchaseCharge.Status)
	default:
		s.FailNowf("unsupported charge type", "charge type %s is not supported", charge.Type())
	}

	s.Equal(fmt.Sprint(status), actualStatus)

	return charge
}

func (s *CreditThenInvoiceTestSuite) RequireUsageBasedChargeStatus(chargeID meta.ChargeID, status usagebased.Status) usagebased.Charge {
	s.T().Helper()

	charge, err := s.RequireChargeStatus(chargeID, status).AsUsageBasedCharge()
	s.NoError(err)

	return charge
}

func (s *CreditThenInvoiceTestSuite) mustGetUsageBasedChargeByIDWithExpands(chargeID meta.ChargeID, expands meta.Expands) usagebased.Charge {
	s.T().Helper()

	charge, err := s.Charges.GetByID(s.T().Context(), charges.GetByIDInput{
		ChargeID: chargeID,
		Expands:  expands,
	})
	s.NoError(err)

	usageBasedCharge, err := charge.AsUsageBasedCharge()
	s.NoError(err)

	return usageBasedCharge
}
