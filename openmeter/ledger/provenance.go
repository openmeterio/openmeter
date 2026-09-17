package ledger

import (
	"errors"
	"fmt"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	AnnotationOriginTracked          = "ledger.origin_tracked"
	AnnotationBackfillCreditPriority = "ledger.backfill.credit_priority"
)

type Provenance struct {
	CollectionOriginID *string
	SourceChargeID     *string
	SpendChargeID      *string
}

func (p Provenance) Filter() ProvenanceFilter {
	return ProvenanceFilter{
		CollectionOriginID: mo.Some(p.CollectionOriginID),
		SourceChargeID:     mo.Some(p.SourceChargeID),
		SpendChargeID:      mo.Some(p.SpendChargeID),
	}
}

// Absent fields match any value; Some(nil) matches entries without that ID.
type ProvenanceFilter struct {
	CollectionOriginID mo.Option[*string]
	SourceChargeID     mo.Option[*string]
	SpendChargeID      mo.Option[*string]
}

func (f ProvenanceFilter) Validate() error {
	var errs []error

	if err := validateOptionalProvenanceID("collection_origin_id", f.CollectionOriginID); err != nil {
		errs = append(errs, err)
	}

	if err := validateOptionalProvenanceID("source_charge_id", f.SourceChargeID); err != nil {
		errs = append(errs, err)
	}

	if err := validateOptionalProvenanceID("spend_charge_id", f.SpendChargeID); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func validateOptionalProvenanceID(name string, filter mo.Option[*string]) error {
	value, _ := filter.Get()
	if value != nil && *value == "" {
		return fmt.Errorf("%s must not be empty", name)
	}

	return nil
}
