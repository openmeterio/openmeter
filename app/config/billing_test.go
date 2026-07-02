package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBillingFeatureSwitchesConfigurationValidate(t *testing.T) {
	t.Run("zero max collected invoice lines is valid", func(t *testing.T) {
		require.NoError(t, BillingFeatureSwitchesConfiguration{
			MaxLinesPerCollectedInvoice: 0,
		}.Validate())
	})

	t.Run("positive max collected invoice lines is valid", func(t *testing.T) {
		require.NoError(t, BillingFeatureSwitchesConfiguration{
			MaxLinesPerCollectedInvoice: 10,
		}.Validate())
	})

	t.Run("negative max collected invoice lines is invalid", func(t *testing.T) {
		err := BillingFeatureSwitchesConfiguration{
			MaxLinesPerCollectedInvoice: -1,
		}.Validate()

		require.ErrorContains(t, err, "maxLinesPerCollectedInvoice must not be negative")
	})
}
