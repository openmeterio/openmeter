package usagebased

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	UsageBasedService
	GetLineEngine() billing.LineEngine
}

type UsageBasedService interface {
	// Create returns one result for each input intent, preserving input order.
	// Pending gathering-line creation relies on this to pair charge targets with
	// their source intents.
	Create(ctx context.Context, input CreateInput) ([]ChargeWithGatheringLine, error)
	// GetByIDs loads usage-based charges. Request realization expansions when
	// callers need run, detailed-line, or credit-allocation state.
	GetByIDs(ctx context.Context, input GetByIDsInput) ([]Charge, error)
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
	// GetCurrentTotals calculates the current customer-facing totals from the
	// effective intent and non-voided realization history.
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
	Intent      OverridableIntent
	Annotations models.Annotations `json:"annotations"`

	FeatureID         string
	RatingEngine      RatingEngine
	ResolvedCostBasis *costbasis.State
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
	// AllowMissing drops IDs with no matching charge instead of failing the
	// read: callers set it when the ID set comes from an earlier search
	// snapshot and charges may have been deleted since.
	AllowMissing bool
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
	ChargeID meta.ChargeID
	// CustomerOverride is an authoritative optional resolution hint. None lets
	// the service resolve the current customer override; Some uses only the
	// supplied snapshot.
	CustomerOverride mo.Option[billing.CustomerOverrideWithDetails]
	// FeatureMeters is an authoritative optional resolution hint. None lets the
	// service resolve the required feature meter; Some resolves exclusively from
	// the supplied collection and returns an error when the feature is absent.
	FeatureMeters mo.Option[feature.FeatureMeters]
}

func (i AdvanceChargeInput) Validate() error {
	var errs []error
	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if customerOverride, ok := i.CustomerOverride.Get(); ok {
		if customerOverride.Customer == nil {
			errs = append(errs, errors.New("expanded customer is required"))
		}

		if err := customerOverride.MergedProfile.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("merged profile is required: %w", err))
		}
	}

	if featureMeters, ok := i.FeatureMeters.Get(); ok && featureMeters == nil {
		errs = append(errs, errors.New("feature meters cannot be nil when provided"))
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
	// MeteredQuantity is the cumulative live metered quantity the due totals
	// were rated from.
	MeteredQuantity alpacadecimal.Decimal
}

func validateExpands(expands meta.Expands) error {
	if err := expands.Validate(); err != nil {
		return err
	}

	if expands.Has(meta.ExpandDetailedLines) && !expands.Has(meta.ExpandRealizations) {
		return fmt.Errorf("%q requires %q", meta.ExpandDetailedLines, meta.ExpandRealizations)
	}

	if expands.Has(meta.ExpandDeletedRealizations) && !expands.Has(meta.ExpandRealizations) {
		return fmt.Errorf("%q requires %q", meta.ExpandDeletedRealizations, meta.ExpandRealizations)
	}

	return nil
}
