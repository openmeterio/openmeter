package invoicemetrics

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type OverdueInvoiceCounts struct {
	Collection  int64
	Advancement int64
}

type CountOverdueInvoicesInput struct {
	AsOf               time.Time
	MinimumAge         time.Duration
	ExcludedNamespaces []string
}

var _ models.Validator = (*CountOverdueInvoicesInput)(nil)

func (i CountOverdueInvoicesInput) Validate() error {
	var errs []error

	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}

	if i.MinimumAge <= 0 {
		errs = append(errs, errors.New("minimum age must be greater than zero"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
