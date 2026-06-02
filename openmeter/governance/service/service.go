package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/governance"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// featureFetchLimit caps the org-wide feature fetch used when no feature filter is given.
// Acceptable for prototype scale; revisit if feature counts grow large.
const featureFetchLimit = 10_000

// Config holds the collaborating services for the governance Service.
type Config struct {
	CustomerService    customer.Service
	EntitlementService entitlement.Service
	FeatureConnector   feature.FeatureConnector
}

func (c Config) Validate() error {
	var errs []error

	if c.CustomerService == nil {
		errs = append(errs, errors.New("customer service is required"))
	}

	if c.EntitlementService == nil {
		errs = append(errs, errors.New("entitlement service is required"))
	}

	if c.FeatureConnector == nil {
		errs = append(errs, errors.New("feature connector is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (governance.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		customerService:    config.CustomerService,
		entitlementService: config.EntitlementService,
		featureConnector:   config.FeatureConnector,
	}, nil
}

type service struct {
	customerService    customer.Service
	entitlementService entitlement.Service
	featureConnector   feature.FeatureConnector
}

var _ governance.Service = (*service)(nil)

// resolvedCustomer groups the matched input keys for a single customer.
type resolvedCustomer struct {
	Customer customer.Customer
	Matched  []string
}

func (s *service) QueryAccess(ctx context.Context, input governance.QueryAccessInput) (governance.QueryResult, error) {
	if err := input.Validate(); err != nil {
		return governance.QueryResult{}, err
	}

	// Resolve each input key to a customer; deduplicate by customer ID.
	customerMap := make(map[string]*resolvedCustomer)
	var queryErrors []governance.QueryError

	for _, key := range input.CustomerKeys {
		cus, err := s.customerService.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
			Namespace: input.Namespace,
			Key:       key,
		})
		if err != nil {
			if models.IsGenericNotFoundError(err) {
				queryErrors = append(queryErrors, governance.QueryError{
					CustomerKey: key,
					Code:        governance.QueryErrorCustomerNotFound,
					Message:     "customer not found",
				})
				continue
			}
			return governance.QueryResult{}, fmt.Errorf("resolve customer key %q: %w", key, err)
		}

		if rc, ok := customerMap[cus.ID]; ok {
			rc.Matched = append(rc.Matched, key)
		} else {
			customerMap[cus.ID] = &resolvedCustomer{
				Customer: *cus,
				Matched:  []string{key},
			}
		}
	}

	// Sort by (CreatedAt, ID) for stable cursor pagination.
	customers := lo.Values(customerMap)
	sort.Slice(customers, func(i, j int) bool {
		ti := customers[i].Customer.CreatedAt
		tj := customers[j].Customer.CreatedAt
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return customers[i].Customer.ID < customers[j].Customer.ID
	})

	customers, hasPrev, hasNext := paginate(customers, input)

	var featureKeys []string
	if len(input.FeatureKeys) > 0 {
		featureKeys = input.FeatureKeys
	}

	now := clock.Now()
	results := make([]governance.CustomerAccess, 0, len(customers))
	for _, rc := range customers {
		access, err := s.entitlementService.GetAccess(ctx, input.Namespace, rc.Customer.ID)
		if err != nil {
			return governance.QueryResult{}, fmt.Errorf("get access for customer %s: %w", rc.Customer.ID, err)
		}

		featureAccess, err := s.buildFeatureAccess(ctx, input.Namespace, featureKeys, access)
		if err != nil {
			return governance.QueryResult{}, fmt.Errorf("build feature access for customer %s: %w", rc.Customer.ID, err)
		}

		results = append(results, governance.CustomerAccess{
			Customer:  rc.Customer,
			Matched:   rc.Matched,
			Features:  featureAccess,
			UpdatedAt: now,
		})
	}

	out := governance.QueryResult{
		Customers: results,
		Errors:    queryErrors,
		HasPrev:   hasPrev,
		HasNext:   hasNext,
	}
	if len(customers) > 0 {
		out.First = lo.ToPtr(cursorFor(customers[0]))
		out.Last = lo.ToPtr(cursorFor(customers[len(customers)-1]))
	}

	return out, nil
}

// cursorFor builds the pagination cursor for a resolved customer. CreatedAt is truncated
// to second precision to match the RFC3339 encoding used by cursor strings.
func cursorFor(rc *resolvedCustomer) pagination.Cursor {
	return pagination.NewCursor(rc.Customer.CreatedAt.Truncate(time.Second), rc.Customer.ID)
}

// paginate applies cursor pagination over the sorted customers and reports whether adjacent
// pages exist. Exactly one of input.After / input.Before may be set (enforced by Validate).
func paginate(customers []*resolvedCustomer, input governance.QueryAccessInput) (page []*resolvedCustomer, hasPrev, hasNext bool) {
	if input.Before != nil {
		// Backward: take the last pageSize items strictly before the cursor.
		bc := *input.Before
		end := 0
		for i, rc := range customers {
			c := cursorFor(rc)
			if c.Time.After(bc.Time) || (c.Time.Equal(bc.Time) && c.ID >= bc.ID) {
				break
			}
			end = i + 1
		}
		candidates := customers[:end]
		hasPrev = len(candidates) > input.PageSize
		if hasPrev {
			candidates = candidates[len(candidates)-input.PageSize:]
		}
		// next is always set in backward mode: the before-cursor item itself is forward.
		return candidates, hasPrev, true
	}

	// Forward (after cursor or first page).
	start := 0
	if input.After != nil {
		ac := *input.After
		start = len(customers) // beyond all items if cursor is past the end
		for i, rc := range customers {
			c := cursorFor(rc)
			if c.Time.After(ac.Time) || (c.Time.Equal(ac.Time) && c.ID > ac.ID) {
				start = i
				break
			}
		}
	}
	hasPrev = start > 0
	page = customers[start:]
	hasNext = len(page) > input.PageSize
	if hasNext {
		page = page[:input.PageSize]
	}
	return page, hasPrev, hasNext
}

// buildFeatureAccess returns the feature access map for a single customer.
// If featureKeys is non-empty, only those keys are evaluated.
// If featureKeys is empty, all non-archived features in the namespace are returned;
// features the customer has no entitlement for are marked feature-unavailable.
func (s *service) buildFeatureAccess(ctx context.Context, ns string, featureKeys []string, access entitlement.Access) (map[string]governance.FeatureAccess, error) {
	result := make(map[string]governance.FeatureAccess)

	if len(featureKeys) == 0 {
		orgFeatures, err := s.listAllOrgFeatures(ctx, ns)
		if err != nil {
			return nil, err
		}
		for _, f := range orgFeatures {
			if ev, ok := access.Entitlements[f.Key]; ok {
				result[f.Key] = mapEntitlementToAccess(ev.Value)
			} else {
				result[f.Key] = featureUnavailable(f.Key)
			}
		}
		return result, nil
	}

	for _, key := range featureKeys {
		ev, ok := access.Entitlements[key]
		if !ok {
			fa, err := s.resolveAbsentFeature(ctx, ns, key)
			if err != nil {
				return nil, err
			}
			result[key] = fa
			continue
		}
		result[key] = mapEntitlementToAccess(ev.Value)
	}

	return result, nil
}

// listAllOrgFeatures fetches all non-archived features in the namespace in one shot.
func (s *service) listAllOrgFeatures(ctx context.Context, ns string) ([]feature.Feature, error) {
	res, err := s.featureConnector.ListFeatures(ctx, feature.ListFeaturesParams{
		Namespace:       ns,
		IncludeArchived: false,
		Limit:           featureFetchLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list org features: %w", err)
	}
	return res.Items, nil
}

// resolveAbsentFeature determines why a requested feature key is absent from GetAccess results:
// either the feature doesn't exist in the org (feature-not-found) or the customer has no
// entitlement for it (feature-unavailable).
func (s *service) resolveAbsentFeature(ctx context.Context, ns, featureKey string) (governance.FeatureAccess, error) {
	_, err := s.featureConnector.GetFeature(ctx, ns, featureKey, feature.IncludeArchivedFeatureFalse)
	if err != nil {
		var fne *feature.FeatureNotFoundError
		if errors.As(err, &fne) || models.IsGenericNotFoundError(err) {
			return governance.FeatureAccess{
				HasAccess: false,
				Reason: &governance.AccessReason{
					Code:    governance.ReasonFeatureNotFound,
					Message: fmt.Sprintf("feature %q not found", featureKey),
				},
			}, nil
		}
		return governance.FeatureAccess{}, fmt.Errorf("get feature %q: %w", featureKey, err)
	}

	return featureUnavailable(featureKey), nil
}

func featureUnavailable(featureKey string) governance.FeatureAccess {
	return governance.FeatureAccess{
		HasAccess: false,
		Reason: &governance.AccessReason{
			Code:    governance.ReasonFeatureUnavailable,
			Message: fmt.Sprintf("feature %q is not available for this customer", featureKey),
		},
	}
}
