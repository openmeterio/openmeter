package clickhouse_connector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	tablePrefix     = "om_"
	EventsTableName = "events"
)

// ClickhouseConnector implements `ingest.Connector“ and `namespace.Handler interfaces.
type ClickhouseConnector struct {
	config ClickhouseConnectorConfig
}

type ClickhouseConnectorConfig struct {
	Logger               *slog.Logger
	ClickHouse           clickhouse.Conn
	Database             string
	Meters               meter.Repository
	CreateOrReplaceMeter bool
	PopulateMeter        bool
}

func NewClickhouseConnector(config ClickhouseConnectorConfig) (*ClickhouseConnector, error) {
	connector := &ClickhouseConnector{
		config: config,
	}

	return connector, nil
}

func (c *ClickhouseConnector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]api.IngestedEvent, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
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

func (c *ClickhouseConnector) CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	err := c.createMeterView(ctx, namespace, meter)
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) DeleteMeter(ctx context.Context, namespace string, meterSlug string) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if meterSlug == "" {
		return fmt.Errorf("slug is required")
	}

	err := c.deleteMeterView(ctx, namespace, meterSlug)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			return err
		}

		return fmt.Errorf("delete meter view: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) QueryMeter(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) ([]models.MeterQueryRow, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	values, err := c.queryMeterView(ctx, namespace, meterSlug, params)
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

func (c *ClickhouseConnector) ListMeterSubjects(ctx context.Context, namespace string, meterSlug string, from *time.Time, to *time.Time) ([]string, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if meterSlug == "" {
		return nil, fmt.Errorf("slug is required")
	}

	subjects, err := c.listMeterViewSubjects(ctx, namespace, meterSlug, from, to)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			return nil, err
		}

		return nil, fmt.Errorf("list meter subjects: %w", err)
	}

	return subjects, nil
}

func (c *ClickhouseConnector) CreateNamespace(ctx context.Context, namespace string) error {
	err := c.createEventsTable(ctx)
	if err != nil {
		return fmt.Errorf("create namespace in clickhouse: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) DeleteNamespace(ctx context.Context, namespace string) error {
	err := c.deleteNamespace(ctx, namespace)
	if err != nil {
		return fmt.Errorf("delete namespace in clickhouse: %w", err)
	}
	return nil
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
		err := c.deleteMeterView(ctx, namespace, meter.Slug)
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

func (c *ClickhouseConnector) CountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
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

func (c *ClickhouseConnector) createEventsTable(ctx context.Context) error {
	table := createEventsTable{
		Database: c.config.Database,
	}

	err := c.config.ClickHouse.Exec(ctx, table.toSQL())
	if err != nil {
		return fmt.Errorf("create events table: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) queryEventsTable(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]api.IngestedEvent, error) {
	table := queryEventsTable{
		Database:  c.config.Database,
		Namespace: namespace,
		From:      params.From,
		To:        params.To,
		Limit:     params.Limit,
	}

	sql, args := table.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, &models.NamespaceNotFoundError{Namespace: namespace}
		}

		return nil, fmt.Errorf("query events table query: %w", err)
	}

	events := []api.IngestedEvent{}

	for rows.Next() {
		var id string
		var eventType string
		var subject string
		var source string
		var eventTime time.Time
		var dataStr string
		var validationError string
		var ingestedAt time.Time
		var createdAt time.Time

		if err = rows.Scan(&id, &eventType, &subject, &source, &eventTime, &dataStr, &validationError, &ingestedAt, &createdAt); err != nil {
			return nil, err
		}

		// Parse data
		var data interface{}
		err := json.Unmarshal([]byte(dataStr), &data)
		if err != nil {
			return nil, fmt.Errorf("query events parse data: %w", err)
		}

		event := event.New()
		event.SetID(id)
		event.SetType(eventType)
		event.SetSubject(subject)
		event.SetSource(source)
		event.SetTime(eventTime)
		err = event.SetData("application/json", data)
		if err != nil {
			return nil, fmt.Errorf("query events set data: %w", err)
		}

		ingestedEvent := api.IngestedEvent{
			Event: event,
		}

		if validationError != "" {
			ingestedEvent.ValidationError = &validationError
		}

		ingestedEvent.IngestedAt = ingestedAt
		ingestedEvent.CreatedAt = createdAt

		events = append(events, ingestedEvent)
	}

	return events, nil
}

func (c *ClickhouseConnector) queryCountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
	table := queryCountEvents{
		Database:  c.config.Database,
		Namespace: namespace,
		From:      params.From,
	}

	sql, args := table.toSQL()

	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, &models.NamespaceNotFoundError{Namespace: namespace}
		}

		return nil, fmt.Errorf("query events count query: %w", err)
	}

	results := []streaming.CountEventRow{}

	for rows.Next() {
		result := streaming.CountEventRow{}

		if err = rows.Scan(&result.Count, &result.Subject, &result.IsError); err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

func (c *ClickhouseConnector) createMeterView(ctx context.Context, namespace string, meter *models.Meter) error {
	// CreateOrReplace is used to force the recreation of the materialized view
	// This is not safe to use in production as it will drop the existing views
	if c.config.CreateOrReplaceMeter {
		err := c.deleteMeterView(ctx, namespace, meter.Slug)
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

func (c *ClickhouseConnector) deleteMeterView(ctx context.Context, namespace string, meterSlug string) error {
	query := deleteMeterView{
		Database:  c.config.Database,
		Namespace: namespace,
		MeterSlug: meterSlug,
	}

	sql := query.toSQL()

	err := c.config.ClickHouse.Exec(ctx, sql)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return &models.MeterNotFoundError{MeterSlug: meterSlug}
		}

		return fmt.Errorf("delete meter view: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) queryMeterView(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) ([]models.MeterQueryRow, error) {
	queryMeter := queryMeterView{
		Database:       c.config.Database,
		Namespace:      namespace,
		MeterSlug:      meterSlug,
		Aggregation:    params.Aggregation,
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
			return nil, &models.MeterNotFoundError{MeterSlug: meterSlug}
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

func (c *ClickhouseConnector) listMeterViewSubjects(ctx context.Context, namespace string, meterSlug string, from *time.Time, to *time.Time) ([]string, error) {
	query := listMeterViewSubjects{
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
