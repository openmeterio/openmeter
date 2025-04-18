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
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoiceline"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicevalidationissue"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingworkflowconfig"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// BillingInvoiceQuery is the builder for querying BillingInvoice entities.
type BillingInvoiceQuery struct {
	config
	ctx                                *QueryContext
	order                              []billinginvoice.OrderOption
	inters                             []Interceptor
	predicates                         []predicate.BillingInvoice
	withSourceBillingProfile           *BillingProfileQuery
	withBillingWorkflowConfig          *BillingWorkflowConfigQuery
	withBillingInvoiceLines            *BillingInvoiceLineQuery
	withBillingInvoiceValidationIssues *BillingInvoiceValidationIssueQuery
	withBillingInvoiceCustomer         *CustomerQuery
	withTaxApp                         *AppQuery
	withInvoicingApp                   *AppQuery
	withPaymentApp                     *AppQuery
	modifiers                          []func(*sql.Selector)
	// intermediate query (i.e. traversal path).
	sql  *sql.Selector
	path func(context.Context) (*sql.Selector, error)
}

// Where adds a new predicate for the BillingInvoiceQuery builder.
func (biq *BillingInvoiceQuery) Where(ps ...predicate.BillingInvoice) *BillingInvoiceQuery {
	biq.predicates = append(biq.predicates, ps...)
	return biq
}

// Limit the number of records to be returned by this query.
func (biq *BillingInvoiceQuery) Limit(limit int) *BillingInvoiceQuery {
	biq.ctx.Limit = &limit
	return biq
}

// Offset to start from.
func (biq *BillingInvoiceQuery) Offset(offset int) *BillingInvoiceQuery {
	biq.ctx.Offset = &offset
	return biq
}

// Unique configures the query builder to filter duplicate records on query.
// By default, unique is set to true, and can be disabled using this method.
func (biq *BillingInvoiceQuery) Unique(unique bool) *BillingInvoiceQuery {
	biq.ctx.Unique = &unique
	return biq
}

// Order specifies how the records should be ordered.
func (biq *BillingInvoiceQuery) Order(o ...billinginvoice.OrderOption) *BillingInvoiceQuery {
	biq.order = append(biq.order, o...)
	return biq
}

// QuerySourceBillingProfile chains the current query on the "source_billing_profile" edge.
func (biq *BillingInvoiceQuery) QuerySourceBillingProfile() *BillingProfileQuery {
	query := (&BillingProfileClient{config: biq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := biq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := biq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoice.Table, billinginvoice.FieldID, selector),
			sqlgraph.To(billingprofile.Table, billingprofile.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, billinginvoice.SourceBillingProfileTable, billinginvoice.SourceBillingProfileColumn),
		)
		fromU = sqlgraph.SetNeighbors(biq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryBillingWorkflowConfig chains the current query on the "billing_workflow_config" edge.
func (biq *BillingInvoiceQuery) QueryBillingWorkflowConfig() *BillingWorkflowConfigQuery {
	query := (&BillingWorkflowConfigClient{config: biq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := biq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := biq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoice.Table, billinginvoice.FieldID, selector),
			sqlgraph.To(billingworkflowconfig.Table, billingworkflowconfig.FieldID),
			sqlgraph.Edge(sqlgraph.O2O, true, billinginvoice.BillingWorkflowConfigTable, billinginvoice.BillingWorkflowConfigColumn),
		)
		fromU = sqlgraph.SetNeighbors(biq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryBillingInvoiceLines chains the current query on the "billing_invoice_lines" edge.
func (biq *BillingInvoiceQuery) QueryBillingInvoiceLines() *BillingInvoiceLineQuery {
	query := (&BillingInvoiceLineClient{config: biq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := biq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := biq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoice.Table, billinginvoice.FieldID, selector),
			sqlgraph.To(billinginvoiceline.Table, billinginvoiceline.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, billinginvoice.BillingInvoiceLinesTable, billinginvoice.BillingInvoiceLinesColumn),
		)
		fromU = sqlgraph.SetNeighbors(biq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryBillingInvoiceValidationIssues chains the current query on the "billing_invoice_validation_issues" edge.
func (biq *BillingInvoiceQuery) QueryBillingInvoiceValidationIssues() *BillingInvoiceValidationIssueQuery {
	query := (&BillingInvoiceValidationIssueClient{config: biq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := biq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := biq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoice.Table, billinginvoice.FieldID, selector),
			sqlgraph.To(billinginvoicevalidationissue.Table, billinginvoicevalidationissue.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, billinginvoice.BillingInvoiceValidationIssuesTable, billinginvoice.BillingInvoiceValidationIssuesColumn),
		)
		fromU = sqlgraph.SetNeighbors(biq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryBillingInvoiceCustomer chains the current query on the "billing_invoice_customer" edge.
func (biq *BillingInvoiceQuery) QueryBillingInvoiceCustomer() *CustomerQuery {
	query := (&CustomerClient{config: biq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := biq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := biq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoice.Table, billinginvoice.FieldID, selector),
			sqlgraph.To(customer.Table, customer.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, billinginvoice.BillingInvoiceCustomerTable, billinginvoice.BillingInvoiceCustomerColumn),
		)
		fromU = sqlgraph.SetNeighbors(biq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryTaxApp chains the current query on the "tax_app" edge.
func (biq *BillingInvoiceQuery) QueryTaxApp() *AppQuery {
	query := (&AppClient{config: biq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := biq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := biq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoice.Table, billinginvoice.FieldID, selector),
			sqlgraph.To(dbapp.Table, dbapp.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, billinginvoice.TaxAppTable, billinginvoice.TaxAppColumn),
		)
		fromU = sqlgraph.SetNeighbors(biq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryInvoicingApp chains the current query on the "invoicing_app" edge.
func (biq *BillingInvoiceQuery) QueryInvoicingApp() *AppQuery {
	query := (&AppClient{config: biq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := biq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := biq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoice.Table, billinginvoice.FieldID, selector),
			sqlgraph.To(dbapp.Table, dbapp.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, billinginvoice.InvoicingAppTable, billinginvoice.InvoicingAppColumn),
		)
		fromU = sqlgraph.SetNeighbors(biq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// QueryPaymentApp chains the current query on the "payment_app" edge.
func (biq *BillingInvoiceQuery) QueryPaymentApp() *AppQuery {
	query := (&AppClient{config: biq.config}).Query()
	query.path = func(ctx context.Context) (fromU *sql.Selector, err error) {
		if err := biq.prepareQuery(ctx); err != nil {
			return nil, err
		}
		selector := biq.sqlQuery(ctx)
		if err := selector.Err(); err != nil {
			return nil, err
		}
		step := sqlgraph.NewStep(
			sqlgraph.From(billinginvoice.Table, billinginvoice.FieldID, selector),
			sqlgraph.To(dbapp.Table, dbapp.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, billinginvoice.PaymentAppTable, billinginvoice.PaymentAppColumn),
		)
		fromU = sqlgraph.SetNeighbors(biq.driver.Dialect(), step)
		return fromU, nil
	}
	return query
}

// First returns the first BillingInvoice entity from the query.
// Returns a *NotFoundError when no BillingInvoice was found.
func (biq *BillingInvoiceQuery) First(ctx context.Context) (*BillingInvoice, error) {
	nodes, err := biq.Limit(1).All(setContextOp(ctx, biq.ctx, ent.OpQueryFirst))
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, &NotFoundError{billinginvoice.Label}
	}
	return nodes[0], nil
}

// FirstX is like First, but panics if an error occurs.
func (biq *BillingInvoiceQuery) FirstX(ctx context.Context) *BillingInvoice {
	node, err := biq.First(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return node
}

// FirstID returns the first BillingInvoice ID from the query.
// Returns a *NotFoundError when no BillingInvoice ID was found.
func (biq *BillingInvoiceQuery) FirstID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = biq.Limit(1).IDs(setContextOp(ctx, biq.ctx, ent.OpQueryFirstID)); err != nil {
		return
	}
	if len(ids) == 0 {
		err = &NotFoundError{billinginvoice.Label}
		return
	}
	return ids[0], nil
}

// FirstIDX is like FirstID, but panics if an error occurs.
func (biq *BillingInvoiceQuery) FirstIDX(ctx context.Context) string {
	id, err := biq.FirstID(ctx)
	if err != nil && !IsNotFound(err) {
		panic(err)
	}
	return id
}

// Only returns a single BillingInvoice entity found by the query, ensuring it only returns one.
// Returns a *NotSingularError when more than one BillingInvoice entity is found.
// Returns a *NotFoundError when no BillingInvoice entities are found.
func (biq *BillingInvoiceQuery) Only(ctx context.Context) (*BillingInvoice, error) {
	nodes, err := biq.Limit(2).All(setContextOp(ctx, biq.ctx, ent.OpQueryOnly))
	if err != nil {
		return nil, err
	}
	switch len(nodes) {
	case 1:
		return nodes[0], nil
	case 0:
		return nil, &NotFoundError{billinginvoice.Label}
	default:
		return nil, &NotSingularError{billinginvoice.Label}
	}
}

// OnlyX is like Only, but panics if an error occurs.
func (biq *BillingInvoiceQuery) OnlyX(ctx context.Context) *BillingInvoice {
	node, err := biq.Only(ctx)
	if err != nil {
		panic(err)
	}
	return node
}

// OnlyID is like Only, but returns the only BillingInvoice ID in the query.
// Returns a *NotSingularError when more than one BillingInvoice ID is found.
// Returns a *NotFoundError when no entities are found.
func (biq *BillingInvoiceQuery) OnlyID(ctx context.Context) (id string, err error) {
	var ids []string
	if ids, err = biq.Limit(2).IDs(setContextOp(ctx, biq.ctx, ent.OpQueryOnlyID)); err != nil {
		return
	}
	switch len(ids) {
	case 1:
		id = ids[0]
	case 0:
		err = &NotFoundError{billinginvoice.Label}
	default:
		err = &NotSingularError{billinginvoice.Label}
	}
	return
}

// OnlyIDX is like OnlyID, but panics if an error occurs.
func (biq *BillingInvoiceQuery) OnlyIDX(ctx context.Context) string {
	id, err := biq.OnlyID(ctx)
	if err != nil {
		panic(err)
	}
	return id
}

// All executes the query and returns a list of BillingInvoices.
func (biq *BillingInvoiceQuery) All(ctx context.Context) ([]*BillingInvoice, error) {
	ctx = setContextOp(ctx, biq.ctx, ent.OpQueryAll)
	if err := biq.prepareQuery(ctx); err != nil {
		return nil, err
	}
	qr := querierAll[[]*BillingInvoice, *BillingInvoiceQuery]()
	return withInterceptors[[]*BillingInvoice](ctx, biq, qr, biq.inters)
}

// AllX is like All, but panics if an error occurs.
func (biq *BillingInvoiceQuery) AllX(ctx context.Context) []*BillingInvoice {
	nodes, err := biq.All(ctx)
	if err != nil {
		panic(err)
	}
	return nodes
}

// IDs executes the query and returns a list of BillingInvoice IDs.
func (biq *BillingInvoiceQuery) IDs(ctx context.Context) (ids []string, err error) {
	if biq.ctx.Unique == nil && biq.path != nil {
		biq.Unique(true)
	}
	ctx = setContextOp(ctx, biq.ctx, ent.OpQueryIDs)
	if err = biq.Select(billinginvoice.FieldID).Scan(ctx, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// IDsX is like IDs, but panics if an error occurs.
func (biq *BillingInvoiceQuery) IDsX(ctx context.Context) []string {
	ids, err := biq.IDs(ctx)
	if err != nil {
		panic(err)
	}
	return ids
}

// Count returns the count of the given query.
func (biq *BillingInvoiceQuery) Count(ctx context.Context) (int, error) {
	ctx = setContextOp(ctx, biq.ctx, ent.OpQueryCount)
	if err := biq.prepareQuery(ctx); err != nil {
		return 0, err
	}
	return withInterceptors[int](ctx, biq, querierCount[*BillingInvoiceQuery](), biq.inters)
}

// CountX is like Count, but panics if an error occurs.
func (biq *BillingInvoiceQuery) CountX(ctx context.Context) int {
	count, err := biq.Count(ctx)
	if err != nil {
		panic(err)
	}
	return count
}

// Exist returns true if the query has elements in the graph.
func (biq *BillingInvoiceQuery) Exist(ctx context.Context) (bool, error) {
	ctx = setContextOp(ctx, biq.ctx, ent.OpQueryExist)
	switch _, err := biq.FirstID(ctx); {
	case IsNotFound(err):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("db: check existence: %w", err)
	default:
		return true, nil
	}
}

// ExistX is like Exist, but panics if an error occurs.
func (biq *BillingInvoiceQuery) ExistX(ctx context.Context) bool {
	exist, err := biq.Exist(ctx)
	if err != nil {
		panic(err)
	}
	return exist
}

// Clone returns a duplicate of the BillingInvoiceQuery builder, including all associated steps. It can be
// used to prepare common query builders and use them differently after the clone is made.
func (biq *BillingInvoiceQuery) Clone() *BillingInvoiceQuery {
	if biq == nil {
		return nil
	}
	return &BillingInvoiceQuery{
		config:                             biq.config,
		ctx:                                biq.ctx.Clone(),
		order:                              append([]billinginvoice.OrderOption{}, biq.order...),
		inters:                             append([]Interceptor{}, biq.inters...),
		predicates:                         append([]predicate.BillingInvoice{}, biq.predicates...),
		withSourceBillingProfile:           biq.withSourceBillingProfile.Clone(),
		withBillingWorkflowConfig:          biq.withBillingWorkflowConfig.Clone(),
		withBillingInvoiceLines:            biq.withBillingInvoiceLines.Clone(),
		withBillingInvoiceValidationIssues: biq.withBillingInvoiceValidationIssues.Clone(),
		withBillingInvoiceCustomer:         biq.withBillingInvoiceCustomer.Clone(),
		withTaxApp:                         biq.withTaxApp.Clone(),
		withInvoicingApp:                   biq.withInvoicingApp.Clone(),
		withPaymentApp:                     biq.withPaymentApp.Clone(),
		// clone intermediate query.
		sql:  biq.sql.Clone(),
		path: biq.path,
	}
}

// WithSourceBillingProfile tells the query-builder to eager-load the nodes that are connected to
// the "source_billing_profile" edge. The optional arguments are used to configure the query builder of the edge.
func (biq *BillingInvoiceQuery) WithSourceBillingProfile(opts ...func(*BillingProfileQuery)) *BillingInvoiceQuery {
	query := (&BillingProfileClient{config: biq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	biq.withSourceBillingProfile = query
	return biq
}

// WithBillingWorkflowConfig tells the query-builder to eager-load the nodes that are connected to
// the "billing_workflow_config" edge. The optional arguments are used to configure the query builder of the edge.
func (biq *BillingInvoiceQuery) WithBillingWorkflowConfig(opts ...func(*BillingWorkflowConfigQuery)) *BillingInvoiceQuery {
	query := (&BillingWorkflowConfigClient{config: biq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	biq.withBillingWorkflowConfig = query
	return biq
}

// WithBillingInvoiceLines tells the query-builder to eager-load the nodes that are connected to
// the "billing_invoice_lines" edge. The optional arguments are used to configure the query builder of the edge.
func (biq *BillingInvoiceQuery) WithBillingInvoiceLines(opts ...func(*BillingInvoiceLineQuery)) *BillingInvoiceQuery {
	query := (&BillingInvoiceLineClient{config: biq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	biq.withBillingInvoiceLines = query
	return biq
}

// WithBillingInvoiceValidationIssues tells the query-builder to eager-load the nodes that are connected to
// the "billing_invoice_validation_issues" edge. The optional arguments are used to configure the query builder of the edge.
func (biq *BillingInvoiceQuery) WithBillingInvoiceValidationIssues(opts ...func(*BillingInvoiceValidationIssueQuery)) *BillingInvoiceQuery {
	query := (&BillingInvoiceValidationIssueClient{config: biq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	biq.withBillingInvoiceValidationIssues = query
	return biq
}

// WithBillingInvoiceCustomer tells the query-builder to eager-load the nodes that are connected to
// the "billing_invoice_customer" edge. The optional arguments are used to configure the query builder of the edge.
func (biq *BillingInvoiceQuery) WithBillingInvoiceCustomer(opts ...func(*CustomerQuery)) *BillingInvoiceQuery {
	query := (&CustomerClient{config: biq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	biq.withBillingInvoiceCustomer = query
	return biq
}

// WithTaxApp tells the query-builder to eager-load the nodes that are connected to
// the "tax_app" edge. The optional arguments are used to configure the query builder of the edge.
func (biq *BillingInvoiceQuery) WithTaxApp(opts ...func(*AppQuery)) *BillingInvoiceQuery {
	query := (&AppClient{config: biq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	biq.withTaxApp = query
	return biq
}

// WithInvoicingApp tells the query-builder to eager-load the nodes that are connected to
// the "invoicing_app" edge. The optional arguments are used to configure the query builder of the edge.
func (biq *BillingInvoiceQuery) WithInvoicingApp(opts ...func(*AppQuery)) *BillingInvoiceQuery {
	query := (&AppClient{config: biq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	biq.withInvoicingApp = query
	return biq
}

// WithPaymentApp tells the query-builder to eager-load the nodes that are connected to
// the "payment_app" edge. The optional arguments are used to configure the query builder of the edge.
func (biq *BillingInvoiceQuery) WithPaymentApp(opts ...func(*AppQuery)) *BillingInvoiceQuery {
	query := (&AppClient{config: biq.config}).Query()
	for _, opt := range opts {
		opt(query)
	}
	biq.withPaymentApp = query
	return biq
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
//	client.BillingInvoice.Query().
//		GroupBy(billinginvoice.FieldNamespace).
//		Aggregate(db.Count()).
//		Scan(ctx, &v)
func (biq *BillingInvoiceQuery) GroupBy(field string, fields ...string) *BillingInvoiceGroupBy {
	biq.ctx.Fields = append([]string{field}, fields...)
	grbuild := &BillingInvoiceGroupBy{build: biq}
	grbuild.flds = &biq.ctx.Fields
	grbuild.label = billinginvoice.Label
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
//	client.BillingInvoice.Query().
//		Select(billinginvoice.FieldNamespace).
//		Scan(ctx, &v)
func (biq *BillingInvoiceQuery) Select(fields ...string) *BillingInvoiceSelect {
	biq.ctx.Fields = append(biq.ctx.Fields, fields...)
	sbuild := &BillingInvoiceSelect{BillingInvoiceQuery: biq}
	sbuild.label = billinginvoice.Label
	sbuild.flds, sbuild.scan = &biq.ctx.Fields, sbuild.Scan
	return sbuild
}

// Aggregate returns a BillingInvoiceSelect configured with the given aggregations.
func (biq *BillingInvoiceQuery) Aggregate(fns ...AggregateFunc) *BillingInvoiceSelect {
	return biq.Select().Aggregate(fns...)
}

func (biq *BillingInvoiceQuery) prepareQuery(ctx context.Context) error {
	for _, inter := range biq.inters {
		if inter == nil {
			return fmt.Errorf("db: uninitialized interceptor (forgotten import db/runtime?)")
		}
		if trv, ok := inter.(Traverser); ok {
			if err := trv.Traverse(ctx, biq); err != nil {
				return err
			}
		}
	}
	for _, f := range biq.ctx.Fields {
		if !billinginvoice.ValidColumn(f) {
			return &ValidationError{Name: f, err: fmt.Errorf("db: invalid field %q for query", f)}
		}
	}
	if biq.path != nil {
		prev, err := biq.path(ctx)
		if err != nil {
			return err
		}
		biq.sql = prev
	}
	return nil
}

func (biq *BillingInvoiceQuery) sqlAll(ctx context.Context, hooks ...queryHook) ([]*BillingInvoice, error) {
	var (
		nodes       = []*BillingInvoice{}
		_spec       = biq.querySpec()
		loadedTypes = [8]bool{
			biq.withSourceBillingProfile != nil,
			biq.withBillingWorkflowConfig != nil,
			biq.withBillingInvoiceLines != nil,
			biq.withBillingInvoiceValidationIssues != nil,
			biq.withBillingInvoiceCustomer != nil,
			biq.withTaxApp != nil,
			biq.withInvoicingApp != nil,
			biq.withPaymentApp != nil,
		}
	)
	_spec.ScanValues = func(columns []string) ([]any, error) {
		return (*BillingInvoice).scanValues(nil, columns)
	}
	_spec.Assign = func(columns []string, values []any) error {
		node := &BillingInvoice{config: biq.config}
		nodes = append(nodes, node)
		node.Edges.loadedTypes = loadedTypes
		return node.assignValues(columns, values)
	}
	if len(biq.modifiers) > 0 {
		_spec.Modifiers = biq.modifiers
	}
	for i := range hooks {
		hooks[i](ctx, _spec)
	}
	if err := sqlgraph.QueryNodes(ctx, biq.driver, _spec); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if query := biq.withSourceBillingProfile; query != nil {
		if err := biq.loadSourceBillingProfile(ctx, query, nodes, nil,
			func(n *BillingInvoice, e *BillingProfile) { n.Edges.SourceBillingProfile = e }); err != nil {
			return nil, err
		}
	}
	if query := biq.withBillingWorkflowConfig; query != nil {
		if err := biq.loadBillingWorkflowConfig(ctx, query, nodes, nil,
			func(n *BillingInvoice, e *BillingWorkflowConfig) { n.Edges.BillingWorkflowConfig = e }); err != nil {
			return nil, err
		}
	}
	if query := biq.withBillingInvoiceLines; query != nil {
		if err := biq.loadBillingInvoiceLines(ctx, query, nodes,
			func(n *BillingInvoice) { n.Edges.BillingInvoiceLines = []*BillingInvoiceLine{} },
			func(n *BillingInvoice, e *BillingInvoiceLine) {
				n.Edges.BillingInvoiceLines = append(n.Edges.BillingInvoiceLines, e)
			}); err != nil {
			return nil, err
		}
	}
	if query := biq.withBillingInvoiceValidationIssues; query != nil {
		if err := biq.loadBillingInvoiceValidationIssues(ctx, query, nodes,
			func(n *BillingInvoice) { n.Edges.BillingInvoiceValidationIssues = []*BillingInvoiceValidationIssue{} },
			func(n *BillingInvoice, e *BillingInvoiceValidationIssue) {
				n.Edges.BillingInvoiceValidationIssues = append(n.Edges.BillingInvoiceValidationIssues, e)
			}); err != nil {
			return nil, err
		}
	}
	if query := biq.withBillingInvoiceCustomer; query != nil {
		if err := biq.loadBillingInvoiceCustomer(ctx, query, nodes, nil,
			func(n *BillingInvoice, e *Customer) { n.Edges.BillingInvoiceCustomer = e }); err != nil {
			return nil, err
		}
	}
	if query := biq.withTaxApp; query != nil {
		if err := biq.loadTaxApp(ctx, query, nodes, nil,
			func(n *BillingInvoice, e *App) { n.Edges.TaxApp = e }); err != nil {
			return nil, err
		}
	}
	if query := biq.withInvoicingApp; query != nil {
		if err := biq.loadInvoicingApp(ctx, query, nodes, nil,
			func(n *BillingInvoice, e *App) { n.Edges.InvoicingApp = e }); err != nil {
			return nil, err
		}
	}
	if query := biq.withPaymentApp; query != nil {
		if err := biq.loadPaymentApp(ctx, query, nodes, nil,
			func(n *BillingInvoice, e *App) { n.Edges.PaymentApp = e }); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

func (biq *BillingInvoiceQuery) loadSourceBillingProfile(ctx context.Context, query *BillingProfileQuery, nodes []*BillingInvoice, init func(*BillingInvoice), assign func(*BillingInvoice, *BillingProfile)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*BillingInvoice)
	for i := range nodes {
		fk := nodes[i].SourceBillingProfileID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(billingprofile.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "source_billing_profile_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (biq *BillingInvoiceQuery) loadBillingWorkflowConfig(ctx context.Context, query *BillingWorkflowConfigQuery, nodes []*BillingInvoice, init func(*BillingInvoice), assign func(*BillingInvoice, *BillingWorkflowConfig)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*BillingInvoice)
	for i := range nodes {
		fk := nodes[i].WorkflowConfigID
		if _, ok := nodeids[fk]; !ok {
			ids = append(ids, fk)
		}
		nodeids[fk] = append(nodeids[fk], nodes[i])
	}
	if len(ids) == 0 {
		return nil
	}
	query.Where(billingworkflowconfig.IDIn(ids...))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		nodes, ok := nodeids[n.ID]
		if !ok {
			return fmt.Errorf(`unexpected foreign-key "workflow_config_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (biq *BillingInvoiceQuery) loadBillingInvoiceLines(ctx context.Context, query *BillingInvoiceLineQuery, nodes []*BillingInvoice, init func(*BillingInvoice), assign func(*BillingInvoice, *BillingInvoiceLine)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*BillingInvoice)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	query.withFKs = true
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(billinginvoiceline.FieldInvoiceID)
	}
	query.Where(predicate.BillingInvoiceLine(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(billinginvoice.BillingInvoiceLinesColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.InvoiceID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "invoice_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (biq *BillingInvoiceQuery) loadBillingInvoiceValidationIssues(ctx context.Context, query *BillingInvoiceValidationIssueQuery, nodes []*BillingInvoice, init func(*BillingInvoice), assign func(*BillingInvoice, *BillingInvoiceValidationIssue)) error {
	fks := make([]driver.Value, 0, len(nodes))
	nodeids := make(map[string]*BillingInvoice)
	for i := range nodes {
		fks = append(fks, nodes[i].ID)
		nodeids[nodes[i].ID] = nodes[i]
		if init != nil {
			init(nodes[i])
		}
	}
	if len(query.ctx.Fields) > 0 {
		query.ctx.AppendFieldOnce(billinginvoicevalidationissue.FieldInvoiceID)
	}
	query.Where(predicate.BillingInvoiceValidationIssue(func(s *sql.Selector) {
		s.Where(sql.InValues(s.C(billinginvoice.BillingInvoiceValidationIssuesColumn), fks...))
	}))
	neighbors, err := query.All(ctx)
	if err != nil {
		return err
	}
	for _, n := range neighbors {
		fk := n.InvoiceID
		node, ok := nodeids[fk]
		if !ok {
			return fmt.Errorf(`unexpected referenced foreign-key "invoice_id" returned %v for node %v`, fk, n.ID)
		}
		assign(node, n)
	}
	return nil
}
func (biq *BillingInvoiceQuery) loadBillingInvoiceCustomer(ctx context.Context, query *CustomerQuery, nodes []*BillingInvoice, init func(*BillingInvoice), assign func(*BillingInvoice, *Customer)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*BillingInvoice)
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
func (biq *BillingInvoiceQuery) loadTaxApp(ctx context.Context, query *AppQuery, nodes []*BillingInvoice, init func(*BillingInvoice), assign func(*BillingInvoice, *App)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*BillingInvoice)
	for i := range nodes {
		fk := nodes[i].TaxAppID
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
			return fmt.Errorf(`unexpected foreign-key "tax_app_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (biq *BillingInvoiceQuery) loadInvoicingApp(ctx context.Context, query *AppQuery, nodes []*BillingInvoice, init func(*BillingInvoice), assign func(*BillingInvoice, *App)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*BillingInvoice)
	for i := range nodes {
		fk := nodes[i].InvoicingAppID
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
			return fmt.Errorf(`unexpected foreign-key "invoicing_app_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}
func (biq *BillingInvoiceQuery) loadPaymentApp(ctx context.Context, query *AppQuery, nodes []*BillingInvoice, init func(*BillingInvoice), assign func(*BillingInvoice, *App)) error {
	ids := make([]string, 0, len(nodes))
	nodeids := make(map[string][]*BillingInvoice)
	for i := range nodes {
		fk := nodes[i].PaymentAppID
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
			return fmt.Errorf(`unexpected foreign-key "payment_app_id" returned %v`, n.ID)
		}
		for i := range nodes {
			assign(nodes[i], n)
		}
	}
	return nil
}

func (biq *BillingInvoiceQuery) sqlCount(ctx context.Context) (int, error) {
	_spec := biq.querySpec()
	if len(biq.modifiers) > 0 {
		_spec.Modifiers = biq.modifiers
	}
	_spec.Node.Columns = biq.ctx.Fields
	if len(biq.ctx.Fields) > 0 {
		_spec.Unique = biq.ctx.Unique != nil && *biq.ctx.Unique
	}
	return sqlgraph.CountNodes(ctx, biq.driver, _spec)
}

func (biq *BillingInvoiceQuery) querySpec() *sqlgraph.QuerySpec {
	_spec := sqlgraph.NewQuerySpec(billinginvoice.Table, billinginvoice.Columns, sqlgraph.NewFieldSpec(billinginvoice.FieldID, field.TypeString))
	_spec.From = biq.sql
	if unique := biq.ctx.Unique; unique != nil {
		_spec.Unique = *unique
	} else if biq.path != nil {
		_spec.Unique = true
	}
	if fields := biq.ctx.Fields; len(fields) > 0 {
		_spec.Node.Columns = make([]string, 0, len(fields))
		_spec.Node.Columns = append(_spec.Node.Columns, billinginvoice.FieldID)
		for i := range fields {
			if fields[i] != billinginvoice.FieldID {
				_spec.Node.Columns = append(_spec.Node.Columns, fields[i])
			}
		}
		if biq.withSourceBillingProfile != nil {
			_spec.Node.AddColumnOnce(billinginvoice.FieldSourceBillingProfileID)
		}
		if biq.withBillingWorkflowConfig != nil {
			_spec.Node.AddColumnOnce(billinginvoice.FieldWorkflowConfigID)
		}
		if biq.withBillingInvoiceCustomer != nil {
			_spec.Node.AddColumnOnce(billinginvoice.FieldCustomerID)
		}
		if biq.withTaxApp != nil {
			_spec.Node.AddColumnOnce(billinginvoice.FieldTaxAppID)
		}
		if biq.withInvoicingApp != nil {
			_spec.Node.AddColumnOnce(billinginvoice.FieldInvoicingAppID)
		}
		if biq.withPaymentApp != nil {
			_spec.Node.AddColumnOnce(billinginvoice.FieldPaymentAppID)
		}
	}
	if ps := biq.predicates; len(ps) > 0 {
		_spec.Predicate = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	if limit := biq.ctx.Limit; limit != nil {
		_spec.Limit = *limit
	}
	if offset := biq.ctx.Offset; offset != nil {
		_spec.Offset = *offset
	}
	if ps := biq.order; len(ps) > 0 {
		_spec.Order = func(selector *sql.Selector) {
			for i := range ps {
				ps[i](selector)
			}
		}
	}
	return _spec
}

func (biq *BillingInvoiceQuery) sqlQuery(ctx context.Context) *sql.Selector {
	builder := sql.Dialect(biq.driver.Dialect())
	t1 := builder.Table(billinginvoice.Table)
	columns := biq.ctx.Fields
	if len(columns) == 0 {
		columns = billinginvoice.Columns
	}
	selector := builder.Select(t1.Columns(columns...)...).From(t1)
	if biq.sql != nil {
		selector = biq.sql
		selector.Select(selector.Columns(columns...)...)
	}
	if biq.ctx.Unique != nil && *biq.ctx.Unique {
		selector.Distinct()
	}
	for _, m := range biq.modifiers {
		m(selector)
	}
	for _, p := range biq.predicates {
		p(selector)
	}
	for _, p := range biq.order {
		p(selector)
	}
	if offset := biq.ctx.Offset; offset != nil {
		// limit is mandatory for offset clause. We start
		// with default value, and override it below if needed.
		selector.Offset(*offset).Limit(math.MaxInt32)
	}
	if limit := biq.ctx.Limit; limit != nil {
		selector.Limit(*limit)
	}
	return selector
}

// ForUpdate locks the selected rows against concurrent updates, and prevent them from being
// updated, deleted or "selected ... for update" by other sessions, until the transaction is
// either committed or rolled-back.
func (biq *BillingInvoiceQuery) ForUpdate(opts ...sql.LockOption) *BillingInvoiceQuery {
	if biq.driver.Dialect() == dialect.Postgres {
		biq.Unique(false)
	}
	biq.modifiers = append(biq.modifiers, func(s *sql.Selector) {
		s.ForUpdate(opts...)
	})
	return biq
}

// ForShare behaves similarly to ForUpdate, except that it acquires a shared mode lock
// on any rows that are read. Other sessions can read the rows, but cannot modify them
// until your transaction commits.
func (biq *BillingInvoiceQuery) ForShare(opts ...sql.LockOption) *BillingInvoiceQuery {
	if biq.driver.Dialect() == dialect.Postgres {
		biq.Unique(false)
	}
	biq.modifiers = append(biq.modifiers, func(s *sql.Selector) {
		s.ForShare(opts...)
	})
	return biq
}

// BillingInvoiceGroupBy is the group-by builder for BillingInvoice entities.
type BillingInvoiceGroupBy struct {
	selector
	build *BillingInvoiceQuery
}

// Aggregate adds the given aggregation functions to the group-by query.
func (bigb *BillingInvoiceGroupBy) Aggregate(fns ...AggregateFunc) *BillingInvoiceGroupBy {
	bigb.fns = append(bigb.fns, fns...)
	return bigb
}

// Scan applies the selector query and scans the result into the given value.
func (bigb *BillingInvoiceGroupBy) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, bigb.build.ctx, ent.OpQueryGroupBy)
	if err := bigb.build.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BillingInvoiceQuery, *BillingInvoiceGroupBy](ctx, bigb.build, bigb, bigb.build.inters, v)
}

func (bigb *BillingInvoiceGroupBy) sqlScan(ctx context.Context, root *BillingInvoiceQuery, v any) error {
	selector := root.sqlQuery(ctx).Select()
	aggregation := make([]string, 0, len(bigb.fns))
	for _, fn := range bigb.fns {
		aggregation = append(aggregation, fn(selector))
	}
	if len(selector.SelectedColumns()) == 0 {
		columns := make([]string, 0, len(*bigb.flds)+len(bigb.fns))
		for _, f := range *bigb.flds {
			columns = append(columns, selector.C(f))
		}
		columns = append(columns, aggregation...)
		selector.Select(columns...)
	}
	selector.GroupBy(selector.Columns(*bigb.flds...)...)
	if err := selector.Err(); err != nil {
		return err
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := bigb.build.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}

// BillingInvoiceSelect is the builder for selecting fields of BillingInvoice entities.
type BillingInvoiceSelect struct {
	*BillingInvoiceQuery
	selector
}

// Aggregate adds the given aggregation functions to the selector query.
func (bis *BillingInvoiceSelect) Aggregate(fns ...AggregateFunc) *BillingInvoiceSelect {
	bis.fns = append(bis.fns, fns...)
	return bis
}

// Scan applies the selector query and scans the result into the given value.
func (bis *BillingInvoiceSelect) Scan(ctx context.Context, v any) error {
	ctx = setContextOp(ctx, bis.ctx, ent.OpQuerySelect)
	if err := bis.prepareQuery(ctx); err != nil {
		return err
	}
	return scanWithInterceptors[*BillingInvoiceQuery, *BillingInvoiceSelect](ctx, bis.BillingInvoiceQuery, bis, bis.inters, v)
}

func (bis *BillingInvoiceSelect) sqlScan(ctx context.Context, root *BillingInvoiceQuery, v any) error {
	selector := root.sqlQuery(ctx)
	aggregation := make([]string, 0, len(bis.fns))
	for _, fn := range bis.fns {
		aggregation = append(aggregation, fn(selector))
	}
	switch n := len(*bis.selector.flds); {
	case n == 0 && len(aggregation) > 0:
		selector.Select(aggregation...)
	case n != 0 && len(aggregation) > 0:
		selector.AppendSelect(aggregation...)
	}
	rows := &sql.Rows{}
	query, args := selector.Query()
	if err := bis.driver.Query(ctx, query, args, rows); err != nil {
		return err
	}
	defer rows.Close()
	return sql.ScanSlice(rows, v)
}
