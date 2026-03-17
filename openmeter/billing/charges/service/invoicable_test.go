package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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

func (s *InvoicableChargesTestSuite) TeardownTest() {
	s.BaseSuite.TeardownTest()
}

func (s *InvoicableChargesTestSuite) TestFlatFeePartialCreditRealizations() {
	ctx := context.Background()
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

	s.Run("create new upcoming charges", func() {
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
	s.Run("invoice the charge", func() {
		defer s.FlatFeeTestHandler.Reset()

		testTrnsGroupID := ulid.Make().String()
		creditRealizationCallbackInvocations := 0
		s.FlatFeeTestHandler.onAssignedToInvoice = func(ctx context.Context, input flatfee.OnAssignedToInvoiceInput) ([]creditrealization.CreateInput, error) {
			creditRealizationCallbackInvocations++

			return []creditrealization.CreateInput{
				{
					ServicePeriod: input.ServicePeriod,
					Amount:        input.PreTaxTotalAmount.Mul(alpacadecimal.NewFromFloat(0.3)), // 30% as credits
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: testTrnsGroupID,
					},
				},
			}, nil
		}

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

		s.Equal(1, creditRealizationCallbackInvocations)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		s.Equal(flatFeeChargeID.ID, updatedFlatFeeCharge.ID)

		// Validate the credit realizations
		// The charge should have $30 realized as credits
		s.Len(updatedFlatFeeCharge.State.CreditRealizations, 1)
		creditRealization := updatedFlatFeeCharge.State.CreditRealizations[0]
		s.Equal(testTrnsGroupID, creditRealization.LedgerTransaction.TransactionGroupID)
		s.Equal(servicePeriod.From, creditRealization.ServicePeriod.From)
		s.Equal(servicePeriod.To, creditRealization.ServicePeriod.To)
		s.Equal(float64(30), creditRealization.Amount.InexactFloat64())

		// Validate the standard invoice's contents
		// Invoice totals should be $70
		s.Equal(float64(70), invoice.Totals.Total.InexactFloat64())
		s.Equal(float64(30), invoice.Totals.CreditsTotal.InexactFloat64())

		// Validate the standard line's contents
		// Line totals should be $70
		s.Equal(float64(30), stdLine.Totals.CreditsTotal.InexactFloat64())
		s.Equal(float64(70), stdLine.Totals.Total.InexactFloat64())

		// The line should have a credit realization intent
		s.Len(stdLine.CreditsApplied, 1)
		creditRealizationIntent := stdLine.CreditsApplied[0]
		s.Equal(float64(30), creditRealizationIntent.Amount.InexactFloat64())
		s.Equal(creditRealization.ID, creditRealizationIntent.CreditRealizationID)

		// The line should have a single detailed line
		s.Len(stdLine.DetailedLines, 1)
		detailedLine := stdLine.DetailedLines[0]
		s.Equal(float64(70), detailedLine.Totals.Total.InexactFloat64())
		s.Equal(float64(30), detailedLine.Totals.CreditsTotal.InexactFloat64())

		// The detailed line should have a credit realization intent
		s.Len(detailedLine.CreditsApplied, 1)
		creditRealizationDetail := detailedLine.CreditsApplied[0]
		s.Equal(float64(30), creditRealizationDetail.Amount.InexactFloat64())
		s.Equal(creditRealization.ID, creditRealizationDetail.CreditRealizationID)

		stdInvoiceID = invoice.GetInvoiceID()
		s.NotEmpty(stdInvoiceID)
		s.Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	})

	s.Run("advance the invoice and authorize payment", func() {
		defer s.FlatFeeTestHandler.Reset()

		authorizedCallback := newCountedLedgerTransactionCallback[flatfee.Charge]()
		s.FlatFeeTestHandler.onPaymentAuthorized = authorizedCallback.Handler(s.T())

		invoiceUsageAccruedCallback := newCountedLedgerTransactionCallback[flatfee.OnInvoiceUsageAccruedInput]()
		s.FlatFeeTestHandler.onInvoiceUsageAccrued = invoiceUsageAccruedCallback.Handler(s.T())

		invoice, err := s.BillingService.ApproveInvoice(ctx, stdInvoiceID)
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)

		s.Equal(1, authorizedCallback.nrInvocations)
		s.Equal(1, invoiceUsageAccruedCallback.nrInvocations)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)

		// Invoice usage accrued callback should have been invoked
		accruedUsage := updatedFlatFeeCharge.State.AccruedUsage
		s.NotNil(accruedUsage)
		s.Equal(invoiceUsageAccruedCallback.id, accruedUsage.LedgerTransaction.TransactionGroupID, "ledger transaction gets recorded")
		s.Equal(servicePeriod, accruedUsage.ServicePeriod, "service period should be the same as the input")
		s.False(accruedUsage.Mutable, "accrued usage should not be mutable")
		s.NotNil(accruedUsage.LineID, "line ID should be set")
		s.Equal(stdLineID.ID, *accruedUsage.LineID, "line ID should be the same as the standard line")
		s.Equal(float64(70), accruedUsage.Totals.Total.InexactFloat64(), "totals should be the same as the input")
		s.Equal(float64(30), accruedUsage.Totals.CreditsTotal.InexactFloat64(), "totals should be the same as the input")

		// Authorization callback should have been invoked
		s.Equal(authorizedCallback.id, updatedFlatFeeCharge.State.Payment.Authorized.TransactionGroupID)
		s.Equal(meta.ChargeStatusActive, updatedFlatFeeCharge.Status)
	})

	s.Run("payment is settled", func() {
		defer s.FlatFeeTestHandler.Reset()

		settledCallback := newCountedLedgerTransactionCallback[flatfee.Charge]()
		s.FlatFeeTestHandler.onPaymentSettled = settledCallback.Handler(s.T())

		invoice, err := customInvoicing.Service.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: stdInvoiceID,
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)

		charge := s.mustGetChargeByID(flatFeeChargeID)
		updatedFlatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(settledCallback.id, updatedFlatFeeCharge.State.Payment.Settled.TransactionGroupID)
		s.Equal(meta.ChargeStatusFinal, updatedFlatFeeCharge.Status)
	})
}
