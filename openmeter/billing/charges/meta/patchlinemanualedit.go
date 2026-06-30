package meta

import (
	"errors"
	"fmt"

	"github.com/qmuntal/stateless"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ Patch = (*PatchLineManualEdit)(nil)

type PatchLineManualEdit struct {
	changeSource billing.ChangeSource
	override     billing.InvoiceLineOverride
}

type NewPatchLineManualEditInput struct {
	ChangeSource billing.ChangeSource
	Override     billing.InvoiceLineOverride
}

func (i NewPatchLineManualEditInput) Validate() error {
	var errs []error

	if err := i.ChangeSource.Require(billing.ChangeSourceAPIRequest); err != nil {
		errs = append(errs, fmt.Errorf("change source: %w", err))
	}

	if err := i.Override.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("override: %w", err))
	}

	if err := ValidateInvoiceLineOverrideDoesNotChangeImmutableChargeIntentFields(i.Override); err != nil {
		errs = append(errs, fmt.Errorf("override: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func ValidateInvoiceLineOverrideDoesNotChangeImmutableChargeIntentFields(override billing.InvoiceLineOverride) error {
	lineID := ""
	if override.ExistingLine != nil {
		lineID = override.ExistingLine.GetID()
	}

	// Feature key and tax config are immutable charge intent fields. Letting
	// invoice-line overrides mutate them would make ledger provenance point at
	// a charge whose base billing context no longer matches the edited line.
	if override.ChangesToApply.FeatureKey.IsPresent() {
		return fmt.Errorf("line[%s]: %w", lineID, billing.ErrInvoiceLineFeatureKeyEditNotSupported)
	}

	if override.ChangesToApply.TaxConfig.IsPresent() {
		return fmt.Errorf("line[%s]: %w", lineID, billing.ErrInvoiceLineTaxConfigEditNotSupported)
	}

	return nil
}

func NewPatchLineManualEdit(input NewPatchLineManualEditInput) (PatchLineManualEdit, error) {
	if err := input.Validate(); err != nil {
		return PatchLineManualEdit{}, err
	}

	patch := PatchLineManualEdit{
		changeSource: input.ChangeSource,
		override:     input.Override,
	}
	if err := patch.Validate(); err != nil {
		return PatchLineManualEdit{}, err
	}

	return patch, nil
}

func (p PatchLineManualEdit) GetOverride() billing.InvoiceLineOverride {
	return p.override
}

func (p PatchLineManualEdit) GetChangeSource() billing.ChangeSource {
	return p.changeSource
}

func (p PatchLineManualEdit) GetTargetLayer(intent LayeredIntentReader) (ChangeTarget, error) {
	if err := p.GetChangeSource().Require(billing.ChangeSourceAPIRequest); err != nil {
		return "", fmt.Errorf("change source: %w", err)
	}

	return apiPatchTargetLayer(intent)
}

func (p PatchLineManualEdit) Op() PatchType {
	return PatchTypeLineManualEdit
}

func (p PatchLineManualEdit) Trigger() stateless.Trigger {
	return TriggerLineManualEdit
}

func (p PatchLineManualEdit) Validate() error {
	return NewPatchLineManualEditInput{
		ChangeSource: p.GetChangeSource(),
		Override:     p.GetOverride(),
	}.Validate()
}
