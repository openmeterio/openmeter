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
	dbapp "github.com/openmeterio/openmeter/openmeter/ent/db/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// AppStripeQuery is the builder for querying AppStripe entities.
type AppStripeQuery struct {
	config
	ctx              *QueryContext
	order            []appstripe.OrderOption
	inters           []Interceptor
	predicates       []predicate.AppStripe
	withCustomerApps *AppStripeCustomerQuery
	withApp          *AppQuery
	modifiers        []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the AppStripeQuery builder.
func (_q *AppStripeQuery) Where(ps ...predicate.AppStripe) *AppStripeQuery {
	_q.predicates = append(_q.predicates, ps...)
	return _q
}

// Limit the number of records to be returned by this query.
func (_q *AppStripeQuery) Limit(limit int) *AppStripeQuery {
	_q.ctx.Limit = &limit
	return _q
}

// Offset to start from.
func (_q *AppStripeQuery) Offset(offset int) *AppStripeQuery {
	_q.ctx.Offset = &offset
	return _q
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (_q *AppStripeQuery) Unique(unique bool) *AppStripeQuery {
	_q.ctx.Unique = &unique
	return _q
}

// Order specifies how the records should be ordered.
func (_q *AppStripeQuery) Order(o ...appstripe.OrderOption) *AppStripeQuery {
	_q.order = append(_q.order, o...)
	return _q
}

// QueryCustomerApps chains the current query on the "customer_apps" edge.
func (_q *AppStripeQuery) QueryCustomerApps() *AppStripeCustomerQuery {
	query := (&AppStripeCustomerClient{config: _q.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := _q.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := _q.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(appstripe.Table, appstripe.FieldID, selector),
			sqlgraph.To(appstripecustomer.Table, appstripecustomer.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, appstripe.CustomerAppsTable, appstripe.CustomerAppsColumn),
		)
		fromU = sqlgraph.SetNeighbors(_q.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryApp chains the current query on the "app" edge.
func (_q *AppStripeQuery) QueryApp() *AppQuery {
	query := (&AppClient{config: _q.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := _q.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := _q.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(appstripe.Table, appstripe.FieldID, selector),
			sqlgraph.To(dbapp.Table, dbapp.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, appstripe.AppTable, appstripe.AppColumn),
		)
		fromU = sqlgraph.SetNeighbors(_q.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first AppStripe entity from the query.
// Returns a *NotFoundError when no AppStripe was found.
func (_q *AppStripeQuery) First(ctx context.Context) (*AppStripe, error) {
	nodes, err := _q.Limit(1).All(setContextOp(ctx, _q.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{appstripe.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (_q *AppStripeQuery) FirstX(ctx context.Context) *AppStripe {
	node, err := _q.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first AppStripe ID from the query.
// Returns a *NotFoundError when no AppStripe ID was found.
func (_q *AppStripeQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = _q.Limit(1).IDs(setContextOp(ctx, _q.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{appstripe.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (_q *AppStripeQuery) FirstIDX(ctx context.Context) string {
	id, err := _q.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single AppStripe entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one AppStripe entity is found.
// Returns a *NotFoundError when no AppStripe entities are found.
func (_q *AppStripeQuery) Only(ctx context.Context) (*AppStripe, error) {
	nodes, err := _q.Limit(2).All(setContextOp(ctx, _q.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{appstripe.Label}
	default:
		return nil, &NotSingularError{appstripe.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (_q *AppStripeQuery) OnlyX(ctx context.Context) *AppStripe {
	node, err := _q.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only AppStripe ID in the query.
// Returns a *NotSingularError when more than one AppStripe ID is found.
// Returns a *NotFoundError when no entities are found.
func (_q *AppStripeQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = _q.Limit(2).IDs(setContextOp(ctx, _q.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{appstripe.Label}
	default:
		err = &NotSingularError{appstripe.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (_q *AppStripeQuery) OnlyIDX(ctx context.Context) string {
	id, err := _q.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of AppStripes.
func (_q *AppStripeQuery) All(ctx context.Context) ([]*AppStripe, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryAll)
	if err := _q.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*AppStripe, *AppStripeQuery]()
	return withInterceptors[[]*AppStripe](ctx, _q, qr, _q.inters)
}

// AllX is like All, but panics if an error occurs.
func (_q *AppStripeQuery) AllX(ctx context.Context) []*AppStripe {
	nodes, err := _q.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of AppStripe IDs.
func (_q *AppStripeQuery) IDs(ctx context.Context) (ids []string, err error) {
	if _q.ctx.Unique == nil && _q.path != nil {
		_q.Unique(true)
	}
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryIDs)
	if err = _q.Select(appstripe.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (_q *AppStripeQuery) IDsX(ctx context.Context) []string {
	ids, err := _q.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (_q *AppStripeQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryCount)
	if err := _q.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, _q, querierCount[*AppStripeQuery](), _q.inters)
}

// CountX is like Count, but panics if an error occurs.
func (_q *AppStripeQuery) CountX(ctx context.Context) int {
	count, err := _q.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (_q *AppStripeQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryExist)
	switch _, err := _q.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (_q *AppStripeQuery) ExistX(ctx context.Context) bool {
	exist, err := _q.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the AppStripeQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (_q *AppStripeQuery) Clone() *AppStripeQuery {
	if _q == nil {
		return nil
	}
	return &AppStripeQuery{
		config:           _q.config,
		ctx:              _q.ctx.Clone(),
		order:            append([]appstripe.OrderOption{}, _q.order...),
		inters:           append([]Interceptor{}, _q.inters...),
		predicates:       append([]predicate.AppStripe{}, _q.predicates...),
		withCustomerApps: _q.withCustomerApps.Clone(),
		withApp:          _q.withApp.Clone(),
		// clone intermediate query.
		sql:  _q.sql.Clone(),
		path: _q.path,
	}
}

// WithCustomerApps tells the query-builder to eager-load the nodes that are connected to
// the "customer_apps" edge. The optional arguments are used to configure the query builder of the edge.
func (_q *AppStripeQuery) WithCustomerApps(opts ...func(*AppStripeCustomerQuery)) *AppStripeQuery {
	query := (&AppStripeCustomerClient{config: _q.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	_q.withCustomerApps = query
	return _q
}

// WithApp tells the query-builder to eager-load the nodes that are connected to
// the "app" edge. The optional arguments are used to configure the query builder of the edge.
func (_q *AppStripeQuery) WithApp(opts ...func(*AppQuery)) *AppStripeQuery {
	query := (&AppClient{config: _q.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	_q.withApp = query
	return _q
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
//	client.AppStripe.Query().
//		GroupBy(appstripe.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (_q *AppStripeQuery) GroupBy(field string, fields ...string) *AppStripeGroupBy {
	_q.ctx.Fields = append([]string{field}, fields...)
	grbuild := &AppStripeGroupBy{build: _q}
	grbuild.flds = &_q.ctx.Fields
	grbuild.label = appstripe.Label
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
//	client.AppStripe.Query().
//		Select(appstripe.FieldNamespace).
//		Scan(ctx, &v)
func (_q *AppStripeQuery) Select(fields ...string) *AppStripeSelect {
	_q.ctx.Fields = append(_q.ctx.Fields, fields...)
	sbuild := &AppStripeSelect{AppStripeQuery: _q}
	sbuild.label = appstripe.Label
	sbuild.flds, sbuild.scan = &_q.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a AppStripeSelect configured with the given aggregations.
func (_q *AppStripeQuery) Aggregate(fns ...AggregateFunc) *AppStripeSelect {
	return _q.Select().Aggregate(fns...)
}

func (_q *AppStripeQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range _q.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, _q); err != nil {
				return err
			}
		}
	}
	for _, f := range _q.ctx.Fields {
		if !appstripe.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if _q.path != nil {
		prev, err := _q.path(ctx)
		if err != nil {
			return err
		}
		_q.sql = prev
	}
	return nil
}

func (_q *AppStripeQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*AppStripe, error) {
	var (
		nodes       = []*AppStripe{}
		_spec       = _q.querySpec()
		loadedTypes = [2]bool{
			_q.withCustomerApps != nil,
			_q.withApp != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*AppStripe).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &AppStripe{config: _q.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(_q.modifiers) > 0 {
		_spec.Modifiers = _q.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, _q.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := _q.withCustomerApps; query != nil {
		if err := _q.loadCustomerApps(ctx, query, nodes,
			func(n *AppStripe) { n.Edges.CustomerApps = []*AppStripeCustomer{} },
			func(n *AppStripe, e *AppStripeCustomer) { n.Edges.CustomerApps = append(n.Edges.CustomerApps, e) }); err != nil {
			return nil, err
		}
	}
	if query := _q.withApp; query != nil {
		if err := _q.loadApp(ctx, query, nodes, nil,
			func(n *AppStripe, e *App) { n.Edges.App = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (_q *AppStripeQuery) loadCustomerApps(ctx context.Context, query *AppStripeCustomerQuery, nodes []*AppStripe, init func(*AppStripe), assign func(*AppStripe, *AppStripeCustomer)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*AppStripe)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(appstripecustomer.FieldAppID)
	}
	query.Where(predicate.AppStripeCustomer(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(appstripe.CustomerAppsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.AppID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "app_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (_q *AppStripeQuery) loadApp(ctx context.Context, query *AppQuery, nodes []*AppStripe, init func(*AppStripe), assign func(*AppStripe, *App)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*AppStripe)
	for i := range nodes {
		fk := nodes[i].ID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(dbapp.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (_q *AppStripeQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := _q.querySpec()
	if len(_q.modifiers) > 0 {
		_spec.Modifiers = _q.modifiers
	}
	_spec.Node.Columns = _q.ctx.Fields
	if len(_q.ctx.Fields) > 0 {
		_spec.Unique = _q.ctx.Unique != nil && *_q.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, _q.driver, _spec)
}

func (_q *AppStripeQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(appstripe.Table, appstripe.Columns, sqlgraph.NewFieldSpec(appstripe.FieldID, field.TypeString))
	_spec.From = _q.sql
	if unique := _q.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if _q.path != nil {
		_spec.Unique = true
	}
	if fields := _q.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, appstripe.FieldID)
		for i := range fields {
			if fields[i] != appstripe.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := _q.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := _q.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := _q.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := _q.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (_q *AppStripeQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(_q.driver.Dialect())
	t1 := builder.Table(appstripe.Table)
	columns := _q.ctx.Fields
	if len(columns) == 0 {
		columns = appstripe.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if _q.sql != nil {
		selector = _q.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if _q.ctx.Unique != nil && *_q.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range _q.modifiers {
		m(selector)
	}
	for _, p := range _q.predicates {
		p(selector)
	}
	for _, p := range _q.order {
		p(selector)
	}
	if offset := _q.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := _q.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (_q *AppStripeQuery) ForUpdate(opts ...sql.LockOption) *AppStripeQuery {
	if _q.driver.Dialect() == dialect.Postgres {
		_q.Unique(false)
	}
	_q.modifiers = append(_q.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return _q
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (_q *AppStripeQuery) ForShare(opts ...sql.LockOption) *AppStripeQuery {
	if _q.driver.Dialect() == dialect.Postgres {
		_q.Unique(false)
	}
	_q.modifiers = append(_q.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return _q
}

// AppStripeGroupBy is the group-by builder for AppStripe entities.
type AppStripeGroupBy struct {
	selector
	build *AppStripeQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (asgb *AppStripeGroupBy) Aggregate(fns ...AggregateFunc) *AppStripeGroupBy {
	asgb.fns = append(asgb.fns, fns...)
	return asgb
}

// Scan applies the selector query and scans the result into the given value.
func (asgb *AppStripeGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, asgb.build.ctx, ent.OpQueryGroupBy)
	if err := asgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*AppStripeQuery, *AppStripeGroupBy](ctx, asgb.build, asgb, asgb.build.inters, v)
}

func (asgb *AppStripeGroupBy) sqlScan(ctx context.Context, root *AppStripeQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(asgb.fns))
	for _, fn := range asgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*asgb.flds)+len(asgb.fns))
		for _, f := range *asgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*asgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := asgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// AppStripeSelect is the builder for selecting fields of AppStripe entities.
type AppStripeSelect struct {
	*AppStripeQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (ass *AppStripeSelect) Aggregate(fns ...AggregateFunc) *AppStripeSelect {
	ass.fns = append(ass.fns, fns...)
	return ass
}

// Scan applies the selector query and scans the result into the given value.
func (ass *AppStripeSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, ass.ctx, ent.OpQuerySelect)
	if err := ass.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*AppStripeQuery, *AppStripeSelect](ctx, ass.AppStripeQuery, ass, ass.inters, v)
}

func (ass *AppStripeSelect) sqlScan(ctx context.Context, root *AppStripeQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(ass.fns))
	for _, fn := range ass.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*ass.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := ass.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
