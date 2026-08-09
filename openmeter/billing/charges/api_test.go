package charges

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteCustomerChargeInputValidate(t *testing.T) {
	validInput := DeleteCustomerChargeInput{
		Namespace:         "namespace",
		CustomerID:        "customer-id",
		ChargeID:          "charge-id",
		PaymentAdjustment: PaymentAdjustmentNone,
	}

	tests := []struct {
		name    string
		mutate  func(*DeleteCustomerChargeInput)
		wantErr bool
	}{
		{
			name: "valid",
		},
		{
			name: "missing namespace",
			mutate: func(input *DeleteCustomerChargeInput) {
				input.Namespace = ""
			},
			wantErr: true,
		},
		{
			name: "missing customer ID",
			mutate: func(input *DeleteCustomerChargeInput) {
				input.CustomerID = ""
			},
			wantErr: true,
		},
		{
			name: "missing charge ID",
			mutate: func(input *DeleteCustomerChargeInput) {
				input.ChargeID = ""
			},
			wantErr: true,
		},
		{
			name: "missing payment adjustment",
			mutate: func(input *DeleteCustomerChargeInput) {
				input.PaymentAdjustment = ""
			},
			wantErr: true,
		},
		{
			name: "invalid payment adjustment",
			mutate: func(input *DeleteCustomerChargeInput) {
				input.PaymentAdjustment = PaymentAdjustment("invalid")
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validInput
			if test.mutate != nil {
				test.mutate(&input)
			}

			err := input.Validate()
			if test.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
