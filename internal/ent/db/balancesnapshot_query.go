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
	"github.com/openmeterio/openmeter/internal/ent/db/balancesnapshot"
	"github.com/openmeterio/openmeter/internal/ent/db/predicate"
)

// BalanceSnapshotQuery is the builder for querying BalanceSnapshot entities.
type BalanceSnapshotQuery struct {
	config
	ctx        *QueryContext
	order      []balancesnapshot.OrderOption
	inters     []Interceptor
	predicates []predicate.BalanceSnapshot
	modifiers  []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the BalanceSnapshotQuery builder.
func (bsq *BalanceSnapshotQuery) Where(ps ...predicate.BalanceSnapshot) *BalanceSnapshotQuery {
	bsq.predicates = append(bsq.predicates, ps...)
	return bsq
}

// Limit the number of records to be returned by this query.
func (bsq *BalanceSnapshotQuery) Limit(limit int) *BalanceSnapshotQuery {
	bsq.ctx.Limit = &limit
	return bsq
}

// Offset to start from.
func (bsq *BalanceSnapshotQuery) Offset(offset int) *BalanceSnapshotQuery {
	bsq.ctx.Offset = &offset
	return bsq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (bsq *BalanceSnapshotQuery) Unique(unique bool) *BalanceSnapshotQuery {
	bsq.ctx.Unique = &unique
	return bsq
}

// Order specifies how the records should be ordered.
func (bsq *BalanceSnapshotQuery) Order(o ...balancesnapshot.OrderOption) *BalanceSnapshotQuery {
	bsq.order = append(bsq.order, o...)
	return bsq
}

// First returns the first BalanceSnapshot entity from the query.
// Returns a *NotFoundError when no BalanceSnapshot was found.
func (bsq *BalanceSnapshotQuery) First(ctx context.Context) (*BalanceSnapshot, error) {
	nodes, err := bsq.Limit(1).All(setContextOp(ctx, bsq.ctx, "First"))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{balancesnapshot.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (bsq *BalanceSnapshotQuery) FirstX(ctx context.Context) *BalanceSnapshot {
	node, err := bsq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first BalanceSnapshot ID from the query.
// Returns a *NotFoundError when no BalanceSnapshot ID was found.
func (bsq *BalanceSnapshotQuery) FirstID(ctx context.Context) (id int, err error) {
	var ids []int
	if ids, err = bsq.Limit(1).IDs(setContextOp(ctx, bsq.ctx, "FirstID")); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{balancesnapshot.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (bsq *BalanceSnapshotQuery) FirstIDX(ctx context.Context) int {
	id, err := bsq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single BalanceSnapshot entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one BalanceSnapshot entity is found.
// Returns a *NotFoundError when no BalanceSnapshot entities are found.
func (bsq *BalanceSnapshotQuery) Only(ctx context.Context) (*BalanceSnapshot, error) {
	nodes, err := bsq.Limit(2).All(setContextOp(ctx, bsq.ctx, "Only"))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{balancesnapshot.Label}
	default:
		return nil, &NotSingularError{balancesnapshot.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (bsq *BalanceSnapshotQuery) OnlyX(ctx context.Context) *BalanceSnapshot {
	node, err := bsq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only BalanceSnapshot ID in the query.
// Returns a *NotSingularError when more than one BalanceSnapshot ID is found.
// Returns a *NotFoundError when no entities are found.
func (bsq *BalanceSnapshotQuery) OnlyID(ctx context.Context) (id int, err error) {
	var ids []int
	if ids, err = bsq.Limit(2).IDs(setContextOp(ctx, bsq.ctx, "OnlyID")); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{balancesnapshot.Label}
	default:
		err = &NotSingularError{balancesnapshot.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (bsq *BalanceSnapshotQuery) OnlyIDX(ctx context.Context) int {
	id, err := bsq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of BalanceSnapshots.
func (bsq *BalanceSnapshotQuery) All(ctx context.Context) ([]*BalanceSnapshot, error) {
	ctx = setContextOp(ctx, bsq.ctx, "All")
	if err := bsq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*BalanceSnapshot, *BalanceSnapshotQuery]()
	return withInterceptors[[]*BalanceSnapshot](ctx, bsq, qr, bsq.inters)
}

// AllX is like All, but panics if an error occurs.
func (bsq *BalanceSnapshotQuery) AllX(ctx context.Context) []*BalanceSnapshot {
	nodes, err := bsq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of BalanceSnapshot IDs.
func (bsq *BalanceSnapshotQuery) IDs(ctx context.Context) (ids []int, err error) {
	if bsq.ctx.Unique == nil && bsq.path != nil {
		bsq.Unique(true)
	}
	ctx = setContextOp(ctx, bsq.ctx, "IDs")
	if err = bsq.Select(balancesnapshot.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (bsq *BalanceSnapshotQuery) IDsX(ctx context.Context) []int {
	ids, err := bsq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (bsq *BalanceSnapshotQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, bsq.ctx, "Count")
	if err := bsq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, bsq, querierCount[*BalanceSnapshotQuery](), bsq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (bsq *BalanceSnapshotQuery) CountX(ctx context.Context) int {
	count, err := bsq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (bsq *BalanceSnapshotQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, bsq.ctx, "Exist")
	switch _, err := bsq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (bsq *BalanceSnapshotQuery) ExistX(ctx context.Context) bool {
	exist, err := bsq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the BalanceSnapshotQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (bsq *BalanceSnapshotQuery) Clone() *BalanceSnapshotQuery {
	if bsq == nil {
		return nil
	}
	return &BalanceSnapshotQuery{
		config:     bsq.config,
		ctx:        bsq.ctx.Clone(),
		order:      append([]balancesnapshot.OrderOption{}, bsq.order...),
		inters:     append([]Interceptor{}, bsq.inters...),
		predicates: append([]predicate.BalanceSnapshot{}, bsq.predicates...),
		// clone intermediate query.
		sql:  bsq.sql.Clone(),
		path: bsq.path,
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
//	client.BalanceSnapshot.Query().
//		GroupBy(balancesnapshot.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (bsq *BalanceSnapshotQuery) GroupBy(field string, fields ...string) *BalanceSnapshotGroupBy {
	bsq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &BalanceSnapshotGroupBy{build: bsq}
	grbuild.flds = &bsq.ctx.Fields
	grbuild.label = balancesnapshot.Label
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
//	client.BalanceSnapshot.Query().
//		Select(balancesnapshot.FieldNamespace).
//		Scan(ctx, &v)
func (bsq *BalanceSnapshotQuery) Select(fields ...string) *BalanceSnapshotSelect {
	bsq.ctx.Fields = append(bsq.ctx.Fields, fields...)
	sbuild := &BalanceSnapshotSelect{BalanceSnapshotQuery: bsq}
	sbuild.label = balancesnapshot.Label
	sbuild.flds, sbuild.scan = &bsq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a BalanceSnapshotSelect configured with the given aggregations.
func (bsq *BalanceSnapshotQuery) Aggregate(fns ...AggregateFunc) *BalanceSnapshotSelect {
	return bsq.Select().Aggregate(fns...)
}

func (bsq *BalanceSnapshotQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range bsq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, bsq); err != nil {
				return err
			}
		}
	}
	for _, f := range bsq.ctx.Fields {
		if !balancesnapshot.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if bsq.path != nil {
		prev, err := bsq.path(ctx)
		if err != nil {
			return err
		}
		bsq.sql = prev
	}
	return nil
}

func (bsq *BalanceSnapshotQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*BalanceSnapshot, error) {
	var (
		nodes = []*BalanceSnapshot{}
		_spec = bsq.querySpec()
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*BalanceSnapshot).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &BalanceSnapshot{config: bsq.config}
		nodes = append(nodes, node)
		return node.assignValues(columns, values)
	}
	if len(bsq.modifiers) > 0 {
		_spec.Modifiers = bsq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, bsq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	return nodes, nil
}

func (bsq *BalanceSnapshotQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := bsq.querySpec()
	if len(bsq.modifiers) > 0 {
		_spec.Modifiers = bsq.modifiers
	}
	_spec.Node.Columns = bsq.ctx.Fields
	if len(bsq.ctx.Fields) > 0 {
		_spec.Unique = bsq.ctx.Unique != nil && *bsq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, bsq.driver, _spec)
}

func (bsq *BalanceSnapshotQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(balancesnapshot.Table, balancesnapshot.Columns, sqlgraph.NewFieldSpec(balancesnapshot.FieldID, field.TypeInt))
	_spec.From = bsq.sql
	if unique := bsq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if bsq.path != nil {
		_spec.Unique = true
	}
	if fields := bsq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, balancesnapshot.FieldID)
		for i := range fields {
			if fields[i] != balancesnapshot.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := bsq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := bsq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := bsq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := bsq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (bsq *BalanceSnapshotQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(bsq.driver.Dialect())
	t1 := builder.Table(balancesnapshot.Table)
	columns := bsq.ctx.Fields
	if len(columns) == 0 {
		columns = balancesnapshot.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if bsq.sql != nil {
		selector = bsq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if bsq.ctx.Unique != nil && *bsq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range bsq.modifiers {
		m(selector)
	}
	for _, p := range bsq.predicates {
		p(selector)
	}
	for _, p := range bsq.order {
		p(selector)
	}
	if offset := bsq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := bsq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (bsq *BalanceSnapshotQuery) ForUpdate(opts ...sql.LockOption) *BalanceSnapshotQuery {
	if bsq.driver.Dialect() == dialect.Postgres {
		bsq.Unique(false)
	}
	bsq.modifiers = append(bsq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return bsq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (bsq *BalanceSnapshotQuery) ForShare(opts ...sql.LockOption) *BalanceSnapshotQuery {
	if bsq.driver.Dialect() == dialect.Postgres {
		bsq.Unique(false)
	}
	bsq.modifiers = append(bsq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return bsq
}

// BalanceSnapshotGroupBy is the group-by builder for BalanceSnapshot entities.
type BalanceSnapshotGroupBy struct {
	selector
	build *BalanceSnapshotQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (bsgb *BalanceSnapshotGroupBy) Aggregate(fns ...AggregateFunc) *BalanceSnapshotGroupBy {
	bsgb.fns = append(bsgb.fns, fns...)
	return bsgb
}

// Scan applies the selector query and scans the result into the given value.
func (bsgb *BalanceSnapshotGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, bsgb.build.ctx, "GroupBy")
	if err := bsgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BalanceSnapshotQuery, *BalanceSnapshotGroupBy](ctx, bsgb.build, bsgb, bsgb.build.inters, v)
}

func (bsgb *BalanceSnapshotGroupBy) sqlScan(ctx context.Context, root *BalanceSnapshotQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(bsgb.fns))
	for _, fn := range bsgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*bsgb.flds)+len(bsgb.fns))
		for _, f := range *bsgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*bsgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := bsgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// BalanceSnapshotSelect is the builder for selecting fields of BalanceSnapshot entities.
type BalanceSnapshotSelect struct {
	*BalanceSnapshotQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (bss *BalanceSnapshotSelect) Aggregate(fns ...AggregateFunc) *BalanceSnapshotSelect {
	bss.fns = append(bss.fns, fns...)
	return bss
}

// Scan applies the selector query and scans the result into the given value.
func (bss *BalanceSnapshotSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, bss.ctx, "Select")
	if err := bss.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BalanceSnapshotQuery, *BalanceSnapshotSelect](ctx, bss.BalanceSnapshotQuery, bss, bss.inters, v)
}

func (bss *BalanceSnapshotSelect) sqlScan(ctx context.Context, root *BalanceSnapshotQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(bss.fns))
	for _, fn := range bss.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*bss.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := bss.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
