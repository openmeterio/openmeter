package taxcode

import (
	"context"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomeroverride"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingworkflowconfig"
	taxcodedomain "github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ taxcodedomain.DeleteValidator = (*Validator)(nil)

type Validator struct {
	taxcodedomain.NoopDeleteValidator
	db *entdb.Client
}

func NewValidator(db *entdb.Client) (*Validator, error) {
	if db == nil {
		return nil, fmt.Errorf("db is required")
	}
	return &Validator{db: db}, nil
}

func (v *Validator) ValidateDeleteTaxCode(ctx context.Context, input taxcodedomain.DeleteTaxCodeInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	profileCount, err := v.db.BillingProfile.Query().Where(
		billingprofile.Namespace(input.Namespace),
		billingprofile.DeletedAtIsNil(),
		billingprofile.HasWorkflowConfigWith(
			billingworkflowconfig.TaxCodeID(input.ID),
			billingworkflowconfig.DeletedAtIsNil(),
		),
	).Count(ctx)
	if err != nil {
		return fmt.Errorf("querying billing profiles for tax code %s: %w", input.ID, err)
	}

	overrideCount, err := v.db.BillingCustomerOverride.Query().Where(
		billingcustomeroverride.Namespace(input.Namespace),
		billingcustomeroverride.DeletedAtIsNil(),
		billingcustomeroverride.TaxCodeID(input.ID),
	).Count(ctx)
	if err != nil {
		return fmt.Errorf("querying billing customer overrides for tax code %s: %w", input.ID, err)
	}

	if profileCount == 0 && overrideCount == 0 {
		return nil
	}

	return models.NewGenericConflictError(fmt.Errorf("tax code %s is referenced by %d billing profile(s) and %d customer override(s); remove the references before deleting it", input.ID, profileCount, overrideCount))
}
