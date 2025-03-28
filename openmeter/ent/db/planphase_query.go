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
	"github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// PlanPhaseQuery is the builder for querying PlanPhase entities.
type PlanPhaseQuery struct {
	config
	ctx           *QueryContext
	order         []planphase.OrderOption
	inters        []Interceptor
	predicates    []predicate.PlanPhase
	withPlan      *PlanQuery
	withRatecards *PlanRateCardQuery
	modifiers     []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the PlanPhaseQuery builder.
func (ppq *PlanPhaseQuery) Where(ps ...predicate.PlanPhase) *PlanPhaseQuery {
	ppq.predicates = append(ppq.predicates, ps...)
	return ppq
}

// Limit the number of records to be returned by this query.
func (ppq *PlanPhaseQuery) Limit(limit int) *PlanPhaseQuery {
	ppq.ctx.Limit = &limit
	return ppq
}

// Offset to start from.
func (ppq *PlanPhaseQuery) Offset(offset int) *PlanPhaseQuery {
	ppq.ctx.Offset = &offset
	return ppq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (ppq *PlanPhaseQuery) Unique(unique bool) *PlanPhaseQuery {
	ppq.ctx.Unique = &unique
	return ppq
}

// Order specifies how the records should be ordered.
func (ppq *PlanPhaseQuery) Order(o ...planphase.OrderOption) *PlanPhaseQuery {
	ppq.order = append(ppq.order, o...)
	return ppq
}

// QueryPlan chains the current query on the "plan" edge.
func (ppq *PlanPhaseQuery) QueryPlan() *PlanQuery {
	query := (&PlanClient{config: ppq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := ppq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := ppq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(planphase.Table, planphase.FieldID, selector),
			sqlgraph.To(plan.Table, plan.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, planphase.PlanTable, planphase.PlanColumn),
		)
		fromU = sqlgraph.SetNeighbors(ppq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryRatecards chains the current query on the "ratecards" edge.
func (ppq *PlanPhaseQuery) QueryRatecards() *PlanRateCardQuery {
	query := (&PlanRateCardClient{config: ppq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := ppq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := ppq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(planphase.Table, planphase.FieldID, selector),
			sqlgraph.To(planratecard.Table, planratecard.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, planphase.RatecardsTable, planphase.RatecardsColumn),
		)
		fromU = sqlgraph.SetNeighbors(ppq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first PlanPhase entity from the query.
// Returns a *NotFoundError when no PlanPhase was found.
func (ppq *PlanPhaseQuery) First(ctx context.Context) (*PlanPhase, error) {
	nodes, err := ppq.Limit(1).All(setContextOp(ctx, ppq.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{planphase.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (ppq *PlanPhaseQuery) FirstX(ctx context.Context) *PlanPhase {
	node, err := ppq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first PlanPhase ID from the query.
// Returns a *NotFoundError when no PlanPhase ID was found.
func (ppq *PlanPhaseQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = ppq.Limit(1).IDs(setContextOp(ctx, ppq.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{planphase.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (ppq *PlanPhaseQuery) FirstIDX(ctx context.Context) string {
	id, err := ppq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single PlanPhase entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one PlanPhase entity is found.
// Returns a *NotFoundError when no PlanPhase entities are found.
func (ppq *PlanPhaseQuery) Only(ctx context.Context) (*PlanPhase, error) {
	nodes, err := ppq.Limit(2).All(setContextOp(ctx, ppq.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{planphase.Label}
	default:
		return nil, &NotSingularError{planphase.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (ppq *PlanPhaseQuery) OnlyX(ctx context.Context) *PlanPhase {
	node, err := ppq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only PlanPhase ID in the query.
// Returns a *NotSingularError when more than one PlanPhase ID is found.
// Returns a *NotFoundError when no entities are found.
func (ppq *PlanPhaseQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = ppq.Limit(2).IDs(setContextOp(ctx, ppq.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{planphase.Label}
	default:
		err = &NotSingularError{planphase.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (ppq *PlanPhaseQuery) OnlyIDX(ctx context.Context) string {
	id, err := ppq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of PlanPhases.
func (ppq *PlanPhaseQuery) All(ctx context.Context) ([]*PlanPhase, error) {
	ctx = setContextOp(ctx, ppq.ctx, ent.OpQueryAll)
	if err := ppq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*PlanPhase, *PlanPhaseQuery]()
	return withInterceptors[[]*PlanPhase](ctx, ppq, qr, ppq.inters)
}

// AllX is like All, but panics if an error occurs.
func (ppq *PlanPhaseQuery) AllX(ctx context.Context) []*PlanPhase {
	nodes, err := ppq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of PlanPhase IDs.
func (ppq *PlanPhaseQuery) IDs(ctx context.Context) (ids []string, err error) {
	if ppq.ctx.Unique == nil && ppq.path != nil {
		ppq.Unique(true)
	}
	ctx = setContextOp(ctx, ppq.ctx, ent.OpQueryIDs)
	if err = ppq.Select(planphase.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (ppq *PlanPhaseQuery) IDsX(ctx context.Context) []string {
	ids, err := ppq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (ppq *PlanPhaseQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, ppq.ctx, ent.OpQueryCount)
	if err := ppq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, ppq, querierCount[*PlanPhaseQuery](), ppq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (ppq *PlanPhaseQuery) CountX(ctx context.Context) int {
	count, err := ppq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (ppq *PlanPhaseQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, ppq.ctx, ent.OpQueryExist)
	switch _, err := ppq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (ppq *PlanPhaseQuery) ExistX(ctx context.Context) bool {
	exist, err := ppq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the PlanPhaseQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (ppq *PlanPhaseQuery) Clone() *PlanPhaseQuery {
	if ppq == nil {
		return nil
	}
	return &PlanPhaseQuery{
		config:        ppq.config,
		ctx:           ppq.ctx.Clone(),
		order:         append([]planphase.OrderOption{}, ppq.order...),
		inters:        append([]Interceptor{}, ppq.inters...),
		predicates:    append([]predicate.PlanPhase{}, ppq.predicates...),
		withPlan:      ppq.withPlan.Clone(),
		withRatecards: ppq.withRatecards.Clone(),
		// clone intermediate query.
		sql:  ppq.sql.Clone(),
		path: ppq.path,
	}
}

// WithPlan tells the query-builder to eager-load the nodes that are connected to
// the "plan" edge. The optional arguments are used to configure the query builder of the edge.
func (ppq *PlanPhaseQuery) WithPlan(opts ...func(*PlanQuery)) *PlanPhaseQuery {
	query := (&PlanClient{config: ppq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	ppq.withPlan = query
	return ppq
}

// WithRatecards tells the query-builder to eager-load the nodes that are connected to
// the "ratecards" edge. The optional arguments are used to configure the query builder of the edge.
func (ppq *PlanPhaseQuery) WithRatecards(opts ...func(*PlanRateCardQuery)) *PlanPhaseQuery {
	query := (&PlanRateCardClient{config: ppq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	ppq.withRatecards = query
	return ppq
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
//	client.PlanPhase.Query().
//		GroupBy(planphase.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (ppq *PlanPhaseQuery) GroupBy(field string, fields ...string) *PlanPhaseGroupBy {
	ppq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &PlanPhaseGroupBy{build: ppq}
	grbuild.flds = &ppq.ctx.Fields
	grbuild.label = planphase.Label
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
//	client.PlanPhase.Query().
//		Select(planphase.FieldNamespace).
//		Scan(ctx, &v)
func (ppq *PlanPhaseQuery) Select(fields ...string) *PlanPhaseSelect {
	ppq.ctx.Fields = append(ppq.ctx.Fields, fields...)
	sbuild := &PlanPhaseSelect{PlanPhaseQuery: ppq}
	sbuild.label = planphase.Label
	sbuild.flds, sbuild.scan = &ppq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a PlanPhaseSelect configured with the given aggregations.
func (ppq *PlanPhaseQuery) Aggregate(fns ...AggregateFunc) *PlanPhaseSelect {
	return ppq.Select().Aggregate(fns...)
}

func (ppq *PlanPhaseQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range ppq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, ppq); err != nil {
				return err
			}
		}
	}
	for _, f := range ppq.ctx.Fields {
		if !planphase.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if ppq.path != nil {
		prev, err := ppq.path(ctx)
		if err != nil {
			return err
		}
		ppq.sql = prev
	}
	return nil
}

func (ppq *PlanPhaseQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*PlanPhase, error) {
	var (
		nodes       = []*PlanPhase{}
		_spec       = ppq.querySpec()
		loadedTypes = [2]bool{
			ppq.withPlan != nil,
			ppq.withRatecards != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*PlanPhase).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &PlanPhase{config: ppq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(ppq.modifiers) > 0 {
		_spec.Modifiers = ppq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, ppq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := ppq.withPlan; query != nil {
		if err := ppq.loadPlan(ctx, query, nodes, nil,
			func(n *PlanPhase, e *Plan) { n.Edges.Plan = e }); err != nil {
			return nil, err
		}
	}
	if query := ppq.withRatecards; query != nil {
		if err := ppq.loadRatecards(ctx, query, nodes,
			func(n *PlanPhase) { n.Edges.Ratecards = []*PlanRateCard{} },
			func(n *PlanPhase, e *PlanRateCard) { n.Edges.Ratecards = append(n.Edges.Ratecards, e) }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (ppq *PlanPhaseQuery) loadPlan(ctx context.Context, query *PlanQuery, nodes []*PlanPhase, init func(*PlanPhase), assign func(*PlanPhase, *Plan)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*PlanPhase)
	for i := range nodes {
		fk := nodes[i].PlanID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(plan.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "plan_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (ppq *PlanPhaseQuery) loadRatecards(ctx context.Context, query *PlanRateCardQuery, nodes []*PlanPhase, init func(*PlanPhase), assign func(*PlanPhase, *PlanRateCard)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*PlanPhase)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(planratecard.FieldPhaseID)
	}
	query.Where(predicate.PlanRateCard(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(planphase.RatecardsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.PhaseID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "phase_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}

func (ppq *PlanPhaseQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := ppq.querySpec()
	if len(ppq.modifiers) > 0 {
		_spec.Modifiers = ppq.modifiers
	}
	_spec.Node.Columns = ppq.ctx.Fields
	if len(ppq.ctx.Fields) > 0 {
		_spec.Unique = ppq.ctx.Unique != nil && *ppq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, ppq.driver, _spec)
}

func (ppq *PlanPhaseQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(planphase.Table, planphase.Columns, sqlgraph.NewFieldSpec(planphase.FieldID, field.TypeString))
	_spec.From = ppq.sql
	if unique := ppq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if ppq.path != nil {
		_spec.Unique = true
	}
	if fields := ppq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, planphase.FieldID)
		for i := range fields {
			if fields[i] != planphase.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if ppq.withPlan != nil {
			_spec.Node.AddColumnOnce(planphase.FieldPlanID)
		}
	}
	if ps := ppq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := ppq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := ppq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := ppq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (ppq *PlanPhaseQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(ppq.driver.Dialect())
	t1 := builder.Table(planphase.Table)
	columns := ppq.ctx.Fields
	if len(columns) == 0 {
		columns = planphase.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if ppq.sql != nil {
		selector = ppq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if ppq.ctx.Unique != nil && *ppq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range ppq.modifiers {
		m(selector)
	}
	for _, p := range ppq.predicates {
		p(selector)
	}
	for _, p := range ppq.order {
		p(selector)
	}
	if offset := ppq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := ppq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (ppq *PlanPhaseQuery) ForUpdate(opts ...sql.LockOption) *PlanPhaseQuery {
	if ppq.driver.Dialect() == dialect.Postgres {
		ppq.Unique(false)
	}
	ppq.modifiers = append(ppq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return ppq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (ppq *PlanPhaseQuery) ForShare(opts ...sql.LockOption) *PlanPhaseQuery {
	if ppq.driver.Dialect() == dialect.Postgres {
		ppq.Unique(false)
	}
	ppq.modifiers = append(ppq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return ppq
}

// PlanPhaseGroupBy is the group-by builder for PlanPhase entities.
type PlanPhaseGroupBy struct {
	selector
	build *PlanPhaseQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (ppgb *PlanPhaseGroupBy) Aggregate(fns ...AggregateFunc) *PlanPhaseGroupBy {
	ppgb.fns = append(ppgb.fns, fns...)
	return ppgb
}

// Scan applies the selector query and scans the result into the given value.
func (ppgb *PlanPhaseGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, ppgb.build.ctx, ent.OpQueryGroupBy)
	if err := ppgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*PlanPhaseQuery, *PlanPhaseGroupBy](ctx, ppgb.build, ppgb, ppgb.build.inters, v)
}

func (ppgb *PlanPhaseGroupBy) sqlScan(ctx context.Context, root *PlanPhaseQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(ppgb.fns))
	for _, fn := range ppgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*ppgb.flds)+len(ppgb.fns))
		for _, f := range *ppgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*ppgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := ppgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// PlanPhaseSelect is the builder for selecting fields of PlanPhase entities.
type PlanPhaseSelect struct {
	*PlanPhaseQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (pps *PlanPhaseSelect) Aggregate(fns ...AggregateFunc) *PlanPhaseSelect {
	pps.fns = append(pps.fns, fns...)
	return pps
}

// Scan applies the selector query and scans the result into the given value.
func (pps *PlanPhaseSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, pps.ctx, ent.OpQuerySelect)
	if err := pps.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*PlanPhaseQuery, *PlanPhaseSelect](ctx, pps.PlanPhaseQuery, pps, pps.inters, v)
}

func (pps *PlanPhaseSelect) sqlScan(ctx context.Context, root *PlanPhaseQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(pps.fns))
	for _, fn := range pps.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*pps.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := pps.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
