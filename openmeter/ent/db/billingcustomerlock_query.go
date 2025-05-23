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
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomerlock"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// BillingCustomerLockQuery is the builder for querying BillingCustomerLock entities.
type BillingCustomerLockQuery struct {
	config
	ctx        *QueryContext
	order      []billingcustomerlock.OrderOption
	inters     []Interceptor
	predicates []predicate.BillingCustomerLock
	modifiers  []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the BillingCustomerLockQuery builder.
func (_q *BillingCustomerLockQuery) Where(ps ...predicate.BillingCustomerLock) *BillingCustomerLockQuery {
	_q.predicates = append(_q.predicates, ps...)
	return _q
}

// Limit the number of records to be returned by this query.
func (_q *BillingCustomerLockQuery) Limit(limit int) *BillingCustomerLockQuery {
	_q.ctx.Limit = &limit
	return _q
}

// Offset to start from.
func (_q *BillingCustomerLockQuery) Offset(offset int) *BillingCustomerLockQuery {
	_q.ctx.Offset = &offset
	return _q
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (_q *BillingCustomerLockQuery) Unique(unique bool) *BillingCustomerLockQuery {
	_q.ctx.Unique = &unique
	return _q
}

// Order specifies how the records should be ordered.
func (_q *BillingCustomerLockQuery) Order(o ...billingcustomerlock.OrderOption) *BillingCustomerLockQuery {
	_q.order = append(_q.order, o...)
	return _q
}

// First returns the first BillingCustomerLock entity from the query.
// Returns a *NotFoundError when no BillingCustomerLock was found.
func (_q *BillingCustomerLockQuery) First(ctx context.Context) (*BillingCustomerLock, error) {
	nodes, err := _q.Limit(1).All(setContextOp(ctx, _q.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{billingcustomerlock.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (_q *BillingCustomerLockQuery) FirstX(ctx context.Context) *BillingCustomerLock {
	node, err := _q.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first BillingCustomerLock ID from the query.
// Returns a *NotFoundError when no BillingCustomerLock ID was found.
func (_q *BillingCustomerLockQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = _q.Limit(1).IDs(setContextOp(ctx, _q.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{billingcustomerlock.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (_q *BillingCustomerLockQuery) FirstIDX(ctx context.Context) string {
	id, err := _q.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single BillingCustomerLock entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one BillingCustomerLock entity is found.
// Returns a *NotFoundError when no BillingCustomerLock entities are found.
func (_q *BillingCustomerLockQuery) Only(ctx context.Context) (*BillingCustomerLock, error) {
	nodes, err := _q.Limit(2).All(setContextOp(ctx, _q.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{billingcustomerlock.Label}
	default:
		return nil, &NotSingularError{billingcustomerlock.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (_q *BillingCustomerLockQuery) OnlyX(ctx context.Context) *BillingCustomerLock {
	node, err := _q.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only BillingCustomerLock ID in the query.
// Returns a *NotSingularError when more than one BillingCustomerLock ID is found.
// Returns a *NotFoundError when no entities are found.
func (_q *BillingCustomerLockQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = _q.Limit(2).IDs(setContextOp(ctx, _q.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{billingcustomerlock.Label}
	default:
		err = &NotSingularError{billingcustomerlock.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (_q *BillingCustomerLockQuery) OnlyIDX(ctx context.Context) string {
	id, err := _q.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of BillingCustomerLocks.
func (_q *BillingCustomerLockQuery) All(ctx context.Context) ([]*BillingCustomerLock, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryAll)
	if err := _q.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*BillingCustomerLock, *BillingCustomerLockQuery]()
	return withInterceptors[[]*BillingCustomerLock](ctx, _q, qr, _q.inters)
}

// AllX is like All, but panics if an error occurs.
func (_q *BillingCustomerLockQuery) AllX(ctx context.Context) []*BillingCustomerLock {
	nodes, err := _q.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of BillingCustomerLock IDs.
func (_q *BillingCustomerLockQuery) IDs(ctx context.Context) (ids []string, err error) {
	if _q.ctx.Unique == nil && _q.path != nil {
		_q.Unique(true)
	}
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryIDs)
	if err = _q.Select(billingcustomerlock.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (_q *BillingCustomerLockQuery) IDsX(ctx context.Context) []string {
	ids, err := _q.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (_q *BillingCustomerLockQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryCount)
	if err := _q.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, _q, querierCount[*BillingCustomerLockQuery](), _q.inters)
}

// CountX is like Count, but panics if an error occurs.
func (_q *BillingCustomerLockQuery) CountX(ctx context.Context) int {
	count, err := _q.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (_q *BillingCustomerLockQuery) Exist(ctx context.Context) (bool, error) {
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
func (_q *BillingCustomerLockQuery) ExistX(ctx context.Context) bool {
	exist, err := _q.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the BillingCustomerLockQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (_q *BillingCustomerLockQuery) Clone() *BillingCustomerLockQuery {
	if _q == nil {
		return nil
	}
	return &BillingCustomerLockQuery{
		config:     _q.config,
		ctx:        _q.ctx.Clone(),
		order:      append([]billingcustomerlock.OrderOption{}, _q.order...),
		inters:     append([]Interceptor{}, _q.inters...),
		predicates: append([]predicate.BillingCustomerLock{}, _q.predicates...),
		// clone intermediate query.
		sql:  _q.sql.Clone(),
		path: _q.path,
	}
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
//	client.BillingCustomerLock.Query().
//		GroupBy(billingcustomerlock.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (_q *BillingCustomerLockQuery) GroupBy(field string, fields ...string) *BillingCustomerLockGroupBy {
	_q.ctx.Fields = append([]string{field}, fields...)
	grbuild := &BillingCustomerLockGroupBy{build: _q}
	grbuild.flds = &_q.ctx.Fields
	grbuild.label = billingcustomerlock.Label
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
//	client.BillingCustomerLock.Query().
//		Select(billingcustomerlock.FieldNamespace).
//		Scan(ctx, &v)
func (_q *BillingCustomerLockQuery) Select(fields ...string) *BillingCustomerLockSelect {
	_q.ctx.Fields = append(_q.ctx.Fields, fields...)
	sbuild := &BillingCustomerLockSelect{BillingCustomerLockQuery: _q}
	sbuild.label = billingcustomerlock.Label
	sbuild.flds, sbuild.scan = &_q.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a BillingCustomerLockSelect configured with the given aggregations.
func (_q *BillingCustomerLockQuery) Aggregate(fns ...AggregateFunc) *BillingCustomerLockSelect {
	return _q.Select().Aggregate(fns...)
}

func (_q *BillingCustomerLockQuery) prepareQuery(ctx context.Context) error {
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
		if !billingcustomerlock.ValidColumn(f) {
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

func (_q *BillingCustomerLockQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*BillingCustomerLock, error) {
	var (
		nodes = []*BillingCustomerLock{}
		_spec = _q.querySpec()
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*BillingCustomerLock).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &BillingCustomerLock{config: _q.config}
		nodes = append(nodes, node)
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
	return nodes, nil
}

func (_q *BillingCustomerLockQuery) sqlCount(ctx context.Context) (int, error) {
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

func (_q *BillingCustomerLockQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(billingcustomerlock.Table, billingcustomerlock.Columns, sqlgraph.NewFieldSpec(billingcustomerlock.FieldID, field.TypeString))
	_spec.From = _q.sql
	if unique := _q.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if _q.path != nil {
		_spec.Unique = true
	}
	if fields := _q.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, billingcustomerlock.FieldID)
		for i := range fields {
			if fields[i] != billingcustomerlock.FieldID {
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

func (_q *BillingCustomerLockQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(_q.driver.Dialect())
	t1 := builder.Table(billingcustomerlock.Table)
	columns := _q.ctx.Fields
	if len(columns) == 0 {
		columns = billingcustomerlock.Columns
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
func (_q *BillingCustomerLockQuery) ForUpdate(opts ...sql.LockOption) *BillingCustomerLockQuery {
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
func (_q *BillingCustomerLockQuery) ForShare(opts ...sql.LockOption) *BillingCustomerLockQuery {
	if _q.driver.Dialect() == dialect.Postgres {
		_q.Unique(false)
	}
	_q.modifiers = append(_q.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return _q
}

// BillingCustomerLockGroupBy is the group-by builder for BillingCustomerLock entities.
type BillingCustomerLockGroupBy struct {
	selector
	build *BillingCustomerLockQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (bclgb *BillingCustomerLockGroupBy) Aggregate(fns ...AggregateFunc) *BillingCustomerLockGroupBy {
	bclgb.fns = append(bclgb.fns, fns...)
	return bclgb
}

// Scan applies the selector query and scans the result into the given value.
func (bclgb *BillingCustomerLockGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, bclgb.build.ctx, ent.OpQueryGroupBy)
	if err := bclgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BillingCustomerLockQuery, *BillingCustomerLockGroupBy](ctx, bclgb.build, bclgb, bclgb.build.inters, v)
}

func (bclgb *BillingCustomerLockGroupBy) sqlScan(ctx context.Context, root *BillingCustomerLockQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(bclgb.fns))
	for _, fn := range bclgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*bclgb.flds)+len(bclgb.fns))
		for _, f := range *bclgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*bclgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := bclgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// BillingCustomerLockSelect is the builder for selecting fields of BillingCustomerLock entities.
type BillingCustomerLockSelect struct {
	*BillingCustomerLockQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (bcls *BillingCustomerLockSelect) Aggregate(fns ...AggregateFunc) *BillingCustomerLockSelect {
	bcls.fns = append(bcls.fns, fns...)
	return bcls
}

// Scan applies the selector query and scans the result into the given value.
func (bcls *BillingCustomerLockSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, bcls.ctx, ent.OpQuerySelect)
	if err := bcls.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BillingCustomerLockQuery, *BillingCustomerLockSelect](ctx, bcls.BillingCustomerLockQuery, bcls, bcls.inters, v)
}

func (bcls *BillingCustomerLockSelect) sqlScan(ctx context.Context, root *BillingCustomerLockQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(bcls.fns))
	for _, fn := range bcls.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*bcls.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := bcls.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
