package clickhouse

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/progressmanager"
	progressmanagerentity "github.com/openmeterio/openmeter/openmeter/progressmanager/entity"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/featuregate"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ streaming.Connector = (*Connector)(nil)

// Connector implements `ingest.Connector" and `namespace.Handler interfaces.
type Connector struct {
	config Config
	// queryCacheMetrics is nil when Config.Meter is unset.
	queryCacheMetrics *queryCacheMetrics
	// tracer is Config.Tracer or a noop.
	tracer trace.Tracer
}

type Config struct {
	Logger                 *slog.Logger
	ClickHouse             clickhouse.Conn
	Database               string
	EventsTableName        string
	AsyncInsert            bool
	AsyncInsertWait        bool
	InsertQuerySettings    map[string]string
	MeterQuerySettings     map[string]string
	EnablePrewhere         bool
	EnableDecimalPrecision bool
	ProgressManager        progressmanager.Service
	SkipCreateTables       bool

	// QueryCacheEnabled turns on the optional meter query-result cache: an
	// hourly pre-aggregated rollup table that serves the settled history of
	// cacheable queries while the fresh tail is scanned live.
	//
	// Topology requirement: the cache populates lazily on read (synchronous
	// INSERT...SELECT, deliberately NOT using InsertQuerySettings so async_insert
	// cannot be applied to it) and the merge SELECT that follows must see that
	// insert. On a single node this always holds; behind a load-balanced
	// multi-replica endpoint without quorum writes the merge can read a replica
	// the insert has not replicated to yet and silently undercount the settled
	// range. Point the connector at a single node / sticky replica, or configure
	// quorum + sequential-consistency on the deployment, before enabling.
	QueryCacheEnabled bool
	// QueryCacheMinimumCacheableQueryPeriod is the minimum queried span for a
	// query to be worth caching.
	QueryCacheMinimumCacheableQueryPeriod time.Duration
	// QueryCacheMinimumCacheableUsageAge is the freshness horizon: only windows
	// entirely older than now-this-age are cached. The
	// fresher tail is always served live.
	QueryCacheMinimumCacheableUsageAge time.Duration
	// QueryCacheParityCheckSampleRate (0..1) samples cache-served queries for
	// shadow verification: the live query is re-run off the request path and
	// compared to the cached result; a mismatch is logged, counted
	// (streaming.query_cache.parity_checks{outcome=mismatch}) and self-heals by
	// invalidating the namespace's cache. 0 disables. Each sampled query costs
	// one extra full live query.
	QueryCacheParityCheckSampleRate float64
	// Meter, when set, exposes the query cache's health counters (queries,
	// populate errors, invalidations, parity checks) and duration histograms.
	// Optional: nil disables metrics without affecting behavior.
	Meter metric.Meter
	// Tracer, when set, emits spans for the query cache operations (cached
	// query, populate, invalidate, parity check). Optional: nil falls back to a
	// noop tracer. The live query path is deliberately NOT instrumented here so
	// it stays byte-identical to the pre-cache behavior.
	Tracer trace.Tracer
	// FeatureGate gates the query cache per namespace.
	FeatureGate *featuregate.FeatureGateChecker
}

func (c Config) Validate() error {
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	if c.ClickHouse == nil {
		return fmt.Errorf("clickhouse connection is required")
	}

	if c.Database == "" {
		return fmt.Errorf("database is required")
	}

	if c.EventsTableName == "" {
		return fmt.Errorf("events table is required")
	}

	if c.ProgressManager == nil {
		return fmt.Errorf("progress manager is required")
	}

	if c.QueryCacheEnabled {
		if !c.EnableDecimalPrecision {
			return fmt.Errorf("query cache requires decimal precision to be enabled")
		}

		// The late-event invalidation must run AFTER the event is visible to
		// SELECTs. Async insert without wait acks before the buffer flush, so a
		// read repopulating inside the flush gap would store a coverage claim
		// over rollups missing that event — and no later invalidation ever
		// comes for it.
		if c.AsyncInsert && !c.AsyncInsertWait {
			return fmt.Errorf("query cache requires synchronous event visibility: enable async insert wait or disable async insert")
		}

		if c.QueryCacheMinimumCacheableQueryPeriod <= 0 {
			return fmt.Errorf("minimum cacheable query period is required")
		}

		if c.QueryCacheMinimumCacheableUsageAge <= 0 {
			return fmt.Errorf("minimum cacheable usage age is required")
		}

		if c.QueryCacheParityCheckSampleRate < 0 || c.QueryCacheParityCheckSampleRate > 1 {
			return fmt.Errorf("query cache parity check sample rate must be between 0 and 1")
		}
	}

	return nil
}

func New(ctx context.Context, config Config) (*Connector, error) {
	// Validate the config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	// Create the connector
	connector := &Connector{
		config: config,
		tracer: config.Tracer,
	}

	if connector.tracer == nil {
		connector.tracer = tracenoop.NewTracerProvider().Tracer("openmeter.streaming.clickhouse")
	}

	if config.Meter != nil {
		var err error
		connector.queryCacheMetrics, err = newQueryCacheMetrics(config.Meter)
		if err != nil {
			return nil, fmt.Errorf("create query cache metrics: %w", err)
		}
	}

	if !config.SkipCreateTables {
		if err := connector.createTable(ctx); err != nil {
			return nil, fmt.Errorf("create tables: %w", err)
		}
	}

	return connector, nil
}

// createTable creates the tables in ClickHouse
func (c *Connector) createTable(ctx context.Context) error {
	// Create the events table
	err := c.createEventsTable(ctx)
	if err != nil {
		return fmt.Errorf("create events table in clickhouse: %w", err)
	}

	// Create the query-result cache rollup table only when the cache is enabled,
	// so a disabled cache leaves the schema byte-identical to today.
	if c.config.QueryCacheEnabled {
		if err := c.createMeterQueryRowCacheTable(ctx); err != nil {
			return fmt.Errorf("create meter query cache table in clickhouse: %w", err)
		}
	}

	return nil
}

// createMeterQueryRowCacheTable creates the rollup table and its coverage
// companion; they always exist (and are invalidated) together.
func (c *Connector) createMeterQueryRowCacheTable(ctx context.Context) error {
	table := createMeterQueryRowCacheTable{
		Database:  c.config.Database,
		TableName: meterQueryRowCacheTable,
	}

	if err := c.config.ClickHouse.Exec(ctx, table.toSQL()); err != nil {
		return fmt.Errorf("create meter query cache table: %w", err)
	}

	coverageTable := createMeterQueryRowCacheCoverageTable{
		Database:  c.config.Database,
		TableName: meterQueryRowCacheCoverageTable,
	}

	if err := c.config.ClickHouse.Exec(ctx, coverageTable.toSQL()); err != nil {
		return fmt.Errorf("create meter query cache coverage table: %w", err)
	}

	return nil
}

func (c *Connector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]streaming.RawEvent, error) {
	if namespace == "" {
		return nil, models.NewGenericValidationError(fmt.Errorf("namespace is required"))
	}

	events, err := c.queryEventsTable(ctx, namespace, params)
	if err != nil {
		if _, ok := err.(*models.NamespaceNotFoundError); ok {
			return nil, err
		}

		return nil, fmt.Errorf("query events: %w", err)
	}

	return events, nil
}

func (c *Connector) ListEventsV2(ctx context.Context, params streaming.ListEventsV2Params) ([]streaming.RawEvent, error) {
	if err := params.Validate(); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	events, err := c.queryEventsTableV2(ctx, params)
	if err != nil {
		if _, ok := err.(*models.NamespaceNotFoundError); ok {
			return nil, err
		}

		return nil, fmt.Errorf("query events: %w", err)
	}

	return events, nil
}

func (c *Connector) QueryMeter(ctx context.Context, namespace string, meter meterpkg.Meter, params streaming.QueryParams) ([]meterpkg.MeterQueryRow, error) {
	// Validate params
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validate params: %w", err)
	}

	// We sort the group by keys to ensure the order of the group by columns is deterministic
	// It helps testing the SQL queries.
	groupBy := append([]string(nil), params.GroupBy...)
	sort.Strings(groupBy)

	query := queryMeter{
		Database:               c.config.Database,
		EventsTableName:        c.config.EventsTableName,
		Namespace:              namespace,
		Meter:                  meter,
		From:                   params.From,
		To:                     params.To,
		FilterCustomer:         params.FilterCustomer,
		FilterSubject:          params.FilterSubject,
		FilterGroupBy:          params.FilterGroupBy,
		FilterStoredAt:         params.FilterStoredAt,
		GroupBy:                groupBy,
		WindowSize:             params.WindowSize,
		WindowTimeZone:         params.WindowTimeZone,
		QuerySettings:          c.config.MeterQuerySettings,
		EnablePrewhere:         c.config.EnablePrewhere,
		EnableDecimalPrecision: c.config.EnableDecimalPrecision,
	}

	var err error
	var values []meterpkg.MeterQueryRow

	// If the query is provably reproducible from the hourly rollup, serve it from
	// the cache (settled history from the rollup + fresh tail scanned live,
	// merged in one SQL statement). Otherwise the live path runs unchanged.
	if c.canQueryBeCached(namespace, meter, params) {
		values, err = c.queryMeterCached(ctx, query)
		if err != nil {
			return values, fmt.Errorf("query meter cached: %w", err)
		}
	} else if params.ClientID != nil {
		// If the client ID is set, we track track the progress of the query
		values, err = c.queryMeterWithProgress(ctx, namespace, *params.ClientID, query)
		if err != nil {
			return values, fmt.Errorf("query meter with progress: %w", err)
		}
	} else {
		values, err = c.queryMeter(ctx, query)
		if err != nil {
			return values, fmt.Errorf("query meter: %w", err)
		}
	}

	// If the total usage is queried for a single period (no window size),
	// replace the window start and end with the period for each row.
	// We can still have multiple rows for a single period due to group bys.
	if params.WindowSize == nil {
		for i := range values {
			if params.From != nil {
				values[i].WindowStart = *params.From
			}
			if params.To != nil {
				values[i].WindowEnd = *params.To
			}
		}
	}

	return values, nil
}

func (c *Connector) ListSubjects(ctx context.Context, params streaming.ListSubjectsParams) ([]string, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validate params: %w", err)
	}

	subjects, err := c.listSubjects(ctx, params)
	if err != nil {
		if meterpkg.IsMeterNotFoundError(err) {
			return nil, err
		}

		return nil, fmt.Errorf("list subjects: %w", err)
	}

	return subjects, nil
}

func (c *Connector) ListGroupByValues(ctx context.Context, params streaming.ListGroupByValuesParams) ([]string, error) {
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validate params: %w", err)
	}

	groupByValues, err := c.listGroupByValues(ctx, params)
	if err != nil {
		if meterpkg.IsMeterNotFoundError(err) {
			return nil, err
		}

		return nil, fmt.Errorf("list group by values: %w", err)
	}

	return groupByValues, nil
}

func (c *Connector) CreateNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (c *Connector) DeleteNamespace(ctx context.Context, namespace string) error {
	// We don't delete the event tables as it it reused between namespaces
	return nil
}

func (c *Connector) CountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	rows, err := c.queryCountEvents(ctx, namespace, params)
	if err != nil {
		if _, ok := err.(*models.NamespaceNotFoundError); ok {
			return nil, err
		}

		return nil, fmt.Errorf("query count events: %w", err)
	}

	return rows, nil
}

func (c *Connector) BatchInsert(ctx context.Context, rawEvents []streaming.RawEvent) error {
	var err error

	// Insert raw events
	query := InsertEventsQuery{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Events:          rawEvents,
		QuerySettings:   c.config.InsertQuerySettings,
	}
	sql, args := query.ToSQL()

	// By default, ClickHouse is writing data synchronously.
	// See https://clickhouse.com/docs/en/cloud/bestpractices/asynchronous-inserts
	if c.config.AsyncInsert {
		// With the `wait_for_async_insert` setting, you can configure
		// if you want an insert statement to return with an acknowledgment
		// either immediately after the data got inserted into the buffer.
		err = c.config.ClickHouse.Exec(clickhouse.Context(ctx, clickhouse.WithAsync(c.config.AsyncInsertWait)), sql, args...)
	} else {
		err = c.config.ClickHouse.Exec(ctx, sql, args...)
	}

	if err != nil {
		return fmt.Errorf("failed to batch insert raw events: %w", err)
	}

	// Late events mutate already-settled windows: wipe the affected namespaces' cached rollups.
	namespacesToInvalidate := c.findNamespacesToInvalidateCache(rawEvents)
	if err := c.invalidateMeterQueryRowCache(ctx, namespacesToInvalidate); err != nil {
		// Do not fail the insert on invalidation error.
		// A stale cache is corrected on the next late event or by re-population.
		c.config.Logger.Error("failed to invalidate meter query cache", "error", err, "namespaces", namespacesToInvalidate)
	} else if len(namespacesToInvalidate) > 0 {
		c.queryCacheMetrics.recordInvalidation(ctx, "late_event", len(namespacesToInvalidate))
	}

	return nil
}

func (c *Connector) createEventsTable(ctx context.Context) error {
	table := createEventsTable{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
	}

	err := c.config.ClickHouse.Exec(ctx, table.toSQL())
	if err != nil {
		return fmt.Errorf("create events table: %w", err)
	}

	return nil
}

// ValidateJSONPath checks if the given JSON path is valid by executing a simple query with it.
func (c *Connector) ValidateJSONPath(ctx context.Context, jsonPath string) (bool, error) {
	query := validateJsonPathQuery{
		jsonPath: jsonPath,
	}

	sql, args := query.toSQL()

	err := c.config.ClickHouse.Exec(ctx, sql, args...)
	if err != nil {
		// Code 36 means bad arguments
		// See: https://github.com/ClickHouse/ClickHouse/blob/master/src/Common/ErrorCodes.cpp
		if strings.Contains(err.Error(), "code: 36") {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (c *Connector) queryEventsTable(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]streaming.RawEvent, error) {
	var err error

	table := queryEventsTable{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Namespace:       namespace,
		From:            params.From,
		To:              params.To,
		IngestedAtFrom:  params.IngestedAtFrom,
		IngestedAtTo:    params.IngestedAtTo,
		ID:              params.ID,
		Subject:         params.Subject,
		Customers:       params.Customers,
		Limit:           params.Limit,
	}

	// If the client ID is set, we track track the progress of the query
	if params.ClientID != nil {
		// Build SQL query to count the total number of rows
		countSQL, countArgs := table.toCountRowSQL()

		ctx, err = c.withProgressContext(ctx, namespace, *params.ClientID, countSQL, countArgs)
		// Log error but don't return it
		if err != nil {
			c.config.Logger.Error("failed track progress", "error", err, "clientId", *params.ClientID)
		}
	}

	sql, args := table.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, models.NewNamespaceNotFoundError(namespace)
		}

		return nil, fmt.Errorf("query events table query: %w", err)
	}

	defer rows.Close()

	events := []streaming.RawEvent{}

	for rows.Next() {
		var rawEvent streaming.RawEvent
		if err = rows.ScanStruct(&rawEvent); err != nil {
			return nil, err
		}

		events = append(events, rawEvent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return events, nil
}

// queryEventsTableV2 is similar to queryEventsTable but with advanced filtering options.
func (c *Connector) queryEventsTableV2(ctx context.Context, params streaming.ListEventsV2Params) ([]streaming.RawEvent, error) {
	var err error

	// Create query struct
	queryBuilder := queryEventsTableV2{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Params:          params,
	}

	// If a client ID is provided, track progress
	if params.ClientID != nil {
		// Build SQL query to count the total number of rows
		countSQL, countArgs := queryBuilder.toCountRowSQL()

		ctx, err = c.withProgressContext(ctx, params.Namespace, *params.ClientID, countSQL, countArgs)
		// Log error but don't return it
		if err != nil {
			c.config.Logger.Error("failed track progress", "error", err, "clientId", *params.ClientID)
		}
	}

	sql, args := queryBuilder.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query events table query: %w", err)
	}

	defer rows.Close()

	events := []streaming.RawEvent{}

	for rows.Next() {
		var event streaming.RawEvent

		err = rows.ScanStruct(&event)
		if err != nil {
			return nil, fmt.Errorf("scan raw event: %w", err)
		}

		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return events, nil
}

func (c *Connector) queryCountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
	table := queryCountEvents{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Namespace:       namespace,
		From:            params.From,
	}

	sql, args := table.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, models.NewNamespaceNotFoundError(namespace)
		}

		return nil, fmt.Errorf("query events count query: %w", err)
	}

	defer rows.Close()

	results := []streaming.CountEventRow{}

	for rows.Next() {
		result := streaming.CountEventRow{}

		if err = rows.Scan(&result.Count, &result.Subject); err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// queryMeterWithProgress queries the meter and returns the rows
// It also tracks the progress of the query
func (c *Connector) queryMeterWithProgress(ctx context.Context, namespace string, clientID string, query queryMeter) ([]meterpkg.MeterQueryRow, error) {
	var err error

	// Build SQL query to count the total number of rows
	countSQL, countArgs := query.toCountRowSQL()

	ctx, err = c.withProgressContext(ctx, namespace, clientID, countSQL, countArgs)
	// Log error but don't return it
	if err != nil {
		c.config.Logger.Error("failed track progress", "error", err, "clientId", clientID)
	}

	// Query the meter
	values, err := c.queryMeter(ctx, query)
	if err != nil {
		return values, fmt.Errorf("query meter rows: %w", err)
	}

	return values, nil
}

// queryMeter queries the meter and returns the rows
func (c *Connector) queryMeter(ctx context.Context, query queryMeter) ([]meterpkg.MeterQueryRow, error) {
	// Build the SQL query
	sql, args, err := query.toSQL()
	if err != nil {
		return nil, fmt.Errorf("build sql query: %w", err)
	}

	start := time.Now()

	// Query the meter view
	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, meterpkg.NewMeterNotFoundError(query.Meter.Key)
		}
		return nil, fmt.Errorf("clickhouse query: %w", err)
	}

	defer rows.Close()

	elapsed := time.Since(start)
	c.config.Logger.Debug("clickhouse query executed", "elapsed", elapsed.String(), "sql", sql, "args", args)

	// Scan the rows
	values, err := query.scanRows(rows)
	if err != nil {
		return nil, fmt.Errorf("scan query rows: %w", err)
	}

	return values, nil
}

// listSubjects lists the subjects that have events in the database
func (c *Connector) listSubjects(ctx context.Context, params streaming.ListSubjectsParams) ([]string, error) {
	query := listSubjectsQuery{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Namespace:       params.Namespace,
		Search:          params.Search,
		Meter:           params.Meter,
		From:            params.From,
		To:              params.To,
	}

	sql, args := query.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list subjects: %w", err)
	}

	defer rows.Close()

	subjects := []string{}
	for rows.Next() {
		var subject string
		if err = rows.Scan(&subject); err != nil {
			return nil, err
		}

		subjects = append(subjects, subject)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return subjects, nil
}

// listGroupByValues lists the group by values that have events in the database
func (c *Connector) listGroupByValues(ctx context.Context, params streaming.ListGroupByValuesParams) ([]string, error) {
	query := listGroupByValuesQuery{
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Namespace:       params.Namespace,
		Meter:           params.Meter,
		GroupByKey:      params.GroupByKey,
		From:            params.From,
		To:              params.To,
		Search:          params.Search,
	}

	sql, args := query.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list group by values: %w", err)
	}

	defer rows.Close()

	groupByValues := []string{}
	for rows.Next() {
		var groupByValue string
		if err = rows.Scan(&groupByValue); err != nil {
			return nil, err
		}

		groupByValues = append(groupByValues, groupByValue)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return groupByValues, nil
}

// withProgressContext wraps the context with a progress tracking
func (c *Connector) withProgressContext(ctx context.Context, namespace string, clientID string, countSQL string, countArgs []interface{}) (context.Context, error) {
	totalRows := uint64(0)
	successRows := uint64(0)

	// Count the total number of rows
	countRows, err := c.config.ClickHouse.Query(ctx, countSQL, countArgs...)
	if err != nil {
		return ctx, fmt.Errorf("count query: %w", err)
	}

	defer countRows.Close()

	for countRows.Next() {
		if err := countRows.Scan(&totalRows); err != nil {
			return ctx, fmt.Errorf("count row scan: %w", err)
		}
	}

	if err := countRows.Err(); err != nil {
		return ctx, fmt.Errorf("count rows error: %w", err)
	}

	// Use context to pass a call back for progress and profile info
	ctx = clickhouse.Context(ctx, clickhouse.WithProgress(func(p *clickhouse.Progress) {
		successRows += p.Rows

		progress := progressmanagerentity.Progress{
			ProgressID: progressmanagerentity.ProgressID{
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ID: clientID,
			},
			Total: totalRows,
			// Rows it scans is maybe more than the total rows returned by the count query
			Success:   min(successRows, totalRows),
			UpdatedAt: time.Now(),
		}

		// Update progress
		err = c.config.ProgressManager.UpsertProgress(ctx, progressmanagerentity.UpsertProgressInput{
			Progress: progress,
		})
		// Log error but don't return it
		if err != nil {
			c.config.Logger.Error("failed to upsert progress", "error", err)
		}
	}))

	return ctx, nil
}

func min[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}
