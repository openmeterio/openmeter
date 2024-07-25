package entitlement

import (
	"context"
	"fmt"
	"time"

	eventmodels "github.com/openmeterio/openmeter/internal/event/models"
	"github.com/openmeterio/openmeter/internal/event/publisher"
	"github.com/openmeterio/openmeter/internal/event/spec"
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ListEntitlementsOrderBy string

const (
	ListEntitlementsOrderByCreatedAt ListEntitlementsOrderBy = "created_at"
	ListEntitlementsOrderByUpdatedAt ListEntitlementsOrderBy = "updated_at"
)

type ListEntitlementsParams struct {
	Namespaces     []string
	SubjectKey     string
	FeatureIDs     []string
	FeatureKeys    []string
	Limit          int
	Offset         int
	OrderBy        ListEntitlementsOrderBy
	IncludeDeleted bool
}

type Connector interface {
	CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (*Entitlement, error)
	GetEntitlement(ctx context.Context, namespace string, id string) (*Entitlement, error)
	DeleteEntitlement(ctx context.Context, namespace string, id string) error

	GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (EntitlementValue, error)

	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey) ([]Entitlement, error)
	ListEntitlements(ctx context.Context, params ListEntitlementsParams) ([]Entitlement, error)
}

type entitlementConnector struct {
	meteredEntitlementConnector SubTypeConnector
	staticEntitlementConnector  SubTypeConnector
	booleanEntitlementConnector SubTypeConnector

	entitlementRepo  EntitlementRepo
	featureConnector productcatalog.FeatureConnector
	meterRepo        meter.Repository

	publisher publisher.TopicPublisher
}

func NewEntitlementConnector(
	entitlementRepo EntitlementRepo,
	featureConnector productcatalog.FeatureConnector,
	meterRepo meter.Repository,
	meteredEntitlementConnector SubTypeConnector,
	staticEntitlementConnector SubTypeConnector,
	booleanEntitlementConnector SubTypeConnector,
	publisher publisher.TopicPublisher,
) Connector {
	return &entitlementConnector{
		meteredEntitlementConnector: meteredEntitlementConnector,
		staticEntitlementConnector:  staticEntitlementConnector,
		booleanEntitlementConnector: booleanEntitlementConnector,
		entitlementRepo:             entitlementRepo,
		featureConnector:            featureConnector,
		meterRepo:                   meterRepo,
		publisher:                   publisher,
	}
}

func (c *entitlementConnector) CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (*Entitlement, error) {
	// ID has priority over key
	idOrFeatureKey := input.FeatureID
	if idOrFeatureKey == nil {
		idOrFeatureKey = input.FeatureKey
	}
	if idOrFeatureKey == nil {
		return nil, &models.GenericUserError{Message: "Feature ID or Key is required"}
	}

	feature, err := c.featureConnector.GetFeature(ctx, input.Namespace, *idOrFeatureKey, productcatalog.IncludeArchivedFeatureFalse)
	if err != nil || feature == nil {
		return nil, &productcatalog.FeatureNotFoundError{ID: *idOrFeatureKey}
	}

	// fill featureId and featureKey
	input.FeatureID = &feature.ID
	input.FeatureKey = &feature.Key

	currentEntitlements, err := c.entitlementRepo.GetEntitlementsOfSubject(ctx, input.Namespace, models.SubjectKey(input.SubjectKey))
	if err != nil {
		return nil, err
	}
	for _, ent := range currentEntitlements {
		// you can only have a single entitlemnet per feature key
		if ent.FeatureKey == feature.Key || ent.FeatureID == feature.ID {
			return nil, &AlreadyExistsError{EntitlementID: ent.ID, FeatureID: feature.ID, SubjectKey: input.SubjectKey}
		}
	}

	connector, err := c.getTypeConnector(input)
	if err != nil {
		return nil, err
	}
	repoInputs, err := connector.BeforeCreate(input, *feature)
	if err != nil {
		return nil, err
	}

	ent, err := entutils.StartAndRunTx(ctx, c.entitlementRepo, func(ctx context.Context, tx *entutils.TxDriver) (*Entitlement, error) {
		txCtx := entutils.NewTxContext(ctx, tx)

		ent, err := c.entitlementRepo.WithTx(txCtx, tx).CreateEntitlement(txCtx, *repoInputs)
		if err != nil {
			return nil, err
		}

		err = connector.AfterCreate(txCtx, ent)
		if err != nil {
			return nil, err
		}

		event, err := spec.NewCloudEvent(
			spec.EventSpec{
				Source:  spec.ComposeResourcePath(input.Namespace, spec.EntityEntitlement, ent.ID),
				Subject: spec.ComposeResourcePath(input.Namespace, spec.EntitySubjectKey, ent.SubjectKey),
			},
			EntitlementCreatedEvent{
				Entitlement: *ent,
				Namespace: eventmodels.NamespaceID{
					ID: input.Namespace,
				},
			},
		)
		if err != nil {
			return nil, err
		}

		if err := c.publisher.Publish(event); err != nil {
			return nil, err
		}

		return ent, nil
	})

	return ent, err
}

func (c *entitlementConnector) GetEntitlement(ctx context.Context, namespace string, id string) (*Entitlement, error) {
	return c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
}

func (c *entitlementConnector) DeleteEntitlement(ctx context.Context, namespace string, id string) error {
	_, err := entutils.StartAndRunTx(ctx, c.entitlementRepo, func(ctx context.Context, tx *entutils.TxDriver) (*Entitlement, error) {
		txCtx := entutils.NewTxContext(ctx, tx)

		ent, err := c.entitlementRepo.WithTx(txCtx, tx).GetEntitlement(txCtx, models.NamespacedID{Namespace: namespace, ID: id})
		if err != nil {
			return nil, err
		}

		err = c.entitlementRepo.WithTx(txCtx, tx).DeleteEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
		if err != nil {
			return nil, err
		}

		event, err := spec.NewCloudEvent(
			spec.EventSpec{
				Source:  spec.ComposeResourcePath(namespace, spec.EntityEntitlement, ent.ID),
				Subject: spec.ComposeResourcePath(namespace, spec.EntitySubjectKey, ent.SubjectKey),
			},
			EntitlementDeletedEvent{
				Entitlement: *ent,
				Namespace: eventmodels.NamespaceID{
					ID: namespace,
				},
			},
		)
		if err != nil {
			return nil, err
		}

		if err := c.publisher.Publish(event); err != nil {
			return nil, err
		}

		return ent, nil
	})

	return err
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
