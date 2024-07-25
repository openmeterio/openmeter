// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"fmt"
	"math"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/internal/ent/db/entitlement"
	"github.com/openmeterio/openmeter/internal/ent/db/grant"
	"github.com/openmeterio/openmeter/internal/ent/db/predicate"
)

// GrantQuery is the builder for querying Grant entities.
type GrantQuery struct {
	config
	ctx             *QueryContext
	order           []grant.OrderOption
	inters          []Interceptor
	predicates      []predicate.Grant
	withEntitlement *EntitlementQuery
	modifiers       []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the GrantQuery builder.
func (gq *GrantQuery) Where(ps ...predicate.Grant) *GrantQuery {
	gq.predicates = append(gq.predicates, ps...)
	return gq
}

// Limit the number of records to be returned by this query.
func (gq *GrantQuery) Limit(limit int) *GrantQuery {
	gq.ctx.Limit = &limit
	return gq
}

// Offset to start from.
func (gq *GrantQuery) Offset(offset int) *GrantQuery {
	gq.ctx.Offset = &offset
	return gq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (gq *GrantQuery) Unique(unique bool) *GrantQuery {
	gq.ctx.Unique = &unique
	return gq
}

// Order specifies how the records should be ordered.
func (gq *GrantQuery) Order(o ...grant.OrderOption) *GrantQuery {
	gq.order = append(gq.order, o...)
	return gq
}

// QueryEntitlement chains the current query on the "entitlement" edge.
func (gq *GrantQuery) QueryEntitlement() *EntitlementQuery {
	query := (&EntitlementClient{config: gq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := gq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := gq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(grant.Table, grant.FieldID, selector),
			sqlgraph.To(entitlement.Table, entitlement.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, grant.EntitlementTable, grant.EntitlementColumn),
		)
		fromU = sqlgraph.SetNeighbors(gq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first Grant entity from the query.
// Returns a *NotFoundError when no Grant was found.
func (gq *GrantQuery) First(ctx context.Context) (*Grant, error) {
	nodes, err := gq.Limit(1).All(setContextOp(ctx, gq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{grant.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (gq *GrantQuery) FirstX(ctx context.Context) *Grant {
	node, err := gq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first Grant ID from the query.
// Returns a *NotFoundError when no Grant ID was found.
func (gq *GrantQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = gq.Limit(1).IDs(setContextOp(ctx, gq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{grant.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (gq *GrantQuery) FirstIDX(ctx context.Context) string {
	id, err := gq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single Grant entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one Grant entity is found.
// Returns a *NotFoundError when no Grant entities are found.
func (gq *GrantQuery) Only(ctx context.Context) (*Grant, error) {
	nodes, err := gq.Limit(2).All(setContextOp(ctx, gq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{grant.Label}
	default:
		return nil, &NotSingularError{grant.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (gq *GrantQuery) OnlyX(ctx context.Context) *Grant {
	node, err := gq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only Grant ID in the query.
// Returns a *NotSingularError when more than one Grant ID is found.
// Returns a *NotFoundError when no entities are found.
func (gq *GrantQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = gq.Limit(2).IDs(setContextOp(ctx, gq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{grant.Label}
	default:
		err = &NotSingularError{grant.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (gq *GrantQuery) OnlyIDX(ctx context.Context) string {
	id, err := gq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of Grants.
func (gq *GrantQuery) All(ctx context.Context) ([]*Grant, error) {
	ctx = setContextOp(ctx, gq.ctx, "All")
	if err := gq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*Grant, *GrantQuery]()
	return withInterceptors[[]*Grant](ctx, gq, qr, gq.inters)
}

// AllX is like All, but panics if an error occurs.
func (gq *GrantQuery) AllX(ctx context.Context) []*Grant {
	nodes, err := gq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of Grant IDs.
func (gq *GrantQuery) IDs(ctx context.Context) (ids []string, err error) {
	if gq.ctx.Unique == nil && gq.path != nil {
		gq.Unique(true)
	}
	ctx = setContextOp(ctx, gq.ctx, "IDs")
	if err = gq.Select(grant.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (gq *GrantQuery) IDsX(ctx context.Context) []string {
	ids, err := gq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (gq *GrantQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, gq.ctx, "Count")
	if err := gq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, gq, querierCount[*GrantQuery](), gq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (gq *GrantQuery) CountX(ctx context.Context) int {
	count, err := gq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (gq *GrantQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, gq.ctx, "Exist")
	switch _, err := gq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (gq *GrantQuery) ExistX(ctx context.Context) bool {
	exist, err := gq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the GrantQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (gq *GrantQuery) Clone() *GrantQuery {
	if gq == nil {
		return nil
	}
	return &GrantQuery{
		config:          gq.config,
		ctx:             gq.ctx.Clone(),
		order:           append([]grant.OrderOption{}, gq.order...),
		inters:          append([]Interceptor{}, gq.inters...),
		predicates:      append([]predicate.Grant{}, gq.predicates...),
		withEntitlement: gq.withEntitlement.Clone(),
		// clone intermediate query.
		sql:  gq.sql.Clone(),
		path: gq.path,
	}
}

// WithEntitlement tells the query-builder to eager-load the nodes that are connected to
// the "entitlement" edge. The optional arguments are used to configure the query builder of the edge.
func (gq *GrantQuery) WithEntitlement(opts ...func(*EntitlementQuery)) *GrantQuery {
	query := (&EntitlementClient{config: gq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	gq.withEntitlement = query
	return gq
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
//	client.Grant.Query().
//		GroupBy(grant.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (gq *GrantQuery) GroupBy(field string, fields ...string) *GrantGroupBy {
	gq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &GrantGroupBy{build: gq}
	grbuild.flds = &gq.ctx.Fields
	grbuild.label = grant.Label
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
//	client.Grant.Query().
//		Select(grant.FieldNamespace).
//		Scan(ctx, &v)
func (gq *GrantQuery) Select(fields ...string) *GrantSelect {
	gq.ctx.Fields = append(gq.ctx.Fields, fields...)
	sbuild := &GrantSelect{GrantQuery: gq}
	sbuild.label = grant.Label
	sbuild.flds, sbuild.scan = &gq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a GrantSelect configured with the given aggregations.
func (gq *GrantQuery) Aggregate(fns ...AggregateFunc) *GrantSelect {
	return gq.Select().Aggregate(fns...)
}

func (gq *GrantQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range gq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, gq); err != nil {
				return err
			}
		}
	}
	for _, f := range gq.ctx.Fields {
		if !grant.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if gq.path != nil {
		prev, err := gq.path(ctx)
		if err != nil {
			return err
		}
		gq.sql = prev
	}
	return nil
}

func (gq *GrantQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*Grant, error) {
	var (
		nodes       = []*Grant{}
		_spec       = gq.querySpec()
		loadedTypes = [1]bool{
			gq.withEntitlement != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*Grant).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &Grant{config: gq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(gq.modifiers) > 0 {
		_spec.Modifiers = gq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, gq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := gq.withEntitlement; query != nil {
		if err := gq.loadEntitlement(ctx, query, nodes, nil,
			func(n *Grant, e *Entitlement) { n.Edges.Entitlement = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (gq *GrantQuery) loadEntitlement(ctx context.Context, query *EntitlementQuery, nodes []*Grant, init func(*Grant), assign func(*Grant, *Entitlement)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*Grant)
	for i := range nodes {
		fk := nodes[i].OwnerID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(entitlement.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "owner_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (gq *GrantQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := gq.querySpec()
	if len(gq.modifiers) > 0 {
		_spec.Modifiers = gq.modifiers
	}
	_spec.Node.Columns = gq.ctx.Fields
	if len(gq.ctx.Fields) > 0 {
		_spec.Unique = gq.ctx.Unique != nil && *gq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, gq.driver, _spec)
}

func (gq *GrantQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(grant.Table, grant.Columns, sqlgraph.NewFieldSpec(grant.FieldID, field.TypeString))
	_spec.From = gq.sql
	if unique := gq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if gq.path != nil {
		_spec.Unique = true
	}
	if fields := gq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, grant.FieldID)
		for i := range fields {
			if fields[i] != grant.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if gq.withEntitlement != nil {
			_spec.Node.AddColumnOnce(grant.FieldOwnerID)
		}
	}
	if ps := gq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := gq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := gq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := gq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (gq *GrantQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(gq.driver.Dialect())
	t1 := builder.Table(grant.Table)
	columns := gq.ctx.Fields
	if len(columns) == 0 {
		columns = grant.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if gq.sql != nil {
		selector = gq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if gq.ctx.Unique != nil && *gq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range gq.modifiers {
		m(selector)
	}
	for _, p := range gq.predicates {
		p(selector)
	}
	for _, p := range gq.order {
		p(selector)
	}
	if offset := gq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := gq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (gq *GrantQuery) ForUpdate(opts ...sql.LockOption) *GrantQuery {
	if gq.driver.Dialect() == dialect.Postgres {
		gq.Unique(false)
	}
	gq.modifiers = append(gq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return gq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (gq *GrantQuery) ForShare(opts ...sql.LockOption) *GrantQuery {
	if gq.driver.Dialect() == dialect.Postgres {
		gq.Unique(false)
	}
	gq.modifiers = append(gq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return gq
}

// GrantGroupBy is the group-by builder for Grant entities.
type GrantGroupBy struct {
	selector
	build *GrantQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (ggb *GrantGroupBy) Aggregate(fns ...AggregateFunc) *GrantGroupBy {
	ggb.fns = append(ggb.fns, fns...)
	return ggb
}

// Scan applies the selector query and scans the result into the given value.
func (ggb *GrantGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, ggb.build.ctx, "GroupBy")
	if err := ggb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*GrantQuery, *GrantGroupBy](ctx, ggb.build, ggb, ggb.build.inters, v)
}

func (ggb *GrantGroupBy) sqlScan(ctx context.Context, root *GrantQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(ggb.fns))
	for _, fn := range ggb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*ggb.flds)+len(ggb.fns))
		for _, f := range *ggb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*ggb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := ggb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// GrantSelect is the builder for selecting fields of Grant entities.
type GrantSelect struct {
	*GrantQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (gs *GrantSelect) Aggregate(fns ...AggregateFunc) *GrantSelect {
	gs.fns = append(gs.fns, fns...)
	return gs
}

// Scan applies the selector query and scans the result into the given value.
func (gs *GrantSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, gs.ctx, "Select")
	if err := gs.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*GrantQuery, *GrantSelect](ctx, gs.GrantQuery, gs, gs.inters, v)
}

func (gs *GrantSelect) sqlScan(ctx context.Context, root *GrantQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(gs.fns))
	for _, fn := range gs.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*gs.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := gs.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
