package meta

import (
	"errors"
	"fmt"
	"time"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

// SetOverrideMutableFields captures the behavior needed to validate and
// snapshot a concrete charge's mutable override intent.
type SetOverrideMutableFields[T any] interface {
	models.Validator

	Clone() T
	GetIntentDeletedAt() *time.Time
}

// PatchSetOverride replaces the complete mutable override intent for a charge.
type PatchSetOverride[T SetOverrideMutableFields[T]] struct {
	changeSource        billing.ChangeSource
	intentMutableFields T
}

// NewPatchSetOverrideInput configures a typed override patch.
type NewPatchSetOverrideInput[T SetOverrideMutableFields[T]] struct {
	ChangeSource        billing.ChangeSource
	IntentMutableFields T
}

func (i NewPatchSetOverrideInput[T]) Validate() error {
	var errs []error

	if err := i.ChangeSource.Require(billing.ChangeSourceAPIRequest); err != nil {
		errs = append(errs, fmt.Errorf("change source: %w", err))
	}

	if i.IntentMutableFields.GetIntentDeletedAt() != nil {
		errs = append(errs, errors.New("intent deleted at cannot be set by an override update"))
	}

	if err := i.IntentMutableFields.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent mutable fields: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func NewPatchSetOverride[T SetOverrideMutableFields[T]](input NewPatchSetOverrideInput[T]) (PatchSetOverride[T], error) {
	if err := input.Validate(); err != nil {
		return PatchSetOverride[T]{}, err
	}

	return PatchSetOverride[T]{
		changeSource:        input.ChangeSource,
		intentMutableFields: input.IntentMutableFields.Clone(),
	}, nil
}

func (p PatchSetOverride[T]) GetIntentMutableFields() T {
	return p.intentMutableFields.Clone()
}

func (p PatchSetOverride[T]) GetTargetLayer(intent LayeredIntentReader) (ChangeTarget, error) {
	if err := p.changeSource.Require(billing.ChangeSourceAPIRequest); err != nil {
		return "", fmt.Errorf("change source: %w", err)
	}

	if intent == nil {
		return "", errors.New("intent is required")
	}

	return ChangeTargetOverride, nil
}

func (p PatchSetOverride[T]) Op() PatchType {
	return PatchTypeSetOverride
}

func (p PatchSetOverride[T]) Trigger() stateless.Trigger {
	return TriggerSetOverride
}

func (p PatchSetOverride[T]) Validate() error {
	return NewPatchSetOverrideInput[T]{
		ChangeSource:        p.changeSource,
		IntentMutableFields: p.intentMutableFields,
	}.Validate()
}
