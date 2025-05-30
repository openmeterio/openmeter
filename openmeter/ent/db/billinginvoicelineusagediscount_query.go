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
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicelineusagediscount"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// BillingInvoiceLineUsageDiscountQuery is the builder for querying BillingInvoiceLineUsageDiscount entities.
type BillingInvoiceLineUsageDiscountQuery struct {
	config
	ctx                    *QueryContext
	order                  []billinginvoicelineusagediscount.OrderOption
	inters                 []Interceptor
	predicates             []predicate.BillingInvoiceLineUsageDiscount
	withBillingInvoiceLine *BillingInvoiceLineQuery
	modifiers              []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the BillingInvoiceLineUsageDiscountQuery builder.
func (_q *BillingInvoiceLineUsageDiscountQuery) Where(ps ...predicate.BillingInvoiceLineUsageDiscount) *BillingInvoiceLineUsageDiscountQuery {
	_q.predicates = append(_q.predicates, ps...)
	return _q
}

// Limit the number of records to be returned by this query.
func (_q *BillingInvoiceLineUsageDiscountQuery) Limit(limit int) *BillingInvoiceLineUsageDiscountQuery {
	_q.ctx.Limit = &limit
	return _q
}

// Offset to start from.
func (_q *BillingInvoiceLineUsageDiscountQuery) Offset(offset int) *BillingInvoiceLineUsageDiscountQuery {
	_q.ctx.Offset = &offset
	return _q
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (_q *BillingInvoiceLineUsageDiscountQuery) Unique(unique bool) *BillingInvoiceLineUsageDiscountQuery {
	_q.ctx.Unique = &unique
	return _q
}

// Order specifies how the records should be ordered.
func (_q *BillingInvoiceLineUsageDiscountQuery) Order(o ...billinginvoicelineusagediscount.OrderOption) *BillingInvoiceLineUsageDiscountQuery {
	_q.order = append(_q.order, o...)
	return _q
}

// QueryBillingInvoiceLine chains the current query on the "billing_invoice_line" edge.
func (_q *BillingInvoiceLineUsageDiscountQuery) QueryBillingInvoiceLine() *BillingInvoiceLineQuery {
	query := (&BillingInvoiceLineClient{config: _q.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := _q.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := _q.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoicelineusagediscount.Table, billinginvoicelineusagediscount.FieldID, selector),
			sqlgraph.To(billinginvoiceline.Table, billinginvoiceline.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, billinginvoicelineusagediscount.BillingInvoiceLineTable, billinginvoicelineusagediscount.BillingInvoiceLineColumn),
		)
		fromU = sqlgraph.SetNeighbors(_q.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first BillingInvoiceLineUsageDiscount entity from the query.
// Returns a *NotFoundError when no BillingInvoiceLineUsageDiscount was found.
func (_q *BillingInvoiceLineUsageDiscountQuery) First(ctx context.Context) (*BillingInvoiceLineUsageDiscount, error) {
	nodes, err := _q.Limit(1).All(setContextOp(ctx, _q.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{billinginvoicelineusagediscount.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (_q *BillingInvoiceLineUsageDiscountQuery) FirstX(ctx context.Context) *BillingInvoiceLineUsageDiscount {
	node, err := _q.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first BillingInvoiceLineUsageDiscount ID from the query.
// Returns a *NotFoundError when no BillingInvoiceLineUsageDiscount ID was found.
func (_q *BillingInvoiceLineUsageDiscountQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = _q.Limit(1).IDs(setContextOp(ctx, _q.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{billinginvoicelineusagediscount.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (_q *BillingInvoiceLineUsageDiscountQuery) FirstIDX(ctx context.Context) string {
	id, err := _q.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single BillingInvoiceLineUsageDiscount entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one BillingInvoiceLineUsageDiscount entity is found.
// Returns a *NotFoundError when no BillingInvoiceLineUsageDiscount entities are found.
func (_q *BillingInvoiceLineUsageDiscountQuery) Only(ctx context.Context) (*BillingInvoiceLineUsageDiscount, error) {
	nodes, err := _q.Limit(2).All(setContextOp(ctx, _q.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{billinginvoicelineusagediscount.Label}
	default:
		return nil, &NotSingularError{billinginvoicelineusagediscount.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (_q *BillingInvoiceLineUsageDiscountQuery) OnlyX(ctx context.Context) *BillingInvoiceLineUsageDiscount {
	node, err := _q.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only BillingInvoiceLineUsageDiscount ID in the query.
// Returns a *NotSingularError when more than one BillingInvoiceLineUsageDiscount ID is found.
// Returns a *NotFoundError when no entities are found.
func (_q *BillingInvoiceLineUsageDiscountQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = _q.Limit(2).IDs(setContextOp(ctx, _q.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{billinginvoicelineusagediscount.Label}
	default:
		err = &NotSingularError{billinginvoicelineusagediscount.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (_q *BillingInvoiceLineUsageDiscountQuery) OnlyIDX(ctx context.Context) string {
	id, err := _q.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of BillingInvoiceLineUsageDiscounts.
func (_q *BillingInvoiceLineUsageDiscountQuery) All(ctx context.Context) ([]*BillingInvoiceLineUsageDiscount, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryAll)
	if err := _q.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*BillingInvoiceLineUsageDiscount, *BillingInvoiceLineUsageDiscountQuery]()
	return withInterceptors[[]*BillingInvoiceLineUsageDiscount](ctx, _q, qr, _q.inters)
}

// AllX is like All, but panics if an error occurs.
func (_q *BillingInvoiceLineUsageDiscountQuery) AllX(ctx context.Context) []*BillingInvoiceLineUsageDiscount {
	nodes, err := _q.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of BillingInvoiceLineUsageDiscount IDs.
func (_q *BillingInvoiceLineUsageDiscountQuery) IDs(ctx context.Context) (ids []string, err error) {
	if _q.ctx.Unique == nil && _q.path != nil {
		_q.Unique(true)
	}
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryIDs)
	if err = _q.Select(billinginvoicelineusagediscount.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (_q *BillingInvoiceLineUsageDiscountQuery) IDsX(ctx context.Context) []string {
	ids, err := _q.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (_q *BillingInvoiceLineUsageDiscountQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryCount)
	if err := _q.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, _q, querierCount[*BillingInvoiceLineUsageDiscountQuery](), _q.inters)
}

// CountX is like Count, but panics if an error occurs.
func (_q *BillingInvoiceLineUsageDiscountQuery) CountX(ctx context.Context) int {
	count, err := _q.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (_q *BillingInvoiceLineUsageDiscountQuery) Exist(ctx context.Context) (bool, error) {
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
func (_q *BillingInvoiceLineUsageDiscountQuery) ExistX(ctx context.Context) bool {
	exist, err := _q.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the BillingInvoiceLineUsageDiscountQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (_q *BillingInvoiceLineUsageDiscountQuery) Clone() *BillingInvoiceLineUsageDiscountQuery {
	if _q == nil {
		return nil
	}
	return &BillingInvoiceLineUsageDiscountQuery{
		config:                 _q.config,
		ctx:                    _q.ctx.Clone(),
		order:                  append([]billinginvoicelineusagediscount.OrderOption{}, _q.order...),
		inters:                 append([]Interceptor{}, _q.inters...),
		predicates:             append([]predicate.BillingInvoiceLineUsageDiscount{}, _q.predicates...),
		withBillingInvoiceLine: _q.withBillingInvoiceLine.Clone(),
		// clone intermediate query.
		sql:  _q.sql.Clone(),
		path: _q.path,
	}
}

// WithBillingInvoiceLine tells the query-builder to eager-load the nodes that are connected to
// the "billing_invoice_line" edge. The optional arguments are used to configure the query builder of the edge.
func (_q *BillingInvoiceLineUsageDiscountQuery) WithBillingInvoiceLine(opts ...func(*BillingInvoiceLineQuery)) *BillingInvoiceLineUsageDiscountQuery {
	query := (&BillingInvoiceLineClient{config: _q.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	_q.withBillingInvoiceLine = query
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
//	client.BillingInvoiceLineUsageDiscount.Query().
//		GroupBy(billinginvoicelineusagediscount.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (_q *BillingInvoiceLineUsageDiscountQuery) GroupBy(field string, fields ...string) *BillingInvoiceLineUsageDiscountGroupBy {
	_q.ctx.Fields = append([]string{field}, fields...)
	grbuild := &BillingInvoiceLineUsageDiscountGroupBy{build: _q}
	grbuild.flds = &_q.ctx.Fields
	grbuild.label = billinginvoicelineusagediscount.Label
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
//	client.BillingInvoiceLineUsageDiscount.Query().
//		Select(billinginvoicelineusagediscount.FieldNamespace).
//		Scan(ctx, &v)
func (_q *BillingInvoiceLineUsageDiscountQuery) Select(fields ...string) *BillingInvoiceLineUsageDiscountSelect {
	_q.ctx.Fields = append(_q.ctx.Fields, fields...)
	sbuild := &BillingInvoiceLineUsageDiscountSelect{BillingInvoiceLineUsageDiscountQuery: _q}
	sbuild.label = billinginvoicelineusagediscount.Label
	sbuild.flds, sbuild.scan = &_q.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a BillingInvoiceLineUsageDiscountSelect configured with the given aggregations.
func (_q *BillingInvoiceLineUsageDiscountQuery) Aggregate(fns ...AggregateFunc) *BillingInvoiceLineUsageDiscountSelect {
	return _q.Select().Aggregate(fns...)
}

func (_q *BillingInvoiceLineUsageDiscountQuery) prepareQuery(ctx context.Context) error {
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
		if !billinginvoicelineusagediscount.ValidColumn(f) {
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

func (_q *BillingInvoiceLineUsageDiscountQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*BillingInvoiceLineUsageDiscount, error) {
	var (
		nodes       = []*BillingInvoiceLineUsageDiscount{}
		_spec       = _q.querySpec()
		loadedTypes = [1]bool{
			_q.withBillingInvoiceLine != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*BillingInvoiceLineUsageDiscount).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &BillingInvoiceLineUsageDiscount{config: _q.config}
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
	if query := _q.withBillingInvoiceLine; query != nil {
		if err := _q.loadBillingInvoiceLine(ctx, query, nodes, nil,
			func(n *BillingInvoiceLineUsageDiscount, e *BillingInvoiceLine) { n.Edges.BillingInvoiceLine = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (_q *BillingInvoiceLineUsageDiscountQuery) loadBillingInvoiceLine(ctx context.Context, query *BillingInvoiceLineQuery, nodes []*BillingInvoiceLineUsageDiscount, init func(*BillingInvoiceLineUsageDiscount), assign func(*BillingInvoiceLineUsageDiscount, *BillingInvoiceLine)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*BillingInvoiceLineUsageDiscount)
	for i := range nodes {
		fk := nodes[i].LineID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(billinginvoiceline.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "line_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (_q *BillingInvoiceLineUsageDiscountQuery) sqlCount(ctx context.Context) (int, error) {
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

func (_q *BillingInvoiceLineUsageDiscountQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(billinginvoicelineusagediscount.Table, billinginvoicelineusagediscount.Columns, sqlgraph.NewFieldSpec(billinginvoicelineusagediscount.FieldID, field.TypeString))
	_spec.From = _q.sql
	if unique := _q.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if _q.path != nil {
		_spec.Unique = true
	}
	if fields := _q.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, billinginvoicelineusagediscount.FieldID)
		for i := range fields {
			if fields[i] != billinginvoicelineusagediscount.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if _q.withBillingInvoiceLine != nil {
			_spec.Node.AddColumnOnce(billinginvoicelineusagediscount.FieldLineID)
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

func (_q *BillingInvoiceLineUsageDiscountQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(_q.driver.Dialect())
	t1 := builder.Table(billinginvoicelineusagediscount.Table)
	columns := _q.ctx.Fields
	if len(columns) == 0 {
		columns = billinginvoicelineusagediscount.Columns
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
func (_q *BillingInvoiceLineUsageDiscountQuery) ForUpdate(opts ...sql.LockOption) *BillingInvoiceLineUsageDiscountQuery {
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
func (_q *BillingInvoiceLineUsageDiscountQuery) ForShare(opts ...sql.LockOption) *BillingInvoiceLineUsageDiscountQuery {
	if _q.driver.Dialect() == dialect.Postgres {
		_q.Unique(false)
	}
	_q.modifiers = append(_q.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return _q
}

// BillingInvoiceLineUsageDiscountGroupBy is the group-by builder for BillingInvoiceLineUsageDiscount entities.
type BillingInvoiceLineUsageDiscountGroupBy struct {
	selector
	build *BillingInvoiceLineUsageDiscountQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (biludgb *BillingInvoiceLineUsageDiscountGroupBy) Aggregate(fns ...AggregateFunc) *BillingInvoiceLineUsageDiscountGroupBy {
	biludgb.fns = append(biludgb.fns, fns...)
	return biludgb
}

// Scan applies the selector query and scans the result into the given value.
func (biludgb *BillingInvoiceLineUsageDiscountGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, biludgb.build.ctx, ent.OpQueryGroupBy)
	if err := biludgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BillingInvoiceLineUsageDiscountQuery, *BillingInvoiceLineUsageDiscountGroupBy](ctx, biludgb.build, biludgb, biludgb.build.inters, v)
}

func (biludgb *BillingInvoiceLineUsageDiscountGroupBy) sqlScan(ctx context.Context, root *BillingInvoiceLineUsageDiscountQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(biludgb.fns))
	for _, fn := range biludgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*biludgb.flds)+len(biludgb.fns))
		for _, f := range *biludgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*biludgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := biludgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// BillingInvoiceLineUsageDiscountSelect is the builder for selecting fields of BillingInvoiceLineUsageDiscount entities.
type BillingInvoiceLineUsageDiscountSelect struct {
	*BillingInvoiceLineUsageDiscountQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (biluds *BillingInvoiceLineUsageDiscountSelect) Aggregate(fns ...AggregateFunc) *BillingInvoiceLineUsageDiscountSelect {
	biluds.fns = append(biluds.fns, fns...)
	return biluds
}

// Scan applies the selector query and scans the result into the given value.
func (biluds *BillingInvoiceLineUsageDiscountSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, biluds.ctx, ent.OpQuerySelect)
	if err := biluds.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BillingInvoiceLineUsageDiscountQuery, *BillingInvoiceLineUsageDiscountSelect](ctx, biluds.BillingInvoiceLineUsageDiscountQuery, biluds, biluds.inters, v)
}

func (biluds *BillingInvoiceLineUsageDiscountSelect) sqlScan(ctx context.Context, root *BillingInvoiceLineUsageDiscountQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(biluds.fns))
	for _, fn := range biluds.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*biluds.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := biluds.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
