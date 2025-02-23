package materialized_view

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	raw_events "github.com/openmeterio/openmeter/openmeter/streaming/clickhouse/raw_events"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ streaming.Connector = (*Connector)(nil)

// Connector implements `ingest.Connector“ and `namespace.Handler interfaces.
type Connector struct {
	config            ConnectorConfig
	rawEventConnector *raw_events.Connector
}

type ConnectorConfig struct {
	Logger          *slog.Logger
	ClickHouse      clickhouse.Conn
	Database        string
	EventsTableName string
	Meters          meter.Service

	CreateOrReplaceMeter bool
	PopulateMeter        bool
	AsyncInsert          bool
	AsyncInsertWait      bool
	InsertQuerySettings  map[string]string
	QueryRawEvents       bool
}

func (c ConnectorConfig) Validate() error {
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
		return fmt.Errorf("events table name is required")
	}

	if c.Meters == nil {
		return fmt.Errorf("meters repository is required")
	}

	return nil
}

func NewConnector(ctx context.Context, config ConnectorConfig) (*Connector, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	rawEventConnector, err := raw_events.NewConnector(ctx, raw_events.ConnectorConfig{
		Logger:              config.Logger,
		ClickHouse:          config.ClickHouse,
		Database:            config.Database,
		EventsTableName:     config.EventsTableName,
		AsyncInsert:         config.AsyncInsert,
		AsyncInsertWait:     config.AsyncInsertWait,
		InsertQuerySettings: config.InsertQuerySettings,
	})
	if err != nil {
		return nil, fmt.Errorf("create raw event connector: %w", err)
	}

	connector := &Connector{
		config:            config,
		rawEventConnector: rawEventConnector,
	}

	return connector, nil
}

func (c *Connector) CreateNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (c *Connector) DeleteNamespace(ctx context.Context, namespace string) error {
	err := c.deleteNamespace(ctx, namespace)
	if err != nil {
		return fmt.Errorf("delete namespace in clickhouse: %w", err)
	}
	return nil
}

func (c *Connector) BatchInsert(ctx context.Context, rawEvents []streaming.RawEvent) error {
	return c.rawEventConnector.BatchInsert(ctx, rawEvents)
}

func (c *Connector) CountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
	return c.rawEventConnector.CountEvents(ctx, namespace, params)
}

func (c *Connector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]api.IngestedEvent, error) {
	return c.rawEventConnector.ListEvents(ctx, namespace, params)
}

func (c *Connector) CreateMeter(ctx context.Context, namespace string, meter meterpkg.Meter) error {
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

func (c *Connector) DeleteMeter(ctx context.Context, namespace string, meter meterpkg.Meter) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if meter.Slug == "" {
		return fmt.Errorf("meter is required")
	}

	err := c.deleteMeterView(ctx, namespace, meter)
	if err != nil {
		if meterpkg.IsMeterNotFoundError(err) {
			return err
		}

		return fmt.Errorf("delete meter view: %w", err)
	}

	return nil
}

func (c *Connector) QueryMeter(ctx context.Context, namespace string, meter meterpkg.Meter, params streaming.QueryParams) ([]meterpkg.MeterQueryRow, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if meter.Slug == "" {
		return nil, fmt.Errorf("meter is required")
	}

	if err := params.Validate(meter); err != nil {
		return nil, fmt.Errorf("validate params: %w", err)
	}

	// Query raw events if the flag is set
	if c.config.QueryRawEvents {
		return c.rawEventConnector.QueryMeter(ctx, namespace, meter, params)
	}

	values, err := c.queryMeterView(ctx, namespace, meter, params)
	if err != nil {
		if meterpkg.IsMeterNotFoundError(err) {
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

func (c *Connector) ListMeterSubjects(ctx context.Context, namespace string, meter meterpkg.Meter, params streaming.ListMeterSubjectsParams) ([]string, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if meter.Slug == "" {
		return nil, fmt.Errorf("meter is required")
	}

	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("validate params: %w", err)
	}

	// Query raw events if the flag is set
	if c.config.QueryRawEvents {
		return c.rawEventConnector.ListMeterSubjects(ctx, namespace, meter, params)
	}

	subjects, err := c.listMeterViewSubjects(ctx, namespace, meter, params)
	if err != nil {
		if meterpkg.IsMeterNotFoundError(err) {
			return nil, err
		}

		return nil, fmt.Errorf("list meter subjects: %w", err)
	}

	return subjects, nil
}

// DeleteNamespace deletes the namespace related resources from Clickhouse
// We don't delete the events table as it it reused between namespaces
// We only delete the materialized views for the meters
func (c *Connector) deleteNamespace(ctx context.Context, namespace string) error {
	// Retrieve meters belonging to the namespace
	result, err := c.config.Meters.ListMeters(ctx, meterpkg.ListMetersParams{
		Namespace: namespace,
		Page:      pagination.NewPage(1, 100),
	})
	if err != nil {
		return fmt.Errorf("failed to list meters: %w", err)
	}

	for _, meter := range result.Items {
		err := c.deleteMeterView(ctx, namespace, meter)
		if err != nil {
			// If the meter view does not exist, we ignore the error
			if meterpkg.IsMeterNotFoundError(err) {
				return nil
			}
			return fmt.Errorf("delete meter view: %w", err)
		}
	}

	return nil
}

func (c *Connector) createMeterView(ctx context.Context, namespace string, meter meterpkg.Meter) error {
	// CreateOrReplace is used to force the recreation of the materialized view
	// This is not safe to use in production as it will drop the existing views
	if c.config.CreateOrReplaceMeter {
		err := c.deleteMeterView(ctx, namespace, meter)
		if err != nil {
			return fmt.Errorf("drop meter view: %w", err)
		}
	}

	view := createMeterView{
		Populate:        c.config.PopulateMeter,
		Database:        c.config.Database,
		EventsTableName: c.config.EventsTableName,
		Namespace:       namespace,
		MeterSlug:       meter.Slug,
		Aggregation:     meter.Aggregation,
		EventType:       meter.EventType,
		ValueProperty:   meter.ValueProperty,
		GroupBy:         meter.GroupBy,
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

func (c *Connector) deleteMeterView(ctx context.Context, namespace string, meter meterpkg.Meter) error {
	query := deleteMeterView{
		Database:  c.config.Database,
		Namespace: namespace,
		MeterSlug: meter.Slug,
	}

	sql := query.toSQL()

	err := c.config.ClickHouse.Exec(ctx, sql)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return meterpkg.NewMeterNotFoundError(meter.Slug)
		}

		return fmt.Errorf("delete meter view: %w", err)
	}

	return nil
}

func (c *Connector) queryMeterView(ctx context.Context, namespace string, meter meterpkg.Meter, params streaming.QueryParams) ([]meterpkg.MeterQueryRow, error) {
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

	values := []meterpkg.MeterQueryRow{}

	sql, args, err := queryMeter.toSQL()
	if err != nil {
		return values, fmt.Errorf("query meter view: %w", err)
	}

	start := time.Now()
	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, meterpkg.NewMeterNotFoundError(meter.Slug)
		}

		return values, fmt.Errorf("query meter view query: %w", err)
	}

	defer rows.Close()

	elapsed := time.Since(start)
	slog.Debug("query meter view", "elapsed", elapsed.String(), "sql", sql, "args", args)

	for rows.Next() {
		value := meterpkg.MeterQueryRow{
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

	err = rows.Err()
	if err != nil {
		return values, fmt.Errorf("rows error: %w", err)
	}

	return values, nil
}

func (c *Connector) listMeterViewSubjects(ctx context.Context, namespace string, meter meterpkg.Meter, params streaming.ListMeterSubjectsParams) ([]string, error) {
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
			return nil, meterpkg.NewMeterNotFoundError(meter.Slug)
		}

		return nil, fmt.Errorf("list meter view subjects: %w", err)
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
