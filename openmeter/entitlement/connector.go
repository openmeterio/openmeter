package entitlement

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type ListEntitlementsOrderBy string

const (
	ListEntitlementsOrderByCreatedAt ListEntitlementsOrderBy = "created_at"
	ListEntitlementsOrderByUpdatedAt ListEntitlementsOrderBy = "updated_at"
)

func (o ListEntitlementsOrderBy) Values() []ListEntitlementsOrderBy {
	return []ListEntitlementsOrderBy{
		ListEntitlementsOrderByCreatedAt,
		ListEntitlementsOrderByUpdatedAt,
	}
}

func (o ListEntitlementsOrderBy) StrValues() []string {
	return slicesx.Map(o.Values(), func(v ListEntitlementsOrderBy) string {
		return string(v)
	})
}

type ListEntitlementsParams struct {
	IDs                 []string
	Namespaces          []string
	SubjectKeys         []string
	FeatureIDs          []string
	FeatureKeys         []string
	FeatureIDsOrKeys    []string
	EntitlementTypes    []EntitlementType
	OrderBy             ListEntitlementsOrderBy
	Order               sortx.Order
	IncludeDeleted      bool
	IncludeDeletedAfter time.Time
	Page                pagination.Page
	// will be deprecated
	Limit int
	// will be deprecated
	Offset int
}

type Connector interface {
	CreateEntitlement(ctx context.Context, input CreateEntitlementInputs) (*Entitlement, error)
	OverrideEntitlement(ctx context.Context, subject string, entitlementIdOrFeatureKey string, input CreateEntitlementInputs) (*Entitlement, error)
	GetEntitlement(ctx context.Context, namespace string, id string) (*Entitlement, error)
	DeleteEntitlement(ctx context.Context, namespace string, id string) error

	GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (EntitlementValue, error)

	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey, at time.Time) ([]Entitlement, error)
	ListEntitlements(ctx context.Context, params ListEntitlementsParams) (pagination.PagedResponse[Entitlement], error)
}

type entitlementConnector struct {
	meteredEntitlementConnector SubTypeConnector
	staticEntitlementConnector  SubTypeConnector
	booleanEntitlementConnector SubTypeConnector

	entitlementRepo  EntitlementRepo
	featureConnector feature.FeatureConnector
	meterRepo        meter.Repository

	publisher eventbus.Publisher
}

func NewEntitlementConnector(
	entitlementRepo EntitlementRepo,
	featureConnector feature.FeatureConnector,
	meterRepo meter.Repository,
	meteredEntitlementConnector SubTypeConnector,
	staticEntitlementConnector SubTypeConnector,
	booleanEntitlementConnector SubTypeConnector,
	publisher eventbus.Publisher,
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
	doInTx := func(ctx context.Context) (*Entitlement, error) {
		activeFromTime := defaultx.WithDefault(input.ActiveFrom, clock.Now())

		// ID has priority over key
		featureIdOrKey := input.FeatureID
		if featureIdOrKey == nil {
			featureIdOrKey = input.FeatureKey
		}
		if featureIdOrKey == nil {
			return nil, &models.GenericUserError{Message: "Feature ID or Key is required"}
		}

		feat, err := c.featureConnector.GetFeature(ctx, input.Namespace, *featureIdOrKey, feature.IncludeArchivedFeatureFalse)
		if err != nil || feat == nil {
			return nil, &feature.FeatureNotFoundError{ID: *featureIdOrKey}
		}

		// fill featureId and featureKey
		input.FeatureID = &feat.ID
		input.FeatureKey = &feat.Key

		currentEntitlements, err := c.entitlementRepo.GetActiveEntitlementsOfSubject(ctx, input.Namespace, models.SubjectKey(input.SubjectKey), activeFromTime)
		if err != nil {
			return nil, err
		}
		for _, ent := range currentEntitlements {
			// you can only have a single entitlemnet per feature key
			if ent.FeatureKey == feat.Key || ent.FeatureID == feat.ID {
				return nil, &AlreadyExistsError{EntitlementID: ent.ID, FeatureID: feat.ID, SubjectKey: input.SubjectKey}
			}
		}

		connector, err := c.getTypeConnector(input)
		if err != nil {
			return nil, err
		}
		repoInputs, err := connector.BeforeCreate(input, *feat)
		if err != nil {
			return nil, err
		}

		ent, err := c.entitlementRepo.CreateEntitlement(ctx, *repoInputs)
		if err != nil {
			return nil, err
		}

		err = connector.AfterCreate(ctx, ent)
		if err != nil {
			return nil, err
		}

		err = c.publisher.Publish(ctx, EntitlementCreatedEvent{
			Entitlement: *ent,
			Namespace: eventmodels.NamespaceID{
				ID: input.Namespace,
			},
		})
		if err != nil {
			return nil, err
		}

		return ent, err
	}

	return transaction.Run(ctx, c.entitlementRepo, doInTx)
}

// OverrideEntitlement replaces an existing entitlement with a new one.
func (c *entitlementConnector) OverrideEntitlement(ctx context.Context, subject string, entitlementIdOrFeatureKey string, input CreateEntitlementInputs) (*Entitlement, error) {
	// Validate input
	if subject != input.SubjectKey {
		return nil, &models.GenericUserError{Message: "Subject key in path and body do not match"}
	}

	// Find the entitlement to override
	oldEnt, err := c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: input.Namespace, ID: entitlementIdOrFeatureKey})

	if _, ok := lo.ErrorsAs[*NotFoundError](err); ok {
		oldEnt, err = c.entitlementRepo.GetActiveEntitlementOfSubjectAt(ctx, input.Namespace, input.SubjectKey, entitlementIdOrFeatureKey, clock.Now())
	}

	if err != nil {
		return nil, err
	}

	if oldEnt == nil {
		return nil, fmt.Errorf("inconsistency error, entitlement not found: %s", entitlementIdOrFeatureKey)
	}

	if oldEnt.DeletedAt != nil {
		return nil, fmt.Errorf("inconsistency error, entitlement already deleted: %s", oldEnt.ID)
	}

	// ID has priority over key
	featureIdOrKey := input.FeatureID
	if featureIdOrKey == nil {
		featureIdOrKey = input.FeatureKey
	}
	if featureIdOrKey == nil {
		return nil, &models.GenericUserError{Message: "Feature ID or Key is required"}
	}

	feat, err := c.featureConnector.GetFeature(ctx, input.Namespace, *featureIdOrKey, feature.IncludeArchivedFeatureFalse)
	if err != nil || feat == nil {
		return nil, &feature.FeatureNotFoundError{ID: *featureIdOrKey}
	}

	if feat.ID != oldEnt.FeatureID {
		return nil, &models.GenericUserError{Message: "Feature in path and body do not match"}
	}

	// Do the override in TX
	return transaction.Run(ctx, c.entitlementRepo, func(ctx context.Context) (*Entitlement, error) {
		// Delete previous entitlement
		// FIXME: we publish an event during this even if we fail later
		err := c.DeleteEntitlement(ctx, input.Namespace, oldEnt.ID)
		if err != nil {
			return nil, err
		}

		// Create new entitlement
		return c.CreateEntitlement(ctx, input)
	})
}

func (c *entitlementConnector) GetEntitlement(ctx context.Context, namespace string, id string) (*Entitlement, error) {
	return c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
}

func (c *entitlementConnector) DeleteEntitlement(ctx context.Context, namespace string, id string) error {
	doInTx := func(ctx context.Context) (*Entitlement, error) {
		ent, err := c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
		if err != nil {
			return nil, err
		}

		err = c.entitlementRepo.DeleteEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
		if err != nil {
			return nil, err
		}

		err = c.publisher.Publish(ctx, EntitlementDeletedEvent{
			Entitlement: *ent,
			Namespace: eventmodels.NamespaceID{
				ID: namespace,
			},
		})
		if err != nil {
			return nil, err
		}

		return ent, nil
	}

	_, err := transaction.Run(ctx, c.entitlementRepo, doInTx)
	return err
}

func (c *entitlementConnector) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey models.SubjectKey, at time.Time) ([]Entitlement, error) {
	return c.entitlementRepo.GetActiveEntitlementsOfSubject(ctx, namespace, subjectKey, at)
}

func (c *entitlementConnector) GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (EntitlementValue, error) {
	ent, err := c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: idOrFeatureKey})
	if _, ok := lo.ErrorsAs[*NotFoundError](err); ok {
		ent, err = c.entitlementRepo.GetActiveEntitlementOfSubjectAt(ctx, namespace, subjectKey, idOrFeatureKey, clock.Now())
	}
	if err != nil {
		return nil, err
	}
	connector, err := c.getTypeConnector(ent)
	if err != nil {
		return nil, err
	}
	return connector.GetValue(ent, at)
}

func (c *entitlementConnector) ListEntitlements(ctx context.Context, params ListEntitlementsParams) (pagination.PagedResponse[Entitlement], error) {
	if !params.Page.IsZero() {
		if err := params.Page.Validate(); err != nil {
			return pagination.PagedResponse[Entitlement]{}, err
		}
	}
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
