package clickhouse_connector_mv

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
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
	Logger               *slog.Logger
	ClickHouse           clickhouse.Conn
	Database             string
	Meters               meter.Repository
	CreateOrReplaceMeter bool
	PopulateMeter        bool
	AsyncInsert          bool
	AsyncInsertWait      bool
	InsertQuerySettings  map[string]string
	QueryRawEvents       bool
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

	if c.Meters == nil {
		return fmt.Errorf("meters repository is required")
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

	return connector, nil
}

func (c *ClickhouseConnector) CreateNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (c *ClickhouseConnector) DeleteNamespace(ctx context.Context, namespace string) error {
	err := c.deleteNamespace(ctx, namespace)
	if err != nil {
		return fmt.Errorf("delete namespace in clickhouse: %w", err)
	}
	return nil
}

func (c *ClickhouseConnector) BatchInsert(ctx context.Context, rawEvents []streaming.RawEvent, meterEvents []streaming.MeterEvent) error {
	return c.rawEventConnector.BatchInsert(ctx, rawEvents, meterEvents)
}

func (c *ClickhouseConnector) CountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
	return c.rawEventConnector.CountEvents(ctx, namespace, params)
}

func (c *ClickhouseConnector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]api.IngestedEvent, error) {
	return c.rawEventConnector.ListEvents(ctx, namespace, params)
}

func (c *ClickhouseConnector) CreateMeter(ctx context.Context, namespace string, meter models.Meter) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if meter.Slug == "" {
		return fmt.Errorf("meter is required")
	}

	err := c.createMeterView(ctx, namespace, meter)
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) DeleteMeter(ctx context.Context, namespace string, meter models.Meter) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if meter.Slug == "" {
		return fmt.Errorf("meter is required")
	}

	err := c.deleteMeterView(ctx, namespace, meter)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			return err
		}

		return fmt.Errorf("delete meter view: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) QueryMeter(ctx context.Context, namespace string, meter models.Meter, params streaming.QueryParams) ([]models.MeterQueryRow, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if meter.Slug == "" {
		return nil, fmt.Errorf("meter is required")
	}

	// Query raw events if the flag is set
	if c.config.QueryRawEvents {
		return c.rawEventConnector.QueryMeter(ctx, namespace, meter, params)
	}

	values, err := c.queryMeterView(ctx, namespace, meter, params)
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

	// Query raw events if the flag is set
	if c.config.QueryRawEvents {
		return c.rawEventConnector.ListMeterSubjects(ctx, namespace, meter, params)
	}

	subjects, err := c.listMeterViewSubjects(ctx, namespace, meter, params)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			return nil, err
		}

		return nil, fmt.Errorf("list meter subjects: %w", err)
	}

	return subjects, nil
}

// DeleteNamespace deletes the namespace related resources from Clickhouse
// We don't delete the events table as it it reused between namespaces
// We only delete the materialized views for the meters
func (c *ClickhouseConnector) deleteNamespace(ctx context.Context, namespace string) error {
	// Retrieve meters belonging to the namespace
	meters, err := c.config.Meters.ListMeters(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to list meters: %w", err)
	}

	for _, meter := range meters {
		err := c.deleteMeterView(ctx, namespace, meter)
		if err != nil {
			// If the meter view does not exist, we ignore the error
			if _, ok := err.(*models.MeterNotFoundError); ok {
				return nil
			}
			return fmt.Errorf("delete meter view: %w", err)
		}
	}

	return nil
}

func (c *ClickhouseConnector) createMeterView(ctx context.Context, namespace string, meter models.Meter) error {
	// CreateOrReplace is used to force the recreation of the materialized view
	// This is not safe to use in production as it will drop the existing views
	if c.config.CreateOrReplaceMeter {
		err := c.deleteMeterView(ctx, namespace, meter)
		if err != nil {
			return fmt.Errorf("drop meter view: %w", err)
		}
	}

	view := createMeterView{
		Populate:      c.config.PopulateMeter,
		Database:      c.config.Database,
		Namespace:     namespace,
		MeterSlug:     meter.Slug,
		Aggregation:   meter.Aggregation,
		EventType:     meter.EventType,
		ValueProperty: meter.ValueProperty,
		GroupBy:       meter.GroupBy,
	}
	sql, args, err := view.toSQL()
	if err != nil {
		return fmt.Errorf("create meter view: %w", err)
	}
	err = c.config.ClickHouse.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("create meter view: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) deleteMeterView(ctx context.Context, namespace string, meter models.Meter) error {
	query := deleteMeterView{
		Database:  c.config.Database,
		Namespace: namespace,
		MeterSlug: meter.Slug,
	}

	sql := query.toSQL()

	err := c.config.ClickHouse.Exec(ctx, sql)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return &models.MeterNotFoundError{MeterSlug: meter.Slug}
		}

		return fmt.Errorf("delete meter view: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) queryMeterView(ctx context.Context, namespace string, meter models.Meter, params streaming.QueryParams) ([]models.MeterQueryRow, error) {
	queryMeter := queryMeterView{
		Database:       c.config.Database,
		Namespace:      namespace,
		MeterSlug:      meter.Slug,
		Aggregation:    meter.Aggregation,
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
		value := models.MeterQueryRow{
			GroupBy: map[string]*string{},
		}

		args := []interface{}{&value.WindowStart, &value.WindowEnd, &value.Value}
		argCount := len(args)

		for range queryMeter.GroupBy {
			tmp := ""
			args = append(args, &tmp)
		}

		if err := rows.Scan(args...); err != nil {
			return values, fmt.Errorf("query meter view row scan: %w", err)
		}

		for i, key := range queryMeter.GroupBy {
			if s, ok := args[i+argCount].(*string); ok {
				if key == "subject" {
					value.Subject = s
					continue
				}

				// We treat empty string as nil
				if s != nil && *s == "" {
					value.GroupBy[key] = nil
				} else {
					value.GroupBy[key] = s
				}
			}
		}

		// an empty row is returned when there are no values for the meter
		if value.WindowStart.IsZero() && value.WindowEnd.IsZero() && value.Value == 0 {
			continue
		}

		values = append(values, value)
	}
	rows.Close()
	err = rows.Err()
	if err != nil {
		return values, fmt.Errorf("query meter rows error: %w", err)
	}

	return values, nil
}

func (c *ClickhouseConnector) listMeterViewSubjects(ctx context.Context, namespace string, meter models.Meter, params streaming.ListMeterSubjectsParams) ([]string, error) {
	query := listMeterViewSubjects{
		Database:  c.config.Database,
		Namespace: namespace,
		MeterSlug: meter.Slug,
		From:      params.From,
		To:        params.To,
	}

	sql, args := query.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, &models.MeterNotFoundError{MeterSlug: meter.Slug}
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
