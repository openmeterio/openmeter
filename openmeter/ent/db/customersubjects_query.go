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
	"github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// CustomerSubjectsQuery is the builder for querying CustomerSubjects entities.
type CustomerSubjectsQuery struct {
	config
	ctx          *QueryContext
	order        []customersubjects.OrderOption
	inters       []Interceptor
	predicates   []predicate.CustomerSubjects
	withCustomer *CustomerQuery
	modifiers    []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the CustomerSubjectsQuery builder.
func (csq *CustomerSubjectsQuery) Where(ps ...predicate.CustomerSubjects) *CustomerSubjectsQuery {
	csq.predicates = append(csq.predicates, ps...)
	return csq
}

// Limit the number of records to be returned by this query.
func (csq *CustomerSubjectsQuery) Limit(limit int) *CustomerSubjectsQuery {
	csq.ctx.Limit = &limit
	return csq
}

// Offset to start from.
func (csq *CustomerSubjectsQuery) Offset(offset int) *CustomerSubjectsQuery {
	csq.ctx.Offset = &offset
	return csq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (csq *CustomerSubjectsQuery) Unique(unique bool) *CustomerSubjectsQuery {
	csq.ctx.Unique = &unique
	return csq
}

// Order specifies how the records should be ordered.
func (csq *CustomerSubjectsQuery) Order(o ...customersubjects.OrderOption) *CustomerSubjectsQuery {
	csq.order = append(csq.order, o...)
	return csq
}

// QueryCustomer chains the current query on the "customer" edge.
func (csq *CustomerSubjectsQuery) QueryCustomer() *CustomerQuery {
	query := (&CustomerClient{config: csq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := csq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := csq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(customersubjects.Table, customersubjects.FieldID, selector),
			sqlgraph.To(customer.Table, customer.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, customersubjects.CustomerTable, customersubjects.CustomerColumn),
		)
		fromU = sqlgraph.SetNeighbors(csq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first CustomerSubjects entity from the query.
// Returns a *NotFoundError when no CustomerSubjects was found.
func (csq *CustomerSubjectsQuery) First(ctx context.Context) (*CustomerSubjects, error) {
	nodes, err := csq.Limit(1).All(setContextOp(ctx, csq.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{customersubjects.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (csq *CustomerSubjectsQuery) FirstX(ctx context.Context) *CustomerSubjects {
	node, err := csq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first CustomerSubjects ID from the query.
// Returns a *NotFoundError when no CustomerSubjects ID was found.
func (csq *CustomerSubjectsQuery) FirstID(ctx context.Context) (id int, err error) {
	var ids []int
	if ids, err = csq.Limit(1).IDs(setContextOp(ctx, csq.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{customersubjects.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (csq *CustomerSubjectsQuery) FirstIDX(ctx context.Context) int {
	id, err := csq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single CustomerSubjects entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one CustomerSubjects entity is found.
// Returns a *NotFoundError when no CustomerSubjects entities are found.
func (csq *CustomerSubjectsQuery) Only(ctx context.Context) (*CustomerSubjects, error) {
	nodes, err := csq.Limit(2).All(setContextOp(ctx, csq.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{customersubjects.Label}
	default:
		return nil, &NotSingularError{customersubjects.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (csq *CustomerSubjectsQuery) OnlyX(ctx context.Context) *CustomerSubjects {
	node, err := csq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only CustomerSubjects ID in the query.
// Returns a *NotSingularError when more than one CustomerSubjects ID is found.
// Returns a *NotFoundError when no entities are found.
func (csq *CustomerSubjectsQuery) OnlyID(ctx context.Context) (id int, err error) {
	var ids []int
	if ids, err = csq.Limit(2).IDs(setContextOp(ctx, csq.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{customersubjects.Label}
	default:
		err = &NotSingularError{customersubjects.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (csq *CustomerSubjectsQuery) OnlyIDX(ctx context.Context) int {
	id, err := csq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of CustomerSubjectsSlice.
func (csq *CustomerSubjectsQuery) All(ctx context.Context) ([]*CustomerSubjects, error) {
	ctx = setContextOp(ctx, csq.ctx, ent.OpQueryAll)
	if err := csq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*CustomerSubjects, *CustomerSubjectsQuery]()
	return withInterceptors[[]*CustomerSubjects](ctx, csq, qr, csq.inters)
}

// AllX is like All, but panics if an error occurs.
func (csq *CustomerSubjectsQuery) AllX(ctx context.Context) []*CustomerSubjects {
	nodes, err := csq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of CustomerSubjects IDs.
func (csq *CustomerSubjectsQuery) IDs(ctx context.Context) (ids []int, err error) {
	if csq.ctx.Unique == nil && csq.path != nil {
		csq.Unique(true)
	}
	ctx = setContextOp(ctx, csq.ctx, ent.OpQueryIDs)
	if err = csq.Select(customersubjects.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (csq *CustomerSubjectsQuery) IDsX(ctx context.Context) []int {
	ids, err := csq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (csq *CustomerSubjectsQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, csq.ctx, ent.OpQueryCount)
	if err := csq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, csq, querierCount[*CustomerSubjectsQuery](), csq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (csq *CustomerSubjectsQuery) CountX(ctx context.Context) int {
	count, err := csq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (csq *CustomerSubjectsQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, csq.ctx, ent.OpQueryExist)
	switch _, err := csq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (csq *CustomerSubjectsQuery) ExistX(ctx context.Context) bool {
	exist, err := csq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the CustomerSubjectsQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (csq *CustomerSubjectsQuery) Clone() *CustomerSubjectsQuery {
	if csq == nil {
		return nil
	}
	return &CustomerSubjectsQuery{
		config:       csq.config,
		ctx:          csq.ctx.Clone(),
		order:        append([]customersubjects.OrderOption{}, csq.order...),
		inters:       append([]Interceptor{}, csq.inters...),
		predicates:   append([]predicate.CustomerSubjects{}, csq.predicates...),
		withCustomer: csq.withCustomer.Clone(),
		// clone intermediate query.
		sql:  csq.sql.Clone(),
		path: csq.path,
	}
}

// WithCustomer tells the query-builder to eager-load the nodes that are connected to
// the "customer" edge. The optional arguments are used to configure the query builder of the edge.
func (csq *CustomerSubjectsQuery) WithCustomer(opts ...func(*CustomerQuery)) *CustomerSubjectsQuery {
	query := (&CustomerClient{config: csq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	csq.withCustomer = query
	return csq
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
//	client.CustomerSubjects.Query().
//		GroupBy(customersubjects.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (csq *CustomerSubjectsQuery) GroupBy(field string, fields ...string) *CustomerSubjectsGroupBy {
	csq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &CustomerSubjectsGroupBy{build: csq}
	grbuild.flds = &csq.ctx.Fields
	grbuild.label = customersubjects.Label
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
//	client.CustomerSubjects.Query().
//		Select(customersubjects.FieldNamespace).
//		Scan(ctx, &v)
func (csq *CustomerSubjectsQuery) Select(fields ...string) *CustomerSubjectsSelect {
	csq.ctx.Fields = append(csq.ctx.Fields, fields...)
	sbuild := &CustomerSubjectsSelect{CustomerSubjectsQuery: csq}
	sbuild.label = customersubjects.Label
	sbuild.flds, sbuild.scan = &csq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a CustomerSubjectsSelect configured with the given aggregations.
func (csq *CustomerSubjectsQuery) Aggregate(fns ...AggregateFunc) *CustomerSubjectsSelect {
	return csq.Select().Aggregate(fns...)
}

func (csq *CustomerSubjectsQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range csq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, csq); err != nil {
				return err
			}
		}
	}
	for _, f := range csq.ctx.Fields {
		if !customersubjects.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if csq.path != nil {
		prev, err := csq.path(ctx)
		if err != nil {
			return err
		}
		csq.sql = prev
	}
	return nil
}

func (csq *CustomerSubjectsQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*CustomerSubjects, error) {
	var (
		nodes       = []*CustomerSubjects{}
		_spec       = csq.querySpec()
		loadedTypes = [1]bool{
			csq.withCustomer != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*CustomerSubjects).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &CustomerSubjects{config: csq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(csq.modifiers) > 0 {
		_spec.Modifiers = csq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, csq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := csq.withCustomer; query != nil {
		if err := csq.loadCustomer(ctx, query, nodes, nil,
			func(n *CustomerSubjects, e *Customer) { n.Edges.Customer = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (csq *CustomerSubjectsQuery) loadCustomer(ctx context.Context, query *CustomerQuery, nodes []*CustomerSubjects, init func(*CustomerSubjects), assign func(*CustomerSubjects, *Customer)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*CustomerSubjects)
	for i := range nodes {
		fk := nodes[i].CustomerID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(customer.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "customer_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (csq *CustomerSubjectsQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := csq.querySpec()
	if len(csq.modifiers) > 0 {
		_spec.Modifiers = csq.modifiers
	}
	_spec.Node.Columns = csq.ctx.Fields
	if len(csq.ctx.Fields) > 0 {
		_spec.Unique = csq.ctx.Unique != nil && *csq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, csq.driver, _spec)
}

func (csq *CustomerSubjectsQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(customersubjects.Table, customersubjects.Columns, sqlgraph.NewFieldSpec(customersubjects.FieldID, field.TypeInt))
	_spec.From = csq.sql
	if unique := csq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if csq.path != nil {
		_spec.Unique = true
	}
	if fields := csq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, customersubjects.FieldID)
		for i := range fields {
			if fields[i] != customersubjects.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if csq.withCustomer != nil {
			_spec.Node.AddColumnOnce(customersubjects.FieldCustomerID)
		}
	}
	if ps := csq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := csq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := csq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := csq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (csq *CustomerSubjectsQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(csq.driver.Dialect())
	t1 := builder.Table(customersubjects.Table)
	columns := csq.ctx.Fields
	if len(columns) == 0 {
		columns = customersubjects.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if csq.sql != nil {
		selector = csq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if csq.ctx.Unique != nil && *csq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range csq.modifiers {
		m(selector)
	}
	for _, p := range csq.predicates {
		p(selector)
	}
	for _, p := range csq.order {
		p(selector)
	}
	if offset := csq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := csq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (csq *CustomerSubjectsQuery) ForUpdate(opts ...sql.LockOption) *CustomerSubjectsQuery {
	if csq.driver.Dialect() == dialect.Postgres {
		csq.Unique(false)
	}
	csq.modifiers = append(csq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return csq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (csq *CustomerSubjectsQuery) ForShare(opts ...sql.LockOption) *CustomerSubjectsQuery {
	if csq.driver.Dialect() == dialect.Postgres {
		csq.Unique(false)
	}
	csq.modifiers = append(csq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return csq
}

// CustomerSubjectsGroupBy is the group-by builder for CustomerSubjects entities.
type CustomerSubjectsGroupBy struct {
	selector
	build *CustomerSubjectsQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (csgb *CustomerSubjectsGroupBy) Aggregate(fns ...AggregateFunc) *CustomerSubjectsGroupBy {
	csgb.fns = append(csgb.fns, fns...)
	return csgb
}

// Scan applies the selector query and scans the result into the given value.
func (csgb *CustomerSubjectsGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, csgb.build.ctx, ent.OpQueryGroupBy)
	if err := csgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*CustomerSubjectsQuery, *CustomerSubjectsGroupBy](ctx, csgb.build, csgb, csgb.build.inters, v)
}

func (csgb *CustomerSubjectsGroupBy) sqlScan(ctx context.Context, root *CustomerSubjectsQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(csgb.fns))
	for _, fn := range csgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*csgb.flds)+len(csgb.fns))
		for _, f := range *csgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*csgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := csgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// CustomerSubjectsSelect is the builder for selecting fields of CustomerSubjects entities.
type CustomerSubjectsSelect struct {
	*CustomerSubjectsQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (css *CustomerSubjectsSelect) Aggregate(fns ...AggregateFunc) *CustomerSubjectsSelect {
	css.fns = append(css.fns, fns...)
	return css
}

// Scan applies the selector query and scans the result into the given value.
func (css *CustomerSubjectsSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, css.ctx, ent.OpQuerySelect)
	if err := css.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*CustomerSubjectsQuery, *CustomerSubjectsSelect](ctx, css.CustomerSubjectsQuery, css, css.inters, v)
}

func (css *CustomerSubjectsSelect) sqlScan(ctx context.Context, root *CustomerSubjectsQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(css.fns))
	for _, fn := range css.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*css.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := css.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
