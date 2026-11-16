package customersbilling

import (
	"testing"

	"github.com/stretchr/testify/require"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
)

func TestCheckoutSessionUIModeConversion(t *testing.T) {
	got, err := FromAPIBillingAppStripeCheckoutSessionUIMode(apiv3models.BillingAppStripeCheckoutSessionUIModeEmbedded)
	require.NoError(t, err)
	require.Equal(t, apiv1models.CheckoutSessionUIModeEmbedded, got)

	_, err = FromAPIBillingAppStripeCheckoutSessionUIMode("unknown")
	require.Error(t, err)
}
