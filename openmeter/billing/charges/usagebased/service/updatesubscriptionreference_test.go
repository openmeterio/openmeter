package service

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/subscription/validators/itemreference"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestUpdateSubscriptionReference(t *testing.T) {
	current := meta.SubscriptionReference{
		SubscriptionID: "subscription-1",
		PhaseID:        "phase-1",
		ItemID:         "item-1",
	}
	updated := meta.SubscriptionReference{
		SubscriptionID: "subscription-1",
		PhaseID:        "phase-2",
		ItemID:         "item-2",
	}
	patch, err := meta.NewPatchUpdateSubscriptionReference(meta.NewPatchUpdateSubscriptionReferenceInput{
		PhaseID:            lo.ToPtr(updated.PhaseID),
		SubscriptionItemID: lo.ToPtr(updated.ItemID),
	})
	require.NoError(t, err)

	t.Run("updates the base reference and preserves the override", func(t *testing.T) {
		// Given a subscription-managed charge with an active customer override.
		override := usagebased.IntentMutableFields{}
		charge := usagebased.Charge{ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace-1"},
				ID:              "charge-1",
			},
			Intent: usagebased.NewOverridableIntent(usagebased.Intent{
				Intent: meta.Intent{Subscription: &current},
			}, &override),
		}}
		updater := &subscriptionReferenceUpdaterStub{}
		validator := &itemReferenceValidatorStub{}
		lineReferenceInput := billing.SetLineSubscriptionReferenceByChargeIDInput{
			Namespace:      charge.Namespace,
			ChargeID:       charge.ID,
			SubscriptionID: updated.SubscriptionID,
			PhaseID:        updated.PhaseID,
			ItemID:         updated.ItemID,
		}
		lineReferences := newLineReferenceServiceMock(t, lineReferenceInput)

		// When the system repairs its subscription ownership reference.
		err := (&service{adapter: updater, itemReferenceValidator: validator, lineSubscriptionReferenceService: lineReferences}).updateSubscriptionReference(t.Context(), &charge, patch)

		// Then only the base attribution changes and the override remains present.
		require.NoError(t, err)
		require.Equal(t, meta.UpdateSubscriptionReferenceInput{
			ChargeID: charge.GetChargeID(),
			Target:   updated,
		}, updater.input)
		require.Equal(t, updated, *charge.Intent.GetSubscription())
		require.NotNil(t, charge.Intent.GetOverrideLayerMutableFields())
		require.Equal(t, itemreference.ValidateInput{
			Namespace:      charge.Namespace,
			SubscriptionID: updated.SubscriptionID,
			PhaseID:        updated.PhaseID,
			ItemID:         updated.ItemID,
		}, validator.input)
	})

	t.Run("updates only the requested item ID", func(t *testing.T) {
		// Given an item-only repair for a charge whose phase remains valid.
		itemPatch, err := meta.NewPatchUpdateSubscriptionReference(meta.NewPatchUpdateSubscriptionReferenceInput{
			SubscriptionItemID: lo.ToPtr(updated.ItemID),
		})
		require.NoError(t, err)
		charge := usagebased.Charge{ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace-1"},
				ID:              "charge-1",
			},
			Intent: usagebased.NewOverridableIntent(usagebased.Intent{
				Intent: meta.Intent{Subscription: &current},
			}, nil),
		}}
		updater := &subscriptionReferenceUpdaterStub{}
		expected := current
		expected.ItemID = updated.ItemID
		lineReferences := newLineReferenceServiceMock(t, billing.SetLineSubscriptionReferenceByChargeIDInput{
			Namespace:      charge.Namespace,
			ChargeID:       charge.ID,
			SubscriptionID: expected.SubscriptionID,
			PhaseID:        expected.PhaseID,
			ItemID:         expected.ItemID,
		})

		// When the item-only repair is applied.
		err = (&service{adapter: updater, itemReferenceValidator: &itemReferenceValidatorStub{}, lineSubscriptionReferenceService: lineReferences}).updateSubscriptionReference(t.Context(), &charge, itemPatch)

		// Then the current subscription and phase IDs form the final reference sent for validation.
		require.NoError(t, err)
		require.Equal(t, expected, updater.input.Target)
		require.Equal(t, expected, *charge.Intent.GetSubscription())
	})

	t.Run("is idempotent when the update is already applied", func(t *testing.T) {
		// Given the charge already contains the patch's target reference.
		charge := usagebased.Charge{ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace-1"},
				ID:              "charge-1",
			},
			Intent: usagebased.NewOverridableIntent(usagebased.Intent{
				Intent: meta.Intent{Subscription: &updated},
			}, nil),
		}}
		updater := &subscriptionReferenceUpdaterStub{}
		lineReferences := newLineReferenceServiceMock(t, billing.SetLineSubscriptionReferenceByChargeIDInput{
			Namespace:      charge.Namespace,
			ChargeID:       charge.ID,
			SubscriptionID: updated.SubscriptionID,
			PhaseID:        updated.PhaseID,
			ItemID:         updated.ItemID,
		})

		// When the same repair is retried, then it succeeds without changing the reference.
		require.NoError(t, (&service{adapter: updater, itemReferenceValidator: &itemReferenceValidatorStub{}, lineSubscriptionReferenceService: lineReferences}).updateSubscriptionReference(t.Context(), &charge, patch))
		require.Equal(t, updated, updater.input.Target)
		require.Equal(t, updated, *charge.Intent.GetSubscription())
	})

	t.Run("rejects an invalid final relationship", func(t *testing.T) {
		// Given persistence reports that the patched IDs do not form a valid subscription graph.
		charge := usagebased.Charge{ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace-1"},
				ID:              "charge-1",
			},
			Intent: usagebased.NewOverridableIntent(usagebased.Intent{
				Intent: meta.Intent{Subscription: &current},
			}, nil),
		}}
		updater := &subscriptionReferenceUpdaterStub{}
		validator := &itemReferenceValidatorStub{
			err: models.NewGenericPreConditionFailedError(errors.New("invalid subscription reference")),
		}

		// When the repair is applied.
		err := (&service{adapter: updater, itemReferenceValidator: validator}).updateSubscriptionReference(t.Context(), &charge, patch)

		// Then the precondition fails and the original reference is preserved.
		require.Error(t, err)
		require.True(t, models.IsGenericPreConditionFailedError(err))
		require.Equal(t, current, *charge.Intent.GetSubscription())
		require.Zero(t, updater.calls)
	})

	t.Run("rejects a charge without a subscription", func(t *testing.T) {
		// Given a manually managed charge without subscription attribution.
		charge := usagebased.Charge{ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace-1"},
				ID:              "charge-1",
			},
			Intent: usagebased.NewOverridableIntent(usagebased.Intent{}, nil),
		}}

		// When a subscription repair is attempted, then the precondition fails.
		err := (&service{adapter: &subscriptionReferenceUpdaterStub{}, itemReferenceValidator: &itemReferenceValidatorStub{}}).updateSubscriptionReference(t.Context(), &charge, patch)

		require.Error(t, err)
		require.True(t, models.IsGenericPreConditionFailedError(err))
	})
}

type subscriptionReferenceUpdaterStub struct {
	usagebased.Adapter

	input meta.UpdateSubscriptionReferenceInput
	err   error
	calls int
}

func (s *subscriptionReferenceUpdaterStub) UpdateSubscriptionReference(_ context.Context, input meta.UpdateSubscriptionReferenceInput) error {
	s.calls++
	s.input = input

	return s.err
}

type itemReferenceValidatorStub struct {
	input itemreference.ValidateInput
	err   error
}

type lineReferenceServiceMock struct {
	mock.Mock
}

func newLineReferenceServiceMock(t *testing.T, input billing.SetLineSubscriptionReferenceByChargeIDInput) *lineReferenceServiceMock {
	t.Helper()

	service := &lineReferenceServiceMock{}
	service.On("SetGatheringLineSubscriptionReferenceByChargeID", mock.Anything, input).Return(nil).Once()
	service.On("SetStandardLineSubscriptionReferenceByChargeID", mock.Anything, input).Return(nil).Once()
	t.Cleanup(func() {
		service.AssertExpectations(t)
	})

	return service
}

func (s *lineReferenceServiceMock) SetGatheringLineSubscriptionReferenceByChargeID(ctx context.Context, input billing.SetLineSubscriptionReferenceByChargeIDInput) error {
	return s.Called(ctx, input).Error(0)
}

func (s *lineReferenceServiceMock) SetStandardLineSubscriptionReferenceByChargeID(ctx context.Context, input billing.SetLineSubscriptionReferenceByChargeIDInput) error {
	return s.Called(ctx, input).Error(0)
}

func (s *itemReferenceValidatorStub) ValidateItemReference(_ context.Context, input itemreference.ValidateInput) error {
	s.input = input

	return s.err
}
