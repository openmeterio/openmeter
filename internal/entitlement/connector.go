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
	CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (Entitlement, error)
	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error)
	GetEntitlementValue(ctx context.Context, entitlementId models.NamespacedID, at time.Time) (EntitlementValue, error)

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

func (c *entitlementConnector) CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (Entitlement, error) {
	feature, err := c.featureConnector.GetFeature(ctx, models.NamespacedID{Namespace: input.Namespace, ID: input.FeatureID})
	if err != nil {
		return Entitlement{}, &productcatalog.FeatureNotFoundError{ID: input.FeatureID}
	}
	if feature.ArchivedAt != nil && feature.ArchivedAt.Before(time.Now()) {
		return Entitlement{}, &models.GenericUserError{Message: "Feature is archived"}
	}
	currentEntitlements, err := c.entitlementRepo.GetEntitlementsOfSubject(ctx, input.Namespace, models.SubjectKey(input.SubjectKey))
	if err != nil {
		return Entitlement{}, err
	}
	for _, ent := range currentEntitlements {
		if ent.FeatureID == input.FeatureID {
			return Entitlement{}, &AlreadyExistsError{EntitlementID: ent.ID, FeatureID: input.FeatureID, SubjectKey: input.SubjectKey}
		}
	}

	connector, err := c.getTypeConnector(input)
	if err != nil {
		return Entitlement{}, err
	}
	err = connector.SetDefaultsAndValidate(&input)
	if err != nil {
		return Entitlement{}, err
	}
	err = connector.ValidateForFeature(&input, feature)
	if err != nil {
		return Entitlement{}, err
	}

	ent, err := c.entitlementRepo.CreateEntitlement(ctx, input)
	if err != nil || ent == nil {
		return Entitlement{}, err
	}
	return *ent, nil
}

func (c *entitlementConnector) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error) {
	return c.entitlementRepo.GetEntitlementsOfSubject(ctx, namespace, subjectKey)
}

func (c *entitlementConnector) GetEntitlementValue(ctx context.Context, entitlementId models.NamespacedID, at time.Time) (EntitlementValue, error) {
	ent, err := c.entitlementRepo.GetEntitlement(ctx, entitlementId)
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

func (c *entitlementConnector) getTypeConnector(inp HasType) (SubTypeConnector, error) {
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
