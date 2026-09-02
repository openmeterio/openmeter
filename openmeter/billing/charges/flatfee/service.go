package flatfee

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	FlatFeeService
	GetLineEngine() billing.LineEngine
}

type FlatFeeService interface {
	// Create returns one result for each input intent, preserving input order.
	// Invoice-line create flows rely on this to merge charge target state back
	// onto the matching billing-preallocated line identity.
	Create(ctx context.Context, input CreateInput) ([]ChargeWithGatheringLine, error)
	// GetByIDs loads flat-fee charges. Request realization expansions when the
	// caller needs invoice-line, credit-allocation, or payment lifecycle state.
	GetByIDs(ctx context.Context, input GetByIDsInput) ([]Charge, error)
	// GetByID loads one flat-fee charge. Effective behavior may come from an
	// override layer, while subscription sync should compare the base intent.
	GetByID(ctx context.Context, input GetByIDInput) (Charge, error)
	// UpdateSubscriptionItemID repairs subscription ownership metadata on the
	// base intent; it must not rewrite an active customer-facing override layer.
	UpdateSubscriptionItemID(ctx context.Context, charge Charge, newSubscriptionItemID string) (Charge, error)
	// AdvanceCharge drives one charge until invoice patches are emitted or its
	// lifecycle becomes stable. Callers must apply returned invoice patches
	// before resuming when CanAdvance is true.
	AdvanceCharge(ctx context.Context, input AdvanceChargeInput) (meta.TriggerPatchResult[Charge], error)
	// TriggerPatch applies an explicit base/override target patch and advances
	// until invoice patches are emitted or the lifecycle becomes stable. Callers
	// that apply returned invoice patches must resume with AdvanceCharge.
	TriggerPatch(ctx context.Context, charge meta.ChargeID, patch meta.Patch) (meta.TriggerPatchResult[Charge], error)
}

type CreateInput struct {
	Namespace     string
	Intents       []Intent
	FeatureMeters billingfeaturemeter.FeatureMeters
}

func (i CreateInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for idx, intent := range i.Intents {
		if err := intent.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("intent [%d]: %w", idx, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ChargeWithGatheringLine struct {
	Charge                Charge
	GatheringLineToCreate *billing.GatheringLine
}

type GetByMetasInput struct {
	Namespace string
	Expands   meta.Expands
	Charges   meta.Charges
}

func (i GetByMetasInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Charges.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charges: %w", err))
	}

	if err := i.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type AdvanceChargeInput struct {
	ChargeID meta.ChargeID
}

func (i AdvanceChargeInput) Validate() error {
	return i.ChargeID.Validate()
}
