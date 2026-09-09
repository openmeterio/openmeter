package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestLineEngineDoesNotImplementLineCalculator(t *testing.T) {
	lineEngine := (&service{}).GetLineEngine()

	require.Equal(t, billing.LineEngineTypeChargeCreditPurchase, lineEngine.GetLineEngineType())
	_, implementsLineCalculator := lineEngine.(billing.LineCalculator)
	require.False(t, implementsLineCalculator)
}

func TestAreLinesBillableAsOfPreservesResultsWithValidationIssues(t *testing.T) {
	// Given a batch with valid credit-purchase lines around an unsupported split line.
	firstPeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	}
	lastPeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
	}
	firstChargeID := "charge-1"
	firstLine := billing.GatheringLine{GatheringLineBase: billing.GatheringLineBase{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
			ID:              "line-1",
			Name:            "credit purchase",
		},
		ManagedBy:     billing.SystemManagedLine,
		Engine:        billing.LineEngineTypeChargeCreditPurchase,
		InvoiceID:     "invoice-id",
		Currency:      currencyx.FiatCode("USD"),
		ServicePeriod: firstPeriod,
		InvoiceAt:     firstPeriod.From,
		Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount: alpacadecimal.NewFromInt(10),
		}),
		ChargeID: &firstChargeID,
	}}
	splitLine := firstLine
	splitLine.ID = "line-2"
	splitChargeID := "charge-2"
	splitLine.ChargeID = &splitChargeID
	splitLineGroupID := "split-group"
	splitLine.SplitLineGroupID = &splitLineGroupID
	lastLine := firstLine
	lastLine.ID = "line-3"
	lastChargeID := "charge-3"
	lastLine.ChargeID = &lastChargeID
	lastLine.ServicePeriod = lastPeriod
	lastLine.InvoiceAt = lastPeriod.From
	engine := &LineEngine{}

	// When a valid line, an invalid split line, and another valid line are checked together.
	results, err := engine.AreLinesBillableAsOf(t.Context(), billing.AreLinesBillableAsOfInput{
		Invoice: billing.GatheringInvoice{GatheringInvoiceBase: billing.GatheringInvoiceBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: firstLine.Namespace},
				ID:              firstLine.InvoiceID,
			},
		}},
		AsOf:  lastPeriod.To,
		Lines: billing.GatheringLines{firstLine, splitLine, lastLine},
	})

	// Then the validation issue is returned without losing the ordered per-line results.
	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Equal(t, billing.ValidationIssues{billing.ErrInvoiceProgressiveBillingNotSupported}, issues)
	require.Equal(t, []billing.IsLineBillableAsOfResult{
		{Billable: true, BillablePeriod: firstPeriod},
		{},
		{Billable: true, BillablePeriod: lastPeriod},
	}, results)
}

func TestPopulateInvoiceCreditPurchaseStandardLineRejectsUnresolvedCostBasis(t *testing.T) {
	// given: an invoice credit purchase whose lifecycle left its cost basis unresolved
	charge := newInvoiceStateMachineTestCharge(t, creditpurchase.StatusActivePaymentPending)
	charge.State.ResolvedCostBasis = nil
	line := newInvoiceStateMachineTestLine(t, charge, alpacadecimal.Zero).Line

	// when: the line engine attempts to finalize the provisional standard line
	_, err := populateInvoiceCreditPurchaseStandardLine(populateInvoiceCreditPurchaseStandardLineInput{
		Line:   line,
		Charge: charge,
	})

	// then: the unresolved lifecycle state is rejected instead of being priced
	require.ErrorContains(t, err, "cost basis is unresolved")
}

func TestGetChargesForStandardLinesInputValidate(t *testing.T) {
	charge := newInvoiceStateMachineTestCharge(t, creditpurchase.StatusCreated)
	lineWithHeader := newInvoiceStateMachineTestLine(t, charge, alpacadecimal.NewFromInt(50))
	input := getChargesForStandardLinesInput{
		Invoice: lineWithHeader.Invoice,
		Lines:   billing.StandardLines{lineWithHeader.Line},
		Expands: meta.Expands{meta.ExpandRealizations},
	}

	require.NoError(t, input.Validate())

	t.Run("rejects multiple lines for the same charge", func(t *testing.T) {
		// given: two otherwise valid invoice lines referencing one credit purchase
		duplicateLine := *lineWithHeader.Line
		duplicateLine.ID = "line-2"
		duplicateInput := input
		duplicateInput.Lines = billing.StandardLines{lineWithHeader.Line, &duplicateLine}

		// when: the line event input is validated
		err := duplicateInput.Validate()

		// then: the charge cannot receive the same lifecycle trigger twice
		require.ErrorContains(t, err, "is referenced by multiple standard lines")
	})

	input.Invoice.Namespace = ""
	input.Lines[0].ChargeID = nil
	input.Expands = meta.Expands{meta.Expand("invalid")}

	err := input.Validate()
	require.Error(t, err)
	require.ErrorContains(t, err, "invoice namespace is required")
	require.ErrorContains(t, err, "charge ID is required")
	require.ErrorContains(t, err, "does not match invoice namespace")
	require.ErrorContains(t, err, "invalid expand")
}
