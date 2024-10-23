// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"database/sql/driver"
	"fmt"
	"math"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/balancesnapshot"
	"github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	dbgrant "github.com/openmeterio/openmeter/openmeter/ent/db/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionentitlement"
	"github.com/openmeterio/openmeter/openmeter/ent/db/usagereset"
)

// EntitlementQuery is the builder for querying Entitlement entities.
type EntitlementQuery struct {
	config
	ctx                 *QueryContext
	order               []entitlement.OrderOption
	inters              []Interceptor
	predicates          []predicate.Entitlement
	withUsageReset      *UsageResetQuery
	withGrant           *GrantQuery
	withBalanceSnapshot *BalanceSnapshotQuery
	withSubscription    *SubscriptionEntitlementQuery
	withFeature         *FeatureQuery
	modifiers           []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the EntitlementQuery builder.
func (eq *EntitlementQuery) Where(ps ...predicate.Entitlement) *EntitlementQuery {
	eq.predicates = append(eq.predicates, ps...)
	return eq
}

// Limit the number of records to be returned by this query.
func (eq *EntitlementQuery) Limit(limit int) *EntitlementQuery {
	eq.ctx.Limit = &limit
	return eq
}

// Offset to start from.
func (eq *EntitlementQuery) Offset(offset int) *EntitlementQuery {
	eq.ctx.Offset = &offset
	return eq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (eq *EntitlementQuery) Unique(unique bool) *EntitlementQuery {
	eq.ctx.Unique = &unique
	return eq
}

// Order specifies how the records should be ordered.
func (eq *EntitlementQuery) Order(o ...entitlement.OrderOption) *EntitlementQuery {
	eq.order = append(eq.order, o...)
	return eq
}

// QueryUsageReset chains the current query on the "usage_reset" edge.
func (eq *EntitlementQuery) QueryUsageReset() *UsageResetQuery {
	query := (&UsageResetClient{config: eq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := eq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := eq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(entitlement.Table, entitlement.FieldID, selector),
			sqlgraph.To(usagereset.Table, usagereset.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, entitlement.UsageResetTable, entitlement.UsageResetColumn),
		)
		fromU = sqlgraph.SetNeighbors(eq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryGrant chains the current query on the "grant" edge.
func (eq *EntitlementQuery) QueryGrant() *GrantQuery {
	query := (&GrantClient{config: eq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := eq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := eq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(entitlement.Table, entitlement.FieldID, selector),
			sqlgraph.To(dbgrant.Table, dbgrant.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, entitlement.GrantTable, entitlement.GrantColumn),
		)
		fromU = sqlgraph.SetNeighbors(eq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryBalanceSnapshot chains the current query on the "balance_snapshot" edge.
func (eq *EntitlementQuery) QueryBalanceSnapshot() *BalanceSnapshotQuery {
	query := (&BalanceSnapshotClient{config: eq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := eq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := eq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(entitlement.Table, entitlement.FieldID, selector),
			sqlgraph.To(balancesnapshot.Table, balancesnapshot.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, entitlement.BalanceSnapshotTable, entitlement.BalanceSnapshotColumn),
		)
		fromU = sqlgraph.SetNeighbors(eq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QuerySubscription chains the current query on the "subscription" edge.
func (eq *EntitlementQuery) QuerySubscription() *SubscriptionEntitlementQuery {
	query := (&SubscriptionEntitlementClient{config: eq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := eq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := eq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(entitlement.Table, entitlement.FieldID, selector),
			sqlgraph.To(subscriptionentitlement.Table, subscriptionentitlement.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, false, entitlement.SubscriptionTable, entitlement.SubscriptionColumn),
		)
		fromU = sqlgraph.SetNeighbors(eq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryFeature chains the current query on the "feature" edge.
func (eq *EntitlementQuery) QueryFeature() *FeatureQuery {
	query := (&FeatureClient{config: eq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := eq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := eq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(entitlement.Table, entitlement.FieldID, selector),
			sqlgraph.To(feature.Table, feature.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, entitlement.FeatureTable, entitlement.FeatureColumn),
		)
		fromU = sqlgraph.SetNeighbors(eq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first Entitlement entity from the query.
// Returns a *NotFoundError when no Entitlement was found.
func (eq *EntitlementQuery) First(ctx context.Context) (*Entitlement, error) {
	nodes, err := eq.Limit(1).All(setContextOp(ctx, eq.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{entitlement.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (eq *EntitlementQuery) FirstX(ctx context.Context) *Entitlement {
	node, err := eq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first Entitlement ID from the query.
// Returns a *NotFoundError when no Entitlement ID was found.
func (eq *EntitlementQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = eq.Limit(1).IDs(setContextOp(ctx, eq.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{entitlement.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (eq *EntitlementQuery) FirstIDX(ctx context.Context) string {
	id, err := eq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single Entitlement entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one Entitlement entity is found.
// Returns a *NotFoundError when no Entitlement entities are found.
func (eq *EntitlementQuery) Only(ctx context.Context) (*Entitlement, error) {
	nodes, err := eq.Limit(2).All(setContextOp(ctx, eq.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{entitlement.Label}
	default:
		return nil, &NotSingularError{entitlement.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (eq *EntitlementQuery) OnlyX(ctx context.Context) *Entitlement {
	node, err := eq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only Entitlement ID in the query.
// Returns a *NotSingularError when more than one Entitlement ID is found.
// Returns a *NotFoundError when no entities are found.
func (eq *EntitlementQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = eq.Limit(2).IDs(setContextOp(ctx, eq.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{entitlement.Label}
	default:
		err = &NotSingularError{entitlement.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (eq *EntitlementQuery) OnlyIDX(ctx context.Context) string {
	id, err := eq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of Entitlements.
func (eq *EntitlementQuery) All(ctx context.Context) ([]*Entitlement, error) {
	ctx = setContextOp(ctx, eq.ctx, ent.OpQueryAll)
	if err := eq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*Entitlement, *EntitlementQuery]()
	return withInterceptors[[]*Entitlement](ctx, eq, qr, eq.inters)
}

// AllX is like All, but panics if an error occurs.
func (eq *EntitlementQuery) AllX(ctx context.Context) []*Entitlement {
	nodes, err := eq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of Entitlement IDs.
func (eq *EntitlementQuery) IDs(ctx context.Context) (ids []string, err error) {
	if eq.ctx.Unique == nil && eq.path != nil {
		eq.Unique(true)
	}
	ctx = setContextOp(ctx, eq.ctx, ent.OpQueryIDs)
	if err = eq.Select(entitlement.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (eq *EntitlementQuery) IDsX(ctx context.Context) []string {
	ids, err := eq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (eq *EntitlementQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, eq.ctx, ent.OpQueryCount)
	if err := eq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, eq, querierCount[*EntitlementQuery](), eq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (eq *EntitlementQuery) CountX(ctx context.Context) int {
	count, err := eq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (eq *EntitlementQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, eq.ctx, ent.OpQueryExist)
	switch _, err := eq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (eq *EntitlementQuery) ExistX(ctx context.Context) bool {
	exist, err := eq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the EntitlementQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (eq *EntitlementQuery) Clone() *EntitlementQuery {
	if eq == nil {
		return nil
	}
	return &EntitlementQuery{
		config:              eq.config,
		ctx:                 eq.ctx.Clone(),
		order:               append([]entitlement.OrderOption{}, eq.order...),
		inters:              append([]Interceptor{}, eq.inters...),
		predicates:          append([]predicate.Entitlement{}, eq.predicates...),
		withUsageReset:      eq.withUsageReset.Clone(),
		withGrant:           eq.withGrant.Clone(),
		withBalanceSnapshot: eq.withBalanceSnapshot.Clone(),
		withSubscription:    eq.withSubscription.Clone(),
		withFeature:         eq.withFeature.Clone(),
		// clone intermediate query.
		sql:  eq.sql.Clone(),
		path: eq.path,
	}
}

// WithUsageReset tells the query-builder to eager-load the nodes that are connected to
// the "usage_reset" edge. The optional arguments are used to configure the query builder of the edge.
func (eq *EntitlementQuery) WithUsageReset(opts ...func(*UsageResetQuery)) *EntitlementQuery {
	query := (&UsageResetClient{config: eq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	eq.withUsageReset = query
	return eq
}

// WithGrant tells the query-builder to eager-load the nodes that are connected to
// the "grant" edge. The optional arguments are used to configure the query builder of the edge.
func (eq *EntitlementQuery) WithGrant(opts ...func(*GrantQuery)) *EntitlementQuery {
	query := (&GrantClient{config: eq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	eq.withGrant = query
	return eq
}

// WithBalanceSnapshot tells the query-builder to eager-load the nodes that are connected to
// the "balance_snapshot" edge. The optional arguments are used to configure the query builder of the edge.
func (eq *EntitlementQuery) WithBalanceSnapshot(opts ...func(*BalanceSnapshotQuery)) *EntitlementQuery {
	query := (&BalanceSnapshotClient{config: eq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	eq.withBalanceSnapshot = query
	return eq
}

// WithSubscription tells the query-builder to eager-load the nodes that are connected to
// the "subscription" edge. The optional arguments are used to configure the query builder of the edge.
func (eq *EntitlementQuery) WithSubscription(opts ...func(*SubscriptionEntitlementQuery)) *EntitlementQuery {
	query := (&SubscriptionEntitlementClient{config: eq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	eq.withSubscription = query
	return eq
}

// WithFeature tells the query-builder to eager-load the nodes that are connected to
// the "feature" edge. The optional arguments are used to configure the query builder of the edge.
func (eq *EntitlementQuery) WithFeature(opts ...func(*FeatureQuery)) *EntitlementQuery {
	query := (&FeatureClient{config: eq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	eq.withFeature = query
	return eq
}

// GroupBy is used to group vertices by one or more fields/columns.
// It is often used with aggregate functions, like: count, max, mean, min, sum.
//
// Example:
//
//	var v []struct {
//		Namespace string `json:"namespace,omitempty"`
//		Count int `json:"count,omitempty"`
//	}
//
//	client.Entitlement.Query().
//		GroupBy(entitlement.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (eq *EntitlementQuery) GroupBy(field string, fields ...string) *EntitlementGroupBy {
	eq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &EntitlementGroupBy{build: eq}
	grbuild.flds = &eq.ctx.Fields
	grbuild.label = entitlement.Label
	grbuild.scan = grbuild.Scan
	return grbuild
}

// Select allows the selection one or more fields/columns for the given query,
// instead of selecting all fields in the entity.
//
// Example:
//
//	var v []struct {
//		Namespace string `json:"namespace,omitempty"`
//	}
//
//	client.Entitlement.Query().
//		Select(entitlement.FieldNamespace).
//		Scan(ctx, &v)
func (eq *EntitlementQuery) Select(fields ...string) *EntitlementSelect {
	eq.ctx.Fields = append(eq.ctx.Fields, fields...)
	sbuild := &EntitlementSelect{EntitlementQuery: eq}
	sbuild.label = entitlement.Label
	sbuild.flds, sbuild.scan = &eq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a EntitlementSelect configured with the given aggregations.
func (eq *EntitlementQuery) Aggregate(fns ...AggregateFunc) *EntitlementSelect {
	return eq.Select().Aggregate(fns...)
}

func (eq *EntitlementQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range eq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, eq); err != nil {
				return err
			}
		}
	}
	for _, f := range eq.ctx.Fields {
		if !entitlement.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if eq.path != nil {
		prev, err := eq.path(ctx)
		if err != nil {
			return err
		}
		eq.sql = prev
	}
	return nil
}

func (eq *EntitlementQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*Entitlement, error) {
	var (
		nodes       = []*Entitlement{}
		_spec       = eq.querySpec()
		loadedTypes = [5]bool{
			eq.withUsageReset != nil,
			eq.withGrant != nil,
			eq.withBalanceSnapshot != nil,
			eq.withSubscription != nil,
			eq.withFeature != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*Entitlement).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &Entitlement{config: eq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(eq.modifiers) > 0 {
		_spec.Modifiers = eq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, eq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := eq.withUsageReset; query != nil {
		if err := eq.loadUsageReset(ctx, query, nodes,
			func(n *Entitlement) { n.Edges.UsageReset = []*UsageReset{} },
			func(n *Entitlement, e *UsageReset) { n.Edges.UsageReset = append(n.Edges.UsageReset, e) }); err != nil {
			return nil, err
		}
	}
	if query := eq.withGrant; query != nil {
		if err := eq.loadGrant(ctx, query, nodes,
			func(n *Entitlement) { n.Edges.Grant = []*Grant{} },
			func(n *Entitlement, e *Grant) { n.Edges.Grant = append(n.Edges.Grant, e) }); err != nil {
			return nil, err
		}
	}
	if query := eq.withBalanceSnapshot; query != nil {
		if err := eq.loadBalanceSnapshot(ctx, query, nodes,
			func(n *Entitlement) { n.Edges.BalanceSnapshot = []*BalanceSnapshot{} },
			func(n *Entitlement, e *BalanceSnapshot) { n.Edges.BalanceSnapshot = append(n.Edges.BalanceSnapshot, e) }); err != nil {
			return nil, err
		}
	}
	if query := eq.withSubscription; query != nil {
		if err := eq.loadSubscription(ctx, query, nodes, nil,
			func(n *Entitlement, e *SubscriptionEntitlement) { n.Edges.Subscription = e }); err != nil {
			return nil, err
		}
	}
	if query := eq.withFeature; query != nil {
		if err := eq.loadFeature(ctx, query, nodes, nil,
			func(n *Entitlement, e *Feature) { n.Edges.Feature = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (eq *EntitlementQuery) loadUsageReset(ctx context.Context, query *UsageResetQuery, nodes []*Entitlement, init func(*Entitlement), assign func(*Entitlement, *UsageReset)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*Entitlement)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(usagereset.FieldEntitlementID)
	}
	query.Where(predicate.UsageReset(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(entitlement.UsageResetColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.EntitlementID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "entitlement_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (eq *EntitlementQuery) loadGrant(ctx context.Context, query *GrantQuery, nodes []*Entitlement, init func(*Entitlement), assign func(*Entitlement, *Grant)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*Entitlement)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(dbgrant.FieldOwnerID)
	}
	query.Where(predicate.Grant(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(entitlement.GrantColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.OwnerID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "owner_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (eq *EntitlementQuery) loadBalanceSnapshot(ctx context.Context, query *BalanceSnapshotQuery, nodes []*Entitlement, init func(*Entitlement), assign func(*Entitlement, *BalanceSnapshot)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*Entitlement)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(balancesnapshot.FieldOwnerID)
	}
	query.Where(predicate.BalanceSnapshot(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(entitlement.BalanceSnapshotColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.OwnerID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "owner_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (eq *EntitlementQuery) loadSubscription(ctx context.Context, query *SubscriptionEntitlementQuery, nodes []*Entitlement, init func(*Entitlement), assign func(*Entitlement, *SubscriptionEntitlement)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*Entitlement)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(subscriptionentitlement.FieldEntitlementID)
	}
	query.Where(predicate.SubscriptionEntitlement(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(entitlement.SubscriptionColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.EntitlementID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "entitlement_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (eq *EntitlementQuery) loadFeature(ctx context.Context, query *FeatureQuery, nodes []*Entitlement, init func(*Entitlement), assign func(*Entitlement, *Feature)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*Entitlement)
	for i := range nodes {
		fk := nodes[i].FeatureID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(feature.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "feature_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (eq *EntitlementQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := eq.querySpec()
	if len(eq.modifiers) > 0 {
		_spec.Modifiers = eq.modifiers
	}
	_spec.Node.Columns = eq.ctx.Fields
	if len(eq.ctx.Fields) > 0 {
		_spec.Unique = eq.ctx.Unique != nil && *eq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, eq.driver, _spec)
}

func (eq *EntitlementQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(entitlement.Table, entitlement.Columns, sqlgraph.NewFieldSpec(entitlement.FieldID, field.TypeString))
	_spec.From = eq.sql
	if unique := eq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if eq.path != nil {
		_spec.Unique = true
	}
	if fields := eq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, entitlement.FieldID)
		for i := range fields {
			if fields[i] != entitlement.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if eq.withFeature != nil {
			_spec.Node.AddColumnOnce(entitlement.FieldFeatureID)
		}
	}
	if ps := eq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := eq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := eq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := eq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (eq *EntitlementQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(eq.driver.Dialect())
	t1 := builder.Table(entitlement.Table)
	columns := eq.ctx.Fields
	if len(columns) == 0 {
		columns = entitlement.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if eq.sql != nil {
		selector = eq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if eq.ctx.Unique != nil && *eq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range eq.modifiers {
		m(selector)
	}
	for _, p := range eq.predicates {
		p(selector)
	}
	for _, p := range eq.order {
		p(selector)
	}
	if offset := eq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := eq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (eq *EntitlementQuery) ForUpdate(opts ...sql.LockOption) *EntitlementQuery {
	if eq.driver.Dialect() == dialect.Postgres {
		eq.Unique(false)
	}
	eq.modifiers = append(eq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return eq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (eq *EntitlementQuery) ForShare(opts ...sql.LockOption) *EntitlementQuery {
	if eq.driver.Dialect() == dialect.Postgres {
		eq.Unique(false)
	}
	eq.modifiers = append(eq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return eq
}

// EntitlementGroupBy is the group-by builder for Entitlement entities.
type EntitlementGroupBy struct {
	selector
	build *EntitlementQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (egb *EntitlementGroupBy) Aggregate(fns ...AggregateFunc) *EntitlementGroupBy {
	egb.fns = append(egb.fns, fns...)
	return egb
}

// Scan applies the selector query and scans the result into the given value.
func (egb *EntitlementGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, egb.build.ctx, ent.OpQueryGroupBy)
	if err := egb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*EntitlementQuery, *EntitlementGroupBy](ctx, egb.build, egb, egb.build.inters, v)
}

func (egb *EntitlementGroupBy) sqlScan(ctx context.Context, root *EntitlementQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(egb.fns))
	for _, fn := range egb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*egb.flds)+len(egb.fns))
		for _, f := range *egb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*egb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := egb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// EntitlementSelect is the builder for selecting fields of Entitlement entities.
type EntitlementSelect struct {
	*EntitlementQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (es *EntitlementSelect) Aggregate(fns ...AggregateFunc) *EntitlementSelect {
	es.fns = append(es.fns, fns...)
	return es
}

// Scan applies the selector query and scans the result into the given value.
func (es *EntitlementSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, es.ctx, ent.OpQuerySelect)
	if err := es.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*EntitlementQuery, *EntitlementSelect](ctx, es.EntitlementQuery, es, es.inters, v)
}

func (es *EntitlementSelect) sqlScan(ctx context.Context, root *EntitlementQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(es.fns))
	for _, fn := range es.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*es.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := es.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
