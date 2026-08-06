package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func TestLineEngineUsesChargeOwnedValuation(t *testing.T) {
	lineEngine := (&service{}).GetLineEngine()

	require.Equal(t, billing.LineEngineTypeChargeCreditPurchase, lineEngine.GetLineEngineType())
	_, implementsLineCalculator := lineEngine.(billing.LineCalculator)
	require.False(t, implementsLineCalculator)
}
