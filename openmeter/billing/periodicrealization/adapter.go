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

	// GetRunCovering returns the earliest run whose period ends at or after
	// AsOf, or nil when the namespace has no such run.
	GetRunCovering(ctx context.Context, input GetOrCreateRunAtInput) (*Run, error)
	// GetLatestRun returns the namespace's run with the latest period, or nil
	// when the namespace has no run.
	GetLatestRun(ctx context.Context, namespace string) (*Run, error)
	CreateRun(ctx context.Context, input CreateRunInput) (Run, error)
}

var _ models.Validator = (*CreateRunInput)(nil)

type CreateRunInput struct {
	Namespace string
	// PreviousRunID is nil for the first run of a namespace.
	PreviousRunID *string
	PeriodTo      time.Time
	RealizeAfter  time.Time
}

func (i CreateRunInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.PreviousRunID != nil && *i.PreviousRunID == "" {
		errs = append(errs, errors.New("previous run id must not be empty when set"))
	}

	if i.PeriodTo.IsZero() {
		errs = append(errs, errors.New("period to is required"))
	}

	if i.RealizeAfter.Before(i.PeriodTo) {
		errs = append(errs, errors.New("realize after must not be before period to"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
