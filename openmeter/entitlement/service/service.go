package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	eventmodels "github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type entitlementConnector struct {
	meteredEntitlementConnector entitlement.SubTypeConnector
	staticEntitlementConnector  entitlement.SubTypeConnector
	booleanEntitlementConnector entitlement.SubTypeConnector

	entitlementRepo  entitlement.EntitlementRepo
	featureConnector feature.FeatureConnector
	meterService     meter.Service

	publisher eventbus.Publisher
	locker    *lockr.Locker
}

func NewEntitlementConnector(
	entitlementRepo entitlement.EntitlementRepo,
	featureConnector feature.FeatureConnector,
	meterService meter.Service,
	meteredEntitlementConnector entitlement.SubTypeConnector,
	staticEntitlementConnector entitlement.SubTypeConnector,
	booleanEntitlementConnector entitlement.SubTypeConnector,
	publisher eventbus.Publisher,
	locker *lockr.Locker,
) entitlement.Connector {
	return &entitlementConnector{
		meteredEntitlementConnector: meteredEntitlementConnector,
		staticEntitlementConnector:  staticEntitlementConnector,
		booleanEntitlementConnector: booleanEntitlementConnector,
		entitlementRepo:             entitlementRepo,
		featureConnector:            featureConnector,
		meterService:                meterService,
		publisher:                   publisher,
		locker:                      locker,
	}
}

func (c *entitlementConnector) CreateEntitlement(ctx context.Context, input entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
	if input.ActiveTo != nil || input.ActiveFrom != nil {
		return nil, fmt.Errorf("activeTo and activeFrom are not supported in CreateEntitlement")
	}
	return c.ScheduleEntitlement(ctx, input)
}

// OverrideEntitlement replaces an existing entitlement with a new one.
func (c *entitlementConnector) OverrideEntitlement(ctx context.Context, subject string, entitlementIdOrFeatureKey string, input entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
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

func (c *entitlementConnector) GetEntitlement(ctx context.Context, namespace string, id string) (*entitlement.Entitlement, error) {
	return c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
}

func (c *entitlementConnector) DeleteEntitlement(ctx context.Context, namespace string, id string, at time.Time) error {
	doInTx := func(ctx context.Context) (*entitlement.Entitlement, error) {
		ent, err := c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
		if err != nil {
			return nil, err
		}

		err = c.entitlementRepo.DeleteEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id}, at)
		if err != nil {
			return nil, err
		}

		err = c.publisher.Publish(ctx, entitlement.EntitlementDeletedEvent{
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

func (c *entitlementConnector) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey string, at time.Time) ([]entitlement.Entitlement, error) {
	return c.entitlementRepo.GetActiveEntitlementsOfSubject(ctx, namespace, subjectKey, at)
}

func (c *entitlementConnector) GetEntitlementOfSubjectAt(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (*entitlement.Entitlement, error) {
	ent, err := c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: idOrFeatureKey})
	if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
		ent, err = c.entitlementRepo.GetActiveEntitlementOfSubjectAt(ctx, namespace, subjectKey, idOrFeatureKey, at)
	}
	return ent, err
}

func (c *entitlementConnector) GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (entitlement.EntitlementValue, error) {
	ent, err := c.GetEntitlementOfSubjectAt(ctx, namespace, subjectKey, idOrFeatureKey, at)
	if err != nil {
		return nil, err
	}

	// If the entitlement is not active it cannot provide access
	if !ent.IsActive(at) {
		return &entitlement.NoAccessValue{}, nil
	}

	connector, err := c.getTypeConnector(ent)
	if err != nil {
		return nil, err
	}
	return connector.GetValue(ctx, ent, at)
}

func (c *entitlementConnector) ListEntitlements(ctx context.Context, params entitlement.ListEntitlementsParams) (pagination.PagedResponse[entitlement.Entitlement], error) {
	if !params.Page.IsZero() {
		if err := params.Page.Validate(); err != nil {
			return pagination.PagedResponse[entitlement.Entitlement]{}, err
		}
	}
	return c.entitlementRepo.ListEntitlements(ctx, params)
}

func (c *entitlementConnector) GetAccess(ctx context.Context, namespace string, subjectKey string) (entitlement.Access, error) {
	now := clock.Now()

	entitlements, err := c.GetEntitlementsOfSubject(ctx, namespace, subjectKey, now)
	if err != nil {
		return entitlement.Access{}, err
	}

	if len(entitlements) == 0 {
		return entitlement.Access{}, nil
	}

	var result sync.Map

	g, ctx := errgroup.WithContext(ctx)

	// Let's limit concurrency
	const maxConcurrency = 10 // TODO: Make this configurable
	sem := semaphore.NewWeighted(int64(maxConcurrency))

	// Start a goroutine for each entitlement
	for _, ent := range entitlements {
		// Let's create a local copy for the goroutine
		entit := ent

		weight := int64(1)

		if err := sem.Acquire(ctx, weight); err != nil {
			// Ctx canceled or never rly happens, but critical if does
			return entitlement.Access{}, err
		}

		g.Go(func() error {
			defer sem.Release(weight)

			// Get the entitlement value
			entValue, err := c.GetEntitlementValue(ctx, namespace, subjectKey, entit.ID, now)
			if err != nil {
				return fmt.Errorf("failed to get entitlement value for ID %s: %w", entit.ID, err)
			}

			// Store the result
			result.Store(entit.FeatureKey, entitlement.EntitlementValueWithId{
				Value: entValue,
				ID:    entit.ID,
			})

			return nil
		})
	}

	// Wait for all goroutines to complete and return any error
	if err := g.Wait(); err != nil {
		return entitlement.Access{}, err
	}

	// Convert sync.Map to regular map for return value
	finalResult := make(map[string]entitlement.EntitlementValueWithId)
	var conversionErrors []error

	result.Range(func(key, value any) bool {
		// Better safe than sorry
		k, ok := key.(string)
		if !ok {
			conversionErrors = append(conversionErrors, fmt.Errorf("unexpected key type in entitlement map: %T", key))
			return false
		}

		v, ok := value.(entitlement.EntitlementValueWithId)
		if !ok {
			conversionErrors = append(conversionErrors, fmt.Errorf("unexpected value type in entitlement map for key %s: %T", k, value))
			return false
		}

		finalResult[k] = v
		return true
	})

	// If there were any type assertion errors, return them
	if len(conversionErrors) > 0 {
		return entitlement.Access{}, fmt.Errorf("errors converting entitlement values: %v", conversionErrors)
	}

	return entitlement.Access{Entitlements: finalResult}, nil
}

func (c *entitlementConnector) getTypeConnector(inp entitlement.TypedEntitlement) (entitlement.SubTypeConnector, error) {
	entitlementType := inp.GetType()
	switch entitlementType {
	case entitlement.EntitlementTypeMetered:
		return c.meteredEntitlementConnector, nil
	case entitlement.EntitlementTypeStatic:
		return c.staticEntitlementConnector, nil
	case entitlement.EntitlementTypeBoolean:
		return c.booleanEntitlementConnector, nil
	default:
		return nil, fmt.Errorf("unsupported entitlement type: %s", entitlementType)
	}
}
