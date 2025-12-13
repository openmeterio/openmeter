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
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ServiceConfig struct {
	EntitlementRepo  entitlement.EntitlementRepo
	FeatureConnector feature.FeatureConnector
	MeterService     meter.Service

	MeteredEntitlementConnector meteredentitlement.Connector
	StaticEntitlementConnector  entitlement.SubTypeConnector
	BooleanEntitlementConnector entitlement.SubTypeConnector

	Publisher eventbus.Publisher
	Locker    *lockr.Locker
}

type service struct {
	meteredEntitlementConnector meteredentitlement.Connector
	staticEntitlementConnector  entitlement.SubTypeConnector
	booleanEntitlementConnector entitlement.SubTypeConnector

	entitlementRepo  entitlement.EntitlementRepo
	featureConnector feature.FeatureConnector
	meterService     meter.Service

	hooks models.ServiceHookRegistry[entitlement.Entitlement]

	publisher eventbus.Publisher
	locker    *lockr.Locker
}

func (s *service) RegisterHooks(hooks ...models.ServiceHook[entitlement.Entitlement]) {
	s.hooks.RegisterHooks(hooks...)
}

func NewEntitlementService(
	config ServiceConfig,
) entitlement.Service {
	return &service{
		meteredEntitlementConnector: config.MeteredEntitlementConnector,
		staticEntitlementConnector:  config.StaticEntitlementConnector,
		booleanEntitlementConnector: config.BooleanEntitlementConnector,
		entitlementRepo:             config.EntitlementRepo,
		featureConnector:            config.FeatureConnector,
		meterService:                config.MeterService,
		publisher:                   config.Publisher,
		locker:                      config.Locker,
	}
}

func (c *service) CreateEntitlement(ctx context.Context, input entitlement.CreateEntitlementInputs, grants []entitlement.CreateEntitlementGrantInputs) (*entitlement.Entitlement, error) {
	return transaction.Run(ctx, c.entitlementRepo, func(ctx context.Context) (*entitlement.Entitlement, error) {
		if input.ActiveTo != nil || input.ActiveFrom != nil {
			return nil, fmt.Errorf("activeTo and activeFrom are not supported in CreateEntitlement")
		}

		if len(grants) > 0 && input.EntitlementType != entitlement.EntitlementTypeMetered {
			return nil, entitlement.ErrEntitlementGrantsOnlySupportedForMeteredEntitlements.WithAttr("entitlement_type", input.EntitlementType).WithAttr("grants", grants)
		}

		ent, err := c.ScheduleEntitlement(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, grant := range grants {
			_, err := c.meteredEntitlementConnector.CreateGrant(ctx, ent.Namespace, ent.Customer.ID, ent.ID, grant)
			if err != nil {
				return nil, err
			}
		}

		return ent, nil
	})
}

// OverrideEntitlement replaces an existing entitlement with a new one.
func (c *service) OverrideEntitlement(ctx context.Context, customerID string, entitlementIdOrFeatureKey string, input entitlement.CreateEntitlementInputs, grants []entitlement.CreateEntitlementGrantInputs) (*entitlement.Entitlement, error) {
	return transaction.Run(ctx, c.entitlementRepo, func(ctx context.Context) (*entitlement.Entitlement, error) {
		// Validate customer match in input
		if customerID != input.UsageAttribution.ID {
			return nil, entitlement.ErrEntitlementCreatePropertyMismatch.WithAttr("customer_id", customerID).WithAttr("usage_attribution_id", input.UsageAttribution.ID)
		}

		if len(grants) > 0 && input.EntitlementType != entitlement.EntitlementTypeMetered {
			return nil, entitlement.ErrEntitlementGrantsOnlySupportedForMeteredEntitlements.WithAttr("entitlement_type", input.EntitlementType).WithAttr("grants", grants)
		}

		// Find the entitlement to override
		oldEnt, err := c.GetEntitlementOfCustomerAt(ctx, input.Namespace, customerID, entitlementIdOrFeatureKey, clock.Now())
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

		ent, err := c.SupersedeEntitlement(ctx, oldEnt.ID, input)
		if err != nil {
			return nil, err
		}

		for _, grant := range grants {
			_, err := c.meteredEntitlementConnector.CreateGrant(ctx, ent.Namespace, ent.Customer.ID, ent.ID, grant)
			if err != nil {
				return nil, err
			}
		}

		return ent, nil
	})
}

func (c *service) GetEntitlement(ctx context.Context, namespace string, id string) (*entitlement.Entitlement, error) {
	return c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
}

func (c *service) DeleteEntitlement(ctx context.Context, namespace string, id string, at time.Time) error {
	doInTx := func(ctx context.Context) (*entitlement.Entitlement, error) {
		ent, err := c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id})
		if err != nil {
			return nil, err
		}

		if err := c.hooks.PreDelete(ctx, ent); err != nil {
			return nil, err
		}

		err = c.entitlementRepo.DeleteEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: id}, at)
		if err != nil {
			return nil, err
		}

		err = c.publisher.Publish(ctx, entitlement.NewEntitlementDeletedEventPayloadV2(*ent))
		if err != nil {
			return nil, err
		}

		return ent, nil
	}

	_, err := transaction.Run(ctx, c.entitlementRepo, doInTx)
	return err
}

func (c *service) GetEntitlementsOfCustomer(ctx context.Context, namespace string, customerId string, at time.Time) ([]entitlement.Entitlement, error) {
	ents, err := c.entitlementRepo.ListEntitlements(
		ctx,
		entitlement.ListEntitlementsParams{
			CustomerIDs:         []string{customerId},
			Namespaces:          []string{namespace},
			ActiveAt:            &at,
			IncludeDeleted:      true,
			IncludeDeletedAfter: at,
			// We leave page empty to get all entitlements
		},
	)
	if err != nil {
		return nil, err
	}
	return ents.Items, nil
}

func (c *service) GetEntitlementOfCustomerAt(ctx context.Context, namespace string, customerID string, idOrFeatureKey string, at time.Time) (*entitlement.Entitlement, error) {
	ent, err := c.entitlementRepo.GetEntitlement(ctx, models.NamespacedID{Namespace: namespace, ID: idOrFeatureKey})
	if _, ok := lo.ErrorsAs[*entitlement.NotFoundError](err); ok {
		ent, err = c.entitlementRepo.GetActiveEntitlementOfCustomerAt(ctx, namespace, customerID, idOrFeatureKey, at)
	}
	return ent, err
}

func (c *service) GetEntitlementValue(ctx context.Context, namespace string, customerID string, idOrFeatureKey string, at time.Time) (entitlement.EntitlementValue, error) {
	ent, err := c.GetEntitlementOfCustomerAt(ctx, namespace, customerID, idOrFeatureKey, at)
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

func (c *service) ListEntitlements(ctx context.Context, params entitlement.ListEntitlementsParams) (pagination.Result[entitlement.Entitlement], error) {
	if !params.Page.IsZero() {
		if err := params.Page.Validate(); err != nil {
			return pagination.Result[entitlement.Entitlement]{}, err
		}
	}
	return c.entitlementRepo.ListEntitlements(ctx, params)
}

func (c *service) GetAccess(ctx context.Context, namespace string, customerId string) (entitlement.Access, error) {
	now := clock.Now()

	entitlements, err := c.GetEntitlementsOfCustomer(ctx, namespace, customerId, now)
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
			entValue, err := c.GetEntitlementValue(ctx, namespace, customerId, entit.ID, now)
			if err != nil {
				return fmt.Errorf("failed to get entitlement value for ID %s: %w", entit.ID, err)
			}

			// Store the result
			result.Store(entit.FeatureKey, entitlement.EntitlementValueWithId{
				Type:  entit.GetType(),
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

func (c *service) getTypeConnector(inp entitlement.TypedEntitlement) (entitlement.SubTypeConnector, error) {
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
