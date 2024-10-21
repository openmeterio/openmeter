package clickhouse_connector_parse

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/shopspring/decimal"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	raw_event_connector "github.com/openmeterio/openmeter/openmeter/streaming/clickhouse_connector_raw"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ streaming.Connector = (*ClickhouseConnector)(nil)

// ClickhouseConnector implements `ingest.Connector“ and `namespace.Handler interfaces.
type ClickhouseConnector struct {
	config            ClickhouseConnectorConfig
	rawEventConnector *raw_event_connector.ClickhouseConnector
}

type ClickhouseConnectorConfig struct {
	Logger              *slog.Logger
	ClickHouse          clickhouse.Conn
	Database            string
	AsyncInsert         bool
	AsyncInsertWait     bool
	InsertQuerySettings map[string]string
}

func (c ClickhouseConnectorConfig) Validate() error {
	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	if c.ClickHouse == nil {
		return fmt.Errorf("clickhouse connection is required")
	}

	if c.Database == "" {
		return fmt.Errorf("database is required")
	}

	return nil
}

func NewClickhouseConnector(ctx context.Context, config ClickhouseConnectorConfig) (*ClickhouseConnector, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	rawEventConnector, err := raw_event_connector.NewClickhouseConnector(ctx, raw_event_connector.ClickhouseConnectorConfig{
		Logger:              config.Logger,
		ClickHouse:          config.ClickHouse,
		Database:            config.Database,
		AsyncInsert:         config.AsyncInsert,
		AsyncInsertWait:     config.AsyncInsertWait,
		InsertQuerySettings: config.InsertQuerySettings,
	})
	if err != nil {
		return nil, fmt.Errorf("create raw event connector: %w", err)
	}

	connector := &ClickhouseConnector{
		config:            config,
		rawEventConnector: rawEventConnector,
	}

	err = connector.createMeterEventTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("create meter events table in clickhouse: %w", err)
	}

	return connector, nil
}

func (c *ClickhouseConnector) CreateNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (c *ClickhouseConnector) DeleteNamespace(ctx context.Context, namespace string) error {
	// We don't delete the event tables as it it reused between namespaces
	return nil
}

func (c *ClickhouseConnector) BatchInsert(ctx context.Context, rawEvents []streaming.RawEvent, meterEvents []streaming.MeterEvent) error {
	// Insert raw events
	err := c.rawEventConnector.BatchInsert(ctx, rawEvents, meterEvents)
	if err != nil {
		return fmt.Errorf("failed to batch insert raw events: %w", err)
	}

	// NOTE: The two inserts are not atomic.
	// If the second insert fails, the first insert will not be rolled back.

	// Insert meter events
	if len(meterEvents) == 0 {
		return nil
	}

	query := InsertMeterEventsQuery{
		Database:      c.config.Database,
		MeterEvents:   meterEvents,
		QuerySettings: c.config.InsertQuerySettings,
	}
	sql, args := query.ToSQL()

	if c.config.AsyncInsert {
		err = c.config.ClickHouse.AsyncInsert(ctx, sql, c.config.AsyncInsertWait, args...)
	} else {
		err = c.config.ClickHouse.Exec(ctx, sql, args...)
	}

	if err != nil {
		return fmt.Errorf("failed to batch insert meter events: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) CountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
	return c.rawEventConnector.CountEvents(ctx, namespace, params)
}

func (c *ClickhouseConnector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]api.IngestedEvent, error) {
	return c.rawEventConnector.ListEvents(ctx, namespace, params)
}

func (c *ClickhouseConnector) CreateMeter(ctx context.Context, namespace string, meter models.Meter) error {
	// Do nothing
	return nil
}

func (c *ClickhouseConnector) DeleteMeter(ctx context.Context, namespace string, meter models.Meter) error {
	// Do nothing
	return nil
}

func (c *ClickhouseConnector) QueryMeter(ctx context.Context, namespace string, meter models.Meter, params streaming.QueryParams) ([]models.MeterQueryRow, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	values, err := c.queryMeter(ctx, namespace, meter, params)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			return nil, err
		}

		return nil, fmt.Errorf("get values: %w", err)
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

func (c *ClickhouseConnector) ListMeterSubjects(ctx context.Context, namespace string, meter models.Meter, params streaming.ListMeterSubjectsParams) ([]string, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if meter.Slug == "" {
		return nil, fmt.Errorf("meter is required")
	}

	subjects, err := c.listMeterViewSubjects(ctx, namespace, meter.Slug, params.From, params.To)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			return nil, err
		}

		return nil, fmt.Errorf("list meter subjects: %w", err)
	}

	return subjects, nil
}

func (c *ClickhouseConnector) createMeterEventTable(ctx context.Context) error {
	table := createMeterEventTable{
		Database: c.config.Database,
	}

	err := c.config.ClickHouse.Exec(ctx, table.toSQL())
	if err != nil {
		return fmt.Errorf("create meter event table: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) queryMeter(ctx context.Context, namespace string, meter models.Meter, params streaming.QueryParams) ([]models.MeterQueryRow, error) {
	queryMeter := queryMeter{
		Database:       c.config.Database,
		Namespace:      namespace,
		Meter:          meter,
		From:           params.From,
		To:             params.To,
		Subject:        params.FilterSubject,
		FilterGroupBy:  params.FilterGroupBy,
		GroupBy:        params.GroupBy,
		WindowSize:     params.WindowSize,
		WindowTimeZone: params.WindowTimeZone,
	}

	values := []models.MeterQueryRow{}

	sql, args, err := queryMeter.toSQL()
	if err != nil {
		return values, fmt.Errorf("query meter view: %w", err)
	}

	start := time.Now()
	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, &models.MeterNotFoundError{MeterSlug: meter.Slug}
		}

		return values, fmt.Errorf("query meter view query: %w", err)
	}
	elapsed := time.Since(start)
	slog.Debug("query meter view", "elapsed", elapsed.String(), "sql", sql, "args", args)

	for rows.Next() {
		row := models.MeterQueryRow{
			GroupBy: map[string]*string{},
		}

		var value decimal.Decimal
		args := []interface{}{&row.WindowStart, &row.WindowEnd, &value}
		argCount := len(args)

		for range queryMeter.GroupBy {
			tmp := ""
			args = append(args, &tmp)
		}

		if err := rows.Scan(args...); err != nil {
			return values, fmt.Errorf("query meter view row scan: %w", err)
		}

		// TODO: should we use decima all the way?
		row.Value, _ = value.Float64()

		for i, key := range queryMeter.GroupBy {
			if s, ok := args[i+argCount].(*string); ok {
				if key == "subject" {
					row.Subject = s
					continue
				}

				// We treat empty string as nil
				if s != nil && *s == "" {
					row.GroupBy[key] = nil
				} else {
					row.GroupBy[key] = s
				}
			}
		}

		// an empty row is returned when there are no values for the meter
		if row.WindowStart.IsZero() && row.WindowEnd.IsZero() && row.Value == 0 {
			continue
		}

		values = append(values, row)
	}
	rows.Close()
	err = rows.Err()
	if err != nil {
		return values, fmt.Errorf("query meter rows error: %w", err)
	}

	return values, nil
}

func (c *ClickhouseConnector) listMeterViewSubjects(ctx context.Context, namespace string, meterSlug string, from *time.Time, to *time.Time) ([]string, error) {
	query := listMeterSubjectsQuery{
		Database:  c.config.Database,
		Namespace: namespace,
		MeterSlug: meterSlug,
		From:      from,
		To:        to,
	}

	sql, args := query.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, &models.MeterNotFoundError{MeterSlug: meterSlug}
		}

		return nil, fmt.Errorf("list meter view subjects: %w", err)
	}

	subjects := []string{}
	for rows.Next() {
		var subject string
		if err = rows.Scan(&subject); err != nil {
			return nil, err
		}

		subjects = append(subjects, subject)
	}

	return subjects, nil
}
