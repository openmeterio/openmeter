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
	"github.com/openmeterio/openmeter/openmeter/ent/db/addon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddonquantity"
)

// SubscriptionAddonQuery is the builder for querying SubscriptionAddon entities.
type SubscriptionAddonQuery struct {
	config
	ctx              *QueryContext
	order            []subscriptionaddon.OrderOption
	inters           []Interceptor
	predicates       []predicate.SubscriptionAddon
	withSubscription *SubscriptionQuery
	withQuantities   *SubscriptionAddonQuantityQuery
	withAddon        *AddonQuery
	modifiers        []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the SubscriptionAddonQuery builder.
func (saq *SubscriptionAddonQuery) Where(ps ...predicate.SubscriptionAddon) *SubscriptionAddonQuery {
	saq.predicates = append(saq.predicates, ps...)
	return saq
}

// Limit the number of records to be returned by this query.
func (saq *SubscriptionAddonQuery) Limit(limit int) *SubscriptionAddonQuery {
	saq.ctx.Limit = &limit
	return saq
}

// Offset to start from.
func (saq *SubscriptionAddonQuery) Offset(offset int) *SubscriptionAddonQuery {
	saq.ctx.Offset = &offset
	return saq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (saq *SubscriptionAddonQuery) Unique(unique bool) *SubscriptionAddonQuery {
	saq.ctx.Unique = &unique
	return saq
}

// Order specifies how the records should be ordered.
func (saq *SubscriptionAddonQuery) Order(o ...subscriptionaddon.OrderOption) *SubscriptionAddonQuery {
	saq.order = append(saq.order, o...)
	return saq
}

// QuerySubscription chains the current query on the "subscription" edge.
func (saq *SubscriptionAddonQuery) QuerySubscription() *SubscriptionQuery {
	query := (&SubscriptionClient{config: saq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := saq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := saq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(subscriptionaddon.Table, subscriptionaddon.FieldID, selector),
			sqlgraph.To(subscription.Table, subscription.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, subscriptionaddon.SubscriptionTable, subscriptionaddon.SubscriptionColumn),
		)
		fromU = sqlgraph.SetNeighbors(saq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryQuantities chains the current query on the "quantities" edge.
func (saq *SubscriptionAddonQuery) QueryQuantities() *SubscriptionAddonQuantityQuery {
	query := (&SubscriptionAddonQuantityClient{config: saq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := saq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := saq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(subscriptionaddon.Table, subscriptionaddon.FieldID, selector),
			sqlgraph.To(subscriptionaddonquantity.Table, subscriptionaddonquantity.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, subscriptionaddon.QuantitiesTable, subscriptionaddon.QuantitiesColumn),
		)
		fromU = sqlgraph.SetNeighbors(saq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryAddon chains the current query on the "addon" edge.
func (saq *SubscriptionAddonQuery) QueryAddon() *AddonQuery {
	query := (&AddonClient{config: saq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := saq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := saq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(subscriptionaddon.Table, subscriptionaddon.FieldID, selector),
			sqlgraph.To(addon.Table, addon.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, subscriptionaddon.AddonTable, subscriptionaddon.AddonColumn),
		)
		fromU = sqlgraph.SetNeighbors(saq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first SubscriptionAddon entity from the query.
// Returns a *NotFoundError when no SubscriptionAddon was found.
func (saq *SubscriptionAddonQuery) First(ctx context.Context) (*SubscriptionAddon, error) {
	nodes, err := saq.Limit(1).All(setContextOp(ctx, saq.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{subscriptionaddon.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (saq *SubscriptionAddonQuery) FirstX(ctx context.Context) *SubscriptionAddon {
	node, err := saq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first SubscriptionAddon ID from the query.
// Returns a *NotFoundError when no SubscriptionAddon ID was found.
func (saq *SubscriptionAddonQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = saq.Limit(1).IDs(setContextOp(ctx, saq.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{subscriptionaddon.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (saq *SubscriptionAddonQuery) FirstIDX(ctx context.Context) string {
	id, err := saq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single SubscriptionAddon entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one SubscriptionAddon entity is found.
// Returns a *NotFoundError when no SubscriptionAddon entities are found.
func (saq *SubscriptionAddonQuery) Only(ctx context.Context) (*SubscriptionAddon, error) {
	nodes, err := saq.Limit(2).All(setContextOp(ctx, saq.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{subscriptionaddon.Label}
	default:
		return nil, &NotSingularError{subscriptionaddon.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (saq *SubscriptionAddonQuery) OnlyX(ctx context.Context) *SubscriptionAddon {
	node, err := saq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only SubscriptionAddon ID in the query.
// Returns a *NotSingularError when more than one SubscriptionAddon ID is found.
// Returns a *NotFoundError when no entities are found.
func (saq *SubscriptionAddonQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = saq.Limit(2).IDs(setContextOp(ctx, saq.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{subscriptionaddon.Label}
	default:
		err = &NotSingularError{subscriptionaddon.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (saq *SubscriptionAddonQuery) OnlyIDX(ctx context.Context) string {
	id, err := saq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of SubscriptionAddons.
func (saq *SubscriptionAddonQuery) All(ctx context.Context) ([]*SubscriptionAddon, error) {
	ctx = setContextOp(ctx, saq.ctx, ent.OpQueryAll)
	if err := saq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*SubscriptionAddon, *SubscriptionAddonQuery]()
	return withInterceptors[[]*SubscriptionAddon](ctx, saq, qr, saq.inters)
}

// AllX is like All, but panics if an error occurs.
func (saq *SubscriptionAddonQuery) AllX(ctx context.Context) []*SubscriptionAddon {
	nodes, err := saq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of SubscriptionAddon IDs.
func (saq *SubscriptionAddonQuery) IDs(ctx context.Context) (ids []string, err error) {
	if saq.ctx.Unique == nil && saq.path != nil {
		saq.Unique(true)
	}
	ctx = setContextOp(ctx, saq.ctx, ent.OpQueryIDs)
	if err = saq.Select(subscriptionaddon.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (saq *SubscriptionAddonQuery) IDsX(ctx context.Context) []string {
	ids, err := saq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (saq *SubscriptionAddonQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, saq.ctx, ent.OpQueryCount)
	if err := saq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, saq, querierCount[*SubscriptionAddonQuery](), saq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (saq *SubscriptionAddonQuery) CountX(ctx context.Context) int {
	count, err := saq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (saq *SubscriptionAddonQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, saq.ctx, ent.OpQueryExist)
	switch _, err := saq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (saq *SubscriptionAddonQuery) ExistX(ctx context.Context) bool {
	exist, err := saq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the SubscriptionAddonQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (saq *SubscriptionAddonQuery) Clone() *SubscriptionAddonQuery {
	if saq == nil {
		return nil
	}
	return &SubscriptionAddonQuery{
		config:           saq.config,
		ctx:              saq.ctx.Clone(),
		order:            append([]subscriptionaddon.OrderOption{}, saq.order...),
		inters:           append([]Interceptor{}, saq.inters...),
		predicates:       append([]predicate.SubscriptionAddon{}, saq.predicates...),
		withSubscription: saq.withSubscription.Clone(),
		withQuantities:   saq.withQuantities.Clone(),
		withAddon:        saq.withAddon.Clone(),
		// clone intermediate query.
		sql:  saq.sql.Clone(),
		path: saq.path,
	}
}

// WithSubscription tells the query-builder to eager-load the nodes that are connected to
// the "subscription" edge. The optional arguments are used to configure the query builder of the edge.
func (saq *SubscriptionAddonQuery) WithSubscription(opts ...func(*SubscriptionQuery)) *SubscriptionAddonQuery {
	query := (&SubscriptionClient{config: saq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	saq.withSubscription = query
	return saq
}

// WithQuantities tells the query-builder to eager-load the nodes that are connected to
// the "quantities" edge. The optional arguments are used to configure the query builder of the edge.
func (saq *SubscriptionAddonQuery) WithQuantities(opts ...func(*SubscriptionAddonQuantityQuery)) *SubscriptionAddonQuery {
	query := (&SubscriptionAddonQuantityClient{config: saq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	saq.withQuantities = query
	return saq
}

// WithAddon tells the query-builder to eager-load the nodes that are connected to
// the "addon" edge. The optional arguments are used to configure the query builder of the edge.
func (saq *SubscriptionAddonQuery) WithAddon(opts ...func(*AddonQuery)) *SubscriptionAddonQuery {
	query := (&AddonClient{config: saq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	saq.withAddon = query
	return saq
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
//	client.SubscriptionAddon.Query().
//		GroupBy(subscriptionaddon.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (saq *SubscriptionAddonQuery) GroupBy(field string, fields ...string) *SubscriptionAddonGroupBy {
	saq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &SubscriptionAddonGroupBy{build: saq}
	grbuild.flds = &saq.ctx.Fields
	grbuild.label = subscriptionaddon.Label
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
//	client.SubscriptionAddon.Query().
//		Select(subscriptionaddon.FieldNamespace).
//		Scan(ctx, &v)
func (saq *SubscriptionAddonQuery) Select(fields ...string) *SubscriptionAddonSelect {
	saq.ctx.Fields = append(saq.ctx.Fields, fields...)
	sbuild := &SubscriptionAddonSelect{SubscriptionAddonQuery: saq}
	sbuild.label = subscriptionaddon.Label
	sbuild.flds, sbuild.scan = &saq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a SubscriptionAddonSelect configured with the given aggregations.
func (saq *SubscriptionAddonQuery) Aggregate(fns ...AggregateFunc) *SubscriptionAddonSelect {
	return saq.Select().Aggregate(fns...)
}

func (saq *SubscriptionAddonQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range saq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, saq); err != nil {
				return err
			}
		}
	}
	for _, f := range saq.ctx.Fields {
		if !subscriptionaddon.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if saq.path != nil {
		prev, err := saq.path(ctx)
		if err != nil {
			return err
		}
		saq.sql = prev
	}
	return nil
}

func (saq *SubscriptionAddonQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*SubscriptionAddon, error) {
	var (
		nodes       = []*SubscriptionAddon{}
		_spec       = saq.querySpec()
		loadedTypes = [3]bool{
			saq.withSubscription != nil,
			saq.withQuantities != nil,
			saq.withAddon != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*SubscriptionAddon).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &SubscriptionAddon{config: saq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(saq.modifiers) > 0 {
		_spec.Modifiers = saq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, saq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := saq.withSubscription; query != nil {
		if err := saq.loadSubscription(ctx, query, nodes, nil,
			func(n *SubscriptionAddon, e *Subscription) { n.Edges.Subscription = e }); err != nil {
			return nil, err
		}
	}
	if query := saq.withQuantities; query != nil {
		if err := saq.loadQuantities(ctx, query, nodes,
			func(n *SubscriptionAddon) { n.Edges.Quantities = []*SubscriptionAddonQuantity{} },
			func(n *SubscriptionAddon, e *SubscriptionAddonQuantity) {
				n.Edges.Quantities = append(n.Edges.Quantities, e)
			}); err != nil {
			return nil, err
		}
	}
	if query := saq.withAddon; query != nil {
		if err := saq.loadAddon(ctx, query, nodes, nil,
			func(n *SubscriptionAddon, e *Addon) { n.Edges.Addon = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (saq *SubscriptionAddonQuery) loadSubscription(ctx context.Context, query *SubscriptionQuery, nodes []*SubscriptionAddon, init func(*SubscriptionAddon), assign func(*SubscriptionAddon, *Subscription)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*SubscriptionAddon)
	for i := range nodes {
		fk := nodes[i].SubscriptionID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(subscription.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "subscription_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (saq *SubscriptionAddonQuery) loadQuantities(ctx context.Context, query *SubscriptionAddonQuantityQuery, nodes []*SubscriptionAddon, init func(*SubscriptionAddon), assign func(*SubscriptionAddon, *SubscriptionAddonQuantity)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*SubscriptionAddon)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(subscriptionaddonquantity.FieldSubscriptionAddonID)
	}
	query.Where(predicate.SubscriptionAddonQuantity(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(subscriptionaddon.QuantitiesColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.SubscriptionAddonID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "subscription_addon_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (saq *SubscriptionAddonQuery) loadAddon(ctx context.Context, query *AddonQuery, nodes []*SubscriptionAddon, init func(*SubscriptionAddon), assign func(*SubscriptionAddon, *Addon)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*SubscriptionAddon)
	for i := range nodes {
		fk := nodes[i].AddonID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(addon.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "addon_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (saq *SubscriptionAddonQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := saq.querySpec()
	if len(saq.modifiers) > 0 {
		_spec.Modifiers = saq.modifiers
	}
	_spec.Node.Columns = saq.ctx.Fields
	if len(saq.ctx.Fields) > 0 {
		_spec.Unique = saq.ctx.Unique != nil && *saq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, saq.driver, _spec)
}

func (saq *SubscriptionAddonQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(subscriptionaddon.Table, subscriptionaddon.Columns, sqlgraph.NewFieldSpec(subscriptionaddon.FieldID, field.TypeString))
	_spec.From = saq.sql
	if unique := saq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if saq.path != nil {
		_spec.Unique = true
	}
	if fields := saq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, subscriptionaddon.FieldID)
		for i := range fields {
			if fields[i] != subscriptionaddon.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if saq.withSubscription != nil {
			_spec.Node.AddColumnOnce(subscriptionaddon.FieldSubscriptionID)
		}
		if saq.withAddon != nil {
			_spec.Node.AddColumnOnce(subscriptionaddon.FieldAddonID)
		}
	}
	if ps := saq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := saq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := saq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := saq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (saq *SubscriptionAddonQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(saq.driver.Dialect())
	t1 := builder.Table(subscriptionaddon.Table)
	columns := saq.ctx.Fields
	if len(columns) == 0 {
		columns = subscriptionaddon.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if saq.sql != nil {
		selector = saq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if saq.ctx.Unique != nil && *saq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range saq.modifiers {
		m(selector)
	}
	for _, p := range saq.predicates {
		p(selector)
	}
	for _, p := range saq.order {
		p(selector)
	}
	if offset := saq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := saq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (saq *SubscriptionAddonQuery) ForUpdate(opts ...sql.LockOption) *SubscriptionAddonQuery {
	if saq.driver.Dialect() == dialect.Postgres {
		saq.Unique(false)
	}
	saq.modifiers = append(saq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return saq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (saq *SubscriptionAddonQuery) ForShare(opts ...sql.LockOption) *SubscriptionAddonQuery {
	if saq.driver.Dialect() == dialect.Postgres {
		saq.Unique(false)
	}
	saq.modifiers = append(saq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return saq
}

// SubscriptionAddonGroupBy is the group-by builder for SubscriptionAddon entities.
type SubscriptionAddonGroupBy struct {
	selector
	build *SubscriptionAddonQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (sagb *SubscriptionAddonGroupBy) Aggregate(fns ...AggregateFunc) *SubscriptionAddonGroupBy {
	sagb.fns = append(sagb.fns, fns...)
	return sagb
}

// Scan applies the selector query and scans the result into the given value.
func (sagb *SubscriptionAddonGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, sagb.build.ctx, ent.OpQueryGroupBy)
	if err := sagb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*SubscriptionAddonQuery, *SubscriptionAddonGroupBy](ctx, sagb.build, sagb, sagb.build.inters, v)
}

func (sagb *SubscriptionAddonGroupBy) sqlScan(ctx context.Context, root *SubscriptionAddonQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(sagb.fns))
	for _, fn := range sagb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*sagb.flds)+len(sagb.fns))
		for _, f := range *sagb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*sagb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := sagb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// SubscriptionAddonSelect is the builder for selecting fields of SubscriptionAddon entities.
type SubscriptionAddonSelect struct {
	*SubscriptionAddonQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (sas *SubscriptionAddonSelect) Aggregate(fns ...AggregateFunc) *SubscriptionAddonSelect {
	sas.fns = append(sas.fns, fns...)
	return sas
}

// Scan applies the selector query and scans the result into the given value.
func (sas *SubscriptionAddonSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, sas.ctx, ent.OpQuerySelect)
	if err := sas.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*SubscriptionAddonQuery, *SubscriptionAddonSelect](ctx, sas.SubscriptionAddonQuery, sas, sas.inters, v)
}

func (sas *SubscriptionAddonSelect) sqlScan(ctx context.Context, root *SubscriptionAddonQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(sas.fns))
	for _, fn := range sas.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*sas.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := sas.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
