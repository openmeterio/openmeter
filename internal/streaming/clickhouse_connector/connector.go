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

	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	prefix          = "om"
	eventsTableName = "events"
)

// ClickhouseConnector implements `ingest.Connector“ and `namespace.Handler interfaces.
type ClickhouseConnector struct {
	config ClickhouseConnectorConfig
}

type ClickhouseConnectorConfig struct {
	Logger     *slog.Logger
	ClickHouse clickhouse.Conn
	Database   string
	Meters     meter.Repository
}

func NewClickhouseConnector(config ClickhouseConnectorConfig) (*ClickhouseConnector, error) {
	connector := &ClickhouseConnector{
		config: config,
	}

	return connector, nil
}

func (c *ClickhouseConnector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]event.Event, error) {
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

func (c *ClickhouseConnector) QueryMeter(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) (*streaming.QueryResult, error) {
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

	return &streaming.QueryResult{
		WindowSize: params.WindowSize,
		Values:     values,
	}, nil
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
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	err := c.createEventsTable(ctx, namespace)
	if err != nil {
		return fmt.Errorf("create namespace in clickhouse: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) createEventsTable(ctx context.Context, namespace string) error {
	table := createEventsTable{
		Database:        c.config.Database,
		EventsTableName: GetEventsTableName(namespace),
	}

	err := c.config.ClickHouse.Exec(ctx, table.toSQL())
	if err != nil {
		return fmt.Errorf("create events table: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) queryEventsTable(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]event.Event, error) {
	table := queryEventsTable{
		Database:        c.config.Database,
		EventsTableName: GetEventsTableName(namespace),
		Limit:           params.Limit,
	}

	sql, _, err := table.toSQL()
	if err != nil {
		return nil, fmt.Errorf("query events table to sql: %w", err)
	}
	rows, err := c.config.ClickHouse.Query(ctx, sql)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, &models.NamespaceNotFoundError{Namespace: namespace}
		}

		return nil, fmt.Errorf("query events table query: %w", err)
	}

	events := []event.Event{}

	for rows.Next() {
		var id string
		var eventType string
		var subject string
		var source string
		var time time.Time
		var dataStr string

		if err = rows.Scan(&id, &eventType, &subject, &source, &time, &dataStr); err != nil {
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
		event.SetTime(time)
		err = event.SetData("application/json", data)
		if err != nil {
			return nil, fmt.Errorf("query events set data: %w", err)
		}

		events = append(events, event)
	}

	return events, nil
}

func (c *ClickhouseConnector) createMeterView(ctx context.Context, namespace string, meter *models.Meter) error {
	view := createMeterView{
		Database:        c.config.Database,
		EventsTableName: GetEventsTableName(namespace),
		Aggregation:     meter.Aggregation,
		EventType:       meter.EventType,
		MeterViewName:   getMeterViewNameBySlug(namespace, meter.Slug),
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

func (c *ClickhouseConnector) deleteMeterView(ctx context.Context, namespace string, meterSlug string) error {
	query := deleteMeterView{
		Database:      c.config.Database,
		MeterViewName: getMeterViewNameBySlug(namespace, meterSlug),
	}
	sql, args := query.toSQL()
	err := c.config.ClickHouse.Exec(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return &models.MeterNotFoundError{MeterSlug: meterSlug}
		}

		return fmt.Errorf("delete meter view: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) queryMeterView(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) ([]*models.MeterValue, error) {
	queryMeter := queryMeterView{
		Database:       c.config.Database,
		MeterViewName:  getMeterViewNameBySlug(namespace, meterSlug),
		Aggregation:    params.Aggregation,
		From:           params.From,
		To:             params.To,
		Subject:        params.Subject,
		GroupBySubject: params.GroupBySubject,
		GroupBy:        params.GroupBy,
		WindowSize:     params.WindowSize,
	}

	values := []*models.MeterValue{}

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
		value := &models.MeterValue{
			GroupBy: map[string]string{},
		}

		args := []interface{}{&value.WindowStart, &value.WindowEnd}
		argCount := 2

		if len(params.Subject) > 0 || params.GroupBySubject {
			args = append(args, &value.Subject)
			argCount++
		}

		args = append(args, &value.Value)
		argCount++

		// TODO: do this next part without interface magic
		for range queryMeter.GroupBy {
			tmp := ""
			args = append(args, &tmp)
		}

		if err := rows.Scan(args...); err != nil {
			return values, fmt.Errorf("query meter view row scan: %w", err)
		}

		for i, key := range queryMeter.GroupBy {
			if s, ok := args[i+argCount].(*string); ok {
				value.GroupBy[key] = *s
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
		Database:      c.config.Database,
		MeterViewName: getMeterViewNameBySlug(namespace, meterSlug),
		From:          from,
		To:            to,
	}

	sql, args, err := query.toSQL()
	if err != nil {
		return nil, fmt.Errorf("list meter view subjects: %w", err)
	}

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

func GetEventsTableName(namespace string) string {
	return fmt.Sprintf("%s_%s_%s", prefix, namespace, eventsTableName)
}

func getMeterViewNameBySlug(namespace string, meterSlug string) string {
	return fmt.Sprintf("%s_%s_%s", prefix, namespace, meterSlug)
}
