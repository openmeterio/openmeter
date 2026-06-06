package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

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
	Customer    customer.Service
	Entitlement entitlement.Service
	Feature     feature.FeatureConnector
	Tracer      trace.Tracer
}

func (c Config) Validate() error {
	var errs []error

	if c.Customer == nil {
		errs = append(errs, errors.New("customer service is required"))
	}

	if c.Entitlement == nil {
		errs = append(errs, errors.New("entitlement service is required"))
	}

	if c.Feature == nil {
		errs = append(errs, errors.New("feature connector is required"))
	}

	if c.Tracer == nil {
		errs = append(errs, errors.New("tracer is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (governance.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		customerService:    config.Customer,
		entitlementService: config.Entitlement,
		featureConnector:   config.Feature,
		tracer:             config.Tracer,
	}, nil
}

type service struct {
	customerService    customer.Service
	entitlementService entitlement.Service
	featureConnector   feature.FeatureConnector
	tracer             trace.Tracer
}

var _ governance.Service = (*service)(nil)

// resolvedCustomer groups the matched input keys for a single customer.
type resolvedCustomer struct {
	Customer customer.Customer
	Matched  []string
}

func (s *service) QueryAccess(ctx context.Context, input governance.QueryAccessInput) (governance.QueryResult, error) {
	ctx, span := s.tracer.Start(ctx, "governance.QueryAccess", trace.WithAttributes(
		attribute.String("namespace", input.Namespace),
		attribute.Int("customer_key_count", len(input.CustomerKeys)),
		attribute.Int("feature_key_count", len(input.FeatureKeys)),
		attribute.Int("page.size", input.PageSize),
		attribute.String("direction", paginationDirection(input)),
	))
	defer span.End()

	if err := input.Validate(); err != nil {
		return governance.QueryResult{}, recordSpanError(span, err)
	}

	customerMap, queryErrors, err := s.resolveCustomers(ctx, input)
	if err != nil {
		return governance.QueryResult{}, recordSpanError(span, err)
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

	results, err := s.resolveAccess(ctx, input, customers)
	if err != nil {
		return governance.QueryResult{}, recordSpanError(span, err)
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

// resolveCustomers resolves each input key to a customer, deduplicating by customer ID.
// Keys that resolve to no customer are collected as customer-not-found query errors rather
// than failing the whole request.
func (s *service) resolveCustomers(ctx context.Context, input governance.QueryAccessInput) (map[string]*resolvedCustomer, []governance.QueryError, error) {
	ctx, span := s.tracer.Start(ctx, "governance.resolveCustomers", trace.WithAttributes(
		attribute.Int("requested", len(input.CustomerKeys)),
	))
	defer span.End()

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
			return nil, nil, recordSpanError(span, fmt.Errorf("resolve customer key %q: %w", key, err))
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

	span.SetAttributes(
		attribute.Int("resolved", len(customerMap)),
		attribute.Int("not_found", len(queryErrors)),
	)

	return customerMap, queryErrors, nil
}

// resolveAccess resolves entitlement access for each customer on the current page and maps it
// to feature access. UpdatedAt is stamped once for the whole page.
//
// When no feature filter is given, the org-wide feature list is namespace-scoped, so it is
// fetched once for the whole page rather than per customer.
func (s *service) resolveAccess(ctx context.Context, input governance.QueryAccessInput, customers []*resolvedCustomer) ([]governance.CustomerAccess, error) {
	allOrgFeatures := len(input.FeatureKeys) == 0

	ctx, span := s.tracer.Start(ctx, "governance.resolveAccess", trace.WithAttributes(
		attribute.Int("customer_count", len(customers)),
		attribute.Int("feature_filter_count", len(input.FeatureKeys)),
		attribute.Bool("all_org_features", allOrgFeatures),
	))
	defer span.End()

	// On the all-org path, fetch the namespace-wide feature list once for the whole page.
	var orgFeatures []feature.Feature

	if allOrgFeatures && len(customers) > 0 {
		var err error
		orgFeatures, err = s.listOrgFeatures(ctx, input.Namespace)
		if err != nil {
			return nil, recordSpanError(span, err)
		}
		span.SetAttributes(attribute.Int("org_feature_count", len(orgFeatures)))
	}

	now := clock.Now()
	results := make([]governance.CustomerAccess, 0, len(customers))
	absentFeatureLookups := 0
	featureAccessTotal := 0

	for _, rc := range customers {
		access, err := s.entitlementService.GetAccess(ctx, input.Namespace, rc.Customer.ID)
		if err != nil {
			return nil, recordSpanError(span, fmt.Errorf("get access for customer %s: %w", rc.Customer.ID, err))
		}

		featureAccess, absentLookups, err := s.buildFeatureAccess(ctx, input.Namespace, input.FeatureKeys, orgFeatures, access)
		if err != nil {
			return nil, recordSpanError(span, fmt.Errorf("build feature access for customer %s: %w", rc.Customer.ID, err))
		}

		absentFeatureLookups += absentLookups
		featureAccessTotal += len(featureAccess)

		results = append(results, governance.CustomerAccess{
			Customer:  rc.Customer,
			Matched:   rc.Matched,
			Features:  featureAccess,
			UpdatedAt: now,
		})
	}

	span.SetAttributes(
		attribute.Int("absent_feature_lookups", absentFeatureLookups),
		attribute.Int("feature_access_total", featureAccessTotal),
	)

	return results, nil
}

// recordSpanError marks the span failed and returns the error unchanged for convenient
// `return recordSpanError(span, err)` call sites.
func recordSpanError(span trace.Span, err error) error {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	return err
}

// paginationDirection reports the pagination mode for span attribution.
func paginationDirection(input governance.QueryAccessInput) string {
	switch {
	case input.Before != nil:
		return "before"
	case input.After != nil:
		return "after"
	default:
		return "first"
	}
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

// buildFeatureAccess returns the feature access map for a single customer, along with the
// number of absent-feature lookups it performed (per-feature GetFeature calls), for span
// attribution.
//
// If featureKeys is non-empty, only those keys are evaluated. If featureKeys is empty, the
// pre-fetched orgFeatures slice (namespace-wide, resolved once by the caller) is used; features
// the customer has no entitlement for are marked feature-unavailable.
func (s *service) buildFeatureAccess(ctx context.Context, ns string, featureKeys []string, orgFeatures []feature.Feature, access entitlement.Access) (map[string]governance.FeatureAccess, int, error) {
	result := make(map[string]governance.FeatureAccess)

	if len(featureKeys) == 0 {
		for _, f := range orgFeatures {
			if ev, ok := access.Entitlements[f.Key]; ok {
				result[f.Key] = mapEntitlementToAccess(ev.Value)
			} else {
				result[f.Key] = featureUnavailable(f.Key)
			}
		}

		return result, 0, nil
	}

	absentLookups := 0

	for _, key := range featureKeys {
		ev, ok := access.Entitlements[key]

		if !ok {
			absentLookups++
			fa, err := s.resolveAbsentFeature(ctx, ns, key)
			if err != nil {
				return nil, absentLookups, err
			}

			result[key] = fa

			continue
		}

		result[key] = mapEntitlementToAccess(ev.Value)
	}

	return result, absentLookups, nil
}

// listOrgFeatures fetches all non-archived features in the namespace in one shot.
func (s *service) listOrgFeatures(ctx context.Context, ns string) ([]feature.Feature, error) {
	ctx, span := s.tracer.Start(ctx, "governance.listOrgFeatures", trace.WithAttributes(
		attribute.Int("limit", featureFetchLimit),
	))
	defer span.End()

	res, err := s.featureConnector.ListFeatures(ctx, feature.ListFeaturesParams{
		Namespace:       ns,
		IncludeArchived: false,
		Limit:           featureFetchLimit,
	})
	if err != nil {
		return nil, recordSpanError(span, fmt.Errorf("list org features: %w", err))
	}

	span.SetAttributes(attribute.Int("feature_count", len(res.Items)))

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
