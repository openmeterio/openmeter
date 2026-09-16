package itemreference

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ValidateInput struct {
	Namespace      string
	SubscriptionID string
	PhaseID        string
	ItemID         string
}

var _ models.Validator = ValidateInput{}

func (i ValidateInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if i.SubscriptionID == "" {
		errs = append(errs, errors.New("subscription ID is required"))
	}
	if i.PhaseID == "" {
		errs = append(errs, errors.New("phase ID is required"))
	}
	if i.ItemID == "" {
		errs = append(errs, errors.New("item ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Validator interface {
	ValidateItemReference(ctx context.Context, input ValidateInput) error
}

type Repository interface {
	transaction.Creator
	IsValidItemReference(ctx context.Context, input ValidateInput) (bool, error)
}

func NewValidator(repository Repository) (Validator, error) {
	if repository == nil {
		return nil, errors.New("repository is required")
	}

	return &validator{repository: repository}, nil
}

type validator struct {
	repository Repository
}

func (v *validator) ValidateItemReference(ctx context.Context, input ValidateInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validate input: %w", err)
	}

	return transaction.RunWithNoValue(ctx, v.repository, func(ctx context.Context) error {
		valid, err := v.repository.IsValidItemReference(ctx, input)
		if err != nil {
			return fmt.Errorf("validate subscription item reference: %w", err)
		}

		if !valid {
			return models.NewGenericPreConditionFailedError(fmt.Errorf(
				"subscription item reference is invalid [subscription_id=%s phase_id=%s item_id=%s]",
				input.SubscriptionID,
				input.PhaseID,
				input.ItemID,
			))
		}

		return nil
	})
}
