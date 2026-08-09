package charges

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

func TestSetCustomerChargeOverrideInputValidate(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	validInput := SetCustomerChargeOverrideInput{
		Namespace:  "namespace",
		CustomerID: "customer-id",
		ChargeID:   "charge-id",
		FlatFee: &flatfee.IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "override",
				ServicePeriod:     period,
				FullServicePeriod: period,
				BillingPeriod:     period,
			},
			InvoiceAt:             period.From,
			PaymentTerm:           productcatalog.InAdvancePaymentTerm,
			AmountBeforeProration: alpacadecimal.NewFromInt(10),
		},
	}

	tests := []struct {
		name    string
		mutate  func(*SetCustomerChargeOverrideInput)
		wantErr bool
	}{
		{
			name: "valid flat fee override",
		},
		{
			name: "valid usage based override",
			mutate: func(input *SetCustomerChargeOverrideInput) {
				input.FlatFee = nil
				input.UsageBased = &usagebased.IntentMutableFields{
					IntentMutableFields: meta.IntentMutableFields{
						Name:              "override",
						ServicePeriod:     period,
						FullServicePeriod: period,
						BillingPeriod:     period,
					},
					InvoiceAt: period.To,
					Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
						Amount: alpacadecimal.NewFromInt(10),
					}),
				}
			},
		},
		{
			name: "missing namespace",
			mutate: func(input *SetCustomerChargeOverrideInput) {
				input.Namespace = ""
			},
			wantErr: true,
		},
		{
			name: "missing customer ID",
			mutate: func(input *SetCustomerChargeOverrideInput) {
				input.CustomerID = ""
			},
			wantErr: true,
		},
		{
			name: "missing charge ID",
			mutate: func(input *SetCustomerChargeOverrideInput) {
				input.ChargeID = ""
			},
			wantErr: true,
		},
		{
			name: "missing charge type",
			mutate: func(input *SetCustomerChargeOverrideInput) {
				input.FlatFee = nil
			},
			wantErr: true,
		},
		{
			name: "multiple charge types",
			mutate: func(input *SetCustomerChargeOverrideInput) {
				input.UsageBased = &usagebased.IntentMutableFields{}
			},
			wantErr: true,
		},
		{
			name: "deleted override intent",
			mutate: func(input *SetCustomerChargeOverrideInput) {
				input.FlatFee.IntentDeletedAt = &period.From
			},
			wantErr: true,
		},
		{
			name: "invalid mutable fields",
			mutate: func(input *SetCustomerChargeOverrideInput) {
				input.FlatFee.Name = ""
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validInput
			flatFee := validInput.FlatFee.Clone()
			input.FlatFee = &flatFee
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
