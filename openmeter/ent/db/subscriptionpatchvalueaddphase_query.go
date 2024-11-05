// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"fmt"
	"math"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionpatch"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionpatchvalueaddphase"
)

// SubscriptionPatchValueAddPhaseQuery is the builder for querying SubscriptionPatchValueAddPhase entities.
type SubscriptionPatchValueAddPhaseQuery struct {
	config
	ctx                   *QueryContext
	order                 []subscriptionpatchvalueaddphase.OrderOption
	inters                []Interceptor
	predicates            []predicate.SubscriptionPatchValueAddPhase
	withSubscriptionPatch *SubscriptionPatchQuery
	modifiers             []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the SubscriptionPatchValueAddPhaseQuery builder.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) Where(ps ...predicate.SubscriptionPatchValueAddPhase) *SubscriptionPatchValueAddPhaseQuery {
	spvapq.predicates = append(spvapq.predicates, ps...)
	return spvapq
}

// Limit the number of records to be returned by this query.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) Limit(limit int) *SubscriptionPatchValueAddPhaseQuery {
	spvapq.ctx.Limit = &limit
	return spvapq
}

// Offset to start from.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) Offset(offset int) *SubscriptionPatchValueAddPhaseQuery {
	spvapq.ctx.Offset = &offset
	return spvapq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) Unique(unique bool) *SubscriptionPatchValueAddPhaseQuery {
	spvapq.ctx.Unique = &unique
	return spvapq
}

// Order specifies how the records should be ordered.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) Order(o ...subscriptionpatchvalueaddphase.OrderOption) *SubscriptionPatchValueAddPhaseQuery {
	spvapq.order = append(spvapq.order, o...)
	return spvapq
}

// QuerySubscriptionPatch chains the current query on the "subscription_patch" edge.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) QuerySubscriptionPatch() *SubscriptionPatchQuery {
	query := (&SubscriptionPatchClient{config: spvapq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := spvapq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := spvapq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(subscriptionpatchvalueaddphase.Table, subscriptionpatchvalueaddphase.FieldID, selector),
			sqlgraph.To(subscriptionpatch.Table, subscriptionpatch.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, subscriptionpatchvalueaddphase.SubscriptionPatchTable, subscriptionpatchvalueaddphase.SubscriptionPatchColumn),
		)
		fromU = sqlgraph.SetNeighbors(spvapq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first SubscriptionPatchValueAddPhase entity from the query.
// Returns a *NotFoundError when no SubscriptionPatchValueAddPhase was found.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) First(ctx context.Context) (*SubscriptionPatchValueAddPhase, error) {
	nodes, err := spvapq.Limit(1).All(setContextOp(ctx, spvapq.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{subscriptionpatchvalueaddphase.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) FirstX(ctx context.Context) *SubscriptionPatchValueAddPhase {
	node, err := spvapq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first SubscriptionPatchValueAddPhase ID from the query.
// Returns a *NotFoundError when no SubscriptionPatchValueAddPhase ID was found.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = spvapq.Limit(1).IDs(setContextOp(ctx, spvapq.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{subscriptionpatchvalueaddphase.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) FirstIDX(ctx context.Context) string {
	id, err := spvapq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single SubscriptionPatchValueAddPhase entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one SubscriptionPatchValueAddPhase entity is found.
// Returns a *NotFoundError when no SubscriptionPatchValueAddPhase entities are found.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) Only(ctx context.Context) (*SubscriptionPatchValueAddPhase, error) {
	nodes, err := spvapq.Limit(2).All(setContextOp(ctx, spvapq.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{subscriptionpatchvalueaddphase.Label}
	default:
		return nil, &NotSingularError{subscriptionpatchvalueaddphase.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) OnlyX(ctx context.Context) *SubscriptionPatchValueAddPhase {
	node, err := spvapq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only SubscriptionPatchValueAddPhase ID in the query.
// Returns a *NotSingularError when more than one SubscriptionPatchValueAddPhase ID is found.
// Returns a *NotFoundError when no entities are found.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = spvapq.Limit(2).IDs(setContextOp(ctx, spvapq.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{subscriptionpatchvalueaddphase.Label}
	default:
		err = &NotSingularError{subscriptionpatchvalueaddphase.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) OnlyIDX(ctx context.Context) string {
	id, err := spvapq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of SubscriptionPatchValueAddPhases.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) All(ctx context.Context) ([]*SubscriptionPatchValueAddPhase, error) {
	ctx = setContextOp(ctx, spvapq.ctx, ent.OpQueryAll)
	if err := spvapq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*SubscriptionPatchValueAddPhase, *SubscriptionPatchValueAddPhaseQuery]()
	return withInterceptors[[]*SubscriptionPatchValueAddPhase](ctx, spvapq, qr, spvapq.inters)
}

// AllX is like All, but panics if an error occurs.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) AllX(ctx context.Context) []*SubscriptionPatchValueAddPhase {
	nodes, err := spvapq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of SubscriptionPatchValueAddPhase IDs.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) IDs(ctx context.Context) (ids []string, err error) {
	if spvapq.ctx.Unique == nil && spvapq.path != nil {
		spvapq.Unique(true)
	}
	ctx = setContextOp(ctx, spvapq.ctx, ent.OpQueryIDs)
	if err = spvapq.Select(subscriptionpatchvalueaddphase.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) IDsX(ctx context.Context) []string {
	ids, err := spvapq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, spvapq.ctx, ent.OpQueryCount)
	if err := spvapq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, spvapq, querierCount[*SubscriptionPatchValueAddPhaseQuery](), spvapq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) CountX(ctx context.Context) int {
	count, err := spvapq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, spvapq.ctx, ent.OpQueryExist)
	switch _, err := spvapq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) ExistX(ctx context.Context) bool {
	exist, err := spvapq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the SubscriptionPatchValueAddPhaseQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) Clone() *SubscriptionPatchValueAddPhaseQuery {
	if spvapq == nil {
		return nil
	}
	return &SubscriptionPatchValueAddPhaseQuery{
		config:                spvapq.config,
		ctx:                   spvapq.ctx.Clone(),
		order:                 append([]subscriptionpatchvalueaddphase.OrderOption{}, spvapq.order...),
		inters:                append([]Interceptor{}, spvapq.inters...),
		predicates:            append([]predicate.SubscriptionPatchValueAddPhase{}, spvapq.predicates...),
		withSubscriptionPatch: spvapq.withSubscriptionPatch.Clone(),
		// clone intermediate query.
		sql:  spvapq.sql.Clone(),
		path: spvapq.path,
	}
}

// WithSubscriptionPatch tells the query-builder to eager-load the nodes that are connected to
// the "subscription_patch" edge. The optional arguments are used to configure the query builder of the edge.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) WithSubscriptionPatch(opts ...func(*SubscriptionPatchQuery)) *SubscriptionPatchValueAddPhaseQuery {
	query := (&SubscriptionPatchClient{config: spvapq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	spvapq.withSubscriptionPatch = query
	return spvapq
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
//	client.SubscriptionPatchValueAddPhase.Query().
//		GroupBy(subscriptionpatchvalueaddphase.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (spvapq *SubscriptionPatchValueAddPhaseQuery) GroupBy(field string, fields ...string) *SubscriptionPatchValueAddPhaseGroupBy {
	spvapq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &SubscriptionPatchValueAddPhaseGroupBy{build: spvapq}
	grbuild.flds = &spvapq.ctx.Fields
	grbuild.label = subscriptionpatchvalueaddphase.Label
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
//	client.SubscriptionPatchValueAddPhase.Query().
//		Select(subscriptionpatchvalueaddphase.FieldNamespace).
//		Scan(ctx, &v)
func (spvapq *SubscriptionPatchValueAddPhaseQuery) Select(fields ...string) *SubscriptionPatchValueAddPhaseSelect {
	spvapq.ctx.Fields = append(spvapq.ctx.Fields, fields...)
	sbuild := &SubscriptionPatchValueAddPhaseSelect{SubscriptionPatchValueAddPhaseQuery: spvapq}
	sbuild.label = subscriptionpatchvalueaddphase.Label
	sbuild.flds, sbuild.scan = &spvapq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a SubscriptionPatchValueAddPhaseSelect configured with the given aggregations.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) Aggregate(fns ...AggregateFunc) *SubscriptionPatchValueAddPhaseSelect {
	return spvapq.Select().Aggregate(fns...)
}

func (spvapq *SubscriptionPatchValueAddPhaseQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range spvapq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, spvapq); err != nil {
				return err
			}
		}
	}
	for _, f := range spvapq.ctx.Fields {
		if !subscriptionpatchvalueaddphase.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if spvapq.path != nil {
		prev, err := spvapq.path(ctx)
		if err != nil {
			return err
		}
		spvapq.sql = prev
	}
	return nil
}

func (spvapq *SubscriptionPatchValueAddPhaseQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*SubscriptionPatchValueAddPhase, error) {
	var (
		nodes       = []*SubscriptionPatchValueAddPhase{}
		_spec       = spvapq.querySpec()
		loadedTypes = [1]bool{
			spvapq.withSubscriptionPatch != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*SubscriptionPatchValueAddPhase).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &SubscriptionPatchValueAddPhase{config: spvapq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(spvapq.modifiers) > 0 {
		_spec.Modifiers = spvapq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, spvapq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := spvapq.withSubscriptionPatch; query != nil {
		if err := spvapq.loadSubscriptionPatch(ctx, query, nodes, nil,
			func(n *SubscriptionPatchValueAddPhase, e *SubscriptionPatch) { n.Edges.SubscriptionPatch = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (spvapq *SubscriptionPatchValueAddPhaseQuery) loadSubscriptionPatch(ctx context.Context, query *SubscriptionPatchQuery, nodes []*SubscriptionPatchValueAddPhase, init func(*SubscriptionPatchValueAddPhase), assign func(*SubscriptionPatchValueAddPhase, *SubscriptionPatch)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*SubscriptionPatchValueAddPhase)
	for i := range nodes {
		fk := nodes[i].SubscriptionPatchID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(subscriptionpatch.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "subscription_patch_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (spvapq *SubscriptionPatchValueAddPhaseQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := spvapq.querySpec()
	if len(spvapq.modifiers) > 0 {
		_spec.Modifiers = spvapq.modifiers
	}
	_spec.Node.Columns = spvapq.ctx.Fields
	if len(spvapq.ctx.Fields) > 0 {
		_spec.Unique = spvapq.ctx.Unique != nil && *spvapq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, spvapq.driver, _spec)
}

func (spvapq *SubscriptionPatchValueAddPhaseQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(subscriptionpatchvalueaddphase.Table, subscriptionpatchvalueaddphase.Columns, sqlgraph.NewFieldSpec(subscriptionpatchvalueaddphase.FieldID, field.TypeString))
	_spec.From = spvapq.sql
	if unique := spvapq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if spvapq.path != nil {
		_spec.Unique = true
	}
	if fields := spvapq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, subscriptionpatchvalueaddphase.FieldID)
		for i := range fields {
			if fields[i] != subscriptionpatchvalueaddphase.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if spvapq.withSubscriptionPatch != nil {
			_spec.Node.AddColumnOnce(subscriptionpatchvalueaddphase.FieldSubscriptionPatchID)
		}
	}
	if ps := spvapq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := spvapq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := spvapq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := spvapq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (spvapq *SubscriptionPatchValueAddPhaseQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(spvapq.driver.Dialect())
	t1 := builder.Table(subscriptionpatchvalueaddphase.Table)
	columns := spvapq.ctx.Fields
	if len(columns) == 0 {
		columns = subscriptionpatchvalueaddphase.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if spvapq.sql != nil {
		selector = spvapq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if spvapq.ctx.Unique != nil && *spvapq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range spvapq.modifiers {
		m(selector)
	}
	for _, p := range spvapq.predicates {
		p(selector)
	}
	for _, p := range spvapq.order {
		p(selector)
	}
	if offset := spvapq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := spvapq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) ForUpdate(opts ...sql.LockOption) *SubscriptionPatchValueAddPhaseQuery {
	if spvapq.driver.Dialect() == dialect.Postgres {
		spvapq.Unique(false)
	}
	spvapq.modifiers = append(spvapq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return spvapq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (spvapq *SubscriptionPatchValueAddPhaseQuery) ForShare(opts ...sql.LockOption) *SubscriptionPatchValueAddPhaseQuery {
	if spvapq.driver.Dialect() == dialect.Postgres {
		spvapq.Unique(false)
	}
	spvapq.modifiers = append(spvapq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return spvapq
}

// SubscriptionPatchValueAddPhaseGroupBy is the group-by builder for SubscriptionPatchValueAddPhase entities.
type SubscriptionPatchValueAddPhaseGroupBy struct {
	selector
	build *SubscriptionPatchValueAddPhaseQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (spvapgb *SubscriptionPatchValueAddPhaseGroupBy) Aggregate(fns ...AggregateFunc) *SubscriptionPatchValueAddPhaseGroupBy {
	spvapgb.fns = append(spvapgb.fns, fns...)
	return spvapgb
}

// Scan applies the selector query and scans the result into the given value.
func (spvapgb *SubscriptionPatchValueAddPhaseGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, spvapgb.build.ctx, ent.OpQueryGroupBy)
	if err := spvapgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*SubscriptionPatchValueAddPhaseQuery, *SubscriptionPatchValueAddPhaseGroupBy](ctx, spvapgb.build, spvapgb, spvapgb.build.inters, v)
}

func (spvapgb *SubscriptionPatchValueAddPhaseGroupBy) sqlScan(ctx context.Context, root *SubscriptionPatchValueAddPhaseQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(spvapgb.fns))
	for _, fn := range spvapgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*spvapgb.flds)+len(spvapgb.fns))
		for _, f := range *spvapgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*spvapgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := spvapgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// SubscriptionPatchValueAddPhaseSelect is the builder for selecting fields of SubscriptionPatchValueAddPhase entities.
type SubscriptionPatchValueAddPhaseSelect struct {
	*SubscriptionPatchValueAddPhaseQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (spvaps *SubscriptionPatchValueAddPhaseSelect) Aggregate(fns ...AggregateFunc) *SubscriptionPatchValueAddPhaseSelect {
	spvaps.fns = append(spvaps.fns, fns...)
	return spvaps
}

// Scan applies the selector query and scans the result into the given value.
func (spvaps *SubscriptionPatchValueAddPhaseSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, spvaps.ctx, ent.OpQuerySelect)
	if err := spvaps.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*SubscriptionPatchValueAddPhaseQuery, *SubscriptionPatchValueAddPhaseSelect](ctx, spvaps.SubscriptionPatchValueAddPhaseQuery, spvaps, spvaps.inters, v)
}

func (spvaps *SubscriptionPatchValueAddPhaseSelect) sqlScan(ctx context.Context, root *SubscriptionPatchValueAddPhaseQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(spvaps.fns))
	for _, fn := range spvaps.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*spvaps.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := spvaps.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}