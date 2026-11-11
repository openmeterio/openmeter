package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

func (s *service) publishInputChanges(ctx context.Context, owners []customer.CustomerID) error {
	if s.inputChangePublisher == nil {
		return nil
	}

	for _, owner := range lo.Uniq(owners) {
		if err := s.inputChangePublisher.Publish(ctx, charges.ChargeInputsChangedEvent{Namespace: owner.Namespace, CustomerID: owner.ID}); err != nil {
			return fmt.Errorf("publish charge input change: %w", err)
		}
	}

	return nil
}
