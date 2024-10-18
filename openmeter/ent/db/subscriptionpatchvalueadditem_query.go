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
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionpatchvalueadditem"
)

// SubscriptionPatchValueAddItemQuery is the builder for querying SubscriptionPatchValueAddItem entities.
type SubscriptionPatchValueAddItemQuery struct {
	config
	ctx                   *QueryContext
	order                 []subscriptionpatchvalueadditem.OrderOption
	inters                []Interceptor
	predicates            []predicate.SubscriptionPatchValueAddItem
	withSubscriptionPatch *SubscriptionPatchQuery
	modifiers             []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the SubscriptionPatchValueAddItemQuery builder.
func (spvaiq *SubscriptionPatchValueAddItemQuery) Where(ps ...predicate.SubscriptionPatchValueAddItem) *SubscriptionPatchValueAddItemQuery {
	spvaiq.predicates = append(spvaiq.predicates, ps...)
	return spvaiq
}

// Limit the number of records to be returned by this query.
func (spvaiq *SubscriptionPatchValueAddItemQuery) Limit(limit int) *SubscriptionPatchValueAddItemQuery {
	spvaiq.ctx.Limit = &limit
	return spvaiq
}

// Offset to start from.
func (spvaiq *SubscriptionPatchValueAddItemQuery) Offset(offset int) *SubscriptionPatchValueAddItemQuery {
	spvaiq.ctx.Offset = &offset
	return spvaiq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (spvaiq *SubscriptionPatchValueAddItemQuery) Unique(unique bool) *SubscriptionPatchValueAddItemQuery {
	spvaiq.ctx.Unique = &unique
	return spvaiq
}

// Order specifies how the records should be ordered.
func (spvaiq *SubscriptionPatchValueAddItemQuery) Order(o ...subscriptionpatchvalueadditem.OrderOption) *SubscriptionPatchValueAddItemQuery {
	spvaiq.order = append(spvaiq.order, o...)
	return spvaiq
}

// QuerySubscriptionPatch chains the current query on the "subscription_patch" edge.
func (spvaiq *SubscriptionPatchValueAddItemQuery) QuerySubscriptionPatch() *SubscriptionPatchQuery {
	query := (&SubscriptionPatchClient{config: spvaiq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := spvaiq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := spvaiq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(subscriptionpatchvalueadditem.Table, subscriptionpatchvalueadditem.FieldID, selector),
			sqlgraph.To(subscriptionpatch.Table, subscriptionpatch.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, subscriptionpatchvalueadditem.SubscriptionPatchTable, subscriptionpatchvalueadditem.SubscriptionPatchColumn),
		)
		fromU = sqlgraph.SetNeighbors(spvaiq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first SubscriptionPatchValueAddItem entity from the query.
// Returns a *NotFoundError when no SubscriptionPatchValueAddItem was found.
func (spvaiq *SubscriptionPatchValueAddItemQuery) First(ctx context.Context) (*SubscriptionPatchValueAddItem, error) {
	nodes, err := spvaiq.Limit(1).All(setContextOp(ctx, spvaiq.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{subscriptionpatchvalueadditem.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (spvaiq *SubscriptionPatchValueAddItemQuery) FirstX(ctx context.Context) *SubscriptionPatchValueAddItem {
	node, err := spvaiq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first SubscriptionPatchValueAddItem ID from the query.
// Returns a *NotFoundError when no SubscriptionPatchValueAddItem ID was found.
func (spvaiq *SubscriptionPatchValueAddItemQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = spvaiq.Limit(1).IDs(setContextOp(ctx, spvaiq.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{subscriptionpatchvalueadditem.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (spvaiq *SubscriptionPatchValueAddItemQuery) FirstIDX(ctx context.Context) string {
	id, err := spvaiq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single SubscriptionPatchValueAddItem entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one SubscriptionPatchValueAddItem entity is found.
// Returns a *NotFoundError when no SubscriptionPatchValueAddItem entities are found.
func (spvaiq *SubscriptionPatchValueAddItemQuery) Only(ctx context.Context) (*SubscriptionPatchValueAddItem, error) {
	nodes, err := spvaiq.Limit(2).All(setContextOp(ctx, spvaiq.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{subscriptionpatchvalueadditem.Label}
	default:
		return nil, &NotSingularError{subscriptionpatchvalueadditem.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (spvaiq *SubscriptionPatchValueAddItemQuery) OnlyX(ctx context.Context) *SubscriptionPatchValueAddItem {
	node, err := spvaiq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only SubscriptionPatchValueAddItem ID in the query.
// Returns a *NotSingularError when more than one SubscriptionPatchValueAddItem ID is found.
// Returns a *NotFoundError when no entities are found.
func (spvaiq *SubscriptionPatchValueAddItemQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = spvaiq.Limit(2).IDs(setContextOp(ctx, spvaiq.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{subscriptionpatchvalueadditem.Label}
	default:
		err = &NotSingularError{subscriptionpatchvalueadditem.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (spvaiq *SubscriptionPatchValueAddItemQuery) OnlyIDX(ctx context.Context) string {
	id, err := spvaiq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of SubscriptionPatchValueAddItems.
func (spvaiq *SubscriptionPatchValueAddItemQuery) All(ctx context.Context) ([]*SubscriptionPatchValueAddItem, error) {
	ctx = setContextOp(ctx, spvaiq.ctx, ent.OpQueryAll)
	if err := spvaiq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*SubscriptionPatchValueAddItem, *SubscriptionPatchValueAddItemQuery]()
	return withInterceptors[[]*SubscriptionPatchValueAddItem](ctx, spvaiq, qr, spvaiq.inters)
}

// AllX is like All, but panics if an error occurs.
func (spvaiq *SubscriptionPatchValueAddItemQuery) AllX(ctx context.Context) []*SubscriptionPatchValueAddItem {
	nodes, err := spvaiq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of SubscriptionPatchValueAddItem IDs.
func (spvaiq *SubscriptionPatchValueAddItemQuery) IDs(ctx context.Context) (ids []string, err error) {
	if spvaiq.ctx.Unique == nil && spvaiq.path != nil {
		spvaiq.Unique(true)
	}
	ctx = setContextOp(ctx, spvaiq.ctx, ent.OpQueryIDs)
	if err = spvaiq.Select(subscriptionpatchvalueadditem.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (spvaiq *SubscriptionPatchValueAddItemQuery) IDsX(ctx context.Context) []string {
	ids, err := spvaiq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (spvaiq *SubscriptionPatchValueAddItemQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, spvaiq.ctx, ent.OpQueryCount)
	if err := spvaiq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, spvaiq, querierCount[*SubscriptionPatchValueAddItemQuery](), spvaiq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (spvaiq *SubscriptionPatchValueAddItemQuery) CountX(ctx context.Context) int {
	count, err := spvaiq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (spvaiq *SubscriptionPatchValueAddItemQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, spvaiq.ctx, ent.OpQueryExist)
	switch _, err := spvaiq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (spvaiq *SubscriptionPatchValueAddItemQuery) ExistX(ctx context.Context) bool {
	exist, err := spvaiq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the SubscriptionPatchValueAddItemQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (spvaiq *SubscriptionPatchValueAddItemQuery) Clone() *SubscriptionPatchValueAddItemQuery {
	if spvaiq == nil {
		return nil
	}
	return &SubscriptionPatchValueAddItemQuery{
		config:                spvaiq.config,
		ctx:                   spvaiq.ctx.Clone(),
		order:                 append([]subscriptionpatchvalueadditem.OrderOption{}, spvaiq.order...),
		inters:                append([]Interceptor{}, spvaiq.inters...),
		predicates:            append([]predicate.SubscriptionPatchValueAddItem{}, spvaiq.predicates...),
		withSubscriptionPatch: spvaiq.withSubscriptionPatch.Clone(),
		// clone intermediate query.
		sql:  spvaiq.sql.Clone(),
		path: spvaiq.path,
	}
}

// WithSubscriptionPatch tells the query-builder to eager-load the nodes that are connected to
// the "subscription_patch" edge. The optional arguments are used to configure the query builder of the edge.
func (spvaiq *SubscriptionPatchValueAddItemQuery) WithSubscriptionPatch(opts ...func(*SubscriptionPatchQuery)) *SubscriptionPatchValueAddItemQuery {
	query := (&SubscriptionPatchClient{config: spvaiq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	spvaiq.withSubscriptionPatch = query
	return spvaiq
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
//	client.SubscriptionPatchValueAddItem.Query().
//		GroupBy(subscriptionpatchvalueadditem.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (spvaiq *SubscriptionPatchValueAddItemQuery) GroupBy(field string, fields ...string) *SubscriptionPatchValueAddItemGroupBy {
	spvaiq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &SubscriptionPatchValueAddItemGroupBy{build: spvaiq}
	grbuild.flds = &spvaiq.ctx.Fields
	grbuild.label = subscriptionpatchvalueadditem.Label
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
//	client.SubscriptionPatchValueAddItem.Query().
//		Select(subscriptionpatchvalueadditem.FieldNamespace).
//		Scan(ctx, &v)
func (spvaiq *SubscriptionPatchValueAddItemQuery) Select(fields ...string) *SubscriptionPatchValueAddItemSelect {
	spvaiq.ctx.Fields = append(spvaiq.ctx.Fields, fields...)
	sbuild := &SubscriptionPatchValueAddItemSelect{SubscriptionPatchValueAddItemQuery: spvaiq}
	sbuild.label = subscriptionpatchvalueadditem.Label
	sbuild.flds, sbuild.scan = &spvaiq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a SubscriptionPatchValueAddItemSelect configured with the given aggregations.
func (spvaiq *SubscriptionPatchValueAddItemQuery) Aggregate(fns ...AggregateFunc) *SubscriptionPatchValueAddItemSelect {
	return spvaiq.Select().Aggregate(fns...)
}

func (spvaiq *SubscriptionPatchValueAddItemQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range spvaiq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, spvaiq); err != nil {
				return err
			}
		}
	}
	for _, f := range spvaiq.ctx.Fields {
		if !subscriptionpatchvalueadditem.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if spvaiq.path != nil {
		prev, err := spvaiq.path(ctx)
		if err != nil {
			return err
		}
		spvaiq.sql = prev
	}
	return nil
}

func (spvaiq *SubscriptionPatchValueAddItemQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*SubscriptionPatchValueAddItem, error) {
	var (
		nodes       = []*SubscriptionPatchValueAddItem{}
		_spec       = spvaiq.querySpec()
		loadedTypes = [1]bool{
			spvaiq.withSubscriptionPatch != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*SubscriptionPatchValueAddItem).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &SubscriptionPatchValueAddItem{config: spvaiq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(spvaiq.modifiers) > 0 {
		_spec.Modifiers = spvaiq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, spvaiq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := spvaiq.withSubscriptionPatch; query != nil {
		if err := spvaiq.loadSubscriptionPatch(ctx, query, nodes, nil,
			func(n *SubscriptionPatchValueAddItem, e *SubscriptionPatch) { n.Edges.SubscriptionPatch = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (spvaiq *SubscriptionPatchValueAddItemQuery) loadSubscriptionPatch(ctx context.Context, query *SubscriptionPatchQuery, nodes []*SubscriptionPatchValueAddItem, init func(*SubscriptionPatchValueAddItem), assign func(*SubscriptionPatchValueAddItem, *SubscriptionPatch)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*SubscriptionPatchValueAddItem)
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

func (spvaiq *SubscriptionPatchValueAddItemQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := spvaiq.querySpec()
	if len(spvaiq.modifiers) > 0 {
		_spec.Modifiers = spvaiq.modifiers
	}
	_spec.Node.Columns = spvaiq.ctx.Fields
	if len(spvaiq.ctx.Fields) > 0 {
		_spec.Unique = spvaiq.ctx.Unique != nil && *spvaiq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, spvaiq.driver, _spec)
}

func (spvaiq *SubscriptionPatchValueAddItemQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(subscriptionpatchvalueadditem.Table, subscriptionpatchvalueadditem.Columns, sqlgraph.NewFieldSpec(subscriptionpatchvalueadditem.FieldID, field.TypeString))
	_spec.From = spvaiq.sql
	if unique := spvaiq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if spvaiq.path != nil {
		_spec.Unique = true
	}
	if fields := spvaiq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, subscriptionpatchvalueadditem.FieldID)
		for i := range fields {
			if fields[i] != subscriptionpatchvalueadditem.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if spvaiq.withSubscriptionPatch != nil {
			_spec.Node.AddColumnOnce(subscriptionpatchvalueadditem.FieldSubscriptionPatchID)
		}
	}
	if ps := spvaiq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := spvaiq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := spvaiq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := spvaiq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (spvaiq *SubscriptionPatchValueAddItemQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(spvaiq.driver.Dialect())
	t1 := builder.Table(subscriptionpatchvalueadditem.Table)
	columns := spvaiq.ctx.Fields
	if len(columns) == 0 {
		columns = subscriptionpatchvalueadditem.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if spvaiq.sql != nil {
		selector = spvaiq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if spvaiq.ctx.Unique != nil && *spvaiq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range spvaiq.modifiers {
		m(selector)
	}
	for _, p := range spvaiq.predicates {
		p(selector)
	}
	for _, p := range spvaiq.order {
		p(selector)
	}
	if offset := spvaiq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := spvaiq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (spvaiq *SubscriptionPatchValueAddItemQuery) ForUpdate(opts ...sql.LockOption) *SubscriptionPatchValueAddItemQuery {
	if spvaiq.driver.Dialect() == dialect.Postgres {
		spvaiq.Unique(false)
	}
	spvaiq.modifiers = append(spvaiq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return spvaiq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (spvaiq *SubscriptionPatchValueAddItemQuery) ForShare(opts ...sql.LockOption) *SubscriptionPatchValueAddItemQuery {
	if spvaiq.driver.Dialect() == dialect.Postgres {
		spvaiq.Unique(false)
	}
	spvaiq.modifiers = append(spvaiq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return spvaiq
}

// SubscriptionPatchValueAddItemGroupBy is the group-by builder for SubscriptionPatchValueAddItem entities.
type SubscriptionPatchValueAddItemGroupBy struct {
	selector
	build *SubscriptionPatchValueAddItemQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (spvaigb *SubscriptionPatchValueAddItemGroupBy) Aggregate(fns ...AggregateFunc) *SubscriptionPatchValueAddItemGroupBy {
	spvaigb.fns = append(spvaigb.fns, fns...)
	return spvaigb
}

// Scan applies the selector query and scans the result into the given value.
func (spvaigb *SubscriptionPatchValueAddItemGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, spvaigb.build.ctx, ent.OpQueryGroupBy)
	if err := spvaigb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*SubscriptionPatchValueAddItemQuery, *SubscriptionPatchValueAddItemGroupBy](ctx, spvaigb.build, spvaigb, spvaigb.build.inters, v)
}

func (spvaigb *SubscriptionPatchValueAddItemGroupBy) sqlScan(ctx context.Context, root *SubscriptionPatchValueAddItemQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(spvaigb.fns))
	for _, fn := range spvaigb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*spvaigb.flds)+len(spvaigb.fns))
		for _, f := range *spvaigb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*spvaigb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := spvaigb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// SubscriptionPatchValueAddItemSelect is the builder for selecting fields of SubscriptionPatchValueAddItem entities.
type SubscriptionPatchValueAddItemSelect struct {
	*SubscriptionPatchValueAddItemQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (spvais *SubscriptionPatchValueAddItemSelect) Aggregate(fns ...AggregateFunc) *SubscriptionPatchValueAddItemSelect {
	spvais.fns = append(spvais.fns, fns...)
	return spvais
}

// Scan applies the selector query and scans the result into the given value.
func (spvais *SubscriptionPatchValueAddItemSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, spvais.ctx, ent.OpQuerySelect)
	if err := spvais.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*SubscriptionPatchValueAddItemQuery, *SubscriptionPatchValueAddItemSelect](ctx, spvais.SubscriptionPatchValueAddItemQuery, spvais, spvais.inters, v)
}

func (spvais *SubscriptionPatchValueAddItemSelect) sqlScan(ctx context.Context, root *SubscriptionPatchValueAddItemQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(spvais.fns))
	for _, fn := range spvais.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*spvais.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := spvais.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
