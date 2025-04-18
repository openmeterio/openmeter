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
	"github.com/openmeterio/openmeter/openmeter/ent/db/addonratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planaddon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddon"
)

// AddonQuery is the builder for querying Addon entities.
type AddonQuery struct {
	config
	ctx                    *QueryContext
	order                  []addon.OrderOption
	inters                 []Interceptor
	predicates             []predicate.Addon
	withRatecards          *AddonRateCardQuery
	withPlans              *PlanAddonQuery
	withSubscriptionAddons *SubscriptionAddonQuery
	modifiers              []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the AddonQuery builder.
func (aq *AddonQuery) Where(ps ...predicate.Addon) *AddonQuery {
	aq.predicates = append(aq.predicates, ps...)
	return aq
}

// Limit the number of records to be returned by this query.
func (aq *AddonQuery) Limit(limit int) *AddonQuery {
	aq.ctx.Limit = &limit
	return aq
}

// Offset to start from.
func (aq *AddonQuery) Offset(offset int) *AddonQuery {
	aq.ctx.Offset = &offset
	return aq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (aq *AddonQuery) Unique(unique bool) *AddonQuery {
	aq.ctx.Unique = &unique
	return aq
}

// Order specifies how the records should be ordered.
func (aq *AddonQuery) Order(o ...addon.OrderOption) *AddonQuery {
	aq.order = append(aq.order, o...)
	return aq
}

// QueryRatecards chains the current query on the "ratecards" edge.
func (aq *AddonQuery) QueryRatecards() *AddonRateCardQuery {
	query := (&AddonRateCardClient{config: aq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := aq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := aq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(addon.Table, addon.FieldID, selector),
			sqlgraph.To(addonratecard.Table, addonratecard.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, addon.RatecardsTable, addon.RatecardsColumn),
		)
		fromU = sqlgraph.SetNeighbors(aq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryPlans chains the current query on the "plans" edge.
func (aq *AddonQuery) QueryPlans() *PlanAddonQuery {
	query := (&PlanAddonClient{config: aq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := aq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := aq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(addon.Table, addon.FieldID, selector),
			sqlgraph.To(planaddon.Table, planaddon.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, addon.PlansTable, addon.PlansColumn),
		)
		fromU = sqlgraph.SetNeighbors(aq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QuerySubscriptionAddons chains the current query on the "subscription_addons" edge.
func (aq *AddonQuery) QuerySubscriptionAddons() *SubscriptionAddonQuery {
	query := (&SubscriptionAddonClient{config: aq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := aq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := aq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(addon.Table, addon.FieldID, selector),
			sqlgraph.To(subscriptionaddon.Table, subscriptionaddon.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, addon.SubscriptionAddonsTable, addon.SubscriptionAddonsColumn),
		)
		fromU = sqlgraph.SetNeighbors(aq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first Addon entity from the query.
// Returns a *NotFoundError when no Addon was found.
func (aq *AddonQuery) First(ctx context.Context) (*Addon, error) {
	nodes, err := aq.Limit(1).All(setContextOp(ctx, aq.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{addon.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (aq *AddonQuery) FirstX(ctx context.Context) *Addon {
	node, err := aq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first Addon ID from the query.
// Returns a *NotFoundError when no Addon ID was found.
func (aq *AddonQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = aq.Limit(1).IDs(setContextOp(ctx, aq.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{addon.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (aq *AddonQuery) FirstIDX(ctx context.Context) string {
	id, err := aq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single Addon entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one Addon entity is found.
// Returns a *NotFoundError when no Addon entities are found.
func (aq *AddonQuery) Only(ctx context.Context) (*Addon, error) {
	nodes, err := aq.Limit(2).All(setContextOp(ctx, aq.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{addon.Label}
	default:
		return nil, &NotSingularError{addon.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (aq *AddonQuery) OnlyX(ctx context.Context) *Addon {
	node, err := aq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only Addon ID in the query.
// Returns a *NotSingularError when more than one Addon ID is found.
// Returns a *NotFoundError when no entities are found.
func (aq *AddonQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = aq.Limit(2).IDs(setContextOp(ctx, aq.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{addon.Label}
	default:
		err = &NotSingularError{addon.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (aq *AddonQuery) OnlyIDX(ctx context.Context) string {
	id, err := aq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of Addons.
func (aq *AddonQuery) All(ctx context.Context) ([]*Addon, error) {
	ctx = setContextOp(ctx, aq.ctx, ent.OpQueryAll)
	if err := aq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*Addon, *AddonQuery]()
	return withInterceptors[[]*Addon](ctx, aq, qr, aq.inters)
}

// AllX is like All, but panics if an error occurs.
func (aq *AddonQuery) AllX(ctx context.Context) []*Addon {
	nodes, err := aq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of Addon IDs.
func (aq *AddonQuery) IDs(ctx context.Context) (ids []string, err error) {
	if aq.ctx.Unique == nil && aq.path != nil {
		aq.Unique(true)
	}
	ctx = setContextOp(ctx, aq.ctx, ent.OpQueryIDs)
	if err = aq.Select(addon.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (aq *AddonQuery) IDsX(ctx context.Context) []string {
	ids, err := aq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (aq *AddonQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, aq.ctx, ent.OpQueryCount)
	if err := aq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, aq, querierCount[*AddonQuery](), aq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (aq *AddonQuery) CountX(ctx context.Context) int {
	count, err := aq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (aq *AddonQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, aq.ctx, ent.OpQueryExist)
	switch _, err := aq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (aq *AddonQuery) ExistX(ctx context.Context) bool {
	exist, err := aq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the AddonQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (aq *AddonQuery) Clone() *AddonQuery {
	if aq == nil {
		return nil
	}
	return &AddonQuery{
		config:                 aq.config,
		ctx:                    aq.ctx.Clone(),
		order:                  append([]addon.OrderOption{}, aq.order...),
		inters:                 append([]Interceptor{}, aq.inters...),
		predicates:             append([]predicate.Addon{}, aq.predicates...),
		withRatecards:          aq.withRatecards.Clone(),
		withPlans:              aq.withPlans.Clone(),
		withSubscriptionAddons: aq.withSubscriptionAddons.Clone(),
		// clone intermediate query.
		sql:  aq.sql.Clone(),
		path: aq.path,
	}
}

// WithRatecards tells the query-builder to eager-load the nodes that are connected to
// the "ratecards" edge. The optional arguments are used to configure the query builder of the edge.
func (aq *AddonQuery) WithRatecards(opts ...func(*AddonRateCardQuery)) *AddonQuery {
	query := (&AddonRateCardClient{config: aq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	aq.withRatecards = query
	return aq
}

// WithPlans tells the query-builder to eager-load the nodes that are connected to
// the "plans" edge. The optional arguments are used to configure the query builder of the edge.
func (aq *AddonQuery) WithPlans(opts ...func(*PlanAddonQuery)) *AddonQuery {
	query := (&PlanAddonClient{config: aq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	aq.withPlans = query
	return aq
}

// WithSubscriptionAddons tells the query-builder to eager-load the nodes that are connected to
// the "subscription_addons" edge. The optional arguments are used to configure the query builder of the edge.
func (aq *AddonQuery) WithSubscriptionAddons(opts ...func(*SubscriptionAddonQuery)) *AddonQuery {
	query := (&SubscriptionAddonClient{config: aq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	aq.withSubscriptionAddons = query
	return aq
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
//	client.Addon.Query().
//		GroupBy(addon.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (aq *AddonQuery) GroupBy(field string, fields ...string) *AddonGroupBy {
	aq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &AddonGroupBy{build: aq}
	grbuild.flds = &aq.ctx.Fields
	grbuild.label = addon.Label
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
//	client.Addon.Query().
//		Select(addon.FieldNamespace).
//		Scan(ctx, &v)
func (aq *AddonQuery) Select(fields ...string) *AddonSelect {
	aq.ctx.Fields = append(aq.ctx.Fields, fields...)
	sbuild := &AddonSelect{AddonQuery: aq}
	sbuild.label = addon.Label
	sbuild.flds, sbuild.scan = &aq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a AddonSelect configured with the given aggregations.
func (aq *AddonQuery) Aggregate(fns ...AggregateFunc) *AddonSelect {
	return aq.Select().Aggregate(fns...)
}

func (aq *AddonQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range aq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, aq); err != nil {
				return err
			}
		}
	}
	for _, f := range aq.ctx.Fields {
		if !addon.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if aq.path != nil {
		prev, err := aq.path(ctx)
		if err != nil {
			return err
		}
		aq.sql = prev
	}
	return nil
}

func (aq *AddonQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*Addon, error) {
	var (
		nodes       = []*Addon{}
		_spec       = aq.querySpec()
		loadedTypes = [3]bool{
			aq.withRatecards != nil,
			aq.withPlans != nil,
			aq.withSubscriptionAddons != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*Addon).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &Addon{config: aq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(aq.modifiers) > 0 {
		_spec.Modifiers = aq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, aq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := aq.withRatecards; query != nil {
		if err := aq.loadRatecards(ctx, query, nodes,
			func(n *Addon) { n.Edges.Ratecards = []*AddonRateCard{} },
			func(n *Addon, e *AddonRateCard) { n.Edges.Ratecards = append(n.Edges.Ratecards, e) }); err != nil {
			return nil, err
		}
	}
	if query := aq.withPlans; query != nil {
		if err := aq.loadPlans(ctx, query, nodes,
			func(n *Addon) { n.Edges.Plans = []*PlanAddon{} },
			func(n *Addon, e *PlanAddon) { n.Edges.Plans = append(n.Edges.Plans, e) }); err != nil {
			return nil, err
		}
	}
	if query := aq.withSubscriptionAddons; query != nil {
		if err := aq.loadSubscriptionAddons(ctx, query, nodes,
			func(n *Addon) { n.Edges.SubscriptionAddons = []*SubscriptionAddon{} },
			func(n *Addon, e *SubscriptionAddon) {
				n.Edges.SubscriptionAddons = append(n.Edges.SubscriptionAddons, e)
			}); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (aq *AddonQuery) loadRatecards(ctx context.Context, query *AddonRateCardQuery, nodes []*Addon, init func(*Addon), assign func(*Addon, *AddonRateCard)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*Addon)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(addonratecard.FieldAddonID)
	}
	query.Where(predicate.AddonRateCard(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(addon.RatecardsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.AddonID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "addon_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (aq *AddonQuery) loadPlans(ctx context.Context, query *PlanAddonQuery, nodes []*Addon, init func(*Addon), assign func(*Addon, *PlanAddon)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*Addon)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(planaddon.FieldAddonID)
	}
	query.Where(predicate.PlanAddon(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(addon.PlansColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.AddonID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "addon_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (aq *AddonQuery) loadSubscriptionAddons(ctx context.Context, query *SubscriptionAddonQuery, nodes []*Addon, init func(*Addon), assign func(*Addon, *SubscriptionAddon)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*Addon)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(subscriptionaddon.FieldAddonID)
	}
	query.Where(predicate.SubscriptionAddon(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(addon.SubscriptionAddonsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.AddonID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "addon_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}

func (aq *AddonQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := aq.querySpec()
	if len(aq.modifiers) > 0 {
		_spec.Modifiers = aq.modifiers
	}
	_spec.Node.Columns = aq.ctx.Fields
	if len(aq.ctx.Fields) > 0 {
		_spec.Unique = aq.ctx.Unique != nil && *aq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, aq.driver, _spec)
}

func (aq *AddonQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(addon.Table, addon.Columns, sqlgraph.NewFieldSpec(addon.FieldID, field.TypeString))
	_spec.From = aq.sql
	if unique := aq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if aq.path != nil {
		_spec.Unique = true
	}
	if fields := aq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, addon.FieldID)
		for i := range fields {
			if fields[i] != addon.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := aq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := aq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := aq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := aq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (aq *AddonQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(aq.driver.Dialect())
	t1 := builder.Table(addon.Table)
	columns := aq.ctx.Fields
	if len(columns) == 0 {
		columns = addon.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if aq.sql != nil {
		selector = aq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if aq.ctx.Unique != nil && *aq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range aq.modifiers {
		m(selector)
	}
	for _, p := range aq.predicates {
		p(selector)
	}
	for _, p := range aq.order {
		p(selector)
	}
	if offset := aq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := aq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (aq *AddonQuery) ForUpdate(opts ...sql.LockOption) *AddonQuery {
	if aq.driver.Dialect() == dialect.Postgres {
		aq.Unique(false)
	}
	aq.modifiers = append(aq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return aq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (aq *AddonQuery) ForShare(opts ...sql.LockOption) *AddonQuery {
	if aq.driver.Dialect() == dialect.Postgres {
		aq.Unique(false)
	}
	aq.modifiers = append(aq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return aq
}

// AddonGroupBy is the group-by builder for Addon entities.
type AddonGroupBy struct {
	selector
	build *AddonQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (agb *AddonGroupBy) Aggregate(fns ...AggregateFunc) *AddonGroupBy {
	agb.fns = append(agb.fns, fns...)
	return agb
}

// Scan applies the selector query and scans the result into the given value.
func (agb *AddonGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, agb.build.ctx, ent.OpQueryGroupBy)
	if err := agb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*AddonQuery, *AddonGroupBy](ctx, agb.build, agb, agb.build.inters, v)
}

func (agb *AddonGroupBy) sqlScan(ctx context.Context, root *AddonQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(agb.fns))
	for _, fn := range agb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*agb.flds)+len(agb.fns))
		for _, f := range *agb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*agb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := agb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// AddonSelect is the builder for selecting fields of Addon entities.
type AddonSelect struct {
	*AddonQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (as *AddonSelect) Aggregate(fns ...AggregateFunc) *AddonSelect {
	as.fns = append(as.fns, fns...)
	return as
}

// Scan applies the selector query and scans the result into the given value.
func (as *AddonSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, as.ctx, ent.OpQuerySelect)
	if err := as.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*AddonQuery, *AddonSelect](ctx, as.AddonQuery, as, as.inters, v)
}

func (as *AddonSelect) sqlScan(ctx context.Context, root *AddonQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(as.fns))
	for _, fn := range as.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*as.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := as.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
