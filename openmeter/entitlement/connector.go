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
	ScheduleEntitlement(ctx context.Context, input CreateEntitlementInputs) (*Entitlement, error)
	// OverrideEntitlement replaces a currently active entitlement with a new one.
	OverrideEntitlement(ctx context.Context, subject string, entitlementIdOrFeatureKey string, input CreateEntitlementInputs) (*Entitlement, error)
	// SupersedeEntitlement replaces an entitlement by scheduling a new one
	SupersedeEntitlement(ctx context.Context, entitlementId string, input CreateEntitlementInputs) (*Entitlement, error)

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
	if input.ActiveTo != nil || input.ActiveFrom != nil {
		return nil, fmt.Errorf("activeTo and activeFrom are not supported in CreateEntitlement")
	}
	return c.ScheduleEntitlement(ctx, input)
}

// OverrideEntitlement replaces an existing entitlement with a new one.
func (c *entitlementConnector) OverrideEntitlement(ctx context.Context, subject string, entitlementIdOrFeatureKey string, input CreateEntitlementInputs) (*Entitlement, error) {
	// Validate input
	if subject != input.SubjectKey {
		return nil, &models.GenericUserError{Message: "Subject keys do not match"}
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

	if input.ActiveFrom != nil || input.ActiveTo != nil {
		return nil, &models.GenericUserError{Message: "ActiveFrom and ActiveTo are not supported in OverrideEntitlement"}
	}

	return c.SupersedeEntitlement(ctx, oldEnt.ID, input)
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

	// If the entitlement is not active it cannot provide access
	if !ent.IsActive(at) {
		return &NoAccessValue{}, nil
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
