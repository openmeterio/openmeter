package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

var pendingLinesServicePeriod = timeutil.ClosedPeriod{
	From: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
	To:   time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC),
}

func newManualUsageGatheringLine(ns, featureKey string) billing.GatheringLine {
	return billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: ns,
				Name:      "manual usage",
			}),
			ManagedBy:     billing.ManuallyManagedLine,
			Engine:        billing.LineEngineTypeInvoice,
			Currency:      USD,
			ServicePeriod: pendingLinesServicePeriod,
			InvoiceAt:     pendingLinesServicePeriod.To,
			Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(2),
			})),
			FeatureKey: featureKey,
		},
	}
}

func newManualFlatGatheringLine(ns, name string, perUnitAmount alpacadecimal.Decimal) billing.GatheringLine {
	return billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
		Namespace:     ns,
		Period:        pendingLinesServicePeriod,
		InvoiceAt:     pendingLinesServicePeriod.From,
		ManagedBy:     billing.ManuallyManagedLine,
		Name:          name,
		Currency:      USD,
		PerUnitAmount: perUnitAmount,
		PaymentTerm:   productcatalog.InAdvancePaymentTerm,
	})
}

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesCreatesChargeBackedGatheringLines() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-create-pending-lines")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	clock.FreezeTime(pendingLinesServicePeriod.From)
	defer clock.UnFreeze()

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)
	featureKey := apiRequestsTotal.Feature.Key

	usageLine := newManualUsageGatheringLine(ns, featureKey)
	usageLine.RateCardDiscounts = billing.Discounts{
		Usage: &billing.UsageDiscount{
			UsageDiscount: productcatalog.UsageDiscount{
				Quantity: alpacadecimal.NewFromInt(3),
			},
		},
	}
	usageLine.ChildUniqueReferenceID = lo.ToPtr("manual-usage")

	flatLine := newManualFlatGatheringLine(ns, "manual flat", alpacadecimal.NewFromInt(10))
	flatLine.Engine = billing.LineEngineTypeInvoice
	flatLine.RateCardDiscounts = billing.Discounts{
		Percentage: &billing.PercentageDiscount{
			PercentageDiscount: productcatalog.PercentageDiscount{
				Percentage: models.NewPercentage(10),
			},
		},
	}
	flatLine.ChildUniqueReferenceID = lo.ToPtr("manual-flat")

	result, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: USD,
		Lines: []billing.GatheringLine{
			usageLine,
			flatLine,
		},
	})
	s.NoError(err)
	s.Require().NotNil(result)
	s.NotEmpty(result.Invoice.ID)
	s.Require().Len(result.Lines, 2)

	s.Equal("manual usage", result.Lines[0].Name)
	s.Equal(billing.LineEngineTypeChargeUsageBased, result.Lines[0].Engine)
	s.Require().NotNil(result.Lines[0].ChargeID)

	s.Equal("manual flat", result.Lines[1].Name)
	s.Equal(billing.LineEngineTypeChargeFlatFee, result.Lines[1].Engine)
	s.Require().NotNil(result.Lines[1].ChargeID)

	usageCharge := s.mustGetChargeByID(meta.ChargeID{
		Namespace: ns,
		ID:        lo.FromPtr(result.Lines[0].ChargeID),
	})
	usageBasedCharge, err := usageCharge.AsUsageBasedCharge()
	s.NoError(err)
	s.Equal(productcatalog.CreditThenInvoiceSettlementMode, usageBasedCharge.Intent.SettlementMode)
	s.Equal(billing.ManuallyManagedLine, usageBasedCharge.Intent.ManagedBy)
	s.Equal("manual-usage", lo.FromPtr(usageBasedCharge.Intent.UniqueReferenceID))
	s.Equal(featureKey, usageBasedCharge.Intent.FeatureKey)
	s.Require().NotNil(usageBasedCharge.Intent.Discounts.Usage)
	s.Equal(float64(3), usageBasedCharge.Intent.Discounts.Usage.Quantity.InexactFloat64())

	flatCharge := s.mustGetChargeByID(meta.ChargeID{
		Namespace: ns,
		ID:        lo.FromPtr(result.Lines[1].ChargeID),
	})
	flatFeeCharge, err := flatCharge.AsFlatFeeCharge()
	s.NoError(err)
	s.Equal(productcatalog.CreditThenInvoiceSettlementMode, flatFeeCharge.Intent.SettlementMode)
	s.Equal(billing.ManuallyManagedLine, flatFeeCharge.Intent.ManagedBy)
	s.Equal("manual-flat", lo.FromPtr(flatFeeCharge.Intent.UniqueReferenceID))
	s.Equal(productcatalog.InAdvancePaymentTerm, flatFeeCharge.Intent.PaymentTerm)
	s.Equal(float64(10), flatFeeCharge.Intent.AmountBeforeProration.InexactFloat64())
	s.Require().NotNil(flatFeeCharge.Intent.PercentageDiscounts)
	s.Equal(float64(10), flatFeeCharge.Intent.PercentageDiscounts.Percentage.InexactFloat64())
}

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesRejectsZeroAmountFlatFee() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-create-pending-lines-zero-flat")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	zeroFlatLine := newManualFlatGatheringLine(ns, "manual zero flat", alpacadecimal.Zero)
	zeroFlatLine.Engine = billing.LineEngineTypeInvoice

	result, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: USD,
		Lines: []billing.GatheringLine{
			zeroFlatLine,
		},
	})
	s.Nil(result)
	s.Require().Error(err)
	s.Require().ErrorAs(err, &billing.ValidationError{})
	s.Contains(err.Error(), "zero-amount flat fee is not supported")

	listed, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
	})
	s.NoError(err)
	s.Empty(listed.Items)
}

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesRejectsNonManualInput() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-create-pending-lines-policy")

	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	systemLine := newManualFlatGatheringLine(ns, "system flat", alpacadecimal.NewFromInt(10))
	systemLine.ManagedBy = billing.SystemManagedLine

	_, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: USD,
		Lines: []billing.GatheringLine{
			systemLine,
		},
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "managed by must be manual")

	subscriptionLine := newManualFlatGatheringLine(ns, "subscription flat", alpacadecimal.NewFromInt(10))
	subscriptionLine.Subscription = &billing.SubscriptionReference{
		SubscriptionID: "sub-1",
		PhaseID:        "phase-1",
		ItemID:         "item-1",
		BillingPeriod:  pendingLinesServicePeriod,
	}

	_, err = s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: USD,
		Lines: []billing.GatheringLine{
			subscriptionLine,
		},
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "subscription is not allowed")
}

func (s *InvoicableChargesTestSuite) TestCreatePendingInvoiceLinesRejectsZeroAmountFlatFeeInBatch() {
	// A zero-amount flat fee in a batch rejects the whole batch upfront: neither charges
	// nor gathering invoices persist.
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("charges-service-create-pending-lines-zero-flat-batch")
	s.ProvisionDefaultTaxCodes(ctx, ns)

	customInvoicing := s.SetupCustomInvoicing(ns)
	cust := s.CreateTestCustomer(ns, "test-subject")
	s.NotEmpty(cust.ID)

	_ = s.ProvisionBillingProfile(ctx, ns, customInvoicing.App.GetID(),
		billingtest.WithCollectionInterval(datetime.MustParseDuration(s.T(), "P2D")),
		billingtest.WithManualApproval(),
	)

	apiRequestsTotal := s.SetupApiRequestsTotalFeature(ctx, ns)

	usageLine := newManualUsageGatheringLine(ns, apiRequestsTotal.Feature.Key)

	zeroFlatLine := newManualFlatGatheringLine(ns, "manual zero flat", alpacadecimal.Zero)
	zeroFlatLine.Engine = billing.LineEngineTypeInvoice

	result, err := s.Charges.CreatePendingInvoiceLines(ctx, charges.CreatePendingInvoiceLinesInput{
		Customer: cust.GetID(),
		Currency: USD,
		Lines: []billing.GatheringLine{
			usageLine,
			zeroFlatLine,
		},
	})
	s.Nil(result)
	s.Require().Error(err)
	s.Require().ErrorAs(err, &billing.ValidationError{})
	s.Contains(err.Error(), "zero-amount flat fee is not supported")

	listedCharges, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:   ns,
		CustomerIDs: []string{cust.ID},
	})
	s.NoError(err)
	s.Empty(listedCharges.Items)

	listedInvoices, err := s.BillingService.ListGatheringInvoices(ctx, billing.ListGatheringInvoicesInput{
		Namespaces: []string{ns},
		Customers:  []string{cust.ID},
	})
	s.NoError(err)
	s.Empty(listedInvoices.Items)
}

func TestValidateChargePendingInvoiceLinesInput(t *testing.T) {
	inputWith := func(lines ...billing.GatheringLine) charges.CreatePendingInvoiceLinesInput {
		return charges.CreatePendingInvoiceLinesInput{
			Customer: customer.CustomerID{Namespace: "ns", ID: "customer-1"},
			Currency: USD,
			Lines:    lines,
		}
	}

	t.Run("valid line passes", func(t *testing.T) {
		require.NoError(t, validateChargePendingInvoiceLinesInput(inputWith(newManualFlatGatheringLine("ns", "manual flat", alpacadecimal.NewFromInt(10)))))
	})

	t.Run("empty lines rejected", func(t *testing.T) {
		require.ErrorContains(t, validateChargePendingInvoiceLinesInput(inputWith()), "no lines provided")
	})

	t.Run("charge ID rejected", func(t *testing.T) {
		line := newManualFlatGatheringLine("ns", "manual flat", alpacadecimal.NewFromInt(10))
		line.ChargeID = lo.ToPtr("charge-1")
		require.ErrorContains(t, validateChargePendingInvoiceLinesInput(inputWith(line)), "charge ID is not allowed")
	})

	t.Run("explicit engine rejected", func(t *testing.T) {
		line := newManualFlatGatheringLine("ns", "manual flat", alpacadecimal.NewFromInt(10))
		line.Engine = billing.LineEngineTypeChargeFlatFee
		require.ErrorContains(t, validateChargePendingInvoiceLinesInput(inputWith(line)), "engine is not allowed")
	})

	t.Run("non-manual managed by rejected", func(t *testing.T) {
		line := newManualFlatGatheringLine("ns", "manual flat", alpacadecimal.NewFromInt(10))
		line.ManagedBy = billing.SystemManagedLine
		require.ErrorContains(t, validateChargePendingInvoiceLinesInput(inputWith(line)), "managed by must be manual")
	})

	t.Run("subscription reference rejected", func(t *testing.T) {
		line := newManualFlatGatheringLine("ns", "manual flat", alpacadecimal.NewFromInt(10))
		line.Subscription = &billing.SubscriptionReference{
			SubscriptionID: "sub-1",
			PhaseID:        "phase-1",
			ItemID:         "item-1",
			BillingPeriod:  pendingLinesServicePeriod,
		}
		require.ErrorContains(t, validateChargePendingInvoiceLinesInput(inputWith(line)), "subscription is not allowed")
	})

	t.Run("zero-amount flat fee rejected", func(t *testing.T) {
		line := newManualFlatGatheringLine("ns", "manual zero flat", alpacadecimal.Zero)
		require.ErrorContains(t, validateChargePendingInvoiceLinesInput(inputWith(line)), "zero-amount flat fee is not supported")
	})

	t.Run("usage discount on flat fee rejected", func(t *testing.T) {
		line := newManualFlatGatheringLine("ns", "manual flat", alpacadecimal.NewFromInt(10))
		line.RateCardDiscounts = billing.Discounts{
			Usage: &billing.UsageDiscount{
				UsageDiscount: productcatalog.UsageDiscount{
					Quantity: alpacadecimal.NewFromInt(3),
				},
			},
		}
		require.ErrorContains(t, validateChargePendingInvoiceLinesInput(inputWith(line)), "usage discount is not supported for flat fee lines")
	})
}

func TestOrderPendingLinesByCreatedCharges(t *testing.T) {
	newFlatFeeCharge := func(id string) charges.Charge {
		return charges.NewCharge(flatfee.Charge{
			ChargeBase: flatfee.ChargeBase{
				ManagedResource: meta.ManagedResource{ID: id},
			},
		})
	}
	newUsageBasedCharge := func(id string) charges.Charge {
		return charges.NewCharge(usagebased.Charge{
			ChargeBase: usagebased.ChargeBase{
				ManagedResource: meta.ManagedResource{ID: id},
			},
		})
	}
	lineFor := func(chargeID string, engine billing.LineEngineType) billing.GatheringLine {
		line := billing.GatheringLine{}
		line.Engine = engine
		if chargeID != "" {
			line.ChargeID = lo.ToPtr(chargeID)
		}
		return line
	}

	t.Run("reorders lines to created charge order", func(t *testing.T) {
		out, err := orderPendingLinesByCreatedCharges(
			[]billing.GatheringLine{
				lineFor("charge-2", billing.LineEngineTypeChargeUsageBased),
				lineFor("charge-1", billing.LineEngineTypeChargeFlatFee),
			},
			charges.Charges{newFlatFeeCharge("charge-1"), newUsageBasedCharge("charge-2")},
		)
		require.NoError(t, err)
		require.Len(t, out, 2)
		require.Equal(t, "charge-1", lo.FromPtr(out[0].ChargeID))
		require.Equal(t, "charge-2", lo.FromPtr(out[1].ChargeID))
	})

	t.Run("missing charge ID rejected", func(t *testing.T) {
		_, err := orderPendingLinesByCreatedCharges(
			[]billing.GatheringLine{lineFor("", billing.LineEngineTypeChargeFlatFee)},
			charges.Charges{newFlatFeeCharge("charge-1")},
		)
		require.ErrorContains(t, err, "charge ID is required")
	})

	t.Run("duplicate charge ID rejected", func(t *testing.T) {
		_, err := orderPendingLinesByCreatedCharges(
			[]billing.GatheringLine{
				lineFor("charge-1", billing.LineEngineTypeChargeFlatFee),
				lineFor("charge-1", billing.LineEngineTypeChargeFlatFee),
			},
			charges.Charges{newFlatFeeCharge("charge-1")},
		)
		require.ErrorContains(t, err, "duplicate charge ID")
	})

	t.Run("charge without gathering line rejected", func(t *testing.T) {
		_, err := orderPendingLinesByCreatedCharges(
			[]billing.GatheringLine{lineFor("charge-1", billing.LineEngineTypeChargeFlatFee)},
			charges.Charges{newFlatFeeCharge("charge-1"), newFlatFeeCharge("charge-2")},
		)
		require.ErrorContains(t, err, "gathering line was not created")
	})

	t.Run("unsupported charge type rejected", func(t *testing.T) {
		_, err := orderPendingLinesByCreatedCharges(
			[]billing.GatheringLine{lineFor("charge-1", billing.LineEngineTypeChargeFlatFee)},
			charges.Charges{charges.NewCharge(creditpurchase.Charge{
				ChargeBase: creditpurchase.ChargeBase{
					ManagedResource: meta.ManagedResource{ID: "charge-1"},
				},
			})},
		)
		require.ErrorContains(t, err, "unsupported charge type")
	})

	t.Run("engine mismatch rejected", func(t *testing.T) {
		_, err := orderPendingLinesByCreatedCharges(
			[]billing.GatheringLine{lineFor("charge-1", billing.LineEngineTypeChargeUsageBased)},
			charges.Charges{newFlatFeeCharge("charge-1")},
		)
		require.ErrorContains(t, err, "expected line engine")
	})

	t.Run("unexpected charge ID rejected", func(t *testing.T) {
		_, err := orderPendingLinesByCreatedCharges(
			[]billing.GatheringLine{
				lineFor("charge-1", billing.LineEngineTypeChargeFlatFee),
				lineFor("charge-9", billing.LineEngineTypeChargeFlatFee),
			},
			charges.Charges{newFlatFeeCharge("charge-1")},
		)
		require.ErrorContains(t, err, "unexpected charge ID")
	})
}
