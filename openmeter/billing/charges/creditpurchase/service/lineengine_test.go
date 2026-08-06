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

func TestGetChargesForStandardLinesInputValidate(t *testing.T) {
	charge := newInvoiceStateMachineTestCharge(t, creditpurchase.StatusCreated)
	lineWithHeader := newInvoiceStateMachineTestLine(t, charge, alpacadecimal.NewFromInt(50))
	input := getChargesForStandardLinesInput{
		Invoice: lineWithHeader.Invoice,
		Lines:   billing.StandardLines{lineWithHeader.Line},
		Expands: meta.Expands{meta.ExpandRealizations},
	}

	require.NoError(t, input.Validate())

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
