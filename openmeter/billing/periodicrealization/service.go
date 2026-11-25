package periodicrealization

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	// GetOrCreateRunAt schedules the namespace's run for the period that
	// contains AsOf.
	//
	// Only the first run of a namespace can be scheduled so far: a namespace
	// that already has a run gets a not-implemented error.
	GetOrCreateRunAt(ctx context.Context, input GetOrCreateRunAtInput) (Run, error)
}

var _ models.Validator = (*GetOrCreateRunAtInput)(nil)

type GetOrCreateRunAtInput struct {
	Namespace string
	AsOf      time.Time
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
