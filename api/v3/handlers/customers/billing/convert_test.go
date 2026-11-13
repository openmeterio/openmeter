package customersbilling

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/models"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

func TestCheckoutSessionUIModeConversion(t *testing.T) {
	got, err := FromAPIBillingAppStripeCheckoutSessionUIMode(apiv3.BillingAppStripeCheckoutSessionUIModeEmbedded)
	require.NoError(t, err)
	require.Equal(t, models.CheckoutSessionUIModeEmbedded, got)

	_, err = FromAPIBillingAppStripeCheckoutSessionUIMode("unknown")
	require.Error(t, err)
}
