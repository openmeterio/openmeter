package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *Service) CreateSplitLineGroup(ctx context.Context, input splitlinegroup.SplitLineGroupCreate) (splitlinegroup.SplitLineGroup, error) {
	if err := input.Validate(); err != nil {
		return splitlinegroup.SplitLineGroup{}, billing.ValidationError{Err: err}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (splitlinegroup.SplitLineGroup, error) {
		return s.adapter.CreateSplitLineGroup(ctx, input)
	})
}

func (s *Service) GetSplitLineGroups(ctx context.Context, input splitlinegroup.GetSplitLineGroupsInput) ([]splitlinegroup.SplitLineHierarchy, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{Err: err}
	}

	// The adapter reuses the caller's transaction when present. Standalone legacy
	// calculations retain their existing read boundary without opening a transaction.
	return s.adapter.GetSplitLineGroups(ctx, input)
}

func (s *Service) GetSplitLineGroupsForSubscription(ctx context.Context, input splitlinegroup.GetSplitLineGroupsForSubscriptionInput) ([]splitlinegroup.SplitLineHierarchy, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]splitlinegroup.SplitLineHierarchy, error) {
		return s.adapter.GetSplitLineGroupsForSubscription(ctx, input)
	})
}

func (s *Service) DeleteSplitLineGroup(ctx context.Context, input models.NamespacedID) error {
	if err := input.Validate(); err != nil {
		return billing.ValidationError{
			Err: err,
		}
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		// Let's load the split line group and validate that all of it's children are also deleted
		splitLineGroup, err := s.adapter.GetSplitLineGroup(ctx, input)
		if err != nil {
			return err
		}

		if splitLineGroup.Group.DeletedAt != nil {
			return nil
		}

		// Let's validate that all of it's children are also deleted
		for _, childLine := range splitLineGroup.Lines {
			if childLine.DeletedAt == nil {
				return billing.ValidationError{
					Err: fmt.Errorf("child lines must be deleted, to delete split line group %s", input.ID),
				}
			}
		}

		return s.adapter.DeleteSplitLineGroup(ctx, input)
	})
}

func (s *Service) UpdateSplitLineGroup(ctx context.Context, input splitlinegroup.SplitLineGroupUpdate) (splitlinegroup.SplitLineGroup, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (splitlinegroup.SplitLineGroup, error) {
		splitLineGroup, err := s.adapter.GetSplitLineGroup(ctx, input.NamespacedID)
		if err != nil {
			return splitlinegroup.SplitLineGroup{}, err
		}

		if err := input.ValidateWithPrice(splitLineGroup.Group.Price); err != nil {
			return splitlinegroup.SplitLineGroup{}, billing.ValidationError{
				Err: err,
			}
		}

		return s.adapter.UpdateSplitLineGroup(ctx, input)
	})
}

func (s *Service) GetSplitLineGroup(ctx context.Context, input models.NamespacedID) (splitlinegroup.SplitLineHierarchy, error) {
	if err := input.Validate(); err != nil {
		return splitlinegroup.SplitLineHierarchy{}, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (splitlinegroup.SplitLineHierarchy, error) {
		return s.adapter.GetSplitLineGroup(ctx, input)
	})
}
