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

type chargeBookingInvoice struct {
	Customer customer.CustomerID
	Invoice  billing.StandardInvoice
	Charges  [2]meta.ChargeID
}

func (s *CreditThenInvoiceTestSuite) setupChargeBookingInvoice(chargeType meta.ChargeType) chargeBookingInvoice {
	t := s.T()
	t.Helper()
	ctx := t.Context()

	ns := s.GetUniqueNamespace("issuing-charge-booking-" + string(chargeType))
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), ns)
	invoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateLedgerBackedCustomer(ns, "test-subject")
	s.ProvisionBillingProfile(ctx, ns, invoicing.App.GetID())
	period := timeutil.ClosedPeriod{From: clock.Now(), To: clock.Now().AddDate(0, 1, 0)}
	price := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      alpacadecimal.NewFromInt(5),
		PaymentTerm: productcatalog.InAdvancePaymentTerm,
	})
	var featureKey string
	var processingStatus any = flatfee.StatusActiveRealizationProcessing
	if chargeType == meta.ChargeTypeUsageBased {
		processingStatus = usagebased.StatusActiveRealizationProcessing
		feature := s.SetupApiRequestsTotalFeature(ctx, ns)
		featureKey = feature.Feature.Key
		period = timeutil.ClosedPeriod{From: clock.Now().AddDate(0, -1, 0), To: clock.Now()}
		s.MockStreamingConnector.AddSimpleEvent(featureKey, 5, period.From.Add(15*24*time.Hour))
		price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)})
	}

	intents := make([]charges.ChargeIntent, 0, 2)
	for i := range 2 {
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
		// The clock stores one global frozen value, so restore the caller's test time.
		func() {
			defer clock.FreezeTime(clock.Now())
			clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
			invoice, err = s.BillingService.AdvanceInvoice(ctx, invoice.GetInvoiceID())
			require.NoError(t, err)
		}()
	}

	require.Equal(t, billing.StandardInvoiceStatusDraftWaitingAutoApproval, invoice.Status)
	require.NotNil(t, invoice.DraftUntil)
	require.Len(t, invoice.Lines.OrEmpty(), 2)
	fixture := chargeBookingInvoice{Customer: cust.GetID(), Invoice: invoice}
	for i, line := range invoice.Lines.OrEmpty() {
		require.NotNil(t, line.ChargeID)
		fixture.Charges[i] = meta.ChargeID{Namespace: ns, ID: *line.ChargeID}
		s.RequireChargeStatus(fixture.Charges[i], processingStatus)
		s.RequireTotals(billingtest.ExpectedTotals{Amount: 5, Total: 5}, line.Totals)
	}

	return fixture
}
