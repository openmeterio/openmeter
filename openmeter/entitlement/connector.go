package entitlement

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

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
	ExcludeInactive     bool
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
	DeleteEntitlement(ctx context.Context, namespace string, id string, at time.Time) error

	GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (EntitlementValue, error)

	GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey string, at time.Time) ([]Entitlement, error)
	ListEntitlements(ctx context.Context, params ListEntitlementsParams) (pagination.PagedResponse[Entitlement], error)

	// Attempts to get the entitlement in an ambiguous situation where it's unclear if the entitlement is referenced by ID or FeatureKey + SubjectKey.
	// First attempts to resolve by ID, then by FeatureKey + SubjectKey.
	//
	// For consistency, it is forbidden for entitlements to be created for featueres the keys of which could be mistaken for entitlement IDs.
	GetEntitlementOfSubjectAt(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (*Entitlement, error)

	// GetAccess returns the access of a subject for a given namespace.
	// It returns a map of featureKey to entitlement value + ID.
	GetAccess(ctx context.Context, namespace string, subjectKey string) (Access, error)
}

type entitlementConnector struct {
	meteredEntitlementConnector SubTypeConnector
	staticEntitlementConnector  SubTypeConnector
	booleanEntitlementConnector SubTypeConnector

	entitlementRepo  EntitlementRepo
	featureConnector feature.FeatureConnector
	meterService     meter.Service

	publisher eventbus.Publisher
}

func NewEntitlementConnector(
	entitlementRepo EntitlementRepo,
	featureConnector feature.FeatureConnector,
	meterService meter.Service,
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
		meterService:                meterService,
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
		return nil, models.NewGenericValidationError(fmt.Errorf("subject keys do not match"))
	}

	// Find the entitlement to override
	oldEnt, err := c.GetEntitlementOfSubjectAt(ctx, input.Namespace, subject, entitlementIdOrFeatureKey, clock.Now())
	if err != nil {
		return nil, err
	}

	if oldEnt == nil {
		return nil, fmt.Errorf("inconsistency error, entitlement not found: %s", entitlementIdOrFeatureKey)
	}

	if oldEnt.DeletedAt != nil {
		return nil, models.NewGenericValidationError(fmt.Errorf("entitlement already deleted: %s", oldEnt.ID))
	}

	if input.ActiveFrom != nil || input.ActiveTo != nil {
		return nil, models.NewGenericValidationError(fmt.Errorf("the ActiveFrom and ActiveTo are not supported in OverrideEntitlement"))
	}

	return c.SupersedeEntitlement(ctx, oldEnt.ID, input)
}

func (c *entitlementConnector) GetEntitlement(ctx context.Context, namespace string, id string) (*Entitlement, error) {
	return c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
}

func (c *entitlementConnector) DeleteEntitlement(ctx context.Context, namespace string, id string, at time.Time) error {
	doInTx := func(ctx context.Context) (*Entitlement, error) {
		ent, err := c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
		if err != nil {
			return nil, err
		}

		err = c.entitlementRepo.DeleteEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id}, at)
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

func (c *entitlementConnector) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey string, at time.Time) ([]Entitlement, error) {
	return c.entitlementRepo.GetActiveEntitlementsOfSubject(ctx, namespace, subjectKey, at)
}

func (c *entitlementConnector) GetEntitlementOfSubjectAt(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (*Entitlement, error) {
	ent, err := c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: idOrFeatureKey})
	if _, ok := lo.ErrorsAs[*NotFoundError](err); ok {
		ent, err = c.entitlementRepo.GetActiveEntitlementOfSubjectAt(ctx, namespace, subjectKey, idOrFeatureKey, at)
	}
	return ent, err
}

func (c *entitlementConnector) GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (EntitlementValue, error) {
	ent, err := c.GetEntitlementOfSubjectAt(ctx, namespace, subjectKey, idOrFeatureKey, at)
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
	return connector.GetValue(ctx, ent, at)
}

func (c *entitlementConnector) ListEntitlements(ctx context.Context, params ListEntitlementsParams) (pagination.PagedResponse[Entitlement], error) {
	if !params.Page.IsZero() {
		if err := params.Page.Validate(); err != nil {
			return pagination.PagedResponse[Entitlement]{}, err
		}
	}
	return c.entitlementRepo.ListEntitlements(ctx, params)
}

func (c *entitlementConnector) GetAccess(ctx context.Context, namespace string, subjectKey string) (Access, error) {
	now := clock.Now()

	entitlements, err := c.GetEntitlementsOfSubject(ctx, namespace, subjectKey, now)
	if err != nil {
		return Access{}, err
	}

	if len(entitlements) == 0 {
		return Access{}, nil
	}

	var result sync.Map

	g, ctx := errgroup.WithContext(ctx)

	// Let's limit concurrency
	const maxConcurrency = 10 // TODO: Make this configurable
	sem := semaphore.NewWeighted(int64(maxConcurrency))

	// Start a goroutine for each entitlement
	for _, ent := range entitlements {
		// Let's create a local copy for the goroutine
		entitlement := ent

		weight := int64(1)

		if err := sem.Acquire(ctx, weight); err != nil {
			// Ctx canceled or never rly happens, but critical if does
			return Access{}, err
		}

		g.Go(func() error {
			defer sem.Release(weight)

			// Get the entitlement value
			entValue, err := c.GetEntitlementValue(ctx, namespace, subjectKey, entitlement.ID, now)
			if err != nil {
				return fmt.Errorf("failed to get entitlement value for ID %s: %w", entitlement.ID, err)
			}

			// Store the result
			result.Store(entitlement.FeatureKey, EntitlementValueWithId{
				EntitlementValue: entValue,
				ID:               entitlement.ID,
			})

			return nil
		})
	}

	// Wait for all goroutines to complete and return any error
	if err := g.Wait(); err != nil {
		return Access{}, err
	}

	// Convert sync.Map to regular map for return value
	finalResult := make(map[string]EntitlementValueWithId)
	var conversionErrors []error

	result.Range(func(key, value any) bool {
		// Better safe than sorry
		k, ok := key.(string)
		if !ok {
			conversionErrors = append(conversionErrors, fmt.Errorf("unexpected key type in entitlement map: %T", key))
			return false
		}

		v, ok := value.(EntitlementValueWithId)
		if !ok {
			conversionErrors = append(conversionErrors, fmt.Errorf("unexpected value type in entitlement map for key %s: %T", k, value))
			return false
		}

		finalResult[k] = v
		return true
	})

	// If there were any type assertion errors, return them
	if len(conversionErrors) > 0 {
		return Access{}, fmt.Errorf("errors converting entitlement values: %v", conversionErrors)
	}

	return Access{Entitlements: finalResult}, nil
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
