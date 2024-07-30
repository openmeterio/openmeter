// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/openmeterio/openmeter/internal/ent/db/migrate"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/internal/ent/db/balancesnapshot"
	"github.com/openmeterio/openmeter/internal/ent/db/entitlement"
	"github.com/openmeterio/openmeter/internal/ent/db/feature"
	dbgrant "github.com/openmeterio/openmeter/internal/ent/db/grant"
	"github.com/openmeterio/openmeter/internal/ent/db/usagereset"
)

// Client is the client that holds all ent builders.
type Client struct {
	config
	// Schema is the client for creating, migrating and dropping schema.
	Schema *migrate.Schema
	// BalanceSnapshot is the client for interacting with the BalanceSnapshot builders.
	BalanceSnapshot *BalanceSnapshotClient
	// Entitlement is the client for interacting with the Entitlement builders.
	Entitlement *EntitlementClient
	// Feature is the client for interacting with the Feature builders.
	Feature *FeatureClient
	// Grant is the client for interacting with the Grant builders.
	Grant *GrantClient
	// UsageReset is the client for interacting with the UsageReset builders.
	UsageReset *UsageResetClient
}

// NewClient creates a new client configured with the given options.
func NewClient(opts ...Option) *Client {
	client := &Client{config: newConfig(opts...)}
	client.init()
	return client
}

func (c *Client) init() {
	c.Schema = migrate.NewSchema(c.driver)
	c.BalanceSnapshot = NewBalanceSnapshotClient(c.config)
	c.Entitlement = NewEntitlementClient(c.config)
	c.Feature = NewFeatureClient(c.config)
	c.Grant = NewGrantClient(c.config)
	c.UsageReset = NewUsageResetClient(c.config)
}

type (
	// config is the configuration for the client and its builder.
	config struct {
		// driver used for executing database requests.
		driver dialect.Driver
		// debug enable a debug logging.
		debug bool
		// log used for logging on debug mode.
		log func(...any)
		// hooks to execute on mutations.
		hooks *hooks
		// interceptors to execute on queries.
		inters *inters
	}
	// Option function to configure the client.
	Option func(*config)
)

// newConfig creates a new config for the client.
func newConfig(opts ...Option) config {
	cfg := config{log: log.Println, hooks: &hooks{}, inters: &inters{}}
	cfg.options(opts...)
	return cfg
}

// options applies the options on the config object.
func (c *config) options(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
	if c.debug {
		c.driver = dialect.Debug(c.driver, c.log)
	}
}

// Debug enables debug logging on the ent.Driver.
func Debug() Option {
	return func(c *config) {
		c.debug = true
	}
}

// Log sets the logging function for debug mode.
func Log(fn func(...any)) Option {
	return func(c *config) {
		c.log = fn
	}
}

// Driver configures the client driver.
func Driver(driver dialect.Driver) Option {
	return func(c *config) {
		c.driver = driver
	}
}

// Open opens a database/sql.DB specified by the driver name and
// the data source name, and returns a new client attached to it.
// Optional parameters can be added for configuring the client.
func Open(driverName, dataSourceName string, options ...Option) (*Client, error) {
	switch driverName {
	case dialect.MySQL, dialect.Postgres, dialect.SQLite:
		drv, err := sql.Open(driverName, dataSourceName)
		if err != nil {
			return nil, err
		}
		return NewClient(append(options, Driver(drv))...), nil
	default:
		return nil, fmt.Errorf("unsupported driver: %q", driverName)
	}
}

// ErrTxStarted is returned when trying to start a new transaction from a transactional client.
var ErrTxStarted = errors.New("db: cannot start a transaction within a transaction")

// Tx returns a new transactional client. The provided context
// is used until the transaction is committed or rolled back.
func (c *Client) Tx(ctx context.Context) (*Tx, error) {
	if _, ok := c.driver.(*txDriver); ok {
		return nil, ErrTxStarted
	}
	tx, err := newTx(ctx, c.driver)
	if err != nil {
		return nil, fmt.Errorf("db: starting a transaction: %w", err)
	}
	cfg := c.config
	cfg.driver = tx
	return &Tx{
		ctx:             ctx,
		config:          cfg,
		BalanceSnapshot: NewBalanceSnapshotClient(cfg),
		Entitlement:     NewEntitlementClient(cfg),
		Feature:         NewFeatureClient(cfg),
		Grant:           NewGrantClient(cfg),
		UsageReset:      NewUsageResetClient(cfg),
	}, nil
}

// BeginTx returns a transactional client with specified options.
func (c *Client) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	if _, ok := c.driver.(*txDriver); ok {
		return nil, errors.New("ent: cannot start a transaction within a transaction")
	}
	tx, err := c.driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}).BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("ent: starting a transaction: %w", err)
	}
	cfg := c.config
	cfg.driver = &txDriver{tx: tx, drv: c.driver}
	return &Tx{
		ctx:             ctx,
		config:          cfg,
		BalanceSnapshot: NewBalanceSnapshotClient(cfg),
		Entitlement:     NewEntitlementClient(cfg),
		Feature:         NewFeatureClient(cfg),
		Grant:           NewGrantClient(cfg),
		UsageReset:      NewUsageResetClient(cfg),
	}, nil
}

// Debug returns a new debug-client. It's used to get verbose logging on specific operations.
//
//	client.Debug().
//		BalanceSnapshot.
//		Query().
//		Count(ctx)
func (c *Client) Debug() *Client {
	if c.debug {
		return c
	}
	cfg := c.config
	cfg.driver = dialect.Debug(c.driver, c.log)
	client := &Client{config: cfg}
	client.init()
	return client
}

// Close closes the database connection and prevents new queries from starting.
func (c *Client) Close() error {
	return c.driver.Close()
}

// Use adds the mutation hooks to all the entity clients.
// In order to add hooks to a specific client, call: `client.Node.Use(...)`.
func (c *Client) Use(hooks ...Hook) {
	c.BalanceSnapshot.Use(hooks...)
	c.Entitlement.Use(hooks...)
	c.Feature.Use(hooks...)
	c.Grant.Use(hooks...)
	c.UsageReset.Use(hooks...)
}

// Intercept adds the query interceptors to all the entity clients.
// In order to add interceptors to a specific client, call: `client.Node.Intercept(...)`.
func (c *Client) Intercept(interceptors ...Interceptor) {
	c.BalanceSnapshot.Intercept(interceptors...)
	c.Entitlement.Intercept(interceptors...)
	c.Feature.Intercept(interceptors...)
	c.Grant.Intercept(interceptors...)
	c.UsageReset.Intercept(interceptors...)
}

// Mutate implements the ent.Mutator interface.
func (c *Client) Mutate(ctx context.Context, m Mutation) (Value, error) {
	switch m := m.(type) {
	case *BalanceSnapshotMutation:
		return c.BalanceSnapshot.mutate(ctx, m)
	case *EntitlementMutation:
		return c.Entitlement.mutate(ctx, m)
	case *FeatureMutation:
		return c.Feature.mutate(ctx, m)
	case *GrantMutation:
		return c.Grant.mutate(ctx, m)
	case *UsageResetMutation:
		return c.UsageReset.mutate(ctx, m)
	default:
		return nil, fmt.Errorf("db: unknown mutation type %T", m)
	}
}

// BalanceSnapshotClient is a client for the BalanceSnapshot schema.
type BalanceSnapshotClient struct {
	config
}

// NewBalanceSnapshotClient returns a client for the BalanceSnapshot from the given config.
func NewBalanceSnapshotClient(c config) *BalanceSnapshotClient {
	return &BalanceSnapshotClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `balancesnapshot.Hooks(f(g(h())))`.
func (c *BalanceSnapshotClient) Use(hooks ...Hook) {
	c.hooks.BalanceSnapshot = append(c.hooks.BalanceSnapshot, hooks...)
}

// Intercept adds a list of query interceptors to the interceptors stack.
// A call to `Intercept(f, g, h)` equals to `balancesnapshot.Intercept(f(g(h())))`.
func (c *BalanceSnapshotClient) Intercept(interceptors ...Interceptor) {
	c.inters.BalanceSnapshot = append(c.inters.BalanceSnapshot, interceptors...)
}

// Create returns a builder for creating a BalanceSnapshot entity.
func (c *BalanceSnapshotClient) Create() *BalanceSnapshotCreate {
	mutation := newBalanceSnapshotMutation(c.config, OpCreate)
	return &BalanceSnapshotCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of BalanceSnapshot entities.
func (c *BalanceSnapshotClient) CreateBulk(builders ...*BalanceSnapshotCreate) *BalanceSnapshotCreateBulk {
	return &BalanceSnapshotCreateBulk{config: c.config, builders: builders}
}

// MapCreateBulk creates a bulk creation builder from the given slice. For each item in the slice, the function creates
// a builder and applies setFunc on it.
func (c *BalanceSnapshotClient) MapCreateBulk(slice any, setFunc func(*BalanceSnapshotCreate, int)) *BalanceSnapshotCreateBulk {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return &BalanceSnapshotCreateBulk{err: fmt.Errorf("calling to BalanceSnapshotClient.MapCreateBulk with wrong type %T, need slice", slice)}
	}
	builders := make([]*BalanceSnapshotCreate, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		builders[i] = c.Create()
		setFunc(builders[i], i)
	}
	return &BalanceSnapshotCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for BalanceSnapshot.
func (c *BalanceSnapshotClient) Update() *BalanceSnapshotUpdate {
	mutation := newBalanceSnapshotMutation(c.config, OpUpdate)
	return &BalanceSnapshotUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *BalanceSnapshotClient) UpdateOne(bs *BalanceSnapshot) *BalanceSnapshotUpdateOne {
	mutation := newBalanceSnapshotMutation(c.config, OpUpdateOne, withBalanceSnapshot(bs))
	return &BalanceSnapshotUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *BalanceSnapshotClient) UpdateOneID(id int) *BalanceSnapshotUpdateOne {
	mutation := newBalanceSnapshotMutation(c.config, OpUpdateOne, withBalanceSnapshotID(id))
	return &BalanceSnapshotUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for BalanceSnapshot.
func (c *BalanceSnapshotClient) Delete() *BalanceSnapshotDelete {
	mutation := newBalanceSnapshotMutation(c.config, OpDelete)
	return &BalanceSnapshotDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a builder for deleting the given entity.
func (c *BalanceSnapshotClient) DeleteOne(bs *BalanceSnapshot) *BalanceSnapshotDeleteOne {
	return c.DeleteOneID(bs.ID)
}

// DeleteOneID returns a builder for deleting the given entity by its id.
func (c *BalanceSnapshotClient) DeleteOneID(id int) *BalanceSnapshotDeleteOne {
	builder := c.Delete().Where(balancesnapshot.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &BalanceSnapshotDeleteOne{builder}
}

// Query returns a query builder for BalanceSnapshot.
func (c *BalanceSnapshotClient) Query() *BalanceSnapshotQuery {
	return &BalanceSnapshotQuery{
		config: c.config,
		ctx:    &QueryContext{Type: TypeBalanceSnapshot},
		inters: c.Interceptors(),
	}
}

// Get returns a BalanceSnapshot entity by its id.
func (c *BalanceSnapshotClient) Get(ctx context.Context, id int) (*BalanceSnapshot, error) {
	return c.Query().Where(balancesnapshot.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *BalanceSnapshotClient) GetX(ctx context.Context, id int) *BalanceSnapshot {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// QueryEntitlement queries the entitlement edge of a BalanceSnapshot.
func (c *BalanceSnapshotClient) QueryEntitlement(bs *BalanceSnapshot) *EntitlementQuery {
	query := (&EntitlementClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := bs.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(balancesnapshot.Table, balancesnapshot.FieldID, id),
			sqlgraph.To(entitlement.Table, entitlement.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, balancesnapshot.EntitlementTable, balancesnapshot.EntitlementColumn),
		)
		fromV = sqlgraph.Neighbors(bs.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// Hooks returns the client hooks.
func (c *BalanceSnapshotClient) Hooks() []Hook {
	return c.hooks.BalanceSnapshot
}

// Interceptors returns the client interceptors.
func (c *BalanceSnapshotClient) Interceptors() []Interceptor {
	return c.inters.BalanceSnapshot
}

func (c *BalanceSnapshotClient) mutate(ctx context.Context, m *BalanceSnapshotMutation) (Value, error) {
	switch m.Op() {
	case OpCreate:
		return (&BalanceSnapshotCreate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdate:
		return (&BalanceSnapshotUpdate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdateOne:
		return (&BalanceSnapshotUpdateOne{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpDelete, OpDeleteOne:
		return (&BalanceSnapshotDelete{config: c.config, hooks: c.Hooks(), mutation: m}).Exec(ctx)
	default:
		return nil, fmt.Errorf("db: unknown BalanceSnapshot mutation op: %q", m.Op())
	}
}

// EntitlementClient is a client for the Entitlement schema.
type EntitlementClient struct {
	config
}

// NewEntitlementClient returns a client for the Entitlement from the given config.
func NewEntitlementClient(c config) *EntitlementClient {
	return &EntitlementClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `entitlement.Hooks(f(g(h())))`.
func (c *EntitlementClient) Use(hooks ...Hook) {
	c.hooks.Entitlement = append(c.hooks.Entitlement, hooks...)
}

// Intercept adds a list of query interceptors to the interceptors stack.
// A call to `Intercept(f, g, h)` equals to `entitlement.Intercept(f(g(h())))`.
func (c *EntitlementClient) Intercept(interceptors ...Interceptor) {
	c.inters.Entitlement = append(c.inters.Entitlement, interceptors...)
}

// Create returns a builder for creating a Entitlement entity.
func (c *EntitlementClient) Create() *EntitlementCreate {
	mutation := newEntitlementMutation(c.config, OpCreate)
	return &EntitlementCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of Entitlement entities.
func (c *EntitlementClient) CreateBulk(builders ...*EntitlementCreate) *EntitlementCreateBulk {
	return &EntitlementCreateBulk{config: c.config, builders: builders}
}

// MapCreateBulk creates a bulk creation builder from the given slice. For each item in the slice, the function creates
// a builder and applies setFunc on it.
func (c *EntitlementClient) MapCreateBulk(slice any, setFunc func(*EntitlementCreate, int)) *EntitlementCreateBulk {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return &EntitlementCreateBulk{err: fmt.Errorf("calling to EntitlementClient.MapCreateBulk with wrong type %T, need slice", slice)}
	}
	builders := make([]*EntitlementCreate, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		builders[i] = c.Create()
		setFunc(builders[i], i)
	}
	return &EntitlementCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for Entitlement.
func (c *EntitlementClient) Update() *EntitlementUpdate {
	mutation := newEntitlementMutation(c.config, OpUpdate)
	return &EntitlementUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *EntitlementClient) UpdateOne(e *Entitlement) *EntitlementUpdateOne {
	mutation := newEntitlementMutation(c.config, OpUpdateOne, withEntitlement(e))
	return &EntitlementUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *EntitlementClient) UpdateOneID(id string) *EntitlementUpdateOne {
	mutation := newEntitlementMutation(c.config, OpUpdateOne, withEntitlementID(id))
	return &EntitlementUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for Entitlement.
func (c *EntitlementClient) Delete() *EntitlementDelete {
	mutation := newEntitlementMutation(c.config, OpDelete)
	return &EntitlementDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a builder for deleting the given entity.
func (c *EntitlementClient) DeleteOne(e *Entitlement) *EntitlementDeleteOne {
	return c.DeleteOneID(e.ID)
}

// DeleteOneID returns a builder for deleting the given entity by its id.
func (c *EntitlementClient) DeleteOneID(id string) *EntitlementDeleteOne {
	builder := c.Delete().Where(entitlement.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &EntitlementDeleteOne{builder}
}

// Query returns a query builder for Entitlement.
func (c *EntitlementClient) Query() *EntitlementQuery {
	return &EntitlementQuery{
		config: c.config,
		ctx:    &QueryContext{Type: TypeEntitlement},
		inters: c.Interceptors(),
	}
}

// Get returns a Entitlement entity by its id.
func (c *EntitlementClient) Get(ctx context.Context, id string) (*Entitlement, error) {
	return c.Query().Where(entitlement.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *EntitlementClient) GetX(ctx context.Context, id string) *Entitlement {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// QueryUsageReset queries the usage_reset edge of a Entitlement.
func (c *EntitlementClient) QueryUsageReset(e *Entitlement) *UsageResetQuery {
	query := (&UsageResetClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := e.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(entitlement.Table, entitlement.FieldID, id),
			sqlgraph.To(usagereset.Table, usagereset.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, entitlement.UsageResetTable, entitlement.UsageResetColumn),
		)
		fromV = sqlgraph.Neighbors(e.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// QueryGrant queries the grant edge of a Entitlement.
func (c *EntitlementClient) QueryGrant(e *Entitlement) *GrantQuery {
	query := (&GrantClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := e.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(entitlement.Table, entitlement.FieldID, id),
			sqlgraph.To(dbgrant.Table, dbgrant.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, entitlement.GrantTable, entitlement.GrantColumn),
		)
		fromV = sqlgraph.Neighbors(e.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// QueryBalanceSnapshot queries the balance_snapshot edge of a Entitlement.
func (c *EntitlementClient) QueryBalanceSnapshot(e *Entitlement) *BalanceSnapshotQuery {
	query := (&BalanceSnapshotClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := e.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(entitlement.Table, entitlement.FieldID, id),
			sqlgraph.To(balancesnapshot.Table, balancesnapshot.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, entitlement.BalanceSnapshotTable, entitlement.BalanceSnapshotColumn),
		)
		fromV = sqlgraph.Neighbors(e.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// QueryFeature queries the feature edge of a Entitlement.
func (c *EntitlementClient) QueryFeature(e *Entitlement) *FeatureQuery {
	query := (&FeatureClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := e.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(entitlement.Table, entitlement.FieldID, id),
			sqlgraph.To(feature.Table, feature.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, entitlement.FeatureTable, entitlement.FeatureColumn),
		)
		fromV = sqlgraph.Neighbors(e.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// Hooks returns the client hooks.
func (c *EntitlementClient) Hooks() []Hook {
	return c.hooks.Entitlement
}

// Interceptors returns the client interceptors.
func (c *EntitlementClient) Interceptors() []Interceptor {
	return c.inters.Entitlement
}

func (c *EntitlementClient) mutate(ctx context.Context, m *EntitlementMutation) (Value, error) {
	switch m.Op() {
	case OpCreate:
		return (&EntitlementCreate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdate:
		return (&EntitlementUpdate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdateOne:
		return (&EntitlementUpdateOne{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpDelete, OpDeleteOne:
		return (&EntitlementDelete{config: c.config, hooks: c.Hooks(), mutation: m}).Exec(ctx)
	default:
		return nil, fmt.Errorf("db: unknown Entitlement mutation op: %q", m.Op())
	}
}

// FeatureClient is a client for the Feature schema.
type FeatureClient struct {
	config
}

// NewFeatureClient returns a client for the Feature from the given config.
func NewFeatureClient(c config) *FeatureClient {
	return &FeatureClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `feature.Hooks(f(g(h())))`.
func (c *FeatureClient) Use(hooks ...Hook) {
	c.hooks.Feature = append(c.hooks.Feature, hooks...)
}

// Intercept adds a list of query interceptors to the interceptors stack.
// A call to `Intercept(f, g, h)` equals to `feature.Intercept(f(g(h())))`.
func (c *FeatureClient) Intercept(interceptors ...Interceptor) {
	c.inters.Feature = append(c.inters.Feature, interceptors...)
}

// Create returns a builder for creating a Feature entity.
func (c *FeatureClient) Create() *FeatureCreate {
	mutation := newFeatureMutation(c.config, OpCreate)
	return &FeatureCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of Feature entities.
func (c *FeatureClient) CreateBulk(builders ...*FeatureCreate) *FeatureCreateBulk {
	return &FeatureCreateBulk{config: c.config, builders: builders}
}

// MapCreateBulk creates a bulk creation builder from the given slice. For each item in the slice, the function creates
// a builder and applies setFunc on it.
func (c *FeatureClient) MapCreateBulk(slice any, setFunc func(*FeatureCreate, int)) *FeatureCreateBulk {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return &FeatureCreateBulk{err: fmt.Errorf("calling to FeatureClient.MapCreateBulk with wrong type %T, need slice", slice)}
	}
	builders := make([]*FeatureCreate, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		builders[i] = c.Create()
		setFunc(builders[i], i)
	}
	return &FeatureCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for Feature.
func (c *FeatureClient) Update() *FeatureUpdate {
	mutation := newFeatureMutation(c.config, OpUpdate)
	return &FeatureUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *FeatureClient) UpdateOne(f *Feature) *FeatureUpdateOne {
	mutation := newFeatureMutation(c.config, OpUpdateOne, withFeature(f))
	return &FeatureUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *FeatureClient) UpdateOneID(id string) *FeatureUpdateOne {
	mutation := newFeatureMutation(c.config, OpUpdateOne, withFeatureID(id))
	return &FeatureUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for Feature.
func (c *FeatureClient) Delete() *FeatureDelete {
	mutation := newFeatureMutation(c.config, OpDelete)
	return &FeatureDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a builder for deleting the given entity.
func (c *FeatureClient) DeleteOne(f *Feature) *FeatureDeleteOne {
	return c.DeleteOneID(f.ID)
}

// DeleteOneID returns a builder for deleting the given entity by its id.
func (c *FeatureClient) DeleteOneID(id string) *FeatureDeleteOne {
	builder := c.Delete().Where(feature.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &FeatureDeleteOne{builder}
}

// Query returns a query builder for Feature.
func (c *FeatureClient) Query() *FeatureQuery {
	return &FeatureQuery{
		config: c.config,
		ctx:    &QueryContext{Type: TypeFeature},
		inters: c.Interceptors(),
	}
}

// Get returns a Feature entity by its id.
func (c *FeatureClient) Get(ctx context.Context, id string) (*Feature, error) {
	return c.Query().Where(feature.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *FeatureClient) GetX(ctx context.Context, id string) *Feature {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// QueryEntitlement queries the entitlement edge of a Feature.
func (c *FeatureClient) QueryEntitlement(f *Feature) *EntitlementQuery {
	query := (&EntitlementClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := f.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(feature.Table, feature.FieldID, id),
			sqlgraph.To(entitlement.Table, entitlement.FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, feature.EntitlementTable, feature.EntitlementColumn),
		)
		fromV = sqlgraph.Neighbors(f.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// Hooks returns the client hooks.
func (c *FeatureClient) Hooks() []Hook {
	return c.hooks.Feature
}

// Interceptors returns the client interceptors.
func (c *FeatureClient) Interceptors() []Interceptor {
	return c.inters.Feature
}

func (c *FeatureClient) mutate(ctx context.Context, m *FeatureMutation) (Value, error) {
	switch m.Op() {
	case OpCreate:
		return (&FeatureCreate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdate:
		return (&FeatureUpdate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdateOne:
		return (&FeatureUpdateOne{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpDelete, OpDeleteOne:
		return (&FeatureDelete{config: c.config, hooks: c.Hooks(), mutation: m}).Exec(ctx)
	default:
		return nil, fmt.Errorf("db: unknown Feature mutation op: %q", m.Op())
	}
}

// GrantClient is a client for the Grant schema.
type GrantClient struct {
	config
}

// NewGrantClient returns a client for the Grant from the given config.
func NewGrantClient(c config) *GrantClient {
	return &GrantClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `dbgrant.Hooks(f(g(h())))`.
func (c *GrantClient) Use(hooks ...Hook) {
	c.hooks.Grant = append(c.hooks.Grant, hooks...)
}

// Intercept adds a list of query interceptors to the interceptors stack.
// A call to `Intercept(f, g, h)` equals to `dbgrant.Intercept(f(g(h())))`.
func (c *GrantClient) Intercept(interceptors ...Interceptor) {
	c.inters.Grant = append(c.inters.Grant, interceptors...)
}

// Create returns a builder for creating a Grant entity.
func (c *GrantClient) Create() *GrantCreate {
	mutation := newGrantMutation(c.config, OpCreate)
	return &GrantCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of Grant entities.
func (c *GrantClient) CreateBulk(builders ...*GrantCreate) *GrantCreateBulk {
	return &GrantCreateBulk{config: c.config, builders: builders}
}

// MapCreateBulk creates a bulk creation builder from the given slice. For each item in the slice, the function creates
// a builder and applies setFunc on it.
func (c *GrantClient) MapCreateBulk(slice any, setFunc func(*GrantCreate, int)) *GrantCreateBulk {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return &GrantCreateBulk{err: fmt.Errorf("calling to GrantClient.MapCreateBulk with wrong type %T, need slice", slice)}
	}
	builders := make([]*GrantCreate, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		builders[i] = c.Create()
		setFunc(builders[i], i)
	}
	return &GrantCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for Grant.
func (c *GrantClient) Update() *GrantUpdate {
	mutation := newGrantMutation(c.config, OpUpdate)
	return &GrantUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *GrantClient) UpdateOne(gr *Grant) *GrantUpdateOne {
	mutation := newGrantMutation(c.config, OpUpdateOne, withGrant(gr))
	return &GrantUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *GrantClient) UpdateOneID(id string) *GrantUpdateOne {
	mutation := newGrantMutation(c.config, OpUpdateOne, withGrantID(id))
	return &GrantUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for Grant.
func (c *GrantClient) Delete() *GrantDelete {
	mutation := newGrantMutation(c.config, OpDelete)
	return &GrantDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a builder for deleting the given entity.
func (c *GrantClient) DeleteOne(gr *Grant) *GrantDeleteOne {
	return c.DeleteOneID(gr.ID)
}

// DeleteOneID returns a builder for deleting the given entity by its id.
func (c *GrantClient) DeleteOneID(id string) *GrantDeleteOne {
	builder := c.Delete().Where(dbgrant.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &GrantDeleteOne{builder}
}

// Query returns a query builder for Grant.
func (c *GrantClient) Query() *GrantQuery {
	return &GrantQuery{
		config: c.config,
		ctx:    &QueryContext{Type: TypeGrant},
		inters: c.Interceptors(),
	}
}

// Get returns a Grant entity by its id.
func (c *GrantClient) Get(ctx context.Context, id string) (*Grant, error) {
	return c.Query().Where(dbgrant.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *GrantClient) GetX(ctx context.Context, id string) *Grant {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// QueryEntitlement queries the entitlement edge of a Grant.
func (c *GrantClient) QueryEntitlement(gr *Grant) *EntitlementQuery {
	query := (&EntitlementClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := gr.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(dbgrant.Table, dbgrant.FieldID, id),
			sqlgraph.To(entitlement.Table, entitlement.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, dbgrant.EntitlementTable, dbgrant.EntitlementColumn),
		)
		fromV = sqlgraph.Neighbors(gr.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// Hooks returns the client hooks.
func (c *GrantClient) Hooks() []Hook {
	return c.hooks.Grant
}

// Interceptors returns the client interceptors.
func (c *GrantClient) Interceptors() []Interceptor {
	return c.inters.Grant
}

func (c *GrantClient) mutate(ctx context.Context, m *GrantMutation) (Value, error) {
	switch m.Op() {
	case OpCreate:
		return (&GrantCreate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdate:
		return (&GrantUpdate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdateOne:
		return (&GrantUpdateOne{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpDelete, OpDeleteOne:
		return (&GrantDelete{config: c.config, hooks: c.Hooks(), mutation: m}).Exec(ctx)
	default:
		return nil, fmt.Errorf("db: unknown Grant mutation op: %q", m.Op())
	}
}

// UsageResetClient is a client for the UsageReset schema.
type UsageResetClient struct {
	config
}

// NewUsageResetClient returns a client for the UsageReset from the given config.
func NewUsageResetClient(c config) *UsageResetClient {
	return &UsageResetClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `usagereset.Hooks(f(g(h())))`.
func (c *UsageResetClient) Use(hooks ...Hook) {
	c.hooks.UsageReset = append(c.hooks.UsageReset, hooks...)
}

// Intercept adds a list of query interceptors to the interceptors stack.
// A call to `Intercept(f, g, h)` equals to `usagereset.Intercept(f(g(h())))`.
func (c *UsageResetClient) Intercept(interceptors ...Interceptor) {
	c.inters.UsageReset = append(c.inters.UsageReset, interceptors...)
}

// Create returns a builder for creating a UsageReset entity.
func (c *UsageResetClient) Create() *UsageResetCreate {
	mutation := newUsageResetMutation(c.config, OpCreate)
	return &UsageResetCreate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// CreateBulk returns a builder for creating a bulk of UsageReset entities.
func (c *UsageResetClient) CreateBulk(builders ...*UsageResetCreate) *UsageResetCreateBulk {
	return &UsageResetCreateBulk{config: c.config, builders: builders}
}

// MapCreateBulk creates a bulk creation builder from the given slice. For each item in the slice, the function creates
// a builder and applies setFunc on it.
func (c *UsageResetClient) MapCreateBulk(slice any, setFunc func(*UsageResetCreate, int)) *UsageResetCreateBulk {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return &UsageResetCreateBulk{err: fmt.Errorf("calling to UsageResetClient.MapCreateBulk with wrong type %T, need slice", slice)}
	}
	builders := make([]*UsageResetCreate, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		builders[i] = c.Create()
		setFunc(builders[i], i)
	}
	return &UsageResetCreateBulk{config: c.config, builders: builders}
}

// Update returns an update builder for UsageReset.
func (c *UsageResetClient) Update() *UsageResetUpdate {
	mutation := newUsageResetMutation(c.config, OpUpdate)
	return &UsageResetUpdate{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOne returns an update builder for the given entity.
func (c *UsageResetClient) UpdateOne(ur *UsageReset) *UsageResetUpdateOne {
	mutation := newUsageResetMutation(c.config, OpUpdateOne, withUsageReset(ur))
	return &UsageResetUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// UpdateOneID returns an update builder for the given id.
func (c *UsageResetClient) UpdateOneID(id string) *UsageResetUpdateOne {
	mutation := newUsageResetMutation(c.config, OpUpdateOne, withUsageResetID(id))
	return &UsageResetUpdateOne{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// Delete returns a delete builder for UsageReset.
func (c *UsageResetClient) Delete() *UsageResetDelete {
	mutation := newUsageResetMutation(c.config, OpDelete)
	return &UsageResetDelete{config: c.config, hooks: c.Hooks(), mutation: mutation}
}

// DeleteOne returns a builder for deleting the given entity.
func (c *UsageResetClient) DeleteOne(ur *UsageReset) *UsageResetDeleteOne {
	return c.DeleteOneID(ur.ID)
}

// DeleteOneID returns a builder for deleting the given entity by its id.
func (c *UsageResetClient) DeleteOneID(id string) *UsageResetDeleteOne {
	builder := c.Delete().Where(usagereset.ID(id))
	builder.mutation.id = &id
	builder.mutation.op = OpDeleteOne
	return &UsageResetDeleteOne{builder}
}

// Query returns a query builder for UsageReset.
func (c *UsageResetClient) Query() *UsageResetQuery {
	return &UsageResetQuery{
		config: c.config,
		ctx:    &QueryContext{Type: TypeUsageReset},
		inters: c.Interceptors(),
	}
}

// Get returns a UsageReset entity by its id.
func (c *UsageResetClient) Get(ctx context.Context, id string) (*UsageReset, error) {
	return c.Query().Where(usagereset.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *UsageResetClient) GetX(ctx context.Context, id string) *UsageReset {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
}

// QueryEntitlement queries the entitlement edge of a UsageReset.
func (c *UsageResetClient) QueryEntitlement(ur *UsageReset) *EntitlementQuery {
	query := (&EntitlementClient{config: c.config}).Query()
	query.path = func(context.Context) (fromV *sql.Selector, _ error) {
		id := ur.ID
		step := sqlgraph.NewStep(
			sqlgraph.From(usagereset.Table, usagereset.FieldID, id),
			sqlgraph.To(entitlement.Table, entitlement.FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, usagereset.EntitlementTable, usagereset.EntitlementColumn),
		)
		fromV = sqlgraph.Neighbors(ur.driver.Dialect(), step)
		return fromV, nil
	}
	return query
}

// Hooks returns the client hooks.
func (c *UsageResetClient) Hooks() []Hook {
	return c.hooks.UsageReset
}

// Interceptors returns the client interceptors.
func (c *UsageResetClient) Interceptors() []Interceptor {
	return c.inters.UsageReset
}

func (c *UsageResetClient) mutate(ctx context.Context, m *UsageResetMutation) (Value, error) {
	switch m.Op() {
	case OpCreate:
		return (&UsageResetCreate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdate:
		return (&UsageResetUpdate{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpUpdateOne:
		return (&UsageResetUpdateOne{config: c.config, hooks: c.Hooks(), mutation: m}).Save(ctx)
	case OpDelete, OpDeleteOne:
		return (&UsageResetDelete{config: c.config, hooks: c.Hooks(), mutation: m}).Exec(ctx)
	default:
		return nil, fmt.Errorf("db: unknown UsageReset mutation op: %q", m.Op())
	}
}

// hooks and interceptors per client, for fast access.
type (
	hooks struct {
		BalanceSnapshot, Entitlement, Feature, Grant, UsageReset []ent.Hook
	}
	inters struct {
		BalanceSnapshot, Entitlement, Feature, Grant, UsageReset []ent.Interceptor
	}
)
