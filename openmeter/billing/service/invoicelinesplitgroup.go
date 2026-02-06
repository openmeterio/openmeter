package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *Service) DeleteSplitLineGroup(ctx context.Context, input billing.DeleteSplitLineGroupInput) error {
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
			if childLine.Line.GetDeletedAt() == nil {
				return billing.ValidationError{
					Err: fmt.Errorf("child lines must be deleted, to delete split line group %s", input.ID),
				}
			}
		}

		return s.adapter.DeleteSplitLineGroup(ctx, input)
	})
}

func (s *Service) UpdateSplitLineGroup(ctx context.Context, input billing.UpdateSplitLineGroupInput) (billing.SplitLineGroup, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.SplitLineGroup, error) {
		splitLineGroup, err := s.adapter.GetSplitLineGroup(ctx, input.NamespacedID)
		if err != nil {
			return billing.SplitLineGroup{}, err
		}

		if err := input.ValidateWithPrice(splitLineGroup.Group.Price); err != nil {
			return billing.SplitLineGroup{}, billing.ValidationError{
				Err: err,
			}
		}

		return s.adapter.UpdateSplitLineGroup(ctx, input)
	})
}

func (s *Service) GetSplitLineGroup(ctx context.Context, input billing.GetSplitLineGroupInput) (billing.SplitLineHierarchy, error) {
	if err := input.Validate(); err != nil {
		return billing.SplitLineHierarchy{}, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.SplitLineHierarchy, error) {
		return s.adapter.GetSplitLineGroup(ctx, input)
	})
}
