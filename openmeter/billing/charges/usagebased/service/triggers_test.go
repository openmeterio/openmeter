package service

import (
	"testing"
	"time"

	"github.com/samber/lo"
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
		// - The state-machine configuration is assembled.
		// then:
		// - It retains the supplied snapshot without consulting the feature service.
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

		resolved, err := config.FeatureMeters.Get(charge)
		require.NoError(t, err)
		require.Equal(t, featureMeter, resolved)
	})

	t.Run("when the supplied collection omits the charge feature", func(t *testing.T) {
		// given:
		// - Advancement receives an authoritative feature-meter collection without the charge feature.
		// when:
		// - The state-machine configuration is assembled.
		// then:
		// - It retains the supplied snapshot, and its lookup error does not fall back to the feature service.
		config, err := (&service{}).getStateMachineConfigForChargeWithHints(t.Context(), charge, usagebased.AdvanceChargeInput{
			CustomerOverride: mo.Some(billing.CustomerOverrideWithDetails{}),
			FeatureMeters: mo.Some[billingfeaturemeter.FeatureMeters](featuremeterservice.FeatureMeterCollection{
				ByKey: map[string]billingfeaturemeter.FeatureMeter{},
			}),
		})
		require.NoError(t, err)

		_, err = config.FeatureMeters.Get(charge)
		require.ErrorContains(t, err, "invoice line: feature not found")
		require.ErrorContains(t, err, "[feature_key=feature-key]")
	})
}

func newFeatureMetersForChargeTest(charge usagebased.Charge) billingfeaturemeter.FeatureMeters {
	reference := charge.GetFeatureMeterRef()
	if reference == nil {
		return featuremeterservice.FeatureMeterCollection{
			ByKey: map[string]billingfeaturemeter.FeatureMeter{},
			ByID:  map[string]billingfeaturemeter.FeatureMeter{},
		}
	}

	featureID := reference.IDOrKey.ID
	if featureID == "" {
		featureID = "feature-id"
	}

	featureMeter := billingfeaturemeter.FeatureMeter{
		Feature: feature.Feature{
			ID:  featureID,
			Key: reference.IDOrKey.Key,
		},
		Meter: &meter.Meter{},
	}
	collection := featuremeterservice.FeatureMeterCollection{
		ByKey: map[string]billingfeaturemeter.FeatureMeter{},
		ByID:  map[string]billingfeaturemeter.FeatureMeter{featureID: featureMeter},
	}
	if reference.IDOrKey.Key != "" {
		collection.ByKey[reference.IDOrKey.Key] = featureMeter
	}

	return collection
}

func TestSyncFeatureIDFromFeatureMeterReconcilesProductCatalogIssue(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 7, 8, 8, 45, 3, 0, time.UTC),
		To:   time.Date(2026, 8, 8, 8, 45, 3, 0, time.UTC),
	}
	productCatalogIssue := billing.ValidationIssue{
		Code:      billing.ErrInvoiceLineFeatureHasNoMeters.Code,
		Component: billing.ValidationComponentProductCatalog,
	}
	unrelatedIssue := billing.ValidationIssue{
		Code:      "unrelated",
		Component: "unrelated",
	}

	for _, featureID := range []string{"", "feature-id"} {
		name := "unpinned feature"
		if featureID != "" {
			name = "pinned feature"
		}

		t.Run(name, func(t *testing.T) {
			charge := usagebased.Charge{ChargeBase: usagebased.ChargeBase{
				ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
				Intent:          newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
				Status:          usagebased.StatusActive,
				ValidationIssues: billing.ValidationIssues{
					unrelatedIssue,
					productCatalogIssue,
				},
				State: usagebased.State{FeatureID: featureID},
			}}
			machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)

			err := machine.SyncFeatureIDFromFeatureMeter(t.Context())

			require.NoError(t, err)
			require.Equal(t, "feature-id", machine.Charge.State.FeatureID)
			require.Equal(t, billing.ValidationIssues{unrelatedIssue}, machine.Charge.ValidationIssues)
		})
	}

	t.Run("failed resolution keeps the existing issues", func(t *testing.T) {
		charge := usagebased.Charge{ChargeBase: usagebased.ChargeBase{
			ManagedResource: newUsageBasedChargeTestManagedResource("charge-id"),
			Intent:          newUsageBasedIntentForCreditThenInvoiceTest(t, servicePeriod),
			Status:          usagebased.StatusActive,
			ValidationIssues: billing.ValidationIssues{
				unrelatedIssue,
				productCatalogIssue,
			},
			State: usagebased.State{FeatureID: "feature-id"},
		}}
		machine := newCreditThenInvoiceStateMachineWithChargeForTest(t, charge)
		machine.FeatureMeters = featuremeterservice.FeatureMeterCollection{
			ByKey: map[string]billingfeaturemeter.FeatureMeter{},
			ByID:  map[string]billingfeaturemeter.FeatureMeter{},
		}

		err := machine.SyncFeatureIDFromFeatureMeter(t.Context())

		require.ErrorIs(t, err, billing.ErrInvoiceLineFeatureNotFound)
		require.Equal(t, "feature-id", machine.Charge.State.FeatureID)
		require.Equal(t, billing.ValidationIssues{unrelatedIssue, productCatalogIssue}, machine.Charge.ValidationIssues)
	})
}

func TestClearInvoiceAssignmentIssueWithoutCurrentRun(t *testing.T) {
	lineEngineIssue := billing.ValidationIssue{
		Code:      usagebased.ValidationIssueCodeInvoiceAssignmentBlockedActiveRun,
		Component: usagebased.ValidationIssueComponentLineEngine,
	}
	productCatalogIssue := billing.ValidationIssue{
		Code:      billing.ErrInvoiceLineFeatureHasNoMeters.Code,
		Component: billing.ValidationComponentProductCatalog,
	}
	unrelatedIssue := billing.ValidationIssue{
		Code:      "unrelated",
		Component: "unrelated",
	}

	t.Run("without a current run", func(t *testing.T) {
		base := usagebased.ChargeBase{
			ValidationIssues: billing.ValidationIssues{productCatalogIssue, lineEngineIssue, unrelatedIssue},
		}

		base = clearInvoiceAssignmentIssueWithoutCurrentRun(base)

		require.Equal(t, billing.ValidationIssues{productCatalogIssue, unrelatedIssue}, base.ValidationIssues)
	})

	t.Run("with a current run", func(t *testing.T) {
		base := usagebased.ChargeBase{
			ValidationIssues: billing.ValidationIssues{productCatalogIssue, lineEngineIssue, unrelatedIssue},
			State:            usagebased.State{CurrentRealizationRunID: lo.ToPtr("run-id")},
		}

		base = clearInvoiceAssignmentIssueWithoutCurrentRun(base)

		require.Equal(t, billing.ValidationIssues{productCatalogIssue, lineEngineIssue, unrelatedIssue}, base.ValidationIssues)
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
