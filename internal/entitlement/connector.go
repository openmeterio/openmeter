package entitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ListEntitlementsOrderBy string

const (
	ListEntitlementsOrderByCreatedAt ListEntitlementsOrderBy = "created_at"
	ListEntitlementsOrderByUpdatedAt ListEntitlementsOrderBy = "updated_at"
)

type ListEntitlementsParams struct {
	Namespace string
	Limit     int
	Offset    int
	OrderBy   ListEntitlementsOrderBy
}

type Connector interface {
	// Entitlement Management
	CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (*Entitlement, error)
	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error)
	GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (EntitlementValue, error)

	ListEntitlements(ctx context.Context, params ListEntitlementsParams) ([]Entitlement, error)
}

type entitlementConnector struct {
	meteredEntitlementConnector SubTypeConnector
	staticEntitlementConnector  SubTypeConnector
	booleanEntitlementConnector SubTypeConnector

	entitlementRepo  EntitlementRepo
	featureConnector productcatalog.FeatureConnector
}

func NewEntitlementConnector(
	entitlementRepo EntitlementRepo,
	featureConnector productcatalog.FeatureConnector,
	meteredEntitlementConnector SubTypeConnector,
	staticEntitlementConnector SubTypeConnector,
	booleanEntitlementConnector SubTypeConnector,
) Connector {
	return &entitlementConnector{
		meteredEntitlementConnector: meteredEntitlementConnector,
		staticEntitlementConnector:  staticEntitlementConnector,
		booleanEntitlementConnector: booleanEntitlementConnector,
		entitlementRepo:             entitlementRepo,
		featureConnector:            featureConnector,
	}
}

func (c *entitlementConnector) CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (*Entitlement, error) {
	// ID has precedence over key
	idOrKey := input.FeatureID
	if idOrKey == nil {
		idOrKey = input.FeatureKey
	}
	if idOrKey == nil {
		return nil, &models.GenericUserError{Message: "Feature ID or key is required"}
	}

	feature, err := c.featureConnector.GetFeature(ctx, input.Namespace, *idOrKey)
	if err != nil || feature == nil {
		return nil, &productcatalog.FeatureNotFoundError{ID: *idOrKey}
	}
	if feature.ArchivedAt != nil && feature.ArchivedAt.Before(time.Now()) {
		return nil, &models.GenericUserError{Message: "Feature is archived"}
	}
	currentEntitlements, err := c.entitlementRepo.GetEntitlementsOfSubject(ctx, input.Namespace, models.SubjectKey(input.SubjectKey))
	if err != nil {
		return nil, err
	}
	for _, ent := range currentEntitlements {
		// If we want to access entitlements by featureKey then this has to be unique
		if ent.FeatureKey == feature.Key {
			return nil, &AlreadyExistsError{EntitlementID: ent.ID, FeatureID: feature.ID, FeatureKey: feature.Key, SubjectKey: input.SubjectKey}
		}
	}

	// populate feature id and key
	input.FeatureID = &feature.ID
	input.FeatureKey = &feature.Key

	connector, err := c.getTypeConnector(input)
	if err != nil {
		return nil, err
	}
	err = connector.SetDefaultsAndValidate(&input)
	if err != nil {
		return nil, err
	}
	err = connector.ValidateForFeature(&input, *feature)
	if err != nil {
		return nil, err
	}

	ent, err := c.entitlementRepo.CreateEntitlement(ctx, CreateEntitlementRepoInputs{
		Namespace:        input.Namespace,
		FeatureID:        *input.FeatureID,
		FeatureKey:       *input.FeatureKey,
		SubjectKey:       input.SubjectKey,
		EntitlementType:  input.EntitlementType,
		MeasureUsageFrom: input.MeasureUsageFrom,
		IssueAfterReset:  input.IssueAfterReset,
		IsSoftLimit:      input.IsSoftLimit,
		Config:           input.Config,
		UsagePeriod:      input.UsagePeriod,
	})
	if err != nil || ent == nil {
		return nil, err
	}
	return ent, nil
}

func (c *entitlementConnector) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error) {
	return c.entitlementRepo.GetEntitlementsOfSubject(ctx, namespace, subjectKey)
}

func (c *entitlementConnector) GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (EntitlementValue, error) {
	ent, err := c.entitlementRepo.GetEntitlementOfSubject(ctx, namespace, subjectKey, idOrFeatureKey)
	if err != nil {
		return nil, err
	}
	connector, err := c.getTypeConnector(ent)
	if err != nil {
		return nil, err
	}
	return connector.GetValue(ent, at)
}

func (c *entitlementConnector) ListEntitlements(ctx context.Context, params ListEntitlementsParams) ([]Entitlement, error) {
	return c.entitlementRepo.ListEntitlements(ctx, params)
}

func (c *entitlementConnector) getTypeConnector(inp TypedEntitlement) (SubTypeConnector, error) {
	entitlementType := inp.GetType()
	switch entitlementType {
	case EntitlementTypeMetered:
		return c.meteredEntitlementConnector, nil
	case EntitlementTypeStatic:
		return c.staticEntitlementConnector, nil
	case EntitlementTypeBoolean:
		return c.booleanEntitlementConnector, nil
	default:
		return nil, fmt.Errorf("unsupported entitlement type: %s", entitlementType)
	}
}
