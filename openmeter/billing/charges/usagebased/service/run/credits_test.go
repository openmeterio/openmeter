package run

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChargeCurrencyCreditReconciliationHandlerOwnsCurrencyCalculator(t *testing.T) {
	service := Service{}
	handler := service.NewChargeCurrencyCreditReconciliationHandler(CreditReconciliationHandlerInput{
		Charge:     newUsageBasedCharge(t),
		Run:        newUsageBasedRun("line-1"),
		AllocateAt: time.Now().UTC(),
	})

	require.NoError(t, handler.Validate())
	require.Equal(t, "USD", handler.CurrencyCalculator().Details().Code.String())
}
