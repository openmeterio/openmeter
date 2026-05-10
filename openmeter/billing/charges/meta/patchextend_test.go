package meta

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestPatchExtendValidateWith(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	intent := Intent{
		ServicePeriod: timeutil.ClosedPeriod{
			From: base,
			To:   base.AddDate(0, 1, 0),
		},
		FullServicePeriod: timeutil.ClosedPeriod{
			From: base,
			To:   base.AddDate(0, 2, 0),
		},
		BillingPeriod: timeutil.ClosedPeriod{
			From: base,
			To:   base.AddDate(0, 3, 0),
		},
	}

	tests := []struct {
		name    string
		patch   PatchExtend
		wantErr bool
	}{
		{
			name: "allows service period extension with unchanged full service and billing periods",
			patch: mustNewPatchExtend(t, NewPatchExtendInput{
				NewServicePeriodTo:     intent.ServicePeriod.To.Add(time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.To,
				NewBillingPeriodTo:     intent.BillingPeriod.To,
			}),
		},
		{
			name: "rejects unchanged service period end",
			patch: mustNewPatchExtend(t, NewPatchExtendInput{
				NewServicePeriodTo:     intent.ServicePeriod.To,
				NewFullServicePeriodTo: intent.FullServicePeriod.To,
				NewBillingPeriodTo:     intent.BillingPeriod.To,
			}),
			wantErr: true,
		},
		{
			name: "rejects earlier service period end",
			patch: mustNewPatchExtend(t, NewPatchExtendInput{
				NewServicePeriodTo:     intent.ServicePeriod.To.Add(-time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.To,
				NewBillingPeriodTo:     intent.BillingPeriod.To,
			}),
			wantErr: true,
		},
		{
			name: "rejects earlier full service period end",
			patch: mustNewPatchExtend(t, NewPatchExtendInput{
				NewServicePeriodTo:     intent.ServicePeriod.To.Add(time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.To.Add(-time.Hour),
				NewBillingPeriodTo:     intent.BillingPeriod.To,
			}),
			wantErr: true,
		},
		{
			name: "rejects earlier billing period end",
			patch: mustNewPatchExtend(t, NewPatchExtendInput{
				NewServicePeriodTo:     intent.ServicePeriod.To.Add(time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.To,
				NewBillingPeriodTo:     intent.BillingPeriod.To.Add(-time.Hour),
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.patch.ValidateWith(intent)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func mustNewPatchExtend(t *testing.T, input NewPatchExtendInput) PatchExtend {
	t.Helper()

	patch, err := NewPatchExtend(input)
	require.NoError(t, err)
	return patch
}
