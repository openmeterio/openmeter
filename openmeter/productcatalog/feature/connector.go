package feature

import (
	"context"
	"fmt"
	"slices"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type CreateFeatureInputs struct {
	Name                string              `json:"name"`
	Key                 string              `json:"key"`
	Namespace           string              `json:"namespace"`
	MeterSlug           *string             `json:"meterSlug"`
	MeterGroupByFilters MeterGroupByFilters `json:"meterGroupByFilters"`
	UnitCost            *UnitCost           `json:"unitCost"`
	Metadata            map[string]string   `json:"metadata"`
}

// TODO: refactor to service pattern
type FeatureConnector interface {
	// Feature Management
	CreateFeature(ctx context.Context, feature CreateFeatureInputs) (Feature, error)
	// Should just use deletedAt, there's no real "archiving"
	ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error
	ListFeatures(ctx context.Context, params ListFeaturesParams) (pagination.Result[Feature], error)
	GetFeature(ctx context.Context, namespace string, idOrKey string, includeArchived IncludeArchivedFeature) (*Feature, error)

	// ResolveFeatureMeters resolves the feature meters for a given namespace and feature keys, returning a map of feature key to feature meter.
	// The list contains either the active feature or the most recently archived feature.
	ResolveFeatureMeters(ctx context.Context, namespace string, featureKeys []string) (FeatureMeters, error)

	// ResolveUnitCost resolves the per-unit cost for a feature given group-by dimension values.
	// For manual unit cost: returns the configured fixed amount.
	// For LLM unit cost: looks up the cost from the LLM cost database using the provider, model, and token type
	// extracted from the group-by values via the feature's property mappings.
	ResolveUnitCost(ctx context.Context, input ResolveUnitCostInput) (*ResolvedUnitCost, error)
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
	MeterSlugs      []string
	IncludeArchived bool
	Page            pagination.Page
	OrderBy         FeatureOrderBy
	Order           sortx.Order
	// will be deprecated
	Limit int
	// will be deprecated
	Offset int
}

type featureConnector struct {
	featureRepo    FeatureRepo
	meterService   meterpkg.Service
	llmcostService llmcost.Service
	publisher      eventbus.Publisher

	validMeterAggregations []meterpkg.MeterAggregation
}

func NewFeatureConnector(
	featureRepo FeatureRepo,
	meterService meterpkg.Service,
	publisher eventbus.Publisher,
	llmcostService llmcost.Service,
) FeatureConnector {
	return &featureConnector{
		featureRepo:    featureRepo,
		meterService:   meterService,
		llmcostService: llmcostService,
		publisher:      publisher,

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

	if feature.MeterSlug != nil {
		slug := *feature.MeterSlug

		// nosemgrep: trailofbits.go.invalid-usage-of-modified-variable.invalid-usage-of-modified-variable
		meter, err := c.meterService.GetMeterByIDOrSlug(ctx, meterpkg.GetMeterInput{
			Namespace: feature.Namespace,
			IDOrSlug:  slug,
		})
		if err != nil {
			return Feature{}, meterpkg.NewMeterNotFoundError(slug)
		}

		resolvedMeter = &meter

		if !slices.Contains(c.validMeterAggregations, meter.Aggregation) {
			return Feature{}, &FeatureInvalidMeterAggregationError{Aggregation: meter.Aggregation, MeterSlug: meter.Key, ValidAggregations: c.validMeterAggregations}
		}

		if feature.MeterGroupByFilters != nil {
			err = feature.MeterGroupByFilters.Validate(meter)
			if err != nil {
				return Feature{}, err
			}
		}
		if err != nil {
			return Feature{}, err
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

	// Publish the feature created event
	featureCreatedEvent := NewFeatureCreateEvent(ctx, &createdFeature)
	if err := c.publisher.Publish(ctx, featureCreatedEvent); err != nil {
		return createdFeature, fmt.Errorf("failed to publish feature created event: %w", err)
	}

	return createdFeature, nil
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
	if !params.Page.IsZero() {
		if err := params.Page.Validate(); err != nil {
			return pagination.Result[Feature]{}, err
		}
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

// ResolveUnitCost resolves the per-unit cost for a feature.
func (c *featureConnector) ResolveUnitCost(ctx context.Context, input ResolveUnitCostInput) (*ResolvedUnitCost, error) {
	feat, err := c.featureRepo.GetByIdOrKey(ctx, input.Namespace, input.FeatureIDOrKey, false)
	if err != nil {
		return nil, err
	}

	if feat.UnitCost == nil {
		return nil, nil
	}

	switch feat.UnitCost.Type {
	case UnitCostTypeManual:
		if feat.UnitCost.Manual == nil {
			return nil, fmt.Errorf("feature %s has manual unit cost type but no manual configuration", feat.Key)
		}

		return &ResolvedUnitCost{
			Amount:   feat.UnitCost.Manual.Amount,
			Currency: "USD",
		}, nil

	case UnitCostTypeLLM:
		if feat.UnitCost.LLM == nil {
			return nil, fmt.Errorf("feature %s has LLM unit cost type but no LLM configuration", feat.Key)
		}

		if c.llmcostService == nil {
			return nil, fmt.Errorf("LLM cost service is not available")
		}

		llmConf := feat.UnitCost.LLM

		// Resolve provider: static value or from group-by
		var provider string
		if llmConf.Provider != "" {
			provider = llmConf.Provider
		} else if llmConf.ProviderProperty != "" {
			var ok bool
			provider, ok = input.GroupByValues[llmConf.ProviderProperty]
			if !ok {
				return nil, fmt.Errorf("group-by value %q (provider) not found in input", llmConf.ProviderProperty)
			}
		} else {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("feature %s has LLM unit cost but neither provider nor provider_property is configured", feat.Key),
			)
		}

		// Resolve model: static value or from group-by
		var modelID string
		if llmConf.Model != "" {
			modelID = llmConf.Model
		} else if llmConf.ModelProperty != "" {
			var ok bool
			modelID, ok = input.GroupByValues[llmConf.ModelProperty]
			if !ok {
				return nil, fmt.Errorf("group-by value %q (model) not found in input", llmConf.ModelProperty)
			}
		} else {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("feature %s has LLM unit cost but neither model nor model_property is configured", feat.Key),
			)
		}

		// Resolve token type: static value or from group-by
		var tokenTypeStr string
		if llmConf.TokenType != "" {
			tokenTypeStr = llmConf.TokenType
		} else if llmConf.TokenTypeProperty != "" {
			var ok bool
			tokenTypeStr, ok = input.GroupByValues[llmConf.TokenTypeProperty]
			if !ok {
				return nil, fmt.Errorf("group-by value %q (token type) not found in input", llmConf.TokenTypeProperty)
			}
		} else {
			return nil, models.NewGenericValidationError(
				fmt.Errorf("feature %s has LLM unit cost but neither token_type nor token_type_property is configured", feat.Key),
			)
		}

		price, err := c.llmcostService.ResolvePrice(ctx, llmcost.ResolvePriceInput{
			Namespace: input.Namespace,
			Provider:  llmcost.Provider(provider),
			ModelID:   modelID,
		})
		if err != nil {
			return nil, fmt.Errorf("resolving LLM price for provider=%s model=%s: %w", provider, modelID, err)
		}

		amount, err := CostPerTokenForType(price.Pricing, LLMTokenType(tokenTypeStr))
		if err != nil {
			return nil, fmt.Errorf("resolving token type cost for provider=%s model=%s type=%s: %w", provider, modelID, tokenTypeStr, err)
		}

		return &ResolvedUnitCost{
			Amount:   amount,
			Currency: price.Currency,
		}, nil

	default:
		return nil, fmt.Errorf("unknown unit cost type: %s", feat.UnitCost.Type)
	}
}
