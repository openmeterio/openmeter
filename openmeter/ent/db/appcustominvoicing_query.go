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
	"github.com/openmeterio/openmeter/openmeter/ent/db/appcustominvoicing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appcustominvoicingcustomer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// AppCustomInvoicingQuery is the builder for querying AppCustomInvoicing entities.
type AppCustomInvoicingQuery struct {
	config
	ctx              *QueryContext
	order            []appcustominvoicing.OrderOption
	inters           []Interceptor
	predicates       []predicate.AppCustomInvoicing
	withCustomerApps *AppCustomInvoicingCustomerQuery
	withApp          *AppQuery
	modifiers        []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the AppCustomInvoicingQuery builder.
func (aciq *AppCustomInvoicingQuery) Where(ps ...predicate.AppCustomInvoicing) *AppCustomInvoicingQuery {
	aciq.predicates = append(aciq.predicates, ps...)
	return aciq
}

// Limit the number of records to be returned by this query.
func (aciq *AppCustomInvoicingQuery) Limit(limit int) *AppCustomInvoicingQuery {
	aciq.ctx.Limit = &limit
	return aciq
}

// Offset to start from.
func (aciq *AppCustomInvoicingQuery) Offset(offset int) *AppCustomInvoicingQuery {
	aciq.ctx.Offset = &offset
	return aciq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (aciq *AppCustomInvoicingQuery) Unique(unique bool) *AppCustomInvoicingQuery {
	aciq.ctx.Unique = &unique
	return aciq
}

// Order specifies how the records should be ordered.
func (aciq *AppCustomInvoicingQuery) Order(o ...appcustominvoicing.OrderOption) *AppCustomInvoicingQuery {
	aciq.order = append(aciq.order, o...)
	return aciq
}

// QueryCustomerApps chains the current query on the "customer_apps" edge.
func (aciq *AppCustomInvoicingQuery) QueryCustomerApps() *AppCustomInvoicingCustomerQuery {
	query := (&AppCustomInvoicingCustomerClient{config: aciq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := aciq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := aciq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(appcustominvoicing.Table, appcustominvoicing.FieldID, selector),
			sqlgraph.To(appcustominvoicingcustomer.Table, appcustominvoicingcustomer.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, appcustominvoicing.CustomerAppsTable, appcustominvoicing.CustomerAppsColumn),
		)
		fromU = sqlgraph.SetNeighbors(aciq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryApp chains the current query on the "app" edge.
func (aciq *AppCustomInvoicingQuery) QueryApp() *AppQuery {
	query := (&AppClient{config: aciq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := aciq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := aciq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(appcustominvoicing.Table, appcustominvoicing.FieldID, selector),
			sqlgraph.To(dbapp.Table, dbapp.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, appcustominvoicing.AppTable, appcustominvoicing.AppColumn),
		)
		fromU = sqlgraph.SetNeighbors(aciq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first AppCustomInvoicing entity from the query.
// Returns a *NotFoundError when no AppCustomInvoicing was found.
func (aciq *AppCustomInvoicingQuery) First(ctx context.Context) (*AppCustomInvoicing, error) {
	nodes, err := aciq.Limit(1).All(setContextOp(ctx, aciq.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{appcustominvoicing.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (aciq *AppCustomInvoicingQuery) FirstX(ctx context.Context) *AppCustomInvoicing {
	node, err := aciq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first AppCustomInvoicing ID from the query.
// Returns a *NotFoundError when no AppCustomInvoicing ID was found.
func (aciq *AppCustomInvoicingQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = aciq.Limit(1).IDs(setContextOp(ctx, aciq.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{appcustominvoicing.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (aciq *AppCustomInvoicingQuery) FirstIDX(ctx context.Context) string {
	id, err := aciq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single AppCustomInvoicing entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one AppCustomInvoicing entity is found.
// Returns a *NotFoundError when no AppCustomInvoicing entities are found.
func (aciq *AppCustomInvoicingQuery) Only(ctx context.Context) (*AppCustomInvoicing, error) {
	nodes, err := aciq.Limit(2).All(setContextOp(ctx, aciq.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{appcustominvoicing.Label}
	default:
		return nil, &NotSingularError{appcustominvoicing.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (aciq *AppCustomInvoicingQuery) OnlyX(ctx context.Context) *AppCustomInvoicing {
	node, err := aciq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only AppCustomInvoicing ID in the query.
// Returns a *NotSingularError when more than one AppCustomInvoicing ID is found.
// Returns a *NotFoundError when no entities are found.
func (aciq *AppCustomInvoicingQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = aciq.Limit(2).IDs(setContextOp(ctx, aciq.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{appcustominvoicing.Label}
	default:
		err = &NotSingularError{appcustominvoicing.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (aciq *AppCustomInvoicingQuery) OnlyIDX(ctx context.Context) string {
	id, err := aciq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of AppCustomInvoicings.
func (aciq *AppCustomInvoicingQuery) All(ctx context.Context) ([]*AppCustomInvoicing, error) {
	ctx = setContextOp(ctx, aciq.ctx, ent.OpQueryAll)
	if err := aciq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*AppCustomInvoicing, *AppCustomInvoicingQuery]()
	return withInterceptors[[]*AppCustomInvoicing](ctx, aciq, qr, aciq.inters)
}

// AllX is like All, but panics if an error occurs.
func (aciq *AppCustomInvoicingQuery) AllX(ctx context.Context) []*AppCustomInvoicing {
	nodes, err := aciq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of AppCustomInvoicing IDs.
func (aciq *AppCustomInvoicingQuery) IDs(ctx context.Context) (ids []string, err error) {
	if aciq.ctx.Unique == nil && aciq.path != nil {
		aciq.Unique(true)
	}
	ctx = setContextOp(ctx, aciq.ctx, ent.OpQueryIDs)
	if err = aciq.Select(appcustominvoicing.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (aciq *AppCustomInvoicingQuery) IDsX(ctx context.Context) []string {
	ids, err := aciq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (aciq *AppCustomInvoicingQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, aciq.ctx, ent.OpQueryCount)
	if err := aciq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, aciq, querierCount[*AppCustomInvoicingQuery](), aciq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (aciq *AppCustomInvoicingQuery) CountX(ctx context.Context) int {
	count, err := aciq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (aciq *AppCustomInvoicingQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, aciq.ctx, ent.OpQueryExist)
	switch _, err := aciq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (aciq *AppCustomInvoicingQuery) ExistX(ctx context.Context) bool {
	exist, err := aciq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the AppCustomInvoicingQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (aciq *AppCustomInvoicingQuery) Clone() *AppCustomInvoicingQuery {
	if aciq == nil {
		return nil
	}
	return &AppCustomInvoicingQuery{
		config:           aciq.config,
		ctx:              aciq.ctx.Clone(),
		order:            append([]appcustominvoicing.OrderOption{}, aciq.order...),
		inters:           append([]Interceptor{}, aciq.inters...),
		predicates:       append([]predicate.AppCustomInvoicing{}, aciq.predicates...),
		withCustomerApps: aciq.withCustomerApps.Clone(),
		withApp:          aciq.withApp.Clone(),
		// clone intermediate query.
		sql:  aciq.sql.Clone(),
		path: aciq.path,
	}
}

// WithCustomerApps tells the query-builder to eager-load the nodes that are connected to
// the "customer_apps" edge. The optional arguments are used to configure the query builder of the edge.
func (aciq *AppCustomInvoicingQuery) WithCustomerApps(opts ...func(*AppCustomInvoicingCustomerQuery)) *AppCustomInvoicingQuery {
	query := (&AppCustomInvoicingCustomerClient{config: aciq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	aciq.withCustomerApps = query
	return aciq
}

// WithApp tells the query-builder to eager-load the nodes that are connected to
// the "app" edge. The optional arguments are used to configure the query builder of the edge.
func (aciq *AppCustomInvoicingQuery) WithApp(opts ...func(*AppQuery)) *AppCustomInvoicingQuery {
	query := (&AppClient{config: aciq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	aciq.withApp = query
	return aciq
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
//	client.AppCustomInvoicing.Query().
//		GroupBy(appcustominvoicing.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (aciq *AppCustomInvoicingQuery) GroupBy(field string, fields ...string) *AppCustomInvoicingGroupBy {
	aciq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &AppCustomInvoicingGroupBy{build: aciq}
	grbuild.flds = &aciq.ctx.Fields
	grbuild.label = appcustominvoicing.Label
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
//	client.AppCustomInvoicing.Query().
//		Select(appcustominvoicing.FieldNamespace).
//		Scan(ctx, &v)
func (aciq *AppCustomInvoicingQuery) Select(fields ...string) *AppCustomInvoicingSelect {
	aciq.ctx.Fields = append(aciq.ctx.Fields, fields...)
	sbuild := &AppCustomInvoicingSelect{AppCustomInvoicingQuery: aciq}
	sbuild.label = appcustominvoicing.Label
	sbuild.flds, sbuild.scan = &aciq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a AppCustomInvoicingSelect configured with the given aggregations.
func (aciq *AppCustomInvoicingQuery) Aggregate(fns ...AggregateFunc) *AppCustomInvoicingSelect {
	return aciq.Select().Aggregate(fns...)
}

func (aciq *AppCustomInvoicingQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range aciq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, aciq); err != nil {
				return err
			}
		}
	}
	for _, f := range aciq.ctx.Fields {
		if !appcustominvoicing.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if aciq.path != nil {
		prev, err := aciq.path(ctx)
		if err != nil {
			return err
		}
		aciq.sql = prev
	}
	return nil
}

func (aciq *AppCustomInvoicingQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*AppCustomInvoicing, error) {
	var (
		nodes       = []*AppCustomInvoicing{}
		_spec       = aciq.querySpec()
		loadedTypes = [2]bool{
			aciq.withCustomerApps != nil,
			aciq.withApp != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*AppCustomInvoicing).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &AppCustomInvoicing{config: aciq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(aciq.modifiers) > 0 {
		_spec.Modifiers = aciq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, aciq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := aciq.withCustomerApps; query != nil {
		if err := aciq.loadCustomerApps(ctx, query, nodes,
			func(n *AppCustomInvoicing) { n.Edges.CustomerApps = []*AppCustomInvoicingCustomer{} },
			func(n *AppCustomInvoicing, e *AppCustomInvoicingCustomer) {
				n.Edges.CustomerApps = append(n.Edges.CustomerApps, e)
			}); err != nil {
			return nil, err
		}
	}
	if query := aciq.withApp; query != nil {
		if err := aciq.loadApp(ctx, query, nodes, nil,
			func(n *AppCustomInvoicing, e *App) { n.Edges.App = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (aciq *AppCustomInvoicingQuery) loadCustomerApps(ctx context.Context, query *AppCustomInvoicingCustomerQuery, nodes []*AppCustomInvoicing, init func(*AppCustomInvoicing), assign func(*AppCustomInvoicing, *AppCustomInvoicingCustomer)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*AppCustomInvoicing)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(appcustominvoicingcustomer.FieldAppID)
	}
	query.Where(predicate.AppCustomInvoicingCustomer(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(appcustominvoicing.CustomerAppsColumn), fks...))
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
func (aciq *AppCustomInvoicingQuery) loadApp(ctx context.Context, query *AppQuery, nodes []*AppCustomInvoicing, init func(*AppCustomInvoicing), assign func(*AppCustomInvoicing, *App)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*AppCustomInvoicing)
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

func (aciq *AppCustomInvoicingQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := aciq.querySpec()
	if len(aciq.modifiers) > 0 {
		_spec.Modifiers = aciq.modifiers
	}
	_spec.Node.Columns = aciq.ctx.Fields
	if len(aciq.ctx.Fields) > 0 {
		_spec.Unique = aciq.ctx.Unique != nil && *aciq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, aciq.driver, _spec)
}

func (aciq *AppCustomInvoicingQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(appcustominvoicing.Table, appcustominvoicing.Columns, sqlgraph.NewFieldSpec(appcustominvoicing.FieldID, field.TypeString))
	_spec.From = aciq.sql
	if unique := aciq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if aciq.path != nil {
		_spec.Unique = true
	}
	if fields := aciq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, appcustominvoicing.FieldID)
		for i := range fields {
			if fields[i] != appcustominvoicing.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := aciq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := aciq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := aciq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := aciq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (aciq *AppCustomInvoicingQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(aciq.driver.Dialect())
	t1 := builder.Table(appcustominvoicing.Table)
	columns := aciq.ctx.Fields
	if len(columns) == 0 {
		columns = appcustominvoicing.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if aciq.sql != nil {
		selector = aciq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if aciq.ctx.Unique != nil && *aciq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range aciq.modifiers {
		m(selector)
	}
	for _, p := range aciq.predicates {
		p(selector)
	}
	for _, p := range aciq.order {
		p(selector)
	}
	if offset := aciq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := aciq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (aciq *AppCustomInvoicingQuery) ForUpdate(opts ...sql.LockOption) *AppCustomInvoicingQuery {
	if aciq.driver.Dialect() == dialect.Postgres {
		aciq.Unique(false)
	}
	aciq.modifiers = append(aciq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return aciq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (aciq *AppCustomInvoicingQuery) ForShare(opts ...sql.LockOption) *AppCustomInvoicingQuery {
	if aciq.driver.Dialect() == dialect.Postgres {
		aciq.Unique(false)
	}
	aciq.modifiers = append(aciq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return aciq
}

// AppCustomInvoicingGroupBy is the group-by builder for AppCustomInvoicing entities.
type AppCustomInvoicingGroupBy struct {
	selector
	build *AppCustomInvoicingQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (acigb *AppCustomInvoicingGroupBy) Aggregate(fns ...AggregateFunc) *AppCustomInvoicingGroupBy {
	acigb.fns = append(acigb.fns, fns...)
	return acigb
}

// Scan applies the selector query and scans the result into the given value.
func (acigb *AppCustomInvoicingGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, acigb.build.ctx, ent.OpQueryGroupBy)
	if err := acigb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*AppCustomInvoicingQuery, *AppCustomInvoicingGroupBy](ctx, acigb.build, acigb, acigb.build.inters, v)
}

func (acigb *AppCustomInvoicingGroupBy) sqlScan(ctx context.Context, root *AppCustomInvoicingQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(acigb.fns))
	for _, fn := range acigb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*acigb.flds)+len(acigb.fns))
		for _, f := range *acigb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*acigb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := acigb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// AppCustomInvoicingSelect is the builder for selecting fields of AppCustomInvoicing entities.
type AppCustomInvoicingSelect struct {
	*AppCustomInvoicingQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (acis *AppCustomInvoicingSelect) Aggregate(fns ...AggregateFunc) *AppCustomInvoicingSelect {
	acis.fns = append(acis.fns, fns...)
	return acis
}

// Scan applies the selector query and scans the result into the given value.
func (acis *AppCustomInvoicingSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, acis.ctx, ent.OpQuerySelect)
	if err := acis.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*AppCustomInvoicingQuery, *AppCustomInvoicingSelect](ctx, acis.AppCustomInvoicingQuery, acis, acis.inters, v)
}

func (acis *AppCustomInvoicingSelect) sqlScan(ctx context.Context, root *AppCustomInvoicingQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(acis.fns))
	for _, fn := range acis.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*acis.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := acis.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
