package feature

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/oapi-codegen/nullable"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type CreateFeatureInputs struct {
	Name                string              `json:"name"`
	Description         *string             `json:"description,omitempty"`
	Key                 string              `json:"key"`
	Namespace           string              `json:"namespace"`
	MeterID             *string             `json:"meterID"`
	MeterGroupByFilters MeterGroupByFilters `json:"meterGroupByFilters"`
	UnitCost            *UnitCost           `json:"unitCost"`
	Metadata            map[string]string   `json:"metadata"`
}

type UpdateFeatureInputs struct {
	Namespace string                      `json:"namespace"`
	ID        string                      `json:"id"`
	UnitCost  nullable.Nullable[UnitCost] `json:"unitCost"`
}

func (i UpdateFeatureInputs) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ID == "" {
		errs = append(errs, errors.New("id is required"))
	}

	if !i.UnitCost.IsSpecified() {
		errs = append(errs, errors.New("unitCost is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// TODO: refactor to service pattern
type FeatureConnector interface {
	// Feature Management
	CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error)
	UpdateFeature(ctx context.Context, input UpdateFeatureInputs) (Feature, error)
	// Should just use deletedAt, there's no real "archiving"
	ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) (pagination.Result[Feature], error)
	GetFeature(ctx context.Context, namespace string, idOrKey string, includeArchived IncludeArchivedFeature) (*Feature, error)

	// ResolveFeatureMeters resolves the feature meters for a given namespace and feature refs.
	// Keys always resolve to the latest available feature for that key.
	// Explicit IDs are returned in the ID index, and also in the key index when they are the latest feature for that key.
	ResolveFeatureMeters(ctx context.Context, namespace string, featureRefs ...ref.IDOrKey) (FeatureMeters, error)
}

type IncludeArchivedFeature bool

const (
	IncludeArchivedFeatureTrue  IncludeArchivedFeature = true
	IncludeArchivedFeatureFalse IncludeArchivedFeature = false
)

// FeatureOrderBy is the order by clause for features
type FeatureOrderBy string

const (
	FeatureOrderByKey       FeatureOrderBy = "key"
	FeatureOrderByName      FeatureOrderBy = "name"
	FeatureOrderByCreatedAt FeatureOrderBy = "created_at"
	FeatureOrderByUpdatedAt FeatureOrderBy = "updated_at"
)

func (f FeatureOrderBy) Values() []FeatureOrderBy {
	return []FeatureOrderBy{
		FeatureOrderByKey,
		FeatureOrderByName,
		FeatureOrderByCreatedAt,
		FeatureOrderByUpdatedAt,
	}
}

type ListFeaturesParams struct {
	IDsOrKeys       []string
	Namespace       string
	MeterIDs        *filter.FilterUlid
	MeterSlugs      []string // Kept for ingest pipeline compat (queries via ent edge on meter key)
	IncludeArchived bool
	Page            pagination.Page
	OrderBy         FeatureOrderBy
	Order           sortx.Order
	// will be deprecated
	Limit int
	// will be deprecated
	Offset int
}

func (p ListFeaturesParams) Validate() error {
	var errs []error

	if p.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if p.MeterIDs != nil {
		if err := p.MeterIDs.Validate(); err != nil {
			errs = append(errs, err)
		}
	}
	if !p.Page.IsZero() {
		if err := p.Page.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type featureConnector struct {
	featureRepo  FeatureRepo
	meterService meterpkg.Service
	publisher    eventbus.Publisher

	validMeterAggregations []meterpkg.MeterAggregation
}

func NewFeatureConnector(
	featureRepo FeatureRepo,
	meterService meterpkg.Service,
	publisher eventbus.Publisher,
) FeatureConnector {
	return &featureConnector{
		featureRepo:  featureRepo,
		meterService: meterService,
		publisher:    publisher,

		validMeterAggregations: []meterpkg.MeterAggregation{
			meterpkg.MeterAggregationSum,
			meterpkg.MeterAggregationCount,
			meterpkg.MeterAggregationUniqueCount,
			meterpkg.MeterAggregationLatest,
		},
	}
}

// CreateFeature creates a new feature
func (c *featureConnector) CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error) {
	// Validate meter configuration
	var resolvedMeter *meterpkg.Meter

	if feature.MeterID != nil {
		meterID := *feature.MeterID

		// nosemgrep: trailofbits.go.invalid-usage-of-modified-variable.invalid-usage-of-modified-variable
		meter, err := c.meterService.GetMeterByIDOrSlug(ctx, meterpkg.GetMeterInput{
			Namespace: feature.Namespace,
			IDOrSlug:  meterID,
		})
		if err != nil {
			if meterpkg.IsMeterNotFoundError(err) {
				return Feature{}, meterpkg.NewMeterNotFoundError(meterID)
			}
			return Feature{}, fmt.Errorf("get meter %s: %w", meterID, err)
		}

		// Normalize to meter ID
		feature.MeterID = &meter.ID

		resolvedMeter = &meter

		if !slices.Contains(c.validMeterAggregations, meter.Aggregation) {
			return Feature{}, &FeatureInvalidMeterAggregationError{Aggregation: meter.Aggregation, MeterSlug: meter.Key, ValidAggregations: c.validMeterAggregations}
		}

		if feature.MeterGroupByFilters != nil {
			if err = feature.MeterGroupByFilters.Validate(meter); err != nil {
				return Feature{}, err
			}
		}
	}

	// Validate unit cost
	if feature.UnitCost != nil {
		if err := feature.UnitCost.Validate(); err != nil {
			return Feature{}, models.NewGenericValidationError(err)
		}

		if feature.UnitCost.Type == UnitCostTypeLLM {
			if resolvedMeter == nil {
				return Feature{}, models.NewGenericValidationError(
					fmt.Errorf("LLM unit cost requires a meter to be associated with the feature"),
				)
			}

			if err := feature.UnitCost.ValidateWithMeter(*resolvedMeter); err != nil {
				return Feature{}, models.NewGenericValidationError(err)
			}
		}
	}

	// Validate feature key
	if _, err := ulid.Parse(feature.Key); err == nil {
		return Feature{}, models.NewGenericValidationError(fmt.Errorf("Feature key cannot be a valid ULID"))
	}

	// Check key is not taken
	found, err := c.featureRepo.GetByIdOrKey(ctx, feature.Namespace, feature.Key, false)
	if err != nil {
		if _, ok := err.(*FeatureNotFoundError); !ok {
			return Feature{}, err
		}
	} else {
		return Feature{}, &FeatureWithNameAlreadyExistsError{Name: feature.Key, ID: found.ID}
	}

	// Create the feature
	createdFeature, err := c.featureRepo.CreateFeature(ctx, feature)
	if err != nil {
		return Feature{}, err
	}

	// Populate MeterSlug from resolved meter for v1 API backward compat
	if resolvedMeter != nil {
		createdFeature.MeterSlug = &resolvedMeter.Key
	}

	// Publish the feature created event
	featureCreatedEvent := NewFeatureCreateEvent(ctx, &createdFeature)
	if err := c.publisher.Publish(ctx, featureCreatedEvent); err != nil {
		return createdFeature, fmt.Errorf("failed to publish feature created event: %w", err)
	}

	return createdFeature, nil
}

// UpdateFeature updates a feature's unit cost
func (c *featureConnector) UpdateFeature(ctx context.Context, input UpdateFeatureInputs) (Feature, error) {
	if err := input.Validate(); err != nil {
		return Feature{}, err
	}

	// Get the feature (rejects archived/not found)
	feat, err := c.GetFeature(ctx, input.Namespace, input.ID, IncludeArchivedFeatureFalse)
	if err != nil {
		return Feature{}, err
	}

	// Validate unit cost if a value is provided (not null/clear)
	if !input.UnitCost.IsNull() {
		unitCost, err := input.UnitCost.Get()
		if err != nil {
			return Feature{}, models.NewGenericValidationError(err)
		}

		if err := unitCost.Validate(); err != nil {
			return Feature{}, models.NewGenericValidationError(err)
		}

		if unitCost.Type == UnitCostTypeLLM {
			if feat.MeterSlug == nil {
				return Feature{}, models.NewGenericValidationError(
					fmt.Errorf("LLM unit cost requires a meter to be associated with the feature"),
				)
			}

			meter, err := c.meterService.GetMeterByIDOrSlug(ctx, meterpkg.GetMeterInput{
				Namespace: input.Namespace,
				IDOrSlug:  *feat.MeterSlug,
			})
			if err != nil {
				return Feature{}, err
			}

			if err := unitCost.ValidateWithMeter(meter); err != nil {
				return Feature{}, models.NewGenericValidationError(err)
			}
		}
	}

	updatedFeature, err := c.featureRepo.UpdateFeature(ctx, input)
	if err != nil {
		return Feature{}, err
	}

	// Publish the feature updated event
	featureUpdatedEvent := NewFeatureUpdateEvent(ctx, &updatedFeature)
	if err := c.publisher.Publish(ctx, featureUpdatedEvent); err != nil {
		return updatedFeature, fmt.Errorf("failed to publish feature updated event: %w", err)
	}

	return updatedFeature, nil
}

// ArchiveFeature archives a feature
func (c *featureConnector) ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error {
	// Get the feature
	feat, err := c.GetFeature(ctx, featureID.Namespace, featureID.ID, false)
	if err != nil {
		return err
	}

	archivedAt := lo.ToPtr(clock.Now())

	// Archive the feature
	err = c.featureRepo.ArchiveFeature(ctx, ArchiveFeatureInput{
		Namespace: feat.Namespace,
		ID:        feat.ID,
		At:        archivedAt,
	})
	if err != nil {
		return err
	}

	feat.ArchivedAt = archivedAt

	// Publish the feature archived event
	featureArchivedEvent := NewFeatureArchiveEvent(ctx, feat)
	if err := c.publisher.Publish(ctx, featureArchivedEvent); err != nil {
		return fmt.Errorf("failed to publish feature archived event: %w", err)
	}

	return nil
}

// ListFeatures lists features
func (c *featureConnector) ListFeatures(ctx context.Context, params ListFeaturesParams) (pagination.Result[Feature], error) {
	if err := params.Validate(); err != nil {
		return pagination.Result[Feature]{}, err
	}

	return c.featureRepo.ListFeatures(ctx, params)
}

// GetFeature gets a feature
func (c *featureConnector) GetFeature(ctx context.Context, namespace string, idOrKey string, includeArchived IncludeArchivedFeature) (*Feature, error) {
	feature, err := c.featureRepo.GetByIdOrKey(ctx, namespace, idOrKey, bool(includeArchived))
	if err != nil {
		return nil, err
	}
	return feature, nil
}
