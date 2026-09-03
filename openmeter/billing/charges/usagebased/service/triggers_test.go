package service

import (
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	featuremeterservice "github.com/openmeterio/openmeter/openmeter/billing/featuremeter/service"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestGetStateMachineConfigUsesAuthoritativeFeatureMeters(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 7, 8, 8, 45, 3, 0, time.UTC),
		To:   time.Date(2026, 8, 8, 8, 45, 3, 0, time.UTC),
	}
	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
			Status:          usagebased.StatusCreated,
		},
	}

	t.Run("when the supplied collection contains the charge feature", func(t *testing.T) {
		// given:
		// - Advancement receives an authoritative feature-meter collection containing the charge feature.
		// when:
		// - The state-machine configuration resolves the charge feature.
		// then:
		// - It uses the supplied snapshot without consulting the feature service.
		featureMeter := billingfeaturemeter.FeatureMeter{
			Feature: feature.Feature{
				ID:  "feature-id",
				Key: "feature-key",
			},
			Meter: &meter.Meter{},
		}
		featureMeters := featuremeterservice.FeatureMeterCollection{
			ByKey: map[string]billingfeaturemeter.FeatureMeter{
				featureMeter.Feature.Key: featureMeter,
			},
		}

		config, err := (&service{}).getStateMachineConfigForChargeWithHints(t.Context(), charge, usagebased.AdvanceChargeInput{
			CustomerOverride: mo.Some(billing.CustomerOverrideWithDetails{}),
			FeatureMeters:    mo.Some[billingfeaturemeter.FeatureMeters](featureMeters),
		})
		require.NoError(t, err)
		require.Equal(t, featureMeter, config.FeatureMeter)
	})

	t.Run("when the supplied collection omits the charge feature", func(t *testing.T) {
		// given:
		// - Advancement receives an authoritative feature-meter collection without the charge feature.
		// when:
		// - The state-machine configuration resolves the charge feature.
		// then:
		// - It returns the snapshot lookup error instead of falling back to the feature service.
		_, err := (&service{}).getStateMachineConfigForChargeWithHints(t.Context(), charge, usagebased.AdvanceChargeInput{
			CustomerOverride: mo.Some(billing.CustomerOverrideWithDetails{}),
			FeatureMeters: mo.Some[billingfeaturemeter.FeatureMeters](featuremeterservice.FeatureMeterCollection{
				ByKey: map[string]billingfeaturemeter.FeatureMeter{},
			}),
		})
		require.ErrorContains(t, err, "feature[feature-key]: invoice line: feature not found")
	})
}

func TestApplyBaseIntentPatchForOverriddenChargeShrinksDeletedEffectiveCharge(t *testing.T) {
	baseServicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 7, 8, 8, 45, 3, 0, time.UTC),
		To:   time.Date(2026, 8, 8, 8, 45, 3, 0, time.UTC),
	}
	shrunkServicePeriodTo := time.Date(2026, 7, 8, 17, 18, 9, 0, time.UTC)
	deletedAt := time.Date(2026, 7, 8, 17, 17, 50, 0, time.UTC)

	baseIntent := newUsageBasedIntentForCreditThenInvoiceTest(t, baseServicePeriod)
	override := baseIntent.GetBaseIntent().IntentMutableFields.Clone()
	override.IntentDeletedAt = &deletedAt

	charge := usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          usagebased.NewOverridableIntent(baseIntent.GetBaseIntent(), &override),
			Status:          usagebased.StatusDeleted,
			State: usagebased.State{
				FeatureID:    "feature-id",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
	}

	patch, err := meta.NewPatchShrink(meta.NewPatchShrinkInput{
		ChangeSource:           billing.ChangeSourceSystem,
		NewServicePeriodTo:     shrunkServicePeriodTo,
		NewFullServicePeriodTo: baseServicePeriod.To,
		NewBillingPeriodTo:     shrunkServicePeriodTo,
		NewInvoiceAt:           shrunkServicePeriodTo,
	})
	require.NoError(t, err)

	updatedCharge, err := applyBaseIntentPatchForOverriddenCharge(charge, patch)
	require.NoError(t, err)
	require.NotNil(t, updatedCharge)

	require.Equal(t, usagebased.StatusDeleted, updatedCharge.Status)
	require.Equal(t, shrunkServicePeriodTo, updatedCharge.Intent.GetBaseIntent().ServicePeriod.To)
	require.Equal(t, shrunkServicePeriodTo, updatedCharge.Intent.GetBaseIntent().BillingPeriod.To)
	require.Equal(t, baseServicePeriod.To, updatedCharge.Intent.GetBaseIntent().FullServicePeriod.To)
	require.Equal(t, shrunkServicePeriodTo, updatedCharge.Intent.GetBaseIntent().InvoiceAt)

	overrideAfterPatch := updatedCharge.Intent.GetOverrideLayerMutableFields()
	require.NotNil(t, overrideAfterPatch)
	require.NotNil(t, overrideAfterPatch.IntentDeletedAt)
	require.Equal(t, deletedAt, *overrideAfterPatch.IntentDeletedAt)
	require.Equal(t, baseServicePeriod.To, overrideAfterPatch.ServicePeriod.To)
}

func TestRejectHiddenIntentTargetRejectsBaseLayerWithOverride(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 7, 8, 8, 45, 3, 0, time.UTC),
		To:   time.Date(2026, 8, 8, 8, 45, 3, 0, time.UTC),
	}
	baseIntent := newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod)
	override := baseIntent.GetBaseIntent().IntentMutableFields.Clone()

	machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          usagebased.NewOverridableIntent(baseIntent.GetBaseIntent(), &override),
			Status:          usagebased.StatusActive,
		},
	})

	err := machine.rejectHiddenIntentTarget(meta.ChangeTargetBase)
	require.Error(t, err)
	require.True(t, models.IsGenericPreConditionFailedError(err))
	require.ErrorContains(t, err, "cannot mutate hidden base intent while override intent is active")

	require.NoError(t, machine.rejectHiddenIntentTarget(meta.ChangeTargetOverride))
}
