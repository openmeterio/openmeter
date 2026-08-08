package service

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	chargestatemachine "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCreditThenInvoiceSetOverrideCreatesAndReplacesOverride(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditThenInvoiceSettlementMode, flatfee.StatusCreated, servicePeriod)
	machine := newCreditThenInvoiceSetOverrideStateMachineForTest(t, charge)

	// given:
	// - a subscription-managed, invoice-backed flat fee has only its source layer
	// when:
	// - a complete override snapshot is set, then replaced
	// then:
	// - the source layer is preserved, the same override layer is updated, and
	//   the created charge is rescheduled from the override invoice time
	first := charge.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
	first.Name = "first override"
	first.AmountBeforeProration = alpacadecimal.NewFromInt(20)
	first.InvoiceAt = servicePeriod.From.Add(24 * time.Hour)
	require.NoError(t, fireFlatFeeSetOverride(t, machine, first))

	updated := machine.GetCharge()
	require.Equal(t, "flat fee", updated.Intent.GetBaseIntent().Name)
	firstOverride := updated.Intent.GetOverrideLayerMutableFields()
	require.NotNil(t, firstOverride)
	require.Equal(t, "first override", firstOverride.Name)
	require.Equal(t, 20.0, firstOverride.AmountBeforeProration.InexactFloat64())
	require.Equal(t, first.InvoiceAt, *updated.State.AdvanceAfter)
	require.Len(t, machine.InvoicePatches(), 1)

	second := first.Clone()
	second.Name = "second override"
	second.AmountBeforeProration = alpacadecimal.NewFromInt(30)
	require.NoError(t, fireFlatFeeSetOverride(t, machine, second))

	updated = machine.GetCharge()
	require.Equal(t, "flat fee", updated.Intent.GetBaseIntent().Name)
	override := updated.Intent.GetOverrideLayerMutableFields()
	require.NotNil(t, override)
	require.Equal(t, "second override", override.Name)
	require.Equal(t, 30.0, override.AmountBeforeProration.InexactFloat64())
}

func TestCreditThenInvoiceSetOverrideRejectsBillingOwnedStates(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	for _, status := range []flatfee.Status{
		flatfee.StatusActiveRealizationIssuing,
		flatfee.StatusActiveRealizationZeroFiatAmountOverageCompleted,
		flatfee.StatusActiveRealizationCompleted,
		flatfee.StatusDeleted,
	} {
		t.Run(string(status), func(t *testing.T) {
			// given:
			// - billing owns the current transition or the charge is deleted
			// when:
			// - an API override is requested
			// then:
			// - the state machine rejects it without mutating the source intent
			charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditThenInvoiceSettlementMode, status, servicePeriod)
			machine := newCreditThenInvoiceSetOverrideStateMachineForTest(t, charge)
			fields := charge.Intent.GetEffectiveIntent().IntentMutableFields.Clone()
			fields.Name = "override"

			err := fireFlatFeeSetOverride(t, machine, fields)
			require.Error(t, err)
			require.True(t, models.IsGenericPreConditionFailedError(err))
			require.Nil(t, machine.GetCharge().Intent.GetOverrideLayerMutableFields())
		})
	}
}

func TestCreditThenInvoiceClearOverrideRestoresBaseIntent(t *testing.T) {
	clock.FreezeTime(time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	// given:
	// - a created invoice-backed charge has a manual override layer
	// when:
	// - the override is cleared through the transient reset state
	// then:
	// - the base intent becomes effective and gathering is rescheduled from it
	charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditThenInvoiceSettlementMode, flatfee.StatusCreated, servicePeriod)
	override := charge.Intent.GetBaseIntent().IntentMutableFields.Clone()
	override.Name = "manual override"
	override.AmountBeforeProration = alpacadecimal.NewFromInt(20)
	override.InvoiceAt = servicePeriod.From.Add(24 * time.Hour)
	charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	charge.State.AmountAfterProration = override.AmountBeforeProration
	machine := newCreditThenInvoiceSetOverrideStateMachineForTest(t, charge)

	require.NoError(t, fireFlatFeeClearOverride(t, machine))
	require.Equal(t, flatfee.StatusActiveClearOverride, machine.GetCharge().Status)
	require.NoError(t, machine.FireAndActivate(t.Context(), meta.TriggerNext))

	updated := machine.GetCharge()
	require.Nil(t, updated.Intent.GetOverrideLayerMutableFields())
	require.Equal(t, "flat fee", updated.Intent.GetEffectiveIntent().Name)
	require.Equal(t, 10.0, updated.State.AmountAfterProration.InexactFloat64())
	require.Equal(t, flatfee.StatusCreated, updated.Status)
	require.Equal(t, servicePeriod.From, *updated.State.AdvanceAfter)
	require.Len(t, machine.InvoicePatches(), 1)
}

func TestCreditThenInvoiceClearOverrideSelectsRestoredLifecycleState(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	for _, test := range []struct {
		name           string
		now            time.Time
		baseAmount     alpacadecimal.Decimal
		expectedStatus flatfee.Status
		expectedPatch  invoiceupdater.PatchOperation
	}{
		{
			name:           "before invoice at",
			now:            time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC),
			baseAmount:     alpacadecimal.NewFromInt(10),
			expectedStatus: flatfee.StatusCreated,
			expectedPatch:  invoiceupdater.PatchOpUpsertGatheringLineByChargeID,
		},
		{
			name:           "after invoice at",
			now:            time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			baseAmount:     alpacadecimal.NewFromInt(10),
			expectedStatus: flatfee.StatusActive,
			expectedPatch:  invoiceupdater.PatchOpUpsertGatheringLineByChargeID,
		},
		{
			name:           "zero amount",
			now:            time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC),
			baseAmount:     alpacadecimal.Zero,
			expectedStatus: flatfee.StatusFinal,
			expectedPatch:  invoiceupdater.PatchOpDeleteGatheringLineByChargeID,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			clock.FreezeTime(test.now)
			defer clock.UnFreeze()

			// given:
			// - a live base with a deletion override and no realization work
			// when:
			// - the override is cleared through active.clear_override
			// then:
			// - TriggerNext selects the normal lifecycle from the restored base
			charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditThenInvoiceSettlementMode, flatfee.StatusDeleted, servicePeriod)
			require.NoError(t, charge.Intent.Mutate(meta.ChangeTargetBase, func(fields *flatfee.IntentMutableFields) {
				fields.AmountBeforeProration = test.baseAmount
			}))
			override := charge.Intent.GetBaseIntent().IntentMutableFields.Clone()
			override.AmountBeforeProration = alpacadecimal.NewFromInt(20)
			override.IntentDeletedAt = &servicePeriod.To
			charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
			charge.State.AmountAfterProration = override.AmountBeforeProration
			machine := newCreditThenInvoiceSetOverrideStateMachineForTest(t, charge)

			require.NoError(t, fireFlatFeeClearOverride(t, machine))
			require.Equal(t, flatfee.StatusActiveClearOverride, machine.GetCharge().Status)
			require.NoError(t, machine.FireAndActivate(t.Context(), meta.TriggerNext))
			require.Equal(t, test.expectedStatus, machine.GetCharge().Status)
			require.Len(t, machine.InvoicePatches(), 1)
			require.Equal(t, test.expectedPatch, machine.InvoicePatches()[0].Op())
		})
	}
}

func TestCreditThenInvoiceClearDeletionOverrideRestoresBaseWithPriorInvoiceHistory(t *testing.T) {
	clock.FreezeTime(time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditThenInvoiceSettlementMode, flatfee.StatusDeleted, servicePeriod)
	override := charge.Intent.GetBaseIntent().IntentMutableFields.Clone()
	override.IntentDeletedAt = &servicePeriod.To
	charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	lineID := "line-id"
	invoiceID := "invoice-id"
	priorRun := flatfee.RealizationRun{
		RealizationRunBase: flatfee.RealizationRunBase{
			ID:                   flatfee.RealizationRunID{Namespace: charge.Namespace, ID: "run-id"},
			ManagedModel:         charge.ManagedModel,
			LineID:               &lineID,
			InvoiceID:            &invoiceID,
			Type:                 flatfee.RealizationRunTypeFinalRealization,
			InitialType:          flatfee.RealizationRunTypeFinalRealization,
			ServicePeriod:        servicePeriod,
			AmountAfterProration: alpacadecimal.NewFromInt(10),
			Immutable:            true,
		},
	}
	charge.Realizations.PriorRuns = flatfee.RealizationRuns{priorRun}
	machine := newCreditThenInvoiceSetOverrideStateMachineForTest(t, charge)

	// given:
	// - a deletion override hides a live base after its invoice realization was
	//   detached into history
	// when:
	// - the deletion override is cleared
	// then:
	// - prior invoice history remains detached and the restored base starts new
	//   gathering work
	require.NoError(t, fireFlatFeeClearOverride(t, machine))
	require.Equal(t, flatfee.StatusActiveClearOverride, machine.GetCharge().Status)
	require.NoError(t, machine.FireAndActivate(t.Context(), meta.TriggerNext))

	updated := machine.GetCharge()
	require.Nil(t, updated.Intent.GetOverrideLayerMutableFields())
	require.Equal(t, flatfee.StatusCreated, updated.Status)
	require.Nil(t, updated.Realizations.CurrentRun)
	require.Equal(t, flatfee.RealizationRuns{priorRun}, updated.Realizations.PriorRuns)
	require.Len(t, machine.InvoicePatches(), 1)
	require.Equal(t, invoiceupdater.PatchOpUpsertGatheringLineByChargeID, machine.InvoicePatches()[0].Op())
}

func TestCreditThenInvoiceClearDeletionOverrideRestoresUnrealizedBase(t *testing.T) {
	clock.FreezeTime(time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditThenInvoiceSettlementMode, flatfee.StatusDeleted, servicePeriod)
	override := charge.Intent.GetBaseIntent().IntentMutableFields.Clone()
	override.IntentDeletedAt = &servicePeriod.To
	charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	machine := newCreditThenInvoiceSetOverrideStateMachineForTest(t, charge)

	// given:
	// - a deletion override hides an active base with no invoice realization history
	// when:
	// - the override is cleared
	// then:
	// - the base is restored to the normal gathering lifecycle
	require.NoError(t, fireFlatFeeClearOverride(t, machine))
	require.Equal(t, flatfee.StatusActiveClearOverride, machine.GetCharge().Status)
	require.NoError(t, machine.FireAndActivate(t.Context(), meta.TriggerNext))

	updated := machine.GetCharge()
	require.Equal(t, flatfee.StatusCreated, updated.Status)
	require.Nil(t, updated.Intent.GetOverrideLayerMutableFields())
	require.Equal(t, servicePeriod.From, *updated.State.AdvanceAfter)
	require.Len(t, machine.InvoicePatches(), 1)
}

func TestCreditThenInvoiceClearOverrideTransitionsToDeletedBase(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditThenInvoiceSettlementMode, flatfee.StatusCreated, servicePeriod)
	override := charge.Intent.GetBaseIntent().IntentMutableFields.Clone()
	override.Name = "manual override"
	require.NoError(t, charge.Intent.Mutate(meta.ChangeTargetBase, func(fields *flatfee.IntentMutableFields) {
		fields.IntentDeletedAt = &servicePeriod.To
	}))
	charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	machine := newCreditThenInvoiceSetOverrideStateMachineForTest(t, charge)

	// given:
	// - the active override hides a source intent deleted by reconciliation
	// when:
	// - the override is cleared
	// then:
	// - the source deletion becomes effective through the normal deletion lifecycle
	require.NoError(t, fireFlatFeeClearOverride(t, machine))
	require.Equal(t, flatfee.StatusDeletedClearOverride, machine.GetCharge().Status)
	_, err := machine.AdvanceUntilStateStable(t.Context())
	require.NoError(t, err)

	updated := machine.GetCharge()
	require.Equal(t, flatfee.StatusDeleted, updated.Status)
	require.False(t, updated.Intent.HasOverrideLayer())
	require.Equal(t, servicePeriod.To, *updated.Intent.GetDeletedAt())
	require.Len(t, machine.InvoicePatches(), 1)
	require.Equal(t, invoiceupdater.PatchOpDeleteGatheringLineByChargeID, machine.InvoicePatches()[0].Op())
}

func TestCreditThenInvoiceClearOverrideResetsBillingOwnedStates(t *testing.T) {
	clock.FreezeTime(time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	for _, status := range []flatfee.Status{
		flatfee.StatusActiveRealizationIssuing,
		flatfee.StatusActiveRealizationZeroFiatAmountOverageCompleted,
		flatfee.StatusActiveRealizationCompleted,
	} {
		t.Run(string(status), func(t *testing.T) {
			// given:
			// - an overridden charge has current and prior invoice
			//   realizations, plus already-cancelled history
			// when:
			// - the override is cleared
			// then:
			// - only the current realization line is cancelled and detached while
			//   prior audit history remains untouched
			charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditThenInvoiceSettlementMode, status, servicePeriod)
			override := charge.Intent.GetBaseIntent().IntentMutableFields.Clone()
			override.Name = "manual override"
			charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
			currentRun := newCreditThenInvoiceClearOverrideTestRun(charge, "current-run", "current-line", "current-invoice", true, nil, flatfee.RealizationRunTypeFinalRealization)
			deletedAt := servicePeriod.To
			charge.Realizations.CurrentRun = &currentRun
			charge.Realizations.PriorRuns = flatfee.RealizationRuns{
				newCreditThenInvoiceClearOverrideTestRun(charge, "prior-run", "prior-line", "prior-invoice", false, nil, flatfee.RealizationRunTypeFinalRealization),
				newCreditThenInvoiceClearOverrideTestRun(charge, "deleted-run", "deleted-line", "deleted-invoice", true, &deletedAt, flatfee.RealizationRunTypeFinalRealization),
				newCreditThenInvoiceClearOverrideTestRun(charge, "voided-run", "voided-line", "voided-invoice", true, nil, flatfee.RealizationRunTypeInvalidDueToUnsupportedCreditNote),
			}
			machine := newCreditThenInvoiceSetOverrideStateMachineForTest(t, charge)

			require.NoError(t, fireFlatFeeClearOverride(t, machine))
			require.Equal(t, flatfee.StatusActiveClearOverride, machine.GetCharge().Status)
			require.Nil(t, machine.GetCharge().Intent.GetOverrideLayerMutableFields())
			require.Nil(t, machine.GetCharge().Realizations.CurrentRun)
			require.Len(t, machine.GetCharge().Realizations.PriorRuns, 4)
			require.Len(t, machine.InvoicePatches(), 2)
			require.Equal(t, invoiceupdater.PatchOpLineDelete, machine.InvoicePatches()[0].Op())
			require.Equal(t, invoiceupdater.PatchOpUpsertGatheringLineByChargeID, machine.InvoicePatches()[1].Op())
			currentLinePatch, err := machine.InvoicePatches()[0].AsDeleteLinePatch()
			require.NoError(t, err)
			require.Equal(t, "current-line", currentLinePatch.Line.ID)
			require.NoError(t, machine.FireAndActivate(t.Context(), meta.TriggerNext))
			require.Equal(t, flatfee.StatusCreated, machine.GetCharge().Status)
		})
	}
}

func TestCreditThenInvoiceClearOverrideDetachesCurrentRealizationWithoutInvoiceReferences(t *testing.T) {
	clock.FreezeTime(time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditThenInvoiceSettlementMode, flatfee.StatusActiveRealizationStarted, servicePeriod)
	override := charge.Intent.GetBaseIntent().IntentMutableFields.Clone()
	override.Name = "manual override"
	charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	currentRun := newCreditThenInvoiceClearOverrideTestRun(charge, "current-run", "", "", false, nil, flatfee.RealizationRunTypeFinalRealization)
	currentRun.LineID = nil
	currentRun.InvoiceID = nil
	charge.Realizations.CurrentRun = &currentRun
	machine := newCreditThenInvoiceSetOverrideStateMachineForTest(t, charge)

	// given:
	// - an overridden charge has a current realization without invoice references
	// when:
	// - the override is cleared
	// then:
	// - the run is still detached into history without requesting line deletion
	require.NoError(t, fireFlatFeeClearOverride(t, machine))
	require.Nil(t, machine.GetCharge().Realizations.CurrentRun)
	require.Equal(t, flatfee.RealizationRuns{currentRun}, machine.GetCharge().Realizations.PriorRuns)
	require.Len(t, machine.InvoicePatches(), 1)
	require.Equal(t, invoiceupdater.PatchOpUpsertGatheringLineByChargeID, machine.InvoicePatches()[0].Op())
}

func newCreditThenInvoiceClearOverrideTestRun(
	charge flatfee.Charge,
	id string,
	lineID string,
	invoiceID string,
	immutable bool,
	deletedAt *time.Time,
	runType flatfee.RealizationRunType,
) flatfee.RealizationRun {
	managedModel := charge.ManagedModel
	managedModel.DeletedAt = deletedAt

	return flatfee.RealizationRun{
		RealizationRunBase: flatfee.RealizationRunBase{
			ID:                   flatfee.RealizationRunID{Namespace: charge.Namespace, ID: id},
			ManagedModel:         managedModel,
			LineID:               &lineID,
			InvoiceID:            &invoiceID,
			Type:                 runType,
			InitialType:          flatfee.RealizationRunTypeFinalRealization,
			ServicePeriod:        charge.Intent.GetEffectiveServicePeriod(),
			AmountAfterProration: charge.State.AmountAfterProration,
			Immutable:            immutable,
		},
	}
}

func TestCreditsOnlyClearDeletedOverrideKeepsDeletedBase(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	// given:
	// - a deleted credits-only charge whose source intent is also deleted
	// when:
	// - its deletion override is cleared
	// then:
	// - only the override row is removed; the source deletion remains effective
	charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditOnlySettlementMode, flatfee.StatusDeleted, servicePeriod)
	override := charge.Intent.GetBaseIntent().IntentMutableFields.Clone()
	override.IntentDeletedAt = &servicePeriod.To
	require.NoError(t, charge.Intent.Mutate(meta.ChangeTargetBase, func(fields *flatfee.IntentMutableFields) {
		fields.IntentDeletedAt = &servicePeriod.To
	}))
	charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	machine := newCreditsOnlyClearOverrideStateMachineForTest(t, charge)

	require.NoError(t, fireFlatFeeClearOverride(t, machine))

	updated := machine.GetCharge()
	require.Equal(t, flatfee.StatusDeleted, updated.Status)
	require.NotNil(t, updated.Intent.GetDeletedAt())
	require.Nil(t, updated.Intent.GetOverrideLayerMutableFields())
}

func TestCreditsOnlyClearDeletionOverrideRestoresLiveBase(t *testing.T) {
	clock.FreezeTime(time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC))
	defer clock.UnFreeze()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditOnlySettlementMode, flatfee.StatusDeleted, servicePeriod)
	override := charge.Intent.GetBaseIntent().IntentMutableFields.Clone()
	override.IntentDeletedAt = &servicePeriod.To
	charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	machine := newCreditsOnlyClearOverrideStateMachineForTest(t, charge)

	// given:
	// - a deletion override hides a live credits-only base before its invoice date
	// when:
	// - the override is cleared and restoration advances
	// then:
	// - reconciliation runs in the transient state and the base resumes as created
	require.NoError(t, fireFlatFeeClearOverride(t, machine))
	require.Equal(t, flatfee.StatusActiveClearOverride, machine.GetCharge().Status)
	require.Nil(t, machine.GetCharge().Intent.GetOverrideLayerMutableFields())
	require.NoError(t, machine.FireAndActivate(t.Context(), meta.TriggerNext))

	updated := machine.GetCharge()
	require.Equal(t, flatfee.StatusCreated, updated.Status)
	require.Equal(t, servicePeriod.From, *updated.State.AdvanceAfter)
}

func TestCreditsOnlyClearOverrideTransitionsToDeletedBase(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	// given:
	// - an active credits-only override hides a source intent deleted by reconciliation
	// when:
	// - the override is cleared
	// then:
	// - the source deletion becomes effective through the normal deletion lifecycle
	charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditOnlySettlementMode, flatfee.StatusActive, servicePeriod)
	deletedAt := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	require.NoError(t, charge.Intent.Mutate(meta.ChangeTargetBase, func(fields *flatfee.IntentMutableFields) {
		fields.IntentDeletedAt = &deletedAt
	}))
	override := charge.Intent.GetBaseIntent().IntentMutableFields.Clone()
	override.IntentDeletedAt = nil
	charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	machine := newCreditsOnlyClearOverrideStateMachineForTest(t, charge)

	require.NoError(t, fireFlatFeeClearOverride(t, machine))
	require.Equal(t, flatfee.StatusDeletedClearOverride, machine.GetCharge().Status)
	_, err := machine.AdvanceUntilStateStable(t.Context())
	require.NoError(t, err)

	updated := machine.GetCharge()
	require.Equal(t, flatfee.StatusDeleted, updated.Status)
	require.False(t, updated.Intent.HasOverrideLayer())
	require.Equal(t, deletedAt, *updated.Intent.GetDeletedAt())
}

func TestCreditsOnlyClearOverrideRestoresActiveBaseAndReconcilesRun(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	// given:
	// - an active credits-only charge has an override and a mutable current run
	// when:
	// - clearing restores the base intent
	// then:
	// - the effective amount and the persisted run are rerated from that base
	charge := newFlatFeeSetOverrideTestCharge(t, productcatalog.CreditOnlySettlementMode, flatfee.StatusActive, servicePeriod)
	override := charge.Intent.GetBaseIntent().IntentMutableFields.Clone()
	override.AmountBeforeProration = alpacadecimal.NewFromInt(20)
	charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	charge.State.AmountAfterProration = override.AmountBeforeProration
	charge.Realizations.CurrentRun = &flatfee.RealizationRun{
		RealizationRunBase: flatfee.RealizationRunBase{
			ID: flatfee.RealizationRunID{Namespace: charge.Namespace, ID: "run-id"},
			ManagedModel: models.ManagedModel{
				CreatedAt: servicePeriod.From,
				UpdatedAt: servicePeriod.From,
			},
			Type:                      flatfee.RealizationRunTypeFinalRealization,
			InitialType:               flatfee.RealizationRunTypeFinalRealization,
			ServicePeriod:             servicePeriod,
			AmountAfterProration:      override.AmountBeforeProration,
			NoFiatTransactionRequired: true,
		},
		CreditRealizations: creditrealization.Realizations{
			{
				CreateInput: creditrealization.CreateInput{
					ID:            "realization-id",
					ServicePeriod: servicePeriod,
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: "transaction-group-id",
					},
					Amount: alpacadecimal.NewFromInt(10),
					Type:   creditrealization.TypeAllocation,
				},
			},
		},
	}
	machine, adapter := newCreditsOnlyClearOverrideReratingStateMachineForTest(t, charge)

	require.NoError(t, fireFlatFeeClearOverride(t, machine))

	updated := machine.GetCharge()
	require.Nil(t, updated.Intent.GetOverrideLayerMutableFields())
	require.Equal(t, 10.0, updated.State.AmountAfterProration.InexactFloat64())
	require.Equal(t, 1, adapter.updateRunCalls)
	require.True(t, adapter.updatedRun.AmountAfterProration.IsPresent())
	require.Equal(t, 10.0, adapter.updatedRun.AmountAfterProration.OrEmpty().InexactFloat64())
	require.Equal(t, 1, adapter.upsertDetailedLinesCalls)
}

func fireFlatFeeSetOverride(t *testing.T, machine StateMachine, fields flatfee.IntentMutableFields) error {
	t.Helper()

	patch, err := meta.NewPatchSetOverride(flatfee.NewPatchSetOverrideInput{
		ChangeSource:        billing.ChangeSourceAPIRequest,
		IntentMutableFields: fields,
	})
	require.NoError(t, err)

	return machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
}

func fireFlatFeeClearOverride(t *testing.T, machine StateMachine) error {
	t.Helper()

	patch, err := meta.NewPatchClearOverride(meta.NewPatchClearOverrideInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
	})
	require.NoError(t, err)

	return machine.FireAndActivate(t.Context(), patch.Trigger(), patch)
}

func newCreditThenInvoiceSetOverrideStateMachineForTest(t *testing.T, charge flatfee.Charge) *CreditThenInvoiceStateMachine {
	t.Helper()

	adapter := &flatFeeSetOverrideStateMachineAdapter{charge: charge}
	machine, err := chargestatemachine.New(chargestatemachine.Config[flatfee.Charge, flatfee.ChargeBase, flatfee.Status]{
		Charge: charge,
		Persistence: chargestatemachine.Persistence[flatfee.Charge, flatfee.ChargeBase]{
			UpdateBase: adapter.UpdateCharge,
			Refetch: func(_ context.Context, _ meta.ChargeID) (flatfee.Charge, error) {
				return adapter.charge, nil
			},
		},
	})
	require.NoError(t, err)

	out := &CreditThenInvoiceStateMachine{
		stateMachine: &stateMachine{
			Machine: machine,
			Adapter: adapter,
		},
	}
	out.configureStates()

	return out
}

func newCreditsOnlyClearOverrideStateMachineForTest(t *testing.T, charge flatfee.Charge) *CreditsOnlyStateMachine {
	t.Helper()

	adapter := &flatFeeSetOverrideStateMachineAdapter{charge: charge}
	realizations, err := flatfeerealizations.New(flatfeerealizations.Config{
		Adapter:       adapter,
		Handler:       flatFeeSetOverrideTestHandler{},
		Lineage:       flatFeeSetOverrideTestLineage{},
		RatingService: flatFeeSetOverrideTestRater{},
	})
	require.NoError(t, err)
	machine, err := chargestatemachine.New(chargestatemachine.Config[flatfee.Charge, flatfee.ChargeBase, flatfee.Status]{
		Charge: charge,
		Persistence: chargestatemachine.Persistence[flatfee.Charge, flatfee.ChargeBase]{
			UpdateBase: adapter.UpdateCharge,
			Refetch: func(_ context.Context, _ meta.ChargeID) (flatfee.Charge, error) {
				return adapter.charge, nil
			},
		},
	})
	require.NoError(t, err)

	out := &CreditsOnlyStateMachine{
		stateMachine: &stateMachine{
			Machine:      machine,
			Adapter:      adapter,
			Realizations: realizations,
		},
	}
	out.configureStates()

	return out
}

func newCreditsOnlyClearOverrideReratingStateMachineForTest(t *testing.T, charge flatfee.Charge) (*CreditsOnlyStateMachine, *flatFeeSetOverrideStateMachineAdapter) {
	t.Helper()

	adapter := &flatFeeSetOverrideStateMachineAdapter{
		charge:  charge,
		runBase: charge.Realizations.CurrentRun.RealizationRunBase,
	}
	realizations, err := flatfeerealizations.New(flatfeerealizations.Config{
		Adapter:       adapter,
		Handler:       flatFeeSetOverrideTestHandler{},
		Lineage:       flatFeeSetOverrideTestLineage{},
		RatingService: flatFeeSetOverrideTestRater{},
	})
	require.NoError(t, err)
	machine, err := chargestatemachine.New(chargestatemachine.Config[flatfee.Charge, flatfee.ChargeBase, flatfee.Status]{
		Charge: charge,
		Persistence: chargestatemachine.Persistence[flatfee.Charge, flatfee.ChargeBase]{
			UpdateBase: adapter.UpdateCharge,
			Refetch: func(_ context.Context, _ meta.ChargeID) (flatfee.Charge, error) {
				return adapter.charge, nil
			},
		},
	})
	require.NoError(t, err)

	out := &CreditsOnlyStateMachine{
		stateMachine: &stateMachine{
			Machine:      machine,
			Adapter:      adapter,
			Realizations: realizations,
		},
	}
	out.configureStates()

	return out, adapter
}

func newFlatFeeSetOverrideTestCharge(
	t *testing.T,
	settlementMode productcatalog.SettlementMode,
	status flatfee.Status,
	servicePeriod timeutil.ClosedPeriod,
) flatfee.Charge {
	t.Helper()

	mutable := flatfee.IntentMutableFields{
		IntentMutableFields: meta.IntentMutableFields{
			Name:              "flat fee",
			ServicePeriod:     servicePeriod,
			FullServicePeriod: servicePeriod,
			BillingPeriod:     servicePeriod,
		},
		InvoiceAt:             servicePeriod.From,
		PaymentTerm:           productcatalog.InAdvancePaymentTerm,
		AmountBeforeProration: alpacadecimal.NewFromInt(10),
	}

	return flatfee.Charge{
		ChargeBase: flatfee.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ManagedModel: models.ManagedModel{
					CreatedAt: servicePeriod.From,
					UpdatedAt: servicePeriod.From,
				},
				ID: "charge-id",
			},
			Intent: flatfee.Intent{
				Intent: meta.Intent{
					ManagedBy:  billing.SubscriptionManagedLine,
					CustomerID: "customer-id",
					Currency:   testutils.NewFiatCurrency(t, "USD"),
					TaxConfig: productcatalog.TaxCodeConfig{
						TaxCodeID: "tax-code-id",
					},
				},
				IntentMutableFields: mutable,
				SettlementMode:      settlementMode,
			}.AsOverridableIntent(),
			Status: status,
			State: flatfee.State{
				AmountAfterProration: alpacadecimal.NewFromInt(10),
			},
		},
	}
}

type flatFeeSetOverrideStateMachineAdapter struct {
	flatfee.Adapter

	charge                   flatfee.Charge
	runBase                  flatfee.RealizationRunBase
	updatedRun               flatfee.UpdateRealizationRunInput
	updateRunCalls           int
	upsertDetailedLinesCalls int
	detachCurrentRunCalls    int
}

func (a *flatFeeSetOverrideStateMachineAdapter) CreateChargeOverride(_ context.Context, charge flatfee.ChargeBase, override flatfee.IntentMutableFields) (flatfee.ChargeBase, error) {
	charge.Intent = flatfee.NewOverridableIntent(charge.Intent.GetBaseIntent(), &override)
	return charge, nil
}

func (a *flatFeeSetOverrideStateMachineAdapter) DeleteChargeOverride(_ context.Context, charge flatfee.ChargeBase) (flatfee.ChargeBase, error) {
	charge.Intent = charge.Intent.GetBaseIntent().AsOverridableIntent()
	return charge, nil
}

func (a *flatFeeSetOverrideStateMachineAdapter) UpdateCharge(_ context.Context, charge flatfee.ChargeBase) (flatfee.ChargeBase, error) {
	a.charge.ChargeBase = charge
	return charge, nil
}

func (a *flatFeeSetOverrideStateMachineAdapter) DeleteCharge(_ context.Context, charge flatfee.Charge) error {
	a.charge = charge
	return nil
}

func (a *flatFeeSetOverrideStateMachineAdapter) DetachCurrentRun(_ context.Context, _ meta.ChargeID) error {
	a.detachCurrentRunCalls++
	return nil
}

func (a *flatFeeSetOverrideStateMachineAdapter) UpdateRealizationRun(_ context.Context, input flatfee.UpdateRealizationRunInput) (flatfee.RealizationRunBase, error) {
	a.updateRunCalls++
	a.updatedRun = input
	if input.ServicePeriod.IsPresent() {
		a.runBase.ServicePeriod = input.ServicePeriod.OrEmpty()
	}
	if input.AmountAfterProration.IsPresent() {
		a.runBase.AmountAfterProration = input.AmountAfterProration.OrEmpty()
	}
	if input.Totals.IsPresent() {
		a.runBase.Totals = input.Totals.OrEmpty()
	}
	if input.NoFiatTransactionRequired.IsPresent() {
		a.runBase.NoFiatTransactionRequired = input.NoFiatTransactionRequired.OrEmpty()
	}

	return a.runBase, nil
}

func (a *flatFeeSetOverrideStateMachineAdapter) UpsertDetailedLines(_ context.Context, _ flatfee.RealizationRunID, _ flatfee.DetailedLines) error {
	a.upsertDetailedLinesCalls++
	return nil
}

type flatFeeSetOverrideTestHandler struct {
	flatfee.Handler
}

type flatFeeSetOverrideTestLineage struct {
	lineage.Service
}

type flatFeeSetOverrideTestRater struct {
	billingrating.Service
}

func (flatFeeSetOverrideTestRater) GenerateDetailedLines(_ billingrating.StandardLineAccessor, _ ...billingrating.GenerateDetailedLinesOption) (billingrating.GenerateDetailedLinesResult, error) {
	return billingrating.GenerateDetailedLinesResult{
		Totals: totals.Totals{Total: alpacadecimal.NewFromInt(10)},
	}, nil
}
