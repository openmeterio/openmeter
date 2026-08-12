package subscription_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCreateCostBasisPinEntityInputValidate(t *testing.T) {
	validInput := subscription.CreateCostBasisPinEntityInput{
		Namespace:        "default",
		SubscriptionID:   "subscription-id",
		CustomCurrencyID: "custom-currency-id",
		InvoiceCurrency:  currencyx.Code("USD"),
		CostBasisID:      "cost-basis-id",
	}

	tests := []struct {
		name             string
		mutate           func(*subscription.CreateCostBasisPinEntityInput)
		expectedErrorMsg string
		expectedCause    string
	}{
		{
			name: "valid",
		},
		{
			name: "missing namespace",
			mutate: func(input *subscription.CreateCostBasisPinEntityInput) {
				input.Namespace = ""
			},
			expectedErrorMsg: "namespace is required",
		},
		{
			name: "missing subscription ID",
			mutate: func(input *subscription.CreateCostBasisPinEntityInput) {
				input.SubscriptionID = ""
			},
			expectedErrorMsg: "subscription ID is required",
		},
		{
			name: "missing custom currency ID",
			mutate: func(input *subscription.CreateCostBasisPinEntityInput) {
				input.CustomCurrencyID = ""
			},
			expectedErrorMsg: "custom currency ID is required",
		},
		{
			name: "missing cost basis ID",
			mutate: func(input *subscription.CreateCostBasisPinEntityInput) {
				input.CostBasisID = ""
			},
			expectedErrorMsg: "cost basis ID is required",
		},
		{
			name: "missing invoice currency",
			mutate: func(input *subscription.CreateCostBasisPinEntityInput) {
				input.InvoiceCurrency = ""
			},
			expectedErrorMsg: `invalid invoice currency ""`,
		},
		{
			name: "invalid invoice currency",
			mutate: func(input *subscription.CreateCostBasisPinEntityInput) {
				input.InvoiceCurrency = "US"
			},
			expectedErrorMsg: `invalid invoice currency "US"`,
			expectedCause:    "invalid currency code: it must be between 4 and 24 characters",
		},
		{
			name: "custom invoice currency",
			mutate: func(input *subscription.CreateCostBasisPinEntityInput) {
				input.InvoiceCurrency = "CREDITS"
			},
			expectedErrorMsg: `invalid invoice currency "CREDITS"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validInput
			if tt.mutate != nil {
				tt.mutate(&input)
			}

			err := input.Validate()
			if tt.expectedErrorMsg == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.expectedErrorMsg)
			if tt.expectedCause != "" {
				require.ErrorContains(t, err, tt.expectedCause)
			}
			require.True(t, models.IsGenericValidationError(err))
		})
	}

	t.Run("collects multiple required field errors", func(t *testing.T) {
		input := validInput
		input.Namespace = ""
		input.CostBasisID = ""

		err := input.Validate()
		require.ErrorContains(t, err, "namespace is required")
		require.ErrorContains(t, err, "cost basis ID is required")
		require.True(t, models.IsGenericValidationError(err))
	})
}
