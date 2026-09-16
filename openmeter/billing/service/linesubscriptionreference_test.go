package billingservice

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func TestSetLineSubscriptionReferenceByChargeID(t *testing.T) {
	input := billing.SetLineSubscriptionReferenceByChargeIDInput{
		Namespace:      "namespace-1",
		ChargeID:       "charge-1",
		SubscriptionID: "subscription-1",
		PhaseID:        "phase-1",
		ItemID:         "item-1",
	}

	t.Run("gathering lines", func(t *testing.T) {
		adapter := &lineSubscriptionReferenceAdapterStub{}
		service := &Service{adapter: adapter}

		require.NoError(t, service.SetGatheringLineSubscriptionReferenceByChargeID(t.Context(), input))
		require.Equal(t, input, adapter.gatheringInput)
		require.Zero(t, adapter.standardCalls)
	})

	t.Run("standard lines", func(t *testing.T) {
		adapter := &lineSubscriptionReferenceAdapterStub{}
		service := &Service{adapter: adapter}

		require.NoError(t, service.SetStandardLineSubscriptionReferenceByChargeID(t.Context(), input))
		require.Equal(t, input, adapter.standardInput)
		require.Zero(t, adapter.gatheringCalls)
	})

	t.Run("invalid input", func(t *testing.T) {
		adapter := &lineSubscriptionReferenceAdapterStub{}
		service := &Service{adapter: adapter}

		err := service.SetGatheringLineSubscriptionReferenceByChargeID(t.Context(), billing.SetLineSubscriptionReferenceByChargeIDInput{})
		require.Error(t, err)
		require.Zero(t, adapter.gatheringCalls)
	})
}

type lineSubscriptionReferenceAdapterStub struct {
	billing.Adapter

	gatheringInput billing.SetLineSubscriptionReferenceByChargeIDInput
	standardInput  billing.SetLineSubscriptionReferenceByChargeIDInput
	gatheringCalls int
	standardCalls  int
}

func (s *lineSubscriptionReferenceAdapterStub) SetGatheringLineSubscriptionReferenceByChargeID(_ context.Context, input billing.SetLineSubscriptionReferenceByChargeIDInput) error {
	s.gatheringCalls++
	s.gatheringInput = input

	return nil
}

func (s *lineSubscriptionReferenceAdapterStub) SetStandardLineSubscriptionReferenceByChargeID(_ context.Context, input billing.SetLineSubscriptionReferenceByChargeIDInput) error {
	s.standardCalls++
	s.standardInput = input

	return nil
}
