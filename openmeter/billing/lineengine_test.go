package billing

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestIsLineBillableAsOfResultValidate(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name    string
		result  IsLineBillableAsOfResult
		wantErr bool
	}{
		{
			name:   "not billable without period",
			result: IsLineBillableAsOfResult{},
		},
		{
			name: "billable with period",
			result: IsLineBillableAsOfResult{
				Billable:       true,
				BillablePeriod: period,
			},
		},
		{
			name: "billable without period",
			result: IsLineBillableAsOfResult{
				Billable: true,
			},
			wantErr: true,
		},
		{
			name: "billable with period start only",
			result: IsLineBillableAsOfResult{
				Billable: true,
				BillablePeriod: timeutil.ClosedPeriod{
					From: period.From,
				},
			},
			wantErr: true,
		},
		{
			name: "billable with period end only",
			result: IsLineBillableAsOfResult{
				Billable: true,
				BillablePeriod: timeutil.ClosedPeriod{
					To: period.To,
				},
			},
			wantErr: true,
		},
		{
			name: "not billable with period",
			result: IsLineBillableAsOfResult{
				BillablePeriod: period,
			},
			wantErr: true,
		},
		{
			name: "not billable with period start only",
			result: IsLineBillableAsOfResult{
				BillablePeriod: timeutil.ClosedPeriod{
					From: period.From,
				},
			},
			wantErr: true,
		},
		{
			name: "not billable with period end only",
			result: IsLineBillableAsOfResult{
				BillablePeriod: timeutil.ClosedPeriod{
					To: period.To,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.True(t, models.IsGenericValidationError(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGateInvoiceAssignmentInputValidate(t *testing.T) {
	line := newGateInvoiceAssignmentLine("line-1")

	t.Run("accepts unique lines", func(t *testing.T) {
		require.NoError(t, (GateInvoiceAssignmentInput{Lines: GatheringLines{line}}).Validate())
	})

	t.Run("requires lines", func(t *testing.T) {
		err := (GateInvoiceAssignmentInput{}).Validate()

		require.Error(t, err)
		require.True(t, models.IsGenericValidationError(err))
		require.ErrorContains(t, err, "lines are required")
	})

	t.Run("rejects duplicate line IDs", func(t *testing.T) {
		err := (GateInvoiceAssignmentInput{Lines: GatheringLines{line, line}}).Validate()

		require.Error(t, err)
		require.True(t, models.IsGenericValidationError(err))
		require.ErrorContains(t, err, "line IDs must be unique")
	})
}

func TestGateInvoiceAssignmentResultValidate(t *testing.T) {
	line1 := newGateInvoiceAssignmentLine("line-1")
	line2 := newGateInvoiceAssignmentLine("line-2")
	input := GateInvoiceAssignmentInput{Lines: GatheringLines{line1, line2}}

	t.Run("accepts sparse exclusion responses", func(t *testing.T) {
		result := GateInvoiceAssignmentResult{
			line2.GetLineID(): {ExcludeFromInvoice: true},
		}

		require.NoError(t, result.Validate(input))
		require.False(t, result[line1.GetLineID()].ExcludeFromInvoice)
		require.True(t, result[line2.GetLineID()].ExcludeFromInvoice)
	})

	t.Run("accepts an empty response", func(t *testing.T) {
		require.NoError(t, GateInvoiceAssignmentResult(nil).Validate(input))
	})

	t.Run("rejects unknown decisions", func(t *testing.T) {
		unknownLine := newGateInvoiceAssignmentLine("unknown")
		result := GateInvoiceAssignmentResult{
			unknownLine.GetLineID(): {},
		}

		err := result.Validate(input)

		require.Error(t, err)
		require.True(t, models.IsGenericValidationError(err))
		require.ErrorContains(t, err, "unknown line ID: default/unknown")
	})
}

func newGateInvoiceAssignmentLine(id string) GatheringLine {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	return NewFlatFeeGatheringLine(NewFlatFeeLineInput{
		ID:            id,
		Namespace:     "default",
		Period:        period,
		InvoiceAt:     period.To,
		Name:          id,
		Currency:      currencyx.FiatCode("USD"),
		ManagedBy:     ManuallyManagedLine,
		PerUnitAmount: alpacadecimal.NewFromInt(1),
		PaymentTerm:   productcatalog.InArrearsPaymentTerm,
	})
}
