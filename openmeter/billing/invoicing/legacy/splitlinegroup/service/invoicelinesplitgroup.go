package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/invoicing/legacy/splitlinegroup"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *Service) GetSplitLineGroupsForSubscription(ctx context.Context, input billing.GetLinesForSubscriptionInput) ([]splitlinegroup.SplitLineHierarchy, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]splitlinegroup.SplitLineHierarchy, error) {
		return s.adapter.GetSplitLineGroupsForSubscription(ctx, input)
	})
}

func (s *Service) DeleteSplitLineGroup(ctx context.Context, input splitlinegroup.DeleteSplitLineGroupInput) error {
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
		for _, childLine := range splitLineGroup.StandardLines {
			if childLine.GetDeletedAt() == nil {
				return billing.ValidationError{
					Err: fmt.Errorf("child lines must be deleted, to delete split line group %s", input.ID),
				}
			}
		}

		if splitLineGroup.GatheringLine != nil {
			if splitLineGroup.GatheringLine.GetDeletedAt() == nil {
				return billing.ValidationError{
					Err: fmt.Errorf("gathering line must be deleted, to delete split line group %s", input.ID),
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

func (s *Service) GetSplitLineGroup(ctx context.Context, input splitlinegroup.GetSplitLineGroupInput) (splitlinegroup.SplitLineHierarchy, error) {
	if err := input.Validate(); err != nil {
		return splitlinegroup.SplitLineHierarchy{}, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (splitlinegroup.SplitLineHierarchy, error) {
		return s.adapter.GetSplitLineGroup(ctx, input)
	})
}
