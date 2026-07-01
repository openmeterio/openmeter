package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/governance"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
	"github.com/openmeterio/openmeter/pkg/models"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// featureFetchLimit caps the org-wide feature fetch used when no feature filter is given.
// Acceptable for prototype scale; revisit if feature counts grow large.
const featureFetchLimit = 10_000

// maxCustomerConcurrency bounds how many customers on a page resolve access in parallel.
// It is deliberately conservative because the pressure multiplies: entitlementService.GetAccess
// ALREADY fans a customer's entitlements out ~10-wide internally, so the peak number of
// concurrent per-entitlement DB/ClickHouse operations is roughly maxCustomerConcurrency × 10.
// The Postgres pool defaults to ~NumCPU connections (pgxpool with no pool_max_conns), and each
// metered value calc holds a connection (including a short snapshot lock transaction), so a
// larger value would mostly queue on connection acquisition and, under concurrent request load,
// risk acquire timeouts. Lower it if pool-acquisition latency shows up. A single-request
// benchmark will NOT reveal that pressure, since it never runs two requests at once.
const maxCustomerConcurrency = 5

// Config holds the collaborating services for the governance Service.
type Config struct {
	Customer    customer.Service
	Entitlement entitlement.Service
	Feature     feature.FeatureConnector
	Tracer      trace.Tracer
	Meter       metric.Meter
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

	if c.Meter == nil {
		errs = append(errs, errors.New("meter is required"))
	}

	return errors.Join(errs...)
}

func New(config Config) (governance.Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	metrics, err := newMetrics(config.Meter)
	if err != nil {
		return nil, err
	}

	return &service{
		customerService:    config.Customer,
		entitlementService: config.Entitlement,
		featureConnector:   config.Feature,
		tracer:             config.Tracer,
		metrics:            metrics,
	}, nil
}

type service struct {
	customerService    customer.Service
	entitlementService entitlement.Service
	featureConnector   feature.FeatureConnector
	tracer             trace.Tracer
	metrics            queryMetrics
}

// queryMetrics holds the instruments for the governance query endpoint. These are unsampled
// (unlike spans), so they back SLOs, alerting, and capacity dashboards. Per-request counts
// are recorded as histogram observations rather than attributes to keep series cardinality
// bounded; only low-cardinality enums are used as counter attributes.
type queryMetrics struct {
	// requests counts queries, broken down by pagination direction and whether the
	// all-org-features (no filter) path was taken.
	requests metric.Int64Counter
	// customersNotFound counts input keys that did not resolve to a customer. These return
	// HTTP 200 with a partial error, so they are invisible to HTTP-level metrics.
	customersNotFound metric.Int64Counter
	// featureAccess records the number of feature evaluations per query — the work unit that
	// drives latency (~customers × features).
	featureAccess metric.Int64Histogram
	// customerKeys records the number of customer keys requested per query.
	customerKeys metric.Int64Histogram
}

func newMetrics(meter metric.Meter) (queryMetrics, error) {
	requests, err := meter.Int64Counter(
		"openmeter.governance.query.requests",
		metric.WithDescription("Number of governance access queries"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return queryMetrics{}, fmt.Errorf("failed to create requests counter: %w", err)
	}

	customersNotFound, err := meter.Int64Counter(
		"openmeter.governance.query.customers_not_found",
		metric.WithDescription("Number of customer keys that did not resolve to a customer"),
		metric.WithUnit("{customer}"),
	)
	if err != nil {
		return queryMetrics{}, fmt.Errorf("failed to create customers_not_found counter: %w", err)
	}

	featureAccess, err := meter.Int64Histogram(
		"openmeter.governance.query.feature_access",
		metric.WithDescription("Number of feature evaluations per governance query"),
		metric.WithUnit("{evaluation}"),
	)
	if err != nil {
		return queryMetrics{}, fmt.Errorf("failed to create feature_access histogram: %w", err)
	}

	customerKeys, err := meter.Int64Histogram(
		"openmeter.governance.query.customer_keys",
		metric.WithDescription("Number of customer keys requested per governance query"),
		metric.WithUnit("{key}"),
	)
	if err != nil {
		return queryMetrics{}, fmt.Errorf("failed to create customer_keys histogram: %w", err)
	}

	return queryMetrics{
		requests:          requests,
		customersNotFound: customersNotFound,
		featureAccess:     featureAccess,
		customerKeys:      customerKeys,
	}, nil
}

var _ governance.Service = (*service)(nil)

// QueryAccess evaluates feature access for a caller-supplied set of customer keys.
//
// Pagination is in-memory, by design. The reason is the BOUND: the input is OAS-capped at
// 100 keys (@maxItems), so the resolvable set is ≤100 — sorting and slicing it in memory is
// trivially cheap, and pushing a keyset cursor into the DB would buy nothing. (It is NOT
// because there's no collection to paginate: resolution already runs as a single bulk
// customer.GetCustomersByUsageAttribution query, and customer.ListCustomers is a DB-side,
// orderable, paginated query that could additionally order+limit the key set via a
// `Key $in [...] $or usageAttributionSubjectKey $in [...]` filter. That DB-paginated path
// only becomes worthwhile if the 100-key cap is ever lifted — see below.)
//
// Phase order and what each page actually costs (e.g. page size 10 over 100 resolved):
//  1. resolveCustomers resolves ALL keys (one bulk customer.GetCustomersByUsageAttribution
//     query) — runs in full regardless of page size, because the sort key is (CreatedAt, ID),
//     customer fields: page order can't be established without first resolving every key to a
//     customer. Dedup, the per-input `matched` mapping, and per-key not-found reporting are
//     done in memory, since the input mixes customer keys and subject keys.
//  2. sort the full set in memory.
//  3. paginate slices to PageSize (10).
//  4. resolveAccess runs only over the page (10): the expensive per-customer GetAccess
//     fan-out + one listOrgFeatures. So the dominant cost IS page-limited; only the cheaper
//     full-set resolution is paid in full (and repeated per page across a paging client).
//
// Pagination would only need to move into the adapter (as a keyset query) if the contract
// gained an unbounded mode — "all customers in a namespace", or dropping the 100-key cap.
// Not the case today.
func (s *service) QueryAccess(ctx context.Context, input governance.QueryAccessInput) (governance.QueryResult, error) {
	fn := func(ctx context.Context) (governance.QueryResult, error) {
		if err := input.Validate(); err != nil {
			return governance.QueryResult{}, err
		}

		span := trace.SpanFromContext(ctx)

		span.SetAttributes(
			attribute.String("namespace", input.Namespace),
			attribute.Int("customer_key_count", len(input.CustomerKeys)),
			attribute.Int("feature_key_count", len(input.FeatureKeys)),
			attribute.Int("page.size", input.PageSize),
			attribute.String("direction", paginationDirection(input)),
		)

		customers, err := s.resolveCustomers(ctx, input)
		if err != nil {
			return governance.QueryResult{}, err
		}

		// Sort by (CreatedAt, ID) for stable cursor pagination.
		sortedCustomers := lo.Values(customers.resolvedCustomers)

		sort.Slice(sortedCustomers, func(i, j int) bool {
			ti := sortedCustomers[i].customer.CreatedAt
			tj := sortedCustomers[j].customer.CreatedAt

			if !ti.Equal(tj) {
				return ti.Before(tj)
			}

			return sortedCustomers[i].customer.ID < sortedCustomers[j].customer.ID
		})

		paginatedCustomers := paginate(sortedCustomers, input)

		results, err := s.resolveAccess(ctx, input, paginatedCustomers.customers)
		if err != nil {
			return governance.QueryResult{}, err
		}

		out := governance.QueryResult{
			Customers: results,
			Errors:    customers.queryErrors,
			HasPrev:   paginatedCustomers.hasPrev,
			HasNext:   paginatedCustomers.hasNext,
		}

		if len(paginatedCustomers.customers) > 0 {
			out.First = lo.ToPtr(cursorFor(paginatedCustomers.customers[0]))
			out.Last = lo.ToPtr(cursorFor(paginatedCustomers.customers[len(paginatedCustomers.customers)-1]))
		}

		s.recordQueryMetrics(ctx, input, out, len(customers.queryErrors))

		return out, nil
	}

	return tracex.Start[governance.QueryResult](ctx, s.tracer, "governance.QueryAccess").Wrap(fn)
}

// recordQueryMetrics emits the unsampled query metrics. namespace is a per-tenant label
// (consistent with ingest/sink/balanceworker); per-request counts are histogram values, and
// only low-cardinality enums (direction, all_org_features) are counter attributes.
func (s *service) recordQueryMetrics(ctx context.Context, input governance.QueryAccessInput, out governance.QueryResult, notFound int) {
	namespaceAttr := attribute.String("namespace", input.Namespace)

	s.metrics.requests.Add(ctx, 1, metric.WithAttributes(
		namespaceAttr,
		attribute.String("direction", paginationDirection(input)),
		attribute.Bool("all_org_features", len(input.FeatureKeys) == 0),
	))

	if notFound > 0 {
		s.metrics.customersNotFound.Add(ctx, int64(notFound), metric.WithAttributes(namespaceAttr))
	}

	featureAccessTotal := lo.SumBy(out.Customers, func(c governance.CustomerAccess) int {
		return len(c.Features)
	})

	s.metrics.featureAccess.Record(ctx, int64(featureAccessTotal), metric.WithAttributes(namespaceAttr))
	s.metrics.customerKeys.Record(ctx, int64(len(input.CustomerKeys)), metric.WithAttributes(namespaceAttr))
}

// resolvedCustomer groups the matched input keys for a single customer.
type resolvedCustomer struct {
	customer customer.Customer
	matched  []string
}

type resolveCustomersResult struct {
	resolvedCustomers map[string]*resolvedCustomer
	queryErrors       []governance.QueryError
}

// resolveCustomers resolves the input keys to customers in a single bulk lookup, deduplicating
// by customer ID. Keys that resolve to no customer are collected as customer-not-found query
// errors rather than failing the whole request.
func (s *service) resolveCustomers(ctx context.Context, input governance.QueryAccessInput) (resolveCustomersResult, error) {
	fn := func(ctx context.Context) (resolveCustomersResult, error) {
		span := trace.SpanFromContext(ctx)

		span.SetAttributes(
			attribute.Int("requested", len(input.CustomerKeys)),
		)

		customers, err := s.customerService.GetCustomersByUsageAttribution(ctx, customer.GetCustomersByUsageAttributionInput{
			Namespace: input.Namespace,
			Keys:      input.CustomerKeys,
		})
		if err != nil {
			return resolveCustomersResult{}, fmt.Errorf("failed to resolve customer keys: %w", err)
		}

		// Map each lookup key to the customer it resolved to. A key matches a customer either by
		// the customer's own key or by one of its subject keys. First match wins on the rare
		// collision, mirroring the single-key First() lookup.
		keyToCustomer := make(map[string]*customer.Customer, len(customers))

		for i := range customers {
			cus := &customers[i]

			if cus.Key != nil {
				if _, ok := keyToCustomer[*cus.Key]; !ok {
					keyToCustomer[*cus.Key] = cus
				}
			}

			if cus.UsageAttribution != nil {
				for _, sk := range cus.UsageAttribution.SubjectKeys {
					if _, ok := keyToCustomer[sk]; !ok {
						keyToCustomer[sk] = cus
					}
				}
			}
		}

		customerMap := make(map[string]*resolvedCustomer)
		var queryErrors []governance.QueryError

		// Iterate the input keys in order so matched keys and not-found errors keep input ordering.
		for _, key := range input.CustomerKeys {
			cus, ok := keyToCustomer[key]

			if !ok {
				queryErrors = append(queryErrors, governance.QueryError{
					CustomerKey: key,
					Code:        governance.QueryErrorCustomerNotFound,
					Message:     "customer not found",
				})
				continue
			}

			if rc, ok := customerMap[cus.ID]; ok {
				rc.matched = append(rc.matched, key)
			} else {
				customerMap[cus.ID] = &resolvedCustomer{
					customer: *cus,
					matched:  []string{key},
				}
			}
		}

		span.SetAttributes(
			attribute.Int("resolved", len(customerMap)),
			attribute.Int("not_found", len(queryErrors)),
		)

		return resolveCustomersResult{
			resolvedCustomers: customerMap,
			queryErrors:       queryErrors,
		}, nil
	}

	return tracex.Start[resolveCustomersResult](ctx, s.tracer, "governance.resolveCustomers").Wrap(fn)
}

// resolveAccess resolves entitlement access for each customer on the current page and maps it
// to feature access. UpdatedAt is stamped once for the whole page.
//
// When no feature filter is given, the org-wide feature list is namespace-scoped, so it is
// fetched once for the whole page rather than per customer.
func (s *service) resolveAccess(ctx context.Context, input governance.QueryAccessInput, customers []*resolvedCustomer) ([]governance.CustomerAccess, error) {
	fn := func(ctx context.Context) ([]governance.CustomerAccess, error) {
		allOrgFeatures := len(input.FeatureKeys) == 0

		span := trace.SpanFromContext(ctx)

		span.SetAttributes(
			attribute.Int("customer_count", len(customers)),
			attribute.Int("feature_filter_count", len(input.FeatureKeys)),
			attribute.Bool("all_org_features", allOrgFeatures),
		)

		// On the all-org path, fetch the namespace-wide feature list once for the whole page.
		var orgFeatures []feature.Feature

		if allOrgFeatures && len(customers) > 0 {
			var err error

			orgFeatures, err = s.listOrgFeatures(ctx, input.Namespace)
			if err != nil {
				return nil, err
			}

			span.SetAttributes(
				attribute.Int("org_feature_count", len(orgFeatures)),
			)
		}

		now := clock.Now()

		// Resolve customers concurrently but bounded (see maxCustomerConcurrency). Results and
		// per-customer span counters are written into pre-sized slices by index so the page keeps
		// its (CreatedAt, ID) sort order and there is no shared-state mutation across goroutines.
		results := make([]governance.CustomerAccess, len(customers))
		absentLookupsPer := make([]int, len(customers))
		featureAccessPer := make([]int, len(customers))

		g, ctx := errgroup.WithContext(ctx)
		g.SetLimit(maxCustomerConcurrency)

		for i, rc := range customers {
			g.Go(func() error {
				access, err := s.entitlementService.GetAccess(ctx, input.Namespace, rc.customer.ID)
				if err != nil {
					return fmt.Errorf("failed to get access for customer %s: %w", rc.customer.ID, err)
				}

				featureAccessResult, err := s.buildFeatureAccess(ctx, input.Namespace, input.FeatureKeys, orgFeatures, access)
				if err != nil {
					return fmt.Errorf("failed to build feature access for customer %s: %w", rc.customer.ID, err)
				}

				absentLookupsPer[i] = featureAccessResult.absentLookups
				featureAccessPer[i] = len(featureAccessResult.featureAccess)

				results[i] = governance.CustomerAccess{
					Customer:  rc.customer,
					Matched:   rc.matched,
					Features:  featureAccessResult.featureAccess,
					UpdatedAt: now,
				}

				return nil
			})
		}

		if err := g.Wait(); err != nil {
			return nil, err
		}

		span.SetAttributes(
			attribute.Int("absent_feature_lookups", lo.Sum(absentLookupsPer)),
			attribute.Int("feature_access_total", lo.Sum(featureAccessPer)),
		)

		return results, nil
	}

	return tracex.Start[[]governance.CustomerAccess](ctx, s.tracer, "governance.resolveAccess").Wrap(fn)
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
	return pagination.NewCursor(rc.customer.CreatedAt.Truncate(time.Second), rc.customer.ID)
}

type paginationResult struct {
	customers []*resolvedCustomer
	hasPrev   bool
	hasNext   bool
}

// paginate applies cursor pagination over the sorted customers and reports whether adjacent
// pages exist. Exactly one of input.After / input.Before may be set (enforced by Validate).
func paginate(customers []*resolvedCustomer, input governance.QueryAccessInput) paginationResult {
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
		hasPrev := len(candidates) > input.PageSize

		if hasPrev {
			candidates = candidates[len(candidates)-input.PageSize:]
		}

		// next is always set in backward mode: the before-cursor item itself is forward.
		return paginationResult{
			customers: candidates,
			hasPrev:   hasPrev,
			hasNext:   true,
		}
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

	hasPrev := start > 0
	page := customers[start:]
	hasNext := len(page) > input.PageSize

	if hasNext {
		page = page[:input.PageSize]
	}

	return paginationResult{
		customers: page,
		hasPrev:   hasPrev,
		hasNext:   hasNext,
	}
}

type buildFeatureAccessResult struct {
	featureAccess map[string]governance.FeatureAccess
	absentLookups int
}

// buildFeatureAccess returns the feature access map for a single customer, along with the
// number of absent-feature lookups it performed (per-feature GetFeature calls), for span
// attribution.
//
// If featureKeys is non-empty, only those keys are evaluated. If featureKeys is empty, the
// pre-fetched orgFeatures slice (namespace-wide, resolved once by the caller) is used; features
// the customer has no entitlement for are marked feature-unavailable.
func (s *service) buildFeatureAccess(ctx context.Context, ns string, featureKeys []string, orgFeatures []feature.Feature, access entitlement.Access) (buildFeatureAccessResult, error) {
	result := make(map[string]governance.FeatureAccess)

	if len(featureKeys) == 0 {
		for _, f := range orgFeatures {
			if ev, ok := access.Entitlements[f.Key]; ok {
				result[f.Key] = mapEntitlementToAccess(ev.Value)
			} else {
				result[f.Key] = governance.FeatureAccess{
					HasAccess: false,
					Reason:    governance.AccessReasonFeatureUnavailable,
				}
			}
		}

		return buildFeatureAccessResult{
			featureAccess: result,
			absentLookups: 0,
		}, nil
	}

	absentLookups := 0

	for _, key := range featureKeys {
		ev, ok := access.Entitlements[key]

		if !ok {
			absentLookups++

			fa, err := s.resolveAbsentFeature(ctx, ns, key)
			if err != nil {
				return buildFeatureAccessResult{
					featureAccess: nil,
					absentLookups: absentLookups,
				}, err
			}

			result[key] = fa

			continue
		}

		result[key] = mapEntitlementToAccess(ev.Value)
	}

	return buildFeatureAccessResult{
		featureAccess: result,
		absentLookups: absentLookups,
	}, nil
}

// listOrgFeatures fetches all non-archived features in the namespace in one shot.
func (s *service) listOrgFeatures(ctx context.Context, ns string) ([]feature.Feature, error) {
	fn := func(ctx context.Context) ([]feature.Feature, error) {
		span := trace.SpanFromContext(ctx)

		span.SetAttributes(
			attribute.Int("limit", featureFetchLimit),
		)

		res, err := s.featureConnector.ListFeatures(ctx, feature.ListFeaturesParams{
			Namespace:       ns,
			IncludeArchived: false,
			Limit:           featureFetchLimit,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list org features: %w", err)
		}

		span.SetAttributes(
			attribute.Int("feature_count", len(res.Items)),
		)

		return res.Items, nil
	}

	return tracex.Start[[]feature.Feature](ctx, s.tracer, "governance.listOrgFeatures").Wrap(fn)
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
				Reason:    governance.AccessReasonFeatureNotFound,
			}, nil
		}

		return governance.FeatureAccess{}, fmt.Errorf("failed to get feature %q: %w", featureKey, err)
	}

	return governance.FeatureAccess{
		HasAccess: false,
		Reason:    governance.AccessReasonFeatureUnavailable,
	}, nil
}
