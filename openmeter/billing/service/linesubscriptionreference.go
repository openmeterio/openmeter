package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ billing.SubscriptionReferenceService = (*Service)(nil)

func (s *Service) SetGatheringLineSubscriptionReferenceByChargeID(ctx context.Context, input billing.SetLineSubscriptionReferenceByChargeIDInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{Err: err}
	}

	if err := s.adapter.SetGatheringLineSubscriptionReferenceByChargeID(ctx, input); err != nil {
		return fmt.Errorf("setting gathering line subscription reference: %w", err)
	}

	return nil
}

func (s *Service) SetStandardLineSubscriptionReferenceByChargeID(ctx context.Context, input billing.SetLineSubscriptionReferenceByChargeIDInput) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{Err: err}
	}

	if err := s.adapter.SetStandardLineSubscriptionReferenceByChargeID(ctx, input); err != nil {
		return fmt.Errorf("setting standard line subscription reference: %w", err)
	}

	return nil
}
