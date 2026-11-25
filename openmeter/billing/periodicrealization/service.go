package periodicrealization

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	// GetOrCreateRunAt returns the namespace's run that covers usage before
	// AsOf: the earliest run whose period ends at or after it. Its PeriodTo can
	// be later than the first period end after AsOf, because a period that
	// never got a run is covered by the next run that exists.
	//
	// When no run covers AsOf, the run for the first period end at or after it
	// is created and chained to the namespace's latest run.
	GetOrCreateRunAt(ctx context.Context, input GetOrCreateRunAtInput) (Run, error)
}

var _ models.Validator = (*GetOrCreateRunAtInput)(nil)

type GetOrCreateRunAtInput struct {
	Namespace string
	// AsOf is an exclusive upper bound, like a service period end: the run is
	// the one that covers usage before this instant.
	AsOf time.Time
}

func (i GetOrCreateRunAtInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
