package usagebased

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	UsageBasedService
	GetLineEngine() billing.LineEngine
}

type UsageBasedService interface {
	Create(ctx context.Context, input CreateInput) ([]ChargeWithGatheringLine, error)
	GetByIDs(ctx context.Context, input GetByIDsInput) ([]Charge, error)
	AdvanceCharge(ctx context.Context, input AdvanceChargeInput) (*Charge, error)
	TriggerPatch(ctx context.Context, charge meta.ChargeID, patch meta.Patch) (*Charge, error)
	GetCurrentTotals(ctx context.Context, input GetCurrentTotalsInput) (GetCurrentTotalsResult, error)
}

type CreateInput struct {
	Namespace     string
	Intents       []Intent
	FeatureMeters feature.FeatureMeters
}

func (i CreateInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if len(i.Intents) > 0 && i.FeatureMeters == nil {
		errs = append(errs, errors.New("feature meters are required"))
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

type CreateIntent struct {
	Intent
	FeatureID    string
	RatingEngine RatingEngine
}

func (i CreateIntent) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.FeatureID == "" {
		errs = append(errs, errors.New("feature id is required"))
	}

	if err := i.RatingEngine.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("rating engine: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreateChargesInput struct {
	Namespace string
	Intents   []CreateIntent
}

func (i CreateChargesInput) Validate() error {
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

type GetByIDsInput struct {
	Namespace string
	IDs       []string
	Expands   meta.Expands
}

func (i GetByIDsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := validateExpands(i.Expands); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type AdvanceChargeInput struct {
	ChargeID         meta.ChargeID
	CustomerOverride billing.CustomerOverrideWithDetails
	FeatureMeters    feature.FeatureMeters
}

func (i AdvanceChargeInput) Validate() error {
	var errs []error
	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if i.CustomerOverride.Customer == nil {
		errs = append(errs, errors.New("expanded customer is required"))
	}

	if err := i.CustomerOverride.MergedProfile.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("merged profile is required: %w", err))
	}

	if i.FeatureMeters == nil {
		errs = append(errs, errors.New("feature meters are required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetByIDInput struct {
	ChargeID meta.ChargeID
	Expands  meta.Expands
}

func (i GetByIDInput) Validate() error {
	var errs []error
	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if err := validateExpands(i.Expands); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type GetCurrentTotalsInput struct {
	ChargeID meta.ChargeID
}

func (i GetCurrentTotalsInput) Validate() error {
	if err := i.ChargeID.Validate(); err != nil {
		return models.NewGenericValidationError(fmt.Errorf("charge ID: %w", err))
	}

	return nil
}

type GetCurrentTotalsResult struct {
	Charge    Charge
	DueTotals totals.Totals
}

func validateExpands(expands meta.Expands) error {
	if err := expands.Validate(); err != nil {
		return err
	}

	if expands.Has(meta.ExpandDetailedLines) && !expands.Has(meta.ExpandRealizations) {
		return fmt.Errorf("%q requires %q", meta.ExpandDetailedLines, meta.ExpandRealizations)
	}

	return nil
}
