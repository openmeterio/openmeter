package meta

import (
	"errors"
	"fmt"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ Patch = (*PatchClearOverride)(nil)

// PatchClearOverride removes the API-managed override layer so the base intent
// becomes effective again.
type PatchClearOverride struct {
	changeSource billing.ChangeSource
}

type NewPatchClearOverrideInput struct {
	ChangeSource billing.ChangeSource
}

func (i NewPatchClearOverrideInput) Validate() error {
	if err := i.ChangeSource.Require(billing.ChangeSourceAPIRequest); err != nil {
		return models.NewGenericValidationError(fmt.Errorf("change source: %w", err))
	}

	return nil
}

func NewPatchClearOverride(input NewPatchClearOverrideInput) (PatchClearOverride, error) {
	if err := input.Validate(); err != nil {
		return PatchClearOverride{}, err
	}

	return PatchClearOverride{changeSource: input.ChangeSource}, nil
}

func (p PatchClearOverride) GetTargetLayer(intent LayeredIntentReader) (ChangeTarget, error) {
	if err := p.changeSource.Require(billing.ChangeSourceAPIRequest); err != nil {
		return "", fmt.Errorf("change source: %w", err)
	}

	if intent == nil {
		return "", errors.New("intent is required")
	}

	return ChangeTargetOverride, nil
}

func (p PatchClearOverride) Op() PatchType {
	return PatchTypeClearOverride
}

func (p PatchClearOverride) Trigger() stateless.Trigger {
	return TriggerClearOverride
}

func (p PatchClearOverride) Validate() error {
	return NewPatchClearOverrideInput{ChangeSource: p.changeSource}.Validate()
}
