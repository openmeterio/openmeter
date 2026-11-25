package periodicrealization

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Adapter interface {
	entutils.TxCreator

	// HasRuns reports whether the namespace has any run, whatever its period.
	HasRuns(ctx context.Context, namespace string) (bool, error)
	CreateRun(ctx context.Context, input CreateRunInput) (Run, error)
}

var _ models.Validator = (*CreateRunInput)(nil)

type CreateRunInput struct {
	Namespace    string
	PeriodTo     time.Time
	RealizeAfter time.Time
}

func (i CreateRunInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.PeriodTo.IsZero() {
		errs = append(errs, errors.New("period to is required"))
	}

	if i.RealizeAfter.Before(i.PeriodTo) {
		errs = append(errs, errors.New("realize after must not be before period to"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
