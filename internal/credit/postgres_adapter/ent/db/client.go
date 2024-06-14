// Code generated by ent, DO NOT EDIT.

package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/openmeterio/openmeter/internal/credit/postgres_adapter/ent/db/migrate"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/internal/credit/postgres_adapter/ent/db/balancesnapshot"
	"github.com/openmeterio/openmeter/internal/credit/postgres_adapter/ent/db/grant"
)

// Client is the client that holds all ent builders.
type Client struct {
	config
	// Schema is the client for creating, migrating and dropping schema.
	Schema *migrate.Schema
	// BalanceSnapshot is the client for interacting with the BalanceSnapshot builders.
	BalanceSnapshot *BalanceSnapshotClient
	// Grant is the client for interacting with the Grant builders.
	Grant *GrantClient
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
	c.Grant = NewGrantClient(c.config)
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
		Grant:           NewGrantClient(cfg),
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
		Grant:           NewGrantClient(cfg),
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
	c.Grant.Use(hooks...)
}

// Intercept adds the query interceptors to all the entity clients.
// In order to add interceptors to a specific client, call: `client.Node.Intercept(...)`.
func (c *Client) Intercept(interceptors ...Interceptor) {
	c.BalanceSnapshot.Intercept(interceptors...)
	c.Grant.Intercept(interceptors...)
}

// Mutate implements the ent.Mutator interface.
func (c *Client) Mutate(ctx context.Context, m Mutation) (Value, error) {
	switch m := m.(type) {
	case *BalanceSnapshotMutation:
		return c.BalanceSnapshot.mutate(ctx, m)
	case *GrantMutation:
		return c.Grant.mutate(ctx, m)
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

// GrantClient is a client for the Grant schema.
type GrantClient struct {
	config
}

// NewGrantClient returns a client for the Grant from the given config.
func NewGrantClient(c config) *GrantClient {
	return &GrantClient{config: c}
}

// Use adds a list of mutation hooks to the hooks stack.
// A call to `Use(f, g, h)` equals to `grant.Hooks(f(g(h())))`.
func (c *GrantClient) Use(hooks ...Hook) {
	c.hooks.Grant = append(c.hooks.Grant, hooks...)
}

// Intercept adds a list of query interceptors to the interceptors stack.
// A call to `Intercept(f, g, h)` equals to `grant.Intercept(f(g(h())))`.
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
	builder := c.Delete().Where(grant.ID(id))
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
	return c.Query().Where(grant.ID(id)).Only(ctx)
}

// GetX is like Get, but panics if an error occurs.
func (c *GrantClient) GetX(ctx context.Context, id string) *Grant {
	obj, err := c.Get(ctx, id)
	if err != nil {
		panic(err)
	}
	return obj
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

// hooks and interceptors per client, for fast access.
type (
	hooks struct {
		BalanceSnapshot, Grant []ent.Hook
	}
	inters struct {
		BalanceSnapshot, Grant []ent.Interceptor
	}
)
