package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

func TestApplyInvocableChargePatchesBatchesEachInvoiceEffectRound(t *testing.T) {
	// given
	// - Two patched charges emit invoice effects in the first lifecycle round.
	// - Only one charge emits another invoice effect after the first batch is applied.
	// when:
	// - The root charge service applies and advances the patches.
	// then:
	// - Effects are batched by round and each charge resumes only after its preceding batch.
	patch := meta.PatchDelete{}

	firstCharge := &scriptedInvocableCharge{
		chargeID: meta.ChargeID{Namespace: "ns", ID: "first"},
		triggerResult: TriggerPatchResult{
			InvoicePatches: invoiceupdater.Patches{
				invoiceupdater.NewDeleteGatheringLineByChargeIDPatch("first-round-1"),
			},
			CanAdvance: true,
		},
		advanceResults: []TriggerPatchResult{
			{
				InvoicePatches: invoiceupdater.Patches{
					invoiceupdater.NewDeleteGatheringLineByChargeIDPatch("first-round-2"),
				},
				CanAdvance: true,
			},
			{},
		},
	}
	secondCharge := &scriptedInvocableCharge{
		chargeID: meta.ChargeID{Namespace: "ns", ID: "second"},
		triggerResult: TriggerPatchResult{InvoicePatches: invoiceupdater.Patches{
			invoiceupdater.NewDeleteGatheringLineByChargeIDPatch("second-round-1"),
		}},
		advanceResults: []TriggerPatchResult{{}},
	}

	var appliedBatches [][]string
	updater := recordingInvoiceUpdater{apply: func(_ context.Context, _ customer.CustomerID, patches invoiceupdater.Patches) error {
		if len(appliedBatches) == 0 {
			require.Zero(t, firstCharge.advanceCalls)
			require.Zero(t, secondCharge.advanceCalls)
		} else {
			require.Equal(t, 1, firstCharge.advanceCalls)
			require.Zero(t, secondCharge.advanceCalls)
		}

		chargeIDs := make([]string, 0, len(patches))
		for _, patch := range patches {
			deletePatch, err := patch.AsDeleteGatheringLineByChargeIDPatch()
			require.NoError(t, err)
			chargeIDs = append(chargeIDs, deletePatch.ChargeID)
		}
		appliedBatches = append(appliedBatches, chargeIDs)

		return nil
	}}

	svc := service{invoiceUpdater: updater}
	err := svc.applyInvocableChargePatches(
		t.Context(),
		customer.CustomerID{Namespace: "ns", ID: "customer"},
		map[string]InvocableCharge{
			"first":  firstCharge,
			"second": secondCharge,
		},
		map[string]charges.Patch{
			"first":  patch,
			"second": patch,
		},
	)
	require.NoError(t, err)

	require.Len(t, appliedBatches, 2)
	require.ElementsMatch(t, []string{"first-round-1", "second-round-1"}, appliedBatches[0])
	require.Equal(t, []string{"first-round-2"}, appliedBatches[1])
	require.Equal(t, 1, firstCharge.triggerCalls)
	require.Equal(t, 2, firstCharge.advanceCalls)
	require.Equal(t, 1, secondCharge.triggerCalls)
	require.Zero(t, secondCharge.advanceCalls)
}

type scriptedInvocableCharge struct {
	chargeID       meta.ChargeID
	triggerResult  TriggerPatchResult
	advanceResults []TriggerPatchResult
	triggerCalls   int
	advanceCalls   int
}

func (c *scriptedInvocableCharge) GetChargeID() meta.ChargeID {
	return c.chargeID
}

func (c *scriptedInvocableCharge) TriggerPatch(context.Context, meta.Patch) (TriggerPatchResult, error) {
	c.triggerCalls++
	return c.triggerResult, nil
}

func (c *scriptedInvocableCharge) AdvanceCharge(context.Context) (TriggerPatchResult, error) {
	result := c.advanceResults[c.advanceCalls]
	c.advanceCalls++
	return result, nil
}

type recordingInvoiceUpdater struct {
	apply func(context.Context, customer.CustomerID, invoiceupdater.Patches) error
}

func (u recordingInvoiceUpdater) ApplyPatches(ctx context.Context, customerID customer.CustomerID, patches invoiceupdater.Patches) error {
	return u.apply(ctx, customerID, patches)
}
