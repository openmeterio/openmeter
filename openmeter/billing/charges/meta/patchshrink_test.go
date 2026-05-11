package meta

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestPatchShrinkValidateWith(t *testing.T) {
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
		patch   PatchShrink
		wantErr bool
	}{
		{
			name: "allows service period shrink with unchanged full service and billing periods",
			patch: mustNewPatchShrink(t, NewPatchShrinkInput{
				NewServicePeriodTo:     intent.ServicePeriod.To.Add(-time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.To,
				NewBillingPeriodTo:     intent.BillingPeriod.To,
				NewInvoiceAt:           intent.ServicePeriod.To.Add(-time.Hour),
			}),
		},
		{
			name: "allows full service and billing period shrink",
			patch: mustNewPatchShrink(t, NewPatchShrinkInput{
				NewServicePeriodTo:     intent.ServicePeriod.To.Add(-time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.To.Add(-time.Hour),
				NewBillingPeriodTo:     intent.BillingPeriod.To.Add(-time.Hour),
				NewInvoiceAt:           intent.ServicePeriod.To.Add(-time.Hour),
			}),
		},
		{
			name: "rejects unchanged service period end",
			patch: mustNewPatchShrink(t, NewPatchShrinkInput{
				NewServicePeriodTo:     intent.ServicePeriod.To,
				NewFullServicePeriodTo: intent.FullServicePeriod.To,
				NewBillingPeriodTo:     intent.BillingPeriod.To,
				NewInvoiceAt:           intent.ServicePeriod.To,
			}),
			wantErr: true,
		},
		{
			name: "rejects later service period end",
			patch: mustNewPatchShrink(t, NewPatchShrinkInput{
				NewServicePeriodTo:     intent.ServicePeriod.To.Add(time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.To,
				NewBillingPeriodTo:     intent.BillingPeriod.To,
				NewInvoiceAt:           intent.ServicePeriod.To.Add(time.Hour),
			}),
			wantErr: true,
		},
		{
			name: "rejects service period end at service period start",
			patch: mustNewPatchShrink(t, NewPatchShrinkInput{
				NewServicePeriodTo:     intent.ServicePeriod.From,
				NewFullServicePeriodTo: intent.FullServicePeriod.To,
				NewBillingPeriodTo:     intent.BillingPeriod.To,
				NewInvoiceAt:           intent.ServicePeriod.From,
			}),
			wantErr: true,
		},
		{
			name: "rejects service period end before service period start",
			patch: mustNewPatchShrink(t, NewPatchShrinkInput{
				NewServicePeriodTo:     intent.ServicePeriod.From.Add(-time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.To,
				NewBillingPeriodTo:     intent.BillingPeriod.To,
				NewInvoiceAt:           intent.ServicePeriod.From.Add(-time.Hour),
			}),
			wantErr: true,
		},
		{
			name: "rejects later full service period end",
			patch: mustNewPatchShrink(t, NewPatchShrinkInput{
				NewServicePeriodTo:     intent.ServicePeriod.To.Add(-time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.To.Add(time.Hour),
				NewBillingPeriodTo:     intent.BillingPeriod.To,
				NewInvoiceAt:           intent.ServicePeriod.To.Add(-time.Hour),
			}),
			wantErr: true,
		},
		{
			name: "rejects full service period end at full service period start",
			patch: mustNewPatchShrink(t, NewPatchShrinkInput{
				NewServicePeriodTo:     intent.ServicePeriod.To.Add(-time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.From,
				NewBillingPeriodTo:     intent.BillingPeriod.To,
				NewInvoiceAt:           intent.ServicePeriod.To.Add(-time.Hour),
			}),
			wantErr: true,
		},
		{
			name: "rejects later billing period end",
			patch: mustNewPatchShrink(t, NewPatchShrinkInput{
				NewServicePeriodTo:     intent.ServicePeriod.To.Add(-time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.To,
				NewBillingPeriodTo:     intent.BillingPeriod.To.Add(time.Hour),
				NewInvoiceAt:           intent.ServicePeriod.To.Add(-time.Hour),
			}),
			wantErr: true,
		},
		{
			name: "rejects billing period end at billing period start",
			patch: mustNewPatchShrink(t, NewPatchShrinkInput{
				NewServicePeriodTo:     intent.ServicePeriod.To.Add(-time.Hour),
				NewFullServicePeriodTo: intent.FullServicePeriod.To,
				NewBillingPeriodTo:     intent.BillingPeriod.From,
				NewInvoiceAt:           intent.ServicePeriod.To.Add(-time.Hour),
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

func mustNewPatchShrink(t *testing.T, input NewPatchShrinkInput) PatchShrink {
	t.Helper()

	patch, err := NewPatchShrink(input)
	require.NoError(t, err)
	return patch
}
