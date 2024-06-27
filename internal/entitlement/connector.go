package entitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/recurrence"
)

type ListEntitlementsOrderBy string

const (
	ListEntitlementsOrderByCreatedAt ListEntitlementsOrderBy = "created_at"
	ListEntitlementsOrderByUpdatedAt ListEntitlementsOrderBy = "updated_at"
)

type ListEntitlementsParams struct {
	Namespace      string
	Limit          int
	Offset         int
	OrderBy        ListEntitlementsOrderBy
	IncludeDeleted bool
}

type Connector interface {
	CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (*Entitlement, error)
	GetEntitlement(ctx context.Context, namespace string, id string) (*Entitlement, error)
	DeleteEntitlement(ctx context.Context, namespace string, id string) error

	GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, id string, at time.Time) (EntitlementValue, error)

	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error)
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
	feature, err := c.featureConnector.GetFeature(ctx, input.Namespace, input.FeatureID)
	if err != nil || feature == nil {
		return nil, &productcatalog.FeatureNotFoundError{ID: input.FeatureID}
	}
	if feature.ArchivedAt != nil && feature.ArchivedAt.Before(time.Now()) {
		return nil, &models.GenericUserError{Message: "Feature is archived"}
	}
	currentEntitlements, err := c.entitlementRepo.GetEntitlementsOfSubject(ctx, input.Namespace, models.SubjectKey(input.SubjectKey))
	if err != nil {
		return nil, err
	}
	for _, ent := range currentEntitlements {
		if ent.FeatureID == feature.ID {
			return nil, &AlreadyExistsError{EntitlementID: ent.ID, FeatureID: feature.ID, SubjectKey: input.SubjectKey}
		}
	}

	connector, err := c.getTypeConnector(input)
	if err != nil {
		return nil, err
	}
	err = connector.BeforeCreate(&input, feature)
	if err != nil {
		return nil, err
	}

	var usagePeriod *UsagePeriod
	var currentUsagePeriod *recurrence.Period
	if input.UsagePeriod != nil {
		usagePeriod = input.UsagePeriod
		usagePeriod.Anchor = usagePeriod.Anchor.Truncate(time.Minute)

		calculatedPeriod, err := usagePeriod.GetCurrentPeriod()
		if err != nil {
			return nil, err
		}

		currentUsagePeriod = &calculatedPeriod
	}

	ent, err := c.entitlementRepo.CreateEntitlement(ctx, CreateEntitlementRepoInputs{
		Namespace:          input.Namespace,
		FeatureID:          input.FeatureID,
		SubjectKey:         input.SubjectKey,
		EntitlementType:    input.EntitlementType,
		Metadata:           input.Metadata,
		MeasureUsageFrom:   input.MeasureUsageFrom,
		IssueAfterReset:    input.IssueAfterReset,
		IsSoftLimit:        input.IsSoftLimit,
		Config:             input.Config,
		UsagePeriod:        usagePeriod,
		CurrentUsagePeriod: currentUsagePeriod,
	})
	if err != nil || ent == nil {
		return nil, err
	}

	err = connector.AfterCreate(ent)
	if err != nil {
		return nil, err
	}
	return ent, nil
}

func (c *entitlementConnector) GetEntitlement(ctx context.Context, namespace string, id string) (*Entitlement, error) {
	return c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
}

func (c *entitlementConnector) DeleteEntitlement(ctx context.Context, namespace string, id string) error {
	return c.entitlementRepo.DeleteEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
}

func (c *entitlementConnector) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error) {
	return c.entitlementRepo.GetEntitlementsOfSubject(ctx, namespace, subjectKey)
}

func (c *entitlementConnector) GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, id string, at time.Time) (EntitlementValue, error) {
	ent, err := c.entitlementRepo.GetEntitlementOfSubject(ctx, namespace, subjectKey, id)
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
