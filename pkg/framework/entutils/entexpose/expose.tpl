{{/*
    This template exposes client internals (like the driver) so they can be shared accross instances.
    The intended usecase is for shared transaction management accross multiple db.Client and db.Tx instances
    using the same connection.

    This template has to be included in each entgo codegen that wants to parttake in shared transactions.

    // TODO: Tx.onRollback and Tx.onCommit hooks are ignored when using shared transactions, fix this
*/}}
{{ define "expose" }}


{{/* Add the base header for the generated file */}}
{{ $pkg := base $.Config.Package }}
{{ template "header" $ }}

{{/* Expose internals of a Client so it can be used by transaction management */}}
func (c *Client) GetConfig() *entutils.RawEntConfig {
    return &entutils.RawEntConfig{
        Driver: c.config.driver,
        Debug: c.config.debug,
        Log: c.config.log,
    }
}
{{/* Expose internals of a Transactional Client so it can be used by transaction management */}}
// ignores hooks and intersectors
type ExposedTxDriver struct {
    Driver *txDriver
}

var _ entutils.Transactable = (*ExposedTxDriver)(nil)

func (d *ExposedTxDriver) Rollback() error {
    return d.Driver.tx.Rollback()
}

func (d *ExposedTxDriver) Commit() error {
    return d.Driver.tx.Commit()
}

func (d *ExposedTxDriver) SavePoint(name string) error {
	_, err := d.Driver.ExecContext(context.Background(), "SAVEPOINT " + name)
    return err
}

func (d *ExposedTxDriver) RollbackTo(name string) error {
	_, err := d.Driver.ExecContext(context.Background(), "ROLLBACK TO " + name)
    return err
}

func (d *ExposedTxDriver) Release(name string) error {
	_, err := d.Driver.ExecContext(context.Background(), "RELEASE SAVEPOINT " + name)
    return err
}

// HijackTx returns a new transaction driver with the provided options.
// The returned transaction can later be used to instanciate new clients.
func (c *Client) HijackTx(ctx context.Context, opts *sql.TxOptions) (context.Context, *entutils.RawEntConfig, *ExposedTxDriver, error) {
	if _, ok := c.driver.(*txDriver); ok {
		return nil, nil, nil, errors.New("ent: cannot start a transaction within a transaction")
	}
	tx, err := c.driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}).BeginTx(ctx, opts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("ent: starting a transaction: %w", err)
	}

	driver := &txDriver{tx: tx, drv: c.driver}

	cfg := c.config
	cfg.driver = &txDriver{tx: tx, drv: c.driver}
	return ctx, &entutils.RawEntConfig{
        Driver: cfg.driver,
        Debug: cfg.debug,
        Log: cfg.log,
    }, &ExposedTxDriver{Driver: driver}, nil
}

// NewTxClientFromConfig creates a new transactional client from a (hijacked) configuration.
func NewTxClientFromRawConfig(ctx context.Context, cfg entutils.RawEntConfig) *Tx {
    config := config{
        driver: cfg.Driver,
        debug:  cfg.Debug,
        log:    cfg.Log,
        hooks: &hooks{},
        inters: &inters{},
    }

    // Hooks registered on the originating client are not carried over (see RawEntConfig),
    // so the write observers have to be installed again.
    config.installWriteObservers()

    return &Tx{
		ctx:         ctx,
		config:      config,
        // Clients templated from defined schemas
        {{ range $n := $.Nodes }}
            {{ $n.Name }}: New{{ $n.Name }}Client(config),
        {{ end }}
	}
}

// The write observers are built once and shared by every client: a transactional client
// is created for each repository call, so building them per client would allocate for
// every entity on every call.
//
// Sharing is safe because each slice is full: a client that registers its own hooks or
// interceptors appends into a new array and leaves these untouched.
var (
    writeObserverHooks = hooks{
        {{- range $n := $.MutableNodes }}
            {{ $n.Name }}: []Hook{entutils.WriteObserverHook},
        {{- end }}
    }

    writeObserverInters = inters{
        {{- range $n := $.Nodes }}
            {{ $n.Name }}: []Interceptor{entutils.WriteObserverInterceptor(Type{{ $n.Name }})},
        {{- end }}
    }
)

// installWriteObservers makes every client report the writes it attempts to the write
// observer of the context (see transaction.WithWriteObserver). It runs for every client
// the package can build, so no caller has to remember to register it.
//
// It replaces the hooks and interceptors of the config, so it must run before any other
// is registered.
func (c *config) installWriteObservers() {
    *c.hooks = writeObserverHooks
    *c.inters = writeObserverInters
}

{{/* Expose row locking so entutils.WriteObserverInterceptor can tell locking reads from plain ones */}}
{{ if $.FeatureEnabled "sql/lock" }}
{{ range $n := $.Nodes }}
    {{ $receiver := $n.Receiver }}
    // LocksRows reports whether the query takes row locks (FOR UPDATE / FOR SHARE).
    // Lock clauses are the only selector modifiers the builders accept.
    func ({{ $receiver }} *{{ $n.QueryName }}) LocksRows() bool {
        return len({{ $receiver }}.modifiers) > 0
    }
{{ end }}
{{ end }}

{{ end }}

{{/* Runs inside newConfig, so every client built with NewClient reports its writes */}}
{{ define "config/init/fields/writeobservers" }}
    cfg.installWriteObservers()
{{ end }}
