package service

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

func TestLineEngineDoesNotImplementLineCalculator(t *testing.T) {
	lineEngine := (&service{}).GetLineEngine()

	require.Equal(t, billing.LineEngineTypeChargeCreditPurchase, lineEngine.GetLineEngineType())
	_, implementsLineCalculator := lineEngine.(billing.LineCalculator)
	require.False(t, implementsLineCalculator)
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
