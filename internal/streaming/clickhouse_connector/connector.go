package clickhouse_connector

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"

	"github.com/openmeterio/openmeter/internal/sink"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	prefix          = "om"
	eventsTableName = "events"
)

// List of accepted aggregate functions: https://clickhouse.com/docs/en/sql-reference/aggregate-functions/reference
var aggregationRegexp = regexp.MustCompile(`^AggregateFunction\((avg|sum|min|max|count), Float64\)$`)

type SinkConfig struct {
	DeadLetterQueueTopicName         string
	DeadLetterQueueReplicationFactor int
	DeadLetterQueueContextHeaders    bool
	Database                         string
	Hostname                         string
	Port                             int
	SSL                              bool
	Username                         string
	Password                         string
}

// ClickhouseConnector implements `ingest.Connector“ and `namespace.Handler interfaces.
type ClickhouseConnector struct {
	config ClickhouseConnectorConfig
}

type ClickhouseConnectorConfig struct {
	Logger              *slog.Logger
	KafkaConnect        sink.KafkaConnect
	KafkaConnectEnabled bool
	ClickHouse          clickhouse.Conn
	Database            string
	SinkConfig          SinkConfig
}

func NewClickhouseConnector(config ClickhouseConnectorConfig) (*ClickhouseConnector, error) {
	connector := &ClickhouseConnector{
		config: config,
	}

	return connector, nil
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

func (c *ClickhouseConnector) QueryMeter(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) ([]*models.MeterValue, *models.WindowSize, error) {
	if namespace == "" {
		return nil, nil, fmt.Errorf("namespace is required")
	}

	// ClickHouse connector requires aggregation type to be set
	// If we don't have it we inspect the meter view in ClickHouse
	if params.Aggregation == nil {
		meterView, err := c.describeMeterView(ctx, namespace, meterSlug)
		if err != nil {
			return nil, nil, err
		}

		params.Aggregation = &meterView.Aggregation
	}

	values, err := c.queryMeterView(ctx, namespace, meterSlug, params)
	if err != nil {
		if _, ok := err.(*models.MeterNotFoundError); ok {
			return nil, nil, err
		}

		return nil, nil, fmt.Errorf("get values: %w", err)
	}

	return values, params.WindowSize, nil
}

func (c *ClickhouseConnector) ListMeterSubjects(ctx context.Context, namespace string, meterSlug string) ([]string, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if meterSlug == "" {
		return nil, fmt.Errorf("slug is required")
	}

	subjects, err := c.listMeterViewSubjects(ctx, namespace, meterSlug)
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

	if c.config.KafkaConnectEnabled {
		err = c.createSinkConnector(ctx, namespace)
		if err != nil {
			return fmt.Errorf("create namespace in clickhouse: %w", err)
		}
	}

	return nil
}

func (c *ClickhouseConnector) createEventsTable(ctx context.Context, namespace string) error {
	table := createEventsTable{
		Database:        c.config.Database,
		EventsTableName: getEventsTableName(namespace),
	}

	err := c.config.ClickHouse.Exec(ctx, table.toSQL())
	if err != nil {
		return fmt.Errorf("create events table: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) createMeterView(ctx context.Context, namespace string, meter *models.Meter) error {
	view := createMeterView{
		Database:        c.config.Database,
		EventsTableName: getEventsTableName(namespace),
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

func (c *ClickhouseConnector) describeMeterView(ctx context.Context, namespace string, meterSlug string) (*MeterView, error) {
	query := describeMeterView{
		Database:      c.config.Database,
		MeterViewName: getMeterViewNameBySlug(namespace, meterSlug),
	}
	sql, args := query.toSQL()
	rows, err := c.config.ClickHouse.Query(ctx, sql, args...)
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return nil, &models.MeterNotFoundError{MeterSlug: meterSlug}
		}

		return nil, fmt.Errorf("describe meter view: %w", err)
	}

	// get columns and types
	meterView := &MeterView{
		Slug: meterSlug,
	}
	for rows.Next() {
		var (
			colName string
			colType string
			ignore  string
		)

		if err = rows.Scan(&colName, &colType, &ignore, &ignore, &ignore, &ignore, &ignore); err != nil {
			return nil, err
		}

		if colName == "value" {
			// Parse aggregation type
			tmp := aggregationRegexp.FindStringSubmatch(colType)
			if len(tmp) != 2 || tmp[1] == "" {
				// This should never happen, if it happens it means the view changed fundamanetally
				return nil, fmt.Errorf("aggregation type not found in regex: %s", colType)
			}

			// Validate aggregation type
			aggregationStr := strings.ToUpper(tmp[1])
			if ok := models.MeterAggregation("").IsValid(aggregationStr); !ok {
				return nil, fmt.Errorf("invalid aggregation type: %s", aggregationStr)
			}
			meterView.Aggregation = models.MeterAggregation(aggregationStr)
		}

		if colName != "windowstart" && colName != "windowend" && colName != "subject" && colName != "value" {
			meterView.GroupBy = append(meterView.GroupBy, colName)
		}
	}

	return meterView, nil
}

func (c *ClickhouseConnector) queryMeterView(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) ([]*models.MeterValue, error) {
	if params.Aggregation == nil {
		return nil, fmt.Errorf("aggregation is required")
	}

	queryMeter := queryMeterView{
		Database:       c.config.Database,
		MeterViewName:  getMeterViewNameBySlug(namespace, meterSlug),
		Aggregation:    *params.Aggregation,
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

		args := []interface{}{}
		var argCount int

		if params.WindowSize != nil {
			args = append(args, &value.WindowStart, &value.WindowEnd)
			argCount += 2
		}

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

		values = append(values, value)
	}
	rows.Close()
	err = rows.Err()
	if err != nil {
		return values, fmt.Errorf("query meter rows error: %w", err)
	}

	return values, nil
}

func (c *ClickhouseConnector) listMeterViewSubjects(ctx context.Context, namespace string, meterSlug string) ([]string, error) {
	query := listMeterViewSubjects{
		Database:      c.config.Database,
		MeterViewName: getMeterViewNameBySlug(namespace, meterSlug),
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

func (c *ClickhouseConnector) createSinkConnector(ctx context.Context, namespace string) error {
	connector := sink.Connector{
		Name: "clickhouse",
		Config: map[string]string{
			"connector.class":                   "com.clickhouse.kafka.connect.ClickHouseSinkConnector",
			"database":                          c.config.SinkConfig.Database,
			"errors.retry.timeout":              "30",
			"hostname":                          c.config.SinkConfig.Hostname,
			"port":                              fmt.Sprint(c.config.SinkConfig.Port),
			"ssl":                               fmt.Sprint(c.config.SinkConfig.SSL),
			"username":                          c.config.SinkConfig.Username,
			"password":                          c.config.SinkConfig.Password,
			"key.converter":                     "org.apache.kafka.connect.storage.StringConverter",
			"value.converter":                   "org.apache.kafka.connect.json.JsonConverter",
			"value.converter.schemas.enable":    "false",
			"schemas.enable":                    "false",
			"topics.regex":                      "^om_[A-Za-z0-9]+(?:_[A-Za-z0-9]+)*_events$",
			"errors.tolerance":                  "all",
			"errors.deadletterqueue.topic.name": c.config.SinkConfig.DeadLetterQueueTopicName,
			"errors.deadletterqueue.topic.replication.factor": fmt.Sprint(c.config.SinkConfig.DeadLetterQueueReplicationFactor),
			"errors.deadletterqueue.context.headers.enable":   fmt.Sprint(c.config.SinkConfig.DeadLetterQueueContextHeaders),
		},
	}

	err := c.config.KafkaConnect.CreateConnector(ctx, connector)
	if err != nil {
		return fmt.Errorf("create sink connector: %w", err)
	}

	return nil
}

func getEventsTableName(namespace string) string {
	return fmt.Sprintf("%s_%s_%s", prefix, namespace, eventsTableName)
}

func getMeterViewNameBySlug(namespace string, meterSlug string) string {
	return fmt.Sprintf("%s_%s_%s", prefix, namespace, meterSlug)
}
