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
	"github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// PlanRateCardQuery is the builder for querying PlanRateCard entities.
type PlanRateCardQuery struct {
	config
	ctx          *QueryContext
	order        []planratecard.OrderOption
	inters       []Interceptor
	predicates   []predicate.PlanRateCard
	withPhase    *PlanPhaseQuery
	withFeatures *FeatureQuery
	modifiers    []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the PlanRateCardQuery builder.
func (_q *PlanRateCardQuery) Where(ps ...predicate.PlanRateCard) *PlanRateCardQuery {
	_q.predicates = append(_q.predicates, ps...)
	return _q
}

// Limit the number of records to be returned by this query.
func (_q *PlanRateCardQuery) Limit(limit int) *PlanRateCardQuery {
	_q.ctx.Limit = &limit
	return _q
}

// Offset to start from.
func (_q *PlanRateCardQuery) Offset(offset int) *PlanRateCardQuery {
	_q.ctx.Offset = &offset
	return _q
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (_q *PlanRateCardQuery) Unique(unique bool) *PlanRateCardQuery {
	_q.ctx.Unique = &unique
	return _q
}

// Order specifies how the records should be ordered.
func (_q *PlanRateCardQuery) Order(o ...planratecard.OrderOption) *PlanRateCardQuery {
	_q.order = append(_q.order, o...)
	return _q
}

// QueryPhase chains the current query on the "phase" edge.
func (_q *PlanRateCardQuery) QueryPhase() *PlanPhaseQuery {
	query := (&PlanPhaseClient{config: _q.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := _q.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := _q.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(planratecard.Table, planratecard.FieldID, selector),
			sqlgraph.To(planphase.Table, planphase.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, planratecard.PhaseTable, planratecard.PhaseColumn),
		)
		fromU = sqlgraph.SetNeighbors(_q.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryFeatures chains the current query on the "features" edge.
func (_q *PlanRateCardQuery) QueryFeatures() *FeatureQuery {
	query := (&FeatureClient{config: _q.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := _q.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := _q.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(planratecard.Table, planratecard.FieldID, selector),
			sqlgraph.To(feature.Table, feature.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, planratecard.FeaturesTable, planratecard.FeaturesColumn),
		)
		fromU = sqlgraph.SetNeighbors(_q.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first PlanRateCard entity from the query.
// Returns a *NotFoundError when no PlanRateCard was found.
func (_q *PlanRateCardQuery) First(ctx context.Context) (*PlanRateCard, error) {
	nodes, err := _q.Limit(1).All(setContextOp(ctx, _q.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{planratecard.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (_q *PlanRateCardQuery) FirstX(ctx context.Context) *PlanRateCard {
	node, err := _q.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first PlanRateCard ID from the query.
// Returns a *NotFoundError when no PlanRateCard ID was found.
func (_q *PlanRateCardQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = _q.Limit(1).IDs(setContextOp(ctx, _q.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{planratecard.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (_q *PlanRateCardQuery) FirstIDX(ctx context.Context) string {
	id, err := _q.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single PlanRateCard entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one PlanRateCard entity is found.
// Returns a *NotFoundError when no PlanRateCard entities are found.
func (_q *PlanRateCardQuery) Only(ctx context.Context) (*PlanRateCard, error) {
	nodes, err := _q.Limit(2).All(setContextOp(ctx, _q.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{planratecard.Label}
	default:
		return nil, &NotSingularError{planratecard.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (_q *PlanRateCardQuery) OnlyX(ctx context.Context) *PlanRateCard {
	node, err := _q.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only PlanRateCard ID in the query.
// Returns a *NotSingularError when more than one PlanRateCard ID is found.
// Returns a *NotFoundError when no entities are found.
func (_q *PlanRateCardQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = _q.Limit(2).IDs(setContextOp(ctx, _q.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{planratecard.Label}
	default:
		err = &NotSingularError{planratecard.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (_q *PlanRateCardQuery) OnlyIDX(ctx context.Context) string {
	id, err := _q.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of PlanRateCards.
func (_q *PlanRateCardQuery) All(ctx context.Context) ([]*PlanRateCard, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryAll)
	if err := _q.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*PlanRateCard, *PlanRateCardQuery]()
	return withInterceptors[[]*PlanRateCard](ctx, _q, qr, _q.inters)
}

// AllX is like All, but panics if an error occurs.
func (_q *PlanRateCardQuery) AllX(ctx context.Context) []*PlanRateCard {
	nodes, err := _q.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of PlanRateCard IDs.
func (_q *PlanRateCardQuery) IDs(ctx context.Context) (ids []string, err error) {
	if _q.ctx.Unique == nil && _q.path != nil {
		_q.Unique(true)
	}
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryIDs)
	if err = _q.Select(planratecard.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (_q *PlanRateCardQuery) IDsX(ctx context.Context) []string {
	ids, err := _q.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (_q *PlanRateCardQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, _q.ctx, ent.OpQueryCount)
	if err := _q.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, _q, querierCount[*PlanRateCardQuery](), _q.inters)
}

// CountX is like Count, but panics if an error occurs.
func (_q *PlanRateCardQuery) CountX(ctx context.Context) int {
	count, err := _q.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (_q *PlanRateCardQuery) Exist(ctx context.Context) (bool, error) {
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
func (_q *PlanRateCardQuery) ExistX(ctx context.Context) bool {
	exist, err := _q.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the PlanRateCardQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (_q *PlanRateCardQuery) Clone() *PlanRateCardQuery {
	if _q == nil {
		return nil
	}
	return &PlanRateCardQuery{
		config:       _q.config,
		ctx:          _q.ctx.Clone(),
		order:        append([]planratecard.OrderOption{}, _q.order...),
		inters:       append([]Interceptor{}, _q.inters...),
		predicates:   append([]predicate.PlanRateCard{}, _q.predicates...),
		withPhase:    _q.withPhase.Clone(),
		withFeatures: _q.withFeatures.Clone(),
		// clone intermediate query.
		sql:  _q.sql.Clone(),
		path: _q.path,
	}
}

// WithPhase tells the query-builder to eager-load the nodes that are connected to
// the "phase" edge. The optional arguments are used to configure the query builder of the edge.
func (_q *PlanRateCardQuery) WithPhase(opts ...func(*PlanPhaseQuery)) *PlanRateCardQuery {
	query := (&PlanPhaseClient{config: _q.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	_q.withPhase = query
	return _q
}

// WithFeatures tells the query-builder to eager-load the nodes that are connected to
// the "features" edge. The optional arguments are used to configure the query builder of the edge.
func (_q *PlanRateCardQuery) WithFeatures(opts ...func(*FeatureQuery)) *PlanRateCardQuery {
	query := (&FeatureClient{config: _q.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	_q.withFeatures = query
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
//	client.PlanRateCard.Query().
//		GroupBy(planratecard.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (_q *PlanRateCardQuery) GroupBy(field string, fields ...string) *PlanRateCardGroupBy {
	_q.ctx.Fields = append([]string{field}, fields...)
	grbuild := &PlanRateCardGroupBy{build: _q}
	grbuild.flds = &_q.ctx.Fields
	grbuild.label = planratecard.Label
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
//	client.PlanRateCard.Query().
//		Select(planratecard.FieldNamespace).
//		Scan(ctx, &v)
func (_q *PlanRateCardQuery) Select(fields ...string) *PlanRateCardSelect {
	_q.ctx.Fields = append(_q.ctx.Fields, fields...)
	sbuild := &PlanRateCardSelect{PlanRateCardQuery: _q}
	sbuild.label = planratecard.Label
	sbuild.flds, sbuild.scan = &_q.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a PlanRateCardSelect configured with the given aggregations.
func (_q *PlanRateCardQuery) Aggregate(fns ...AggregateFunc) *PlanRateCardSelect {
	return _q.Select().Aggregate(fns...)
}

func (_q *PlanRateCardQuery) prepareQuery(ctx context.Context) error {
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
		if !planratecard.ValidColumn(f) {
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

func (_q *PlanRateCardQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*PlanRateCard, error) {
	var (
		nodes       = []*PlanRateCard{}
		_spec       = _q.querySpec()
		loadedTypes = [2]bool{
			_q.withPhase != nil,
			_q.withFeatures != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*PlanRateCard).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &PlanRateCard{config: _q.config}
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
	if query := _q.withPhase; query != nil {
		if err := _q.loadPhase(ctx, query, nodes, nil,
			func(n *PlanRateCard, e *PlanPhase) { n.Edges.Phase = e }); err != nil {
			return nil, err
		}
	}
	if query := _q.withFeatures; query != nil {
		if err := _q.loadFeatures(ctx, query, nodes, nil,
			func(n *PlanRateCard, e *Feature) { n.Edges.Features = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (_q *PlanRateCardQuery) loadPhase(ctx context.Context, query *PlanPhaseQuery, nodes []*PlanRateCard, init func(*PlanRateCard), assign func(*PlanRateCard, *PlanPhase)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*PlanRateCard)
	for i := range nodes {
		fk := nodes[i].PhaseID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(planphase.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "phase_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (_q *PlanRateCardQuery) loadFeatures(ctx context.Context, query *FeatureQuery, nodes []*PlanRateCard, init func(*PlanRateCard), assign func(*PlanRateCard, *Feature)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*PlanRateCard)
	for i := range nodes {
		if nodes[i].FeatureID == nil {
			continue
		}
		fk := *nodes[i].FeatureID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(feature.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "feature_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (_q *PlanRateCardQuery) sqlCount(ctx context.Context) (int, error) {
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

func (_q *PlanRateCardQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(planratecard.Table, planratecard.Columns, sqlgraph.NewFieldSpec(planratecard.FieldID, field.TypeString))
	_spec.From = _q.sql
	if unique := _q.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if _q.path != nil {
		_spec.Unique = true
	}
	if fields := _q.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, planratecard.FieldID)
		for i := range fields {
			if fields[i] != planratecard.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if _q.withPhase != nil {
			_spec.Node.AddColumnOnce(planratecard.FieldPhaseID)
		}
		if _q.withFeatures != nil {
			_spec.Node.AddColumnOnce(planratecard.FieldFeatureID)
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

func (_q *PlanRateCardQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(_q.driver.Dialect())
	t1 := builder.Table(planratecard.Table)
	columns := _q.ctx.Fields
	if len(columns) == 0 {
		columns = planratecard.Columns
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
func (_q *PlanRateCardQuery) ForUpdate(opts ...sql.LockOption) *PlanRateCardQuery {
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
func (_q *PlanRateCardQuery) ForShare(opts ...sql.LockOption) *PlanRateCardQuery {
	if _q.driver.Dialect() == dialect.Postgres {
		_q.Unique(false)
	}
	_q.modifiers = append(_q.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return _q
}

// PlanRateCardGroupBy is the group-by builder for PlanRateCard entities.
type PlanRateCardGroupBy struct {
	selector
	build *PlanRateCardQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (prcgb *PlanRateCardGroupBy) Aggregate(fns ...AggregateFunc) *PlanRateCardGroupBy {
	prcgb.fns = append(prcgb.fns, fns...)
	return prcgb
}

// Scan applies the selector query and scans the result into the given value.
func (prcgb *PlanRateCardGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, prcgb.build.ctx, ent.OpQueryGroupBy)
	if err := prcgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*PlanRateCardQuery, *PlanRateCardGroupBy](ctx, prcgb.build, prcgb, prcgb.build.inters, v)
}

func (prcgb *PlanRateCardGroupBy) sqlScan(ctx context.Context, root *PlanRateCardQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(prcgb.fns))
	for _, fn := range prcgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*prcgb.flds)+len(prcgb.fns))
		for _, f := range *prcgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*prcgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := prcgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// PlanRateCardSelect is the builder for selecting fields of PlanRateCard entities.
type PlanRateCardSelect struct {
	*PlanRateCardQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (prcs *PlanRateCardSelect) Aggregate(fns ...AggregateFunc) *PlanRateCardSelect {
	prcs.fns = append(prcs.fns, fns...)
	return prcs
}

// Scan applies the selector query and scans the result into the given value.
func (prcs *PlanRateCardSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, prcs.ctx, ent.OpQuerySelect)
	if err := prcs.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*PlanRateCardQuery, *PlanRateCardSelect](ctx, prcs.PlanRateCardQuery, prcs, prcs.inters, v)
}

func (prcs *PlanRateCardSelect) sqlScan(ctx context.Context, root *PlanRateCardQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(prcs.fns))
	for _, fn := range prcs.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*prcs.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := prcs.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
