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
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceflatfeelineconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicelinediscount"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceusagebasedlineconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// BillingInvoiceLineQuery is the builder for querying BillingInvoiceLine entities.
type BillingInvoiceLineQuery struct {
	config
	ctx                *QueryContext
	order              []billinginvoiceline.OrderOption
	inters             []Interceptor
	predicates         []predicate.BillingInvoiceLine
	withBillingInvoice *BillingInvoiceQuery
	withFlatFeeLine    *BillingInvoiceFlatFeeLineConfigQuery
	withUsageBasedLine *BillingInvoiceUsageBasedLineConfigQuery
	withParentLine     *BillingInvoiceLineQuery
	withDetailedLines  *BillingInvoiceLineQuery
	withLineDiscounts  *BillingInvoiceLineDiscountQuery
	withFKs            bool
	modifiers          []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the BillingInvoiceLineQuery builder.
func (bilq *BillingInvoiceLineQuery) Where(ps ...predicate.BillingInvoiceLine) *BillingInvoiceLineQuery {
	bilq.predicates = append(bilq.predicates, ps...)
	return bilq
}

// Limit the number of records to be returned by this query.
func (bilq *BillingInvoiceLineQuery) Limit(limit int) *BillingInvoiceLineQuery {
	bilq.ctx.Limit = &limit
	return bilq
}

// Offset to start from.
func (bilq *BillingInvoiceLineQuery) Offset(offset int) *BillingInvoiceLineQuery {
	bilq.ctx.Offset = &offset
	return bilq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (bilq *BillingInvoiceLineQuery) Unique(unique bool) *BillingInvoiceLineQuery {
	bilq.ctx.Unique = &unique
	return bilq
}

// Order specifies how the records should be ordered.
func (bilq *BillingInvoiceLineQuery) Order(o ...billinginvoiceline.OrderOption) *BillingInvoiceLineQuery {
	bilq.order = append(bilq.order, o...)
	return bilq
}

// QueryBillingInvoice chains the current query on the "billing_invoice" edge.
func (bilq *BillingInvoiceLineQuery) QueryBillingInvoice() *BillingInvoiceQuery {
	query := (&BillingInvoiceClient{config: bilq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := bilq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := bilq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoiceline.Table, billinginvoiceline.FieldID, selector),
			sqlgraph.To(billinginvoice.Table, billinginvoice.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, billinginvoiceline.BillingInvoiceTable, billinginvoiceline.BillingInvoiceColumn),
		)
		fromU = sqlgraph.SetNeighbors(bilq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryFlatFeeLine chains the current query on the "flat_fee_line" edge.
func (bilq *BillingInvoiceLineQuery) QueryFlatFeeLine() *BillingInvoiceFlatFeeLineConfigQuery {
	query := (&BillingInvoiceFlatFeeLineConfigClient{config: bilq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := bilq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := bilq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoiceline.Table, billinginvoiceline.FieldID, selector),
			sqlgraph.To(billinginvoiceflatfeelineconfig.Table, billinginvoiceflatfeelineconfig.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, billinginvoiceline.FlatFeeLineTable, billinginvoiceline.FlatFeeLineColumn),
		)
		fromU = sqlgraph.SetNeighbors(bilq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryUsageBasedLine chains the current query on the "usage_based_line" edge.
func (bilq *BillingInvoiceLineQuery) QueryUsageBasedLine() *BillingInvoiceUsageBasedLineConfigQuery {
	query := (&BillingInvoiceUsageBasedLineConfigClient{config: bilq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := bilq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := bilq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoiceline.Table, billinginvoiceline.FieldID, selector),
			sqlgraph.To(billinginvoiceusagebasedlineconfig.Table, billinginvoiceusagebasedlineconfig.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, false, billinginvoiceline.UsageBasedLineTable, billinginvoiceline.UsageBasedLineColumn),
		)
		fromU = sqlgraph.SetNeighbors(bilq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryParentLine chains the current query on the "parent_line" edge.
func (bilq *BillingInvoiceLineQuery) QueryParentLine() *BillingInvoiceLineQuery {
	query := (&BillingInvoiceLineClient{config: bilq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := bilq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := bilq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoiceline.Table, billinginvoiceline.FieldID, selector),
			sqlgraph.To(billinginvoiceline.Table, billinginvoiceline.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, billinginvoiceline.ParentLineTable, billinginvoiceline.ParentLineColumn),
		)
		fromU = sqlgraph.SetNeighbors(bilq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryDetailedLines chains the current query on the "detailed_lines" edge.
func (bilq *BillingInvoiceLineQuery) QueryDetailedLines() *BillingInvoiceLineQuery {
	query := (&BillingInvoiceLineClient{config: bilq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := bilq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := bilq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoiceline.Table, billinginvoiceline.FieldID, selector),
			sqlgraph.To(billinginvoiceline.Table, billinginvoiceline.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, billinginvoiceline.DetailedLinesTable, billinginvoiceline.DetailedLinesColumn),
		)
		fromU = sqlgraph.SetNeighbors(bilq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryLineDiscounts chains the current query on the "line_discounts" edge.
func (bilq *BillingInvoiceLineQuery) QueryLineDiscounts() *BillingInvoiceLineDiscountQuery {
	query := (&BillingInvoiceLineDiscountClient{config: bilq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := bilq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := bilq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoiceline.Table, billinginvoiceline.FieldID, selector),
			sqlgraph.To(billinginvoicelinediscount.Table, billinginvoicelinediscount.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, billinginvoiceline.LineDiscountsTable, billinginvoiceline.LineDiscountsColumn),
		)
		fromU = sqlgraph.SetNeighbors(bilq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first BillingInvoiceLine entity from the query.
// Returns a *NotFoundError when no BillingInvoiceLine was found.
func (bilq *BillingInvoiceLineQuery) First(ctx context.Context) (*BillingInvoiceLine, error) {
	nodes, err := bilq.Limit(1).All(setContextOp(ctx, bilq.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{billinginvoiceline.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (bilq *BillingInvoiceLineQuery) FirstX(ctx context.Context) *BillingInvoiceLine {
	node, err := bilq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first BillingInvoiceLine ID from the query.
// Returns a *NotFoundError when no BillingInvoiceLine ID was found.
func (bilq *BillingInvoiceLineQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = bilq.Limit(1).IDs(setContextOp(ctx, bilq.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{billinginvoiceline.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (bilq *BillingInvoiceLineQuery) FirstIDX(ctx context.Context) string {
	id, err := bilq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single BillingInvoiceLine entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one BillingInvoiceLine entity is found.
// Returns a *NotFoundError when no BillingInvoiceLine entities are found.
func (bilq *BillingInvoiceLineQuery) Only(ctx context.Context) (*BillingInvoiceLine, error) {
	nodes, err := bilq.Limit(2).All(setContextOp(ctx, bilq.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{billinginvoiceline.Label}
	default:
		return nil, &NotSingularError{billinginvoiceline.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (bilq *BillingInvoiceLineQuery) OnlyX(ctx context.Context) *BillingInvoiceLine {
	node, err := bilq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only BillingInvoiceLine ID in the query.
// Returns a *NotSingularError when more than one BillingInvoiceLine ID is found.
// Returns a *NotFoundError when no entities are found.
func (bilq *BillingInvoiceLineQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = bilq.Limit(2).IDs(setContextOp(ctx, bilq.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{billinginvoiceline.Label}
	default:
		err = &NotSingularError{billinginvoiceline.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (bilq *BillingInvoiceLineQuery) OnlyIDX(ctx context.Context) string {
	id, err := bilq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of BillingInvoiceLines.
func (bilq *BillingInvoiceLineQuery) All(ctx context.Context) ([]*BillingInvoiceLine, error) {
	ctx = setContextOp(ctx, bilq.ctx, ent.OpQueryAll)
	if err := bilq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*BillingInvoiceLine, *BillingInvoiceLineQuery]()
	return withInterceptors[[]*BillingInvoiceLine](ctx, bilq, qr, bilq.inters)
}

// AllX is like All, but panics if an error occurs.
func (bilq *BillingInvoiceLineQuery) AllX(ctx context.Context) []*BillingInvoiceLine {
	nodes, err := bilq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of BillingInvoiceLine IDs.
func (bilq *BillingInvoiceLineQuery) IDs(ctx context.Context) (ids []string, err error) {
	if bilq.ctx.Unique == nil && bilq.path != nil {
		bilq.Unique(true)
	}
	ctx = setContextOp(ctx, bilq.ctx, ent.OpQueryIDs)
	if err = bilq.Select(billinginvoiceline.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (bilq *BillingInvoiceLineQuery) IDsX(ctx context.Context) []string {
	ids, err := bilq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (bilq *BillingInvoiceLineQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, bilq.ctx, ent.OpQueryCount)
	if err := bilq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, bilq, querierCount[*BillingInvoiceLineQuery](), bilq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (bilq *BillingInvoiceLineQuery) CountX(ctx context.Context) int {
	count, err := bilq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (bilq *BillingInvoiceLineQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, bilq.ctx, ent.OpQueryExist)
	switch _, err := bilq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (bilq *BillingInvoiceLineQuery) ExistX(ctx context.Context) bool {
	exist, err := bilq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the BillingInvoiceLineQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (bilq *BillingInvoiceLineQuery) Clone() *BillingInvoiceLineQuery {
	if bilq == nil {
		return nil
	}
	return &BillingInvoiceLineQuery{
		config:             bilq.config,
		ctx:                bilq.ctx.Clone(),
		order:              append([]billinginvoiceline.OrderOption{}, bilq.order...),
		inters:             append([]Interceptor{}, bilq.inters...),
		predicates:         append([]predicate.BillingInvoiceLine{}, bilq.predicates...),
		withBillingInvoice: bilq.withBillingInvoice.Clone(),
		withFlatFeeLine:    bilq.withFlatFeeLine.Clone(),
		withUsageBasedLine: bilq.withUsageBasedLine.Clone(),
		withParentLine:     bilq.withParentLine.Clone(),
		withDetailedLines:  bilq.withDetailedLines.Clone(),
		withLineDiscounts:  bilq.withLineDiscounts.Clone(),
		// clone intermediate query.
		sql:  bilq.sql.Clone(),
		path: bilq.path,
	}
}

// WithBillingInvoice tells the query-builder to eager-load the nodes that are connected to
// the "billing_invoice" edge. The optional arguments are used to configure the query builder of the edge.
func (bilq *BillingInvoiceLineQuery) WithBillingInvoice(opts ...func(*BillingInvoiceQuery)) *BillingInvoiceLineQuery {
	query := (&BillingInvoiceClient{config: bilq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	bilq.withBillingInvoice = query
	return bilq
}

// WithFlatFeeLine tells the query-builder to eager-load the nodes that are connected to
// the "flat_fee_line" edge. The optional arguments are used to configure the query builder of the edge.
func (bilq *BillingInvoiceLineQuery) WithFlatFeeLine(opts ...func(*BillingInvoiceFlatFeeLineConfigQuery)) *BillingInvoiceLineQuery {
	query := (&BillingInvoiceFlatFeeLineConfigClient{config: bilq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	bilq.withFlatFeeLine = query
	return bilq
}

// WithUsageBasedLine tells the query-builder to eager-load the nodes that are connected to
// the "usage_based_line" edge. The optional arguments are used to configure the query builder of the edge.
func (bilq *BillingInvoiceLineQuery) WithUsageBasedLine(opts ...func(*BillingInvoiceUsageBasedLineConfigQuery)) *BillingInvoiceLineQuery {
	query := (&BillingInvoiceUsageBasedLineConfigClient{config: bilq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	bilq.withUsageBasedLine = query
	return bilq
}

// WithParentLine tells the query-builder to eager-load the nodes that are connected to
// the "parent_line" edge. The optional arguments are used to configure the query builder of the edge.
func (bilq *BillingInvoiceLineQuery) WithParentLine(opts ...func(*BillingInvoiceLineQuery)) *BillingInvoiceLineQuery {
	query := (&BillingInvoiceLineClient{config: bilq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	bilq.withParentLine = query
	return bilq
}

// WithDetailedLines tells the query-builder to eager-load the nodes that are connected to
// the "detailed_lines" edge. The optional arguments are used to configure the query builder of the edge.
func (bilq *BillingInvoiceLineQuery) WithDetailedLines(opts ...func(*BillingInvoiceLineQuery)) *BillingInvoiceLineQuery {
	query := (&BillingInvoiceLineClient{config: bilq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	bilq.withDetailedLines = query
	return bilq
}

// WithLineDiscounts tells the query-builder to eager-load the nodes that are connected to
// the "line_discounts" edge. The optional arguments are used to configure the query builder of the edge.
func (bilq *BillingInvoiceLineQuery) WithLineDiscounts(opts ...func(*BillingInvoiceLineDiscountQuery)) *BillingInvoiceLineQuery {
	query := (&BillingInvoiceLineDiscountClient{config: bilq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	bilq.withLineDiscounts = query
	return bilq
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
//	client.BillingInvoiceLine.Query().
//		GroupBy(billinginvoiceline.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (bilq *BillingInvoiceLineQuery) GroupBy(field string, fields ...string) *BillingInvoiceLineGroupBy {
	bilq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &BillingInvoiceLineGroupBy{build: bilq}
	grbuild.flds = &bilq.ctx.Fields
	grbuild.label = billinginvoiceline.Label
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
//	client.BillingInvoiceLine.Query().
//		Select(billinginvoiceline.FieldNamespace).
//		Scan(ctx, &v)
func (bilq *BillingInvoiceLineQuery) Select(fields ...string) *BillingInvoiceLineSelect {
	bilq.ctx.Fields = append(bilq.ctx.Fields, fields...)
	sbuild := &BillingInvoiceLineSelect{BillingInvoiceLineQuery: bilq}
	sbuild.label = billinginvoiceline.Label
	sbuild.flds, sbuild.scan = &bilq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a BillingInvoiceLineSelect configured with the given aggregations.
func (bilq *BillingInvoiceLineQuery) Aggregate(fns ...AggregateFunc) *BillingInvoiceLineSelect {
	return bilq.Select().Aggregate(fns...)
}

func (bilq *BillingInvoiceLineQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range bilq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, bilq); err != nil {
				return err
			}
		}
	}
	for _, f := range bilq.ctx.Fields {
		if !billinginvoiceline.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if bilq.path != nil {
		prev, err := bilq.path(ctx)
		if err != nil {
			return err
		}
		bilq.sql = prev
	}
	return nil
}

func (bilq *BillingInvoiceLineQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*BillingInvoiceLine, error) {
	var (
		nodes       = []*BillingInvoiceLine{}
		withFKs     = bilq.withFKs
		_spec       = bilq.querySpec()
		loadedTypes = [6]bool{
			bilq.withBillingInvoice != nil,
			bilq.withFlatFeeLine != nil,
			bilq.withUsageBasedLine != nil,
			bilq.withParentLine != nil,
			bilq.withDetailedLines != nil,
			bilq.withLineDiscounts != nil,
		}
	)
	if bilq.withFlatFeeLine != nil || bilq.withUsageBasedLine != nil {
		withFKs = true
	}
	if withFKs {
		_spec.Node.Columns = append(_spec.Node.Columns, billinginvoiceline.ForeignKeys...)
	}
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*BillingInvoiceLine).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &BillingInvoiceLine{config: bilq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(bilq.modifiers) > 0 {
		_spec.Modifiers = bilq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, bilq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := bilq.withBillingInvoice; query != nil {
		if err := bilq.loadBillingInvoice(ctx, query, nodes, nil,
			func(n *BillingInvoiceLine, e *BillingInvoice) { n.Edges.BillingInvoice = e }); err != nil {
			return nil, err
		}
	}
	if query := bilq.withFlatFeeLine; query != nil {
		if err := bilq.loadFlatFeeLine(ctx, query, nodes, nil,
			func(n *BillingInvoiceLine, e *BillingInvoiceFlatFeeLineConfig) { n.Edges.FlatFeeLine = e }); err != nil {
			return nil, err
		}
	}
	if query := bilq.withUsageBasedLine; query != nil {
		if err := bilq.loadUsageBasedLine(ctx, query, nodes, nil,
			func(n *BillingInvoiceLine, e *BillingInvoiceUsageBasedLineConfig) { n.Edges.UsageBasedLine = e }); err != nil {
			return nil, err
		}
	}
	if query := bilq.withParentLine; query != nil {
		if err := bilq.loadParentLine(ctx, query, nodes, nil,
			func(n *BillingInvoiceLine, e *BillingInvoiceLine) { n.Edges.ParentLine = e }); err != nil {
			return nil, err
		}
	}
	if query := bilq.withDetailedLines; query != nil {
		if err := bilq.loadDetailedLines(ctx, query, nodes,
			func(n *BillingInvoiceLine) { n.Edges.DetailedLines = []*BillingInvoiceLine{} },
			func(n *BillingInvoiceLine, e *BillingInvoiceLine) {
				n.Edges.DetailedLines = append(n.Edges.DetailedLines, e)
			}); err != nil {
			return nil, err
		}
	}
	if query := bilq.withLineDiscounts; query != nil {
		if err := bilq.loadLineDiscounts(ctx, query, nodes,
			func(n *BillingInvoiceLine) { n.Edges.LineDiscounts = []*BillingInvoiceLineDiscount{} },
			func(n *BillingInvoiceLine, e *BillingInvoiceLineDiscount) {
				n.Edges.LineDiscounts = append(n.Edges.LineDiscounts, e)
			}); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (bilq *BillingInvoiceLineQuery) loadBillingInvoice(ctx context.Context, query *BillingInvoiceQuery, nodes []*BillingInvoiceLine, init func(*BillingInvoiceLine), assign func(*BillingInvoiceLine, *BillingInvoice)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*BillingInvoiceLine)
	for i := range nodes {
		fk := nodes[i].InvoiceID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(billinginvoice.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "invoice_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (bilq *BillingInvoiceLineQuery) loadFlatFeeLine(ctx context.Context, query *BillingInvoiceFlatFeeLineConfigQuery, nodes []*BillingInvoiceLine, init func(*BillingInvoiceLine), assign func(*BillingInvoiceLine, *BillingInvoiceFlatFeeLineConfig)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*BillingInvoiceLine)
	for i := range nodes {
		if nodes[i].fee_line_config_id == nil {
			continue
		}
		fk := *nodes[i].fee_line_config_id
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(billinginvoiceflatfeelineconfig.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "fee_line_config_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (bilq *BillingInvoiceLineQuery) loadUsageBasedLine(ctx context.Context, query *BillingInvoiceUsageBasedLineConfigQuery, nodes []*BillingInvoiceLine, init func(*BillingInvoiceLine), assign func(*BillingInvoiceLine, *BillingInvoiceUsageBasedLineConfig)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*BillingInvoiceLine)
	for i := range nodes {
		if nodes[i].usage_based_line_config_id == nil {
			continue
		}
		fk := *nodes[i].usage_based_line_config_id
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(billinginvoiceusagebasedlineconfig.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "usage_based_line_config_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (bilq *BillingInvoiceLineQuery) loadParentLine(ctx context.Context, query *BillingInvoiceLineQuery, nodes []*BillingInvoiceLine, init func(*BillingInvoiceLine), assign func(*BillingInvoiceLine, *BillingInvoiceLine)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*BillingInvoiceLine)
	for i := range nodes {
		if nodes[i].ParentLineID == nil {
			continue
		}
		fk := *nodes[i].ParentLineID
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
			return fmt.Errorf(`unexpected foreign-key "parent_line_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (bilq *BillingInvoiceLineQuery) loadDetailedLines(ctx context.Context, query *BillingInvoiceLineQuery, nodes []*BillingInvoiceLine, init func(*BillingInvoiceLine), assign func(*BillingInvoiceLine, *BillingInvoiceLine)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*BillingInvoiceLine)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(billinginvoiceline.FieldParentLineID)
	}
	query.Where(predicate.BillingInvoiceLine(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(billinginvoiceline.DetailedLinesColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.ParentLineID
		if fk == nil {
			return fmt.Errorf(`foreign-key "parent_line_id" is nil for node %v`, n.ID)
		}
		node, ok := nodeids[*fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "parent_line_id" returned %v for node %v`, *fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (bilq *BillingInvoiceLineQuery) loadLineDiscounts(ctx context.Context, query *BillingInvoiceLineDiscountQuery, nodes []*BillingInvoiceLine, init func(*BillingInvoiceLine), assign func(*BillingInvoiceLine, *BillingInvoiceLineDiscount)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*BillingInvoiceLine)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(billinginvoicelinediscount.FieldLineID)
	}
	query.Where(predicate.BillingInvoiceLineDiscount(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(billinginvoiceline.LineDiscountsColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.LineID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "line_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}

func (bilq *BillingInvoiceLineQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := bilq.querySpec()
	if len(bilq.modifiers) > 0 {
		_spec.Modifiers = bilq.modifiers
	}
	_spec.Node.Columns = bilq.ctx.Fields
	if len(bilq.ctx.Fields) > 0 {
		_spec.Unique = bilq.ctx.Unique != nil && *bilq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, bilq.driver, _spec)
}

func (bilq *BillingInvoiceLineQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(billinginvoiceline.Table, billinginvoiceline.Columns, sqlgraph.NewFieldSpec(billinginvoiceline.FieldID, field.TypeString))
	_spec.From = bilq.sql
	if unique := bilq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if bilq.path != nil {
		_spec.Unique = true
	}
	if fields := bilq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, billinginvoiceline.FieldID)
		for i := range fields {
			if fields[i] != billinginvoiceline.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if bilq.withBillingInvoice != nil {
			_spec.Node.AddColumnOnce(billinginvoiceline.FieldInvoiceID)
		}
		if bilq.withParentLine != nil {
			_spec.Node.AddColumnOnce(billinginvoiceline.FieldParentLineID)
		}
	}
	if ps := bilq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := bilq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := bilq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := bilq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (bilq *BillingInvoiceLineQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(bilq.driver.Dialect())
	t1 := builder.Table(billinginvoiceline.Table)
	columns := bilq.ctx.Fields
	if len(columns) == 0 {
		columns = billinginvoiceline.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if bilq.sql != nil {
		selector = bilq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if bilq.ctx.Unique != nil && *bilq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range bilq.modifiers {
		m(selector)
	}
	for _, p := range bilq.predicates {
		p(selector)
	}
	for _, p := range bilq.order {
		p(selector)
	}
	if offset := bilq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := bilq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (bilq *BillingInvoiceLineQuery) ForUpdate(opts ...sql.LockOption) *BillingInvoiceLineQuery {
	if bilq.driver.Dialect() == dialect.Postgres {
		bilq.Unique(false)
	}
	bilq.modifiers = append(bilq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return bilq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (bilq *BillingInvoiceLineQuery) ForShare(opts ...sql.LockOption) *BillingInvoiceLineQuery {
	if bilq.driver.Dialect() == dialect.Postgres {
		bilq.Unique(false)
	}
	bilq.modifiers = append(bilq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return bilq
}

// BillingInvoiceLineGroupBy is the group-by builder for BillingInvoiceLine entities.
type BillingInvoiceLineGroupBy struct {
	selector
	build *BillingInvoiceLineQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (bilgb *BillingInvoiceLineGroupBy) Aggregate(fns ...AggregateFunc) *BillingInvoiceLineGroupBy {
	bilgb.fns = append(bilgb.fns, fns...)
	return bilgb
}

// Scan applies the selector query and scans the result into the given value.
func (bilgb *BillingInvoiceLineGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, bilgb.build.ctx, ent.OpQueryGroupBy)
	if err := bilgb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BillingInvoiceLineQuery, *BillingInvoiceLineGroupBy](ctx, bilgb.build, bilgb, bilgb.build.inters, v)
}

func (bilgb *BillingInvoiceLineGroupBy) sqlScan(ctx context.Context, root *BillingInvoiceLineQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(bilgb.fns))
	for _, fn := range bilgb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*bilgb.flds)+len(bilgb.fns))
		for _, f := range *bilgb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*bilgb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := bilgb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// BillingInvoiceLineSelect is the builder for selecting fields of BillingInvoiceLine entities.
type BillingInvoiceLineSelect struct {
	*BillingInvoiceLineQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (bils *BillingInvoiceLineSelect) Aggregate(fns ...AggregateFunc) *BillingInvoiceLineSelect {
	bils.fns = append(bils.fns, fns...)
	return bils
}

// Scan applies the selector query and scans the result into the given value.
func (bils *BillingInvoiceLineSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, bils.ctx, ent.OpQuerySelect)
	if err := bils.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BillingInvoiceLineQuery, *BillingInvoiceLineSelect](ctx, bils.BillingInvoiceLineQuery, bils, bils.inters, v)
}

func (bils *BillingInvoiceLineSelect) sqlScan(ctx context.Context, root *BillingInvoiceLineQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(bils.fns))
	for _, fn := range bils.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*bils.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := bils.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
