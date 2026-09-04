package billing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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
			name: "not billable with period",
			result: IsLineBillableAsOfResult{
				BillablePeriod: period,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
