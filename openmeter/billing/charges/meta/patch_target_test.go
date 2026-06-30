package meta

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type layeredIntentReaderForTest struct {
	baseManagedBy billing.InvoiceLineManagedBy
	hasOverride   bool
}

func (r layeredIntentReaderForTest) GetBaseManagedBy() billing.InvoiceLineManagedBy {
	return r.baseManagedBy
}

func (r layeredIntentReaderForTest) HasOverrideLayer() bool {
	return r.hasOverride
}

func TestPeriodPatchGetTargetLayer(t *testing.T) {
	patch := PatchExtend{changeSource: billing.ChangeSourceSystem}

	got, err := patch.GetTargetLayer(layeredIntentReaderForTest{
		baseManagedBy: billing.SubscriptionManagedLine,
	})

	require.NoError(t, err)
	require.Equal(t, ChangeTargetBase, got)
}

func TestPeriodPatchValidateRejectsAPIChange(t *testing.T) {
	patch := PatchExtend{changeSource: billing.ChangeSourceAPIRequest}

	require.Error(t, patch.Validate())
}

func TestDeletePatchGetTargetLayer(t *testing.T) {
	tests := []struct {
		name   string
		patch  PatchDelete
		intent layeredIntentReaderForTest
		want   ChangeTarget
	}{
		{
			name: "system change targets base",
			patch: PatchDelete{
				changeSource: billing.ChangeSourceSystem,
			},
			intent: layeredIntentReaderForTest{
				baseManagedBy: billing.SubscriptionManagedLine,
			},
			want: ChangeTargetBase,
		},
		{
			name: "api change on manual base without override targets base",
			patch: PatchDelete{
				changeSource: billing.ChangeSourceAPIRequest,
			},
			intent: layeredIntentReaderForTest{
				baseManagedBy: billing.ManuallyManagedLine,
			},
			want: ChangeTargetBase,
		},
		{
			name: "api change with override targets override",
			patch: PatchDelete{
				changeSource: billing.ChangeSourceAPIRequest,
			},
			intent: layeredIntentReaderForTest{
				baseManagedBy: billing.ManuallyManagedLine,
				hasOverride:   true,
			},
			want: ChangeTargetOverride,
		},
		{
			name: "api change on subscription base targets override",
			patch: PatchDelete{
				changeSource: billing.ChangeSourceAPIRequest,
			},
			intent: layeredIntentReaderForTest{
				baseManagedBy: billing.SubscriptionManagedLine,
			},
			want: ChangeTargetOverride,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.patch.GetTargetLayer(tt.intent)

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDeletePatchGetTargetLayerRejectsMissingAPIIntent(t *testing.T) {
	patch := PatchDelete{changeSource: billing.ChangeSourceAPIRequest}

	_, err := patch.GetTargetLayer(nil)

	require.ErrorContains(t, err, "intent is required")
}

func TestLineManualEditPatchGetTargetLayer(t *testing.T) {
	tests := []struct {
		name   string
		intent layeredIntentReaderForTest
		want   ChangeTarget
	}{
		{
			name: "manual base without override targets base",
			intent: layeredIntentReaderForTest{
				baseManagedBy: billing.ManuallyManagedLine,
			},
			want: ChangeTargetBase,
		},
		{
			name: "manual base with override targets override",
			intent: layeredIntentReaderForTest{
				baseManagedBy: billing.ManuallyManagedLine,
				hasOverride:   true,
			},
			want: ChangeTargetOverride,
		},
		{
			name: "subscription base targets override",
			intent: layeredIntentReaderForTest{
				baseManagedBy: billing.SubscriptionManagedLine,
			},
			want: ChangeTargetOverride,
		},
	}

	patch := PatchLineManualEdit{changeSource: billing.ChangeSourceAPIRequest}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := patch.GetTargetLayer(tt.intent)

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestLineManualEditPatchGetTargetLayerRejectsMissingIntent(t *testing.T) {
	patch := PatchLineManualEdit{changeSource: billing.ChangeSourceAPIRequest}

	_, err := patch.GetTargetLayer(nil)

	require.ErrorContains(t, err, "intent is required")
}

func TestLineManualEditPatchValidateRejectsSystemChange(t *testing.T) {
	patch := PatchLineManualEdit{changeSource: billing.ChangeSourceSystem}

	require.Error(t, patch.Validate())
}
