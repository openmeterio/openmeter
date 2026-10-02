package credits

import (
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

func (s *CreditThenInvoiceTestSuite) TestFlatFeeIssuingSkipsDeletedChargeLine() {
	t := s.T()
	ctx := t.Context()
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	// given a two-charge draft whose automatic approval deadline has passed
	fixture := s.setupDeletedLineIssuanceInvoice(meta.ChargeTypeFlatFee)
	clock.FreezeTime(fixture.Invoice.DraftUntil.Add(time.Second))
	defer clock.UnFreeze()

	// when deleting the first charge causes the remaining invoice to issue
	s.MustRefundCharge(ctx, fixture.Customer, fixture.Charges[0])
	invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: fixture.Invoice.GetInvoiceID(),
		Expand: billing.StandardInvoiceExpands{
			billing.StandardInvoiceExpandLines,
			billing.StandardInvoiceExpandDeletedLines,
		},
	})
	require.NoError(t, err)
	require.Len(t, invoice.Lines.OrEmpty(), 2)
	deletedLine := invoice.Lines.GetByID(fixture.Invoice.Lines.OrEmpty()[0].ID)
	require.NotNil(t, deletedLine)
	require.NotNil(t, deletedLine.DeletedAt)
	s.RequireChargeStatus(fixture.Charges[0], meta.ChargeStatusDeleted)

	// then the deleted charge does not block booking the retained line
	s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
	s.False(invoice.StatusDetails.Failed)
	s.False(invoice.HasCriticalValidationIssues())
	s.RequireTotals(billingtest.ExpectedTotals{Amount: 10, Total: 10}, invoice.Totals)
}

func (s *CreditThenInvoiceTestSuite) TestUsageBasedIssuingSkipsDeletedChargeLine() {
	t := s.T()
	ctx := t.Context()
	clock.FreezeTime(time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()
	defer s.MockStreamingConnector.Reset()

	// given a two-charge draft whose automatic approval deadline has passed
	fixture := s.setupDeletedLineIssuanceInvoice(meta.ChargeTypeUsageBased)
	clock.FreezeTime(fixture.Invoice.DraftUntil.Add(time.Second))
	defer clock.UnFreeze()

	// when deleting the first charge causes the remaining invoice to issue
	s.MustRefundCharge(ctx, fixture.Customer, fixture.Charges[0])
	invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: fixture.Invoice.GetInvoiceID(),
		Expand: billing.StandardInvoiceExpands{
			billing.StandardInvoiceExpandLines,
			billing.StandardInvoiceExpandDeletedLines,
		},
	})
	require.NoError(t, err)
	require.Len(t, invoice.Lines.OrEmpty(), 2)
	deletedLine := invoice.Lines.GetByID(fixture.Invoice.Lines.OrEmpty()[0].ID)
	require.NotNil(t, deletedLine)
	require.NotNil(t, deletedLine.DeletedAt)
	s.RequireChargeStatus(fixture.Charges[0], meta.ChargeStatusDeleted)

	// then the deleted charge does not block booking the retained line
	s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status)
	s.False(invoice.StatusDetails.Failed)
	s.False(invoice.HasCriticalValidationIssues())
	s.RequireTotals(billingtest.ExpectedTotals{Amount: 10, Total: 10}, invoice.Totals)
}

type deletedLineIssuanceInvoice struct {
	Customer customer.CustomerID
	Invoice  billing.StandardInvoice
	Charges  [2]meta.ChargeID
}

func (s *CreditThenInvoiceTestSuite) setupDeletedLineIssuanceInvoice(chargeType meta.ChargeType) deletedLineIssuanceInvoice {
	t := s.T()
	t.Helper()
	ctx := t.Context()

	ns := s.GetUniqueNamespace("deleted-line-issuance-" + string(chargeType))
	s.ProvisionDefaultTaxCodes(ctx, ns)
	invoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
	period := timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now().AddDate(0, 1, 0)}
	var featureKey string
	var processingStatus any = flatfee.StatusActiveRealizationProcessing
	if chargeType == meta.ChargeTypeUsageBased {
		processingStatus = usagebased.StatusActiveRealizationProcessing
		feature := s.SetupApiRequestsTotalFeature(ctx, ns)
		featureKey = feature.Feature.Key
		period = timeutil.ClosedPeriod{From: clock.Now().AddDate(0, -1, 0), To: clock.Now()}
		s.MockStreamingConnector.AddSimpleEvent(featureKey, 5, period.From.Add(15*24*time.Hour))
	}

	intents := make([]charges.ChargeIntent, 0, 2)
	for i, amount := range []int64{5, 10} {
		price := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(amount),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})
		if chargeType == meta.ChargeTypeUsageBased {
			price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(amount / 5)})
		}

		intents = append(intents, s.CreateMockChargeIntent(CreateMockChargeIntentInput{
			Customer:       cust.GetID(),
			Currency:       USD,
			ServicePeriod:  period,
			SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			Price:          price,
			ProRating: productcatalog.ProRatingConfig{
				Enabled: true,
				Mode:    productcatalog.ProRatingModeProratePrices,
			},
			FeatureKey:        featureKey,
			Name:              fmt.Sprintf("charge %d", i),
			ManagedBy:         billing.SubscriptionManagedLine,
			UniqueReferenceID: fmt.Sprintf("charge-%d", i),
		}))
	}

	created, err := s.Charges.Create(ctx, charges.CreateInput{Namespace: ns, Intents: charges.NewCreateChargeIntents(intents...)})
	require.NoError(t, err)
	require.Len(t, created, 2)
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: cust.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	require.NoError(t, err)
	require.Len(t, invoices, 1)
	invoice := invoices[0]
	if chargeType == meta.ChargeTypeUsageBased {
		require.Equal(t, billing.StandardInvoiceStatusDraftWaitingForCollection, invoice.Status)
		clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
		defer clock.UnFreeze()
		invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
		require.NoError(t, err)
	}

	require.Equal(t, billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
	require.NotNil(t, invoice.DraftUntil)
	require.Len(t, invoice.Lines.OrEmpty(), 2)
	fixture := deletedLineIssuanceInvoice{Customer: cust.GetID(), Invoice: invoice}
	for i, line := range invoice.Lines.OrEmpty() {
		require.NotNil(t, line.ChargeID)
		fixture.Charges[i] = meta.ChargeID{Namespace: ns, ID: *line.ChargeID}
		s.RequireChargeStatus(fixture.Charges[i], processingStatus)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: float64((i + 1) * 5), Total: float64((i + 1) * 5)}, line.Totals)
	}

	return fixture
}
