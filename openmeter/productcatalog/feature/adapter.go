package feature

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ArchiveFeatureInput contains the parameters for archiving a feature.
type ArchiveFeatureInput struct {
	Namespace string
	ID        string
	At        *time.Time
}

func (i ArchiveFeatureInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	if i.At != nil && i.At.IsZero() {
		errs = append(errs, errors.New("at must not be zero"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Adapter is the database adapter interface for feature persistence.
type Adapter interface {
	CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error)
	ArchiveFeature(ctx context.Context, params ArchiveFeatureInput) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) (pagination.Result[Feature], error)
	HasActiveFeatureForMeter(ctx context.Context, namespace string, meterSlug string) (bool, error)

	GetByIdOrKey(ctx context.Context, namespace string, idOrKey string, includeArchived bool) (*Feature, error)
	entutils.TxCreator
	entutils.TxUser[Adapter]
}

// FeatureRepo is an alias for Adapter, kept for backward compatibility.
type FeatureRepo = Adapter
