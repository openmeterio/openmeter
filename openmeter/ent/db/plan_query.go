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
	"github.com/openmeterio/openmeter/openmeter/ent/db/planaddon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
)

// PlanQuery is the builder for querying Plan entities.
type PlanQuery struct {
	config
	ctx               *QueryContext
	order             []plan.OrderOption
	inters            []Interceptor
	predicates        []predicate.Plan
	withPhases        *PlanPhaseQuery
	withAddons        *PlanAddonQuery
	withSubscriptions *SubscriptionQuery
	modifiers         []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the PlanQuery builder.
func (pq *PlanQuery) Where(ps ...predicate.Plan) *PlanQuery {
	pq.predicates = append(pq.predicates, ps...)
	return pq
}

// Limit the number of records to be returned by this query.
func (pq *PlanQuery) Limit(limit int) *PlanQuery {
	pq.ctx.Limit = &limit
	return pq
}

// Offset to start from.
func (pq *PlanQuery) Offset(offset int) *PlanQuery {
	pq.ctx.Offset = &offset
	return pq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (pq *PlanQuery) Unique(unique bool) *PlanQuery {
	pq.ctx.Unique = &unique
	return pq
}

// Order specifies how the records should be ordered.
func (pq *PlanQuery) Order(o ...plan.OrderOption) *PlanQuery {
	pq.order = append(pq.order, o...)
	return pq
}

// QueryPhases chains the current query on the "phases" edge.
func (pq *PlanQuery) QueryPhases() *PlanPhaseQuery {
	query := (&PlanPhaseClient{config: pq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := pq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := pq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(plan.Table, plan.FieldID, selector),
			sqlgraph.To(planphase.Table, planphase.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, plan.PhasesTable, plan.PhasesColumn),
		)
		fromU = sqlgraph.SetNeighbors(pq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryAddons chains the current query on the "addons" edge.
func (pq *PlanQuery) QueryAddons() *PlanAddonQuery {
	query := (&PlanAddonClient{config: pq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := pq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := pq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(plan.Table, plan.FieldID, selector),
			sqlgraph.To(planaddon.Table, planaddon.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, plan.AddonsTable, plan.AddonsColumn),
		)
		fromU = sqlgraph.SetNeighbors(pq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QuerySubscriptions chains the current query on the "subscriptions" edge.
func (pq *PlanQuery) QuerySubscriptions() *SubscriptionQuery {
	query := (&SubscriptionClient{config: pq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := pq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := pq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(plan.Table, plan.FieldID, selector),
			sqlgraph.To(subscription.Table, subscription.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, plan.SubscriptionsTable, plan.SubscriptionsColumn),
		)
		fromU = sqlgraph.SetNeighbors(pq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first Plan entity from the query.
// Returns a *NotFoundError when no Plan was found.
func (pq *PlanQuery) First(ctx context.Context) (*Plan, error) {
	nodes, err := pq.Limit(1).All(setContextOp(ctx, pq.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{plan.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (pq *PlanQuery) FirstX(ctx context.Context) *Plan {
	node, err := pq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first Plan ID from the query.
// Returns a *NotFoundError when no Plan ID was found.
func (pq *PlanQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = pq.Limit(1).IDs(setContextOp(ctx, pq.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{plan.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (pq *PlanQuery) FirstIDX(ctx context.Context) string {
	id, err := pq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single Plan entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one Plan entity is found.
// Returns a *NotFoundError when no Plan entities are found.
func (pq *PlanQuery) Only(ctx context.Context) (*Plan, error) {
	nodes, err := pq.Limit(2).All(setContextOp(ctx, pq.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{plan.Label}
	default:
		return nil, &NotSingularError{plan.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (pq *PlanQuery) OnlyX(ctx context.Context) *Plan {
	node, err := pq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only Plan ID in the query.
// Returns a *NotSingularError when more than one Plan ID is found.
// Returns a *NotFoundError when no entities are found.
func (pq *PlanQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = pq.Limit(2).IDs(setContextOp(ctx, pq.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{plan.Label}
	default:
		err = &NotSingularError{plan.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (pq *PlanQuery) OnlyIDX(ctx context.Context) string {
	id, err := pq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of Plans.
func (pq *PlanQuery) All(ctx context.Context) ([]*Plan, error) {
	ctx = setContextOp(ctx, pq.ctx, ent.OpQueryAll)
	if err := pq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*Plan, *PlanQuery]()
	return withInterceptors[[]*Plan](ctx, pq, qr, pq.inters)
}

// AllX is like All, but panics if an error occurs.
func (pq *PlanQuery) AllX(ctx context.Context) []*Plan {
	nodes, err := pq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of Plan IDs.
func (pq *PlanQuery) IDs(ctx context.Context) (ids []string, err error) {
	if pq.ctx.Unique == nil && pq.path != nil {
		pq.Unique(true)
	}
	ctx = setContextOp(ctx, pq.ctx, ent.OpQueryIDs)
	if err = pq.Select(plan.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (pq *PlanQuery) IDsX(ctx context.Context) []string {
	ids, err := pq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (pq *PlanQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, pq.ctx, ent.OpQueryCount)
	if err := pq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, pq, querierCount[*PlanQuery](), pq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (pq *PlanQuery) CountX(ctx context.Context) int {
	count, err := pq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (pq *PlanQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, pq.ctx, ent.OpQueryExist)
	switch _, err := pq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (pq *PlanQuery) ExistX(ctx context.Context) bool {
	exist, err := pq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the PlanQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (pq *PlanQuery) Clone() *PlanQuery {
	if pq == nil {
		return nil
	}
	return &PlanQuery{
		config:            pq.config,
		ctx:               pq.ctx.Clone(),
		order:             append([]plan.OrderOption{}, pq.order...),
		inters:            append([]Interceptor{}, pq.inters...),
		predicates:        append([]predicate.Plan{}, pq.predicates...),
		withPhases:        pq.withPhases.Clone(),
		withAddons:        pq.withAddons.Clone(),
		withSubscriptions: pq.withSubscriptions.Clone(),
		// clone intermediate query.
		sql:  pq.sql.Clone(),
		path: pq.path,
	}
}

// WithPhases tells the query-builder to eager-load the nodes that are connected to
// the "phases" edge. The optional arguments are used to configure the query builder of the edge.
func (pq *PlanQuery) WithPhases(opts ...func(*PlanPhaseQuery)) *PlanQuery {
	query := (&PlanPhaseClient{config: pq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	pq.withPhases = query
	return pq
}

// WithAddons tells the query-builder to eager-load the nodes that are connected to
// the "addons" edge. The optional arguments are used to configure the query builder of the edge.
func (pq *PlanQuery) WithAddons(opts ...func(*PlanAddonQuery)) *PlanQuery {
	query := (&PlanAddonClient{config: pq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	pq.withAddons = query
	return pq
}

// WithSubscriptions tells the query-builder to eager-load the nodes that are connected to
// the "subscriptions" edge. The optional arguments are used to configure the query builder of the edge.
func (pq *PlanQuery) WithSubscriptions(opts ...func(*SubscriptionQuery)) *PlanQuery {
	query := (&SubscriptionClient{config: pq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	pq.withSubscriptions = query
	return pq
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
//	client.Plan.Query().
//		GroupBy(plan.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (pq *PlanQuery) GroupBy(field string, fields ...string) *PlanGroupBy {
	pq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &PlanGroupBy{build: pq}
	grbuild.flds = &pq.ctx.Fields
	grbuild.label = plan.Label
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
//	client.Plan.Query().
//		Select(plan.FieldNamespace).
//		Scan(ctx, &v)
func (pq *PlanQuery) Select(fields ...string) *PlanSelect {
	pq.ctx.Fields = append(pq.ctx.Fields, fields...)
	sbuild := &PlanSelect{PlanQuery: pq}
	sbuild.label = plan.Label
	sbuild.flds, sbuild.scan = &pq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a PlanSelect configured with the given aggregations.
func (pq *PlanQuery) Aggregate(fns ...AggregateFunc) *PlanSelect {
	return pq.Select().Aggregate(fns...)
}

func (pq *PlanQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range pq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, pq); err != nil {
				return err
			}
		}
	}
	for _, f := range pq.ctx.Fields {
		if !plan.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if pq.path != nil {
		prev, err := pq.path(ctx)
		if err != nil {
			return err
		}
		pq.sql = prev
	}
	return nil
}

func (pq *PlanQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*Plan, error) {
	var (
		nodes       = []*Plan{}
		_spec       = pq.querySpec()
		loadedTypes = [3]bool{
			pq.withPhases != nil,
			pq.withAddons != nil,
			pq.withSubscriptions != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*Plan).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &Plan{config: pq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(pq.modifiers) > 0 {
		_spec.Modifiers = pq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, pq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := pq.withPhases; query != nil {
		if err := pq.loadPhases(ctx, query, nodes,
			func(n *Plan) { n.Edges.Phases = []*PlanPhase{} },
			func(n *Plan, e *PlanPhase) { n.Edges.Phases = append(n.Edges.Phases, e) }); err != nil {
			return nil, err
		}
	}
	if query := pq.withAddons; query != nil {
		if err := pq.loadAddons(ctx, query, nodes,
			func(n *Plan) { n.Edges.Addons = []*PlanAddon{} },
			func(n *Plan, e *PlanAddon) { n.Edges.Addons = append(n.Edges.Addons, e) }); err != nil {
			return nil, err
		}
	}
	if query := pq.withSubscriptions; query != nil {
		if err := pq.loadSubscriptions(ctx, query, nodes,
			func(n *Plan) { n.Edges.Subscriptions = []*Subscription{} },
			func(n *Plan, e *Subscription) { n.Edges.Subscriptions = append(n.Edges.Subscriptions, e) }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (pq *PlanQuery) loadPhases(ctx context.Context, query *PlanPhaseQuery, nodes []*Plan, init func(*Plan), assign func(*Plan, *PlanPhase)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*Plan)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(planphase.FieldPlanID)
	}
	query.Where(predicate.PlanPhase(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(plan.PhasesColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.PlanID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "plan_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (pq *PlanQuery) loadAddons(ctx context.Context, query *PlanAddonQuery, nodes []*Plan, init func(*Plan), assign func(*Plan, *PlanAddon)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*Plan)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(planaddon.FieldPlanID)
	}
	query.Where(predicate.PlanAddon(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(plan.AddonsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.PlanID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "plan_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (pq *PlanQuery) loadSubscriptions(ctx context.Context, query *SubscriptionQuery, nodes []*Plan, init func(*Plan), assign func(*Plan, *Subscription)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*Plan)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(subscription.FieldPlanID)
	}
	query.Where(predicate.Subscription(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(plan.SubscriptionsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.PlanID
		if fk == nil {
			return fmt.Errorf(`foreign-key "plan_id" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "plan_id" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}

func (pq *PlanQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := pq.querySpec()
	if len(pq.modifiers) > 0 {
		_spec.Modifiers = pq.modifiers
	}
	_spec.Node.Columns = pq.ctx.Fields
	if len(pq.ctx.Fields) > 0 {
		_spec.Unique = pq.ctx.Unique != nil && *pq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, pq.driver, _spec)
}

func (pq *PlanQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(plan.Table, plan.Columns, sqlgraph.NewFieldSpec(plan.FieldID, field.TypeString))
	_spec.From = pq.sql
	if unique := pq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if pq.path != nil {
		_spec.Unique = true
	}
	if fields := pq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, plan.FieldID)
		for i := range fields {
			if fields[i] != plan.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
	}
	if ps := pq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := pq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := pq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := pq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (pq *PlanQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(pq.driver.Dialect())
	t1 := builder.Table(plan.Table)
	columns := pq.ctx.Fields
	if len(columns) == 0 {
		columns = plan.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if pq.sql != nil {
		selector = pq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if pq.ctx.Unique != nil && *pq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range pq.modifiers {
		m(selector)
	}
	for _, p := range pq.predicates {
		p(selector)
	}
	for _, p := range pq.order {
		p(selector)
	}
	if offset := pq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := pq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (pq *PlanQuery) ForUpdate(opts ...sql.LockOption) *PlanQuery {
	if pq.driver.Dialect() == dialect.Postgres {
		pq.Unique(false)
	}
	pq.modifiers = append(pq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return pq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (pq *PlanQuery) ForShare(opts ...sql.LockOption) *PlanQuery {
	if pq.driver.Dialect() == dialect.Postgres {
		pq.Unique(false)
	}
	pq.modifiers = append(pq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return pq
}

// PlanGroupBy is the group-by builder for Plan entities.
type PlanGroupBy struct {
	selector
	build *PlanQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (pgb *PlanGroupBy) Aggregate(fns ...AggregateFunc) *PlanGroupBy {
	pgb.fns = append(pgb.fns, fns...)
	return pgb
}

// Scan applies the selector query and scans the result into the given value.
func (pgb *PlanGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, pgb.build.ctx, ent.OpQueryGroupBy)
	if err := pgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*PlanQuery, *PlanGroupBy](ctx, pgb.build, pgb, pgb.build.inters, v)
}

func (pgb *PlanGroupBy) sqlScan(ctx context.Context, root *PlanQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(pgb.fns))
	for _, fn := range pgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*pgb.flds)+len(pgb.fns))
		for _, f := range *pgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*pgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := pgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// PlanSelect is the builder for selecting fields of Plan entities.
type PlanSelect struct {
	*PlanQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (ps *PlanSelect) Aggregate(fns ...AggregateFunc) *PlanSelect {
	ps.fns = append(ps.fns, fns...)
	return ps
}

// Scan applies the selector query and scans the result into the given value.
func (ps *PlanSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, ps.ctx, ent.OpQuerySelect)
	if err := ps.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*PlanQuery, *PlanSelect](ctx, ps.PlanQuery, ps, ps.inters, v)
}

func (ps *PlanSelect) sqlScan(ctx context.Context, root *PlanQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(ps.fns))
	for _, fn := range ps.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*ps.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := ps.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
