package taxcode

import (
	"context"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/addon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/addonratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ taxcode.DeleteValidator = (*Validator)(nil)

// Validator blocks deletion of a tax code that is still referenced by non-archived,
// non-soft-deleted plan or addon ratecards. Archived means effective_to < now.
// Returns a 409 conflict error when any live reference is found.
type Validator struct {
	taxcode.NoopDeleteValidator
	db *entdb.Client
}

// NewValidator creates a Validator backed by the given Ent client.
func NewValidator(db *entdb.Client) (*Validator, error) {
	if db == nil {
		return nil, fmt.Errorf("ent client is required")
	}

	return &Validator{db: db}, nil
}

// ValidateDeleteTaxCode returns a conflict error when the tax code being deleted is
// still referenced by at least one non-archived, non-soft-deleted plan or addon ratecard.
func (v *Validator) ValidateDeleteTaxCode(ctx context.Context, input taxcode.DeleteTaxCodeInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	now := clock.Now()

	// Count non-archived (effective_to IS NULL OR effective_to >= now), non-deleted plans
	// that have at least one non-deleted ratecard referencing this tax code.
	planCount, err := v.db.Plan.Query().Where(
		plan.Namespace(input.Namespace),
		plan.DeletedAtIsNil(),
		plan.Or(
			plan.EffectiveToIsNil(),
			plan.EffectiveToGTE(now),
		),
		plan.HasPhasesWith(
			planphase.HasRatecardsWith(
				planratecard.TaxCodeID(input.ID),
				planratecard.DeletedAtIsNil(),
			),
		),
	).Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count plan ratecards referencing tax code %s: %w", input.ID, err)
	}

	// Count non-archived, non-deleted addons that have at least one non-deleted ratecard
	// referencing this tax code.
	addonCount, err := v.db.Addon.Query().Where(
		addon.Namespace(input.Namespace),
		addon.DeletedAtIsNil(),
		addon.Or(
			addon.EffectiveToIsNil(),
			addon.EffectiveToGTE(now),
		),
		addon.HasRatecardsWith(
			addonratecard.TaxCodeID(input.ID),
			addonratecard.DeletedAtIsNil(),
		),
	).Count(ctx)
	if err != nil {
		return fmt.Errorf("failed to count addon ratecards referencing tax code %s: %w", input.ID, err)
	}

	if planCount == 0 && addonCount == 0 {
		return nil
	}

	return models.NewGenericConflictError(fmt.Errorf(
		"tax code %s is referenced by %d plan(s) and %d addon(s); remove the references before deleting it",
		input.ID, planCount, addonCount,
	))
}
