package clickhouse_connector

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/internal/sink"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

var prefix = "om"
var eventsTableName = "events"
var aggregationRegexp = regexp.MustCompile(`^AggregateFunction\((sum|min|max|count), Float64\)$`)

type SinkConfig struct {
	Database string
	Hostname string
	Port     int
	SSL      bool
	Username string
	Password string
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
		return fmt.Errorf("delete meter view: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) QueryMeter(ctx context.Context, namespace string, meterSlug string, params *streaming.GetValuesParams) ([]*models.MeterValue, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	meterView, err := c.describeMeterView(ctx, namespace, meterSlug)
	if err != nil {
		return nil, err
	}

	values, err := c.queryMeterView(ctx, namespace, meterView, params)
	if err != nil {
		return values, fmt.Errorf("get values: %w", err)
	}

	// TODO: aggregate windows in query
	return models.AggregateMeterValues(values, meterView.Aggregation, models.WindowSizeMinute, params.WindowSize)
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
	query, err := streaming.TemplateQuery(createEventsTableTemplate, createEventsTableData{
		Database:        c.config.Database,
		EventsTableName: getEventsTableName(namespace),
	})
	if err != nil {
		return fmt.Errorf("create events table: %w", err)
	}

	return c.config.ClickHouse.Exec(ctx, query)
}

func (c *ClickhouseConnector) createMeterView(ctx context.Context, namespace string, meter *models.Meter) error {
	query, err := streaming.TemplateQuery(createMeterViewTemplate, createMeterViewData{
		Database:        c.config.Database,
		EventsTableName: getEventsTableName(namespace),
		EventType:       meter.EventType,
		MeterViewName:   getMeterViewNameBySlug(namespace, meter.Slug),
		ValueProperty:   meter.ValueProperty,
		GroupBy:         meter.GroupBy,
	})
	if err != nil {
		return err
	}
	err = c.config.ClickHouse.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("create meter view: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) deleteMeterView(ctx context.Context, namespace string, meterSlug string) error {
	query, err := streaming.TemplateQuery(deleteMeterViewTemplate, deleteMeterViewData{
		Database:      c.config.Database,
		MeterViewName: getMeterViewNameBySlug(namespace, meterSlug),
	})
	if err != nil {
		if strings.Contains(err.Error(), "code: 60") {
			return &models.MeterNotFoundError{MeterSlug: meterSlug}
		}

		return err
	}
	err = c.config.ClickHouse.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("delete meter view: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) describeMeterView(ctx context.Context, namespace string, meterSlug string) (*MeterView, error) {
	query, err := streaming.TemplateQuery(describeMeterViewTemplate, describeMeterViewData{
		Database:      c.config.Database,
		MeterViewName: getMeterViewNameBySlug(namespace, meterSlug),
	})
	if err != nil {
		return nil, err
	}
	rows, err := c.config.ClickHouse.Query(ctx, query)
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

func (c *ClickhouseConnector) queryMeterView(ctx context.Context, namespace string, meterView *MeterView, params *streaming.GetValuesParams) ([]*models.MeterValue, error) {
	values := []*models.MeterValue{}

	query, err := streaming.TemplateQuery(queryMeterViewTemplate, queryMeterViewData{
		Database:      c.config.Database,
		MeterViewName: getMeterViewNameBySlug(namespace, meterView.Slug),
		Subject:       params.Subject,
		From:          params.From,
		To:            params.To,
		GroupBy:       meterView.GroupBy,
		// TODO: implement window size
		WindowSize: params.WindowSize,
	})
	if err != nil {
		return values, err
	}
	rows, err := c.config.ClickHouse.Query(ctx, query)
	if err != nil {
		return values, fmt.Errorf("query meter view query: %w", err)
	}

	for rows.Next() {
		value := &models.MeterValue{
			GroupBy: map[string]string{},
		}
		args := []interface{}{&value.WindowStart, &value.WindowEnd, &value.Subject, &value.Value}
		// TODO: do this next part without interface magic
		for range meterView.GroupBy {
			tmp := ""
			args = append(args, &tmp)
		}

		if err := rows.Scan(args...); err != nil {
			return values, fmt.Errorf("query meter view row scan: %w", err)
		}

		for i, key := range meterView.GroupBy {
			if s, ok := args[i+4].(*string); ok {
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

func (c *ClickhouseConnector) createSinkConnector(ctx context.Context, namespace string) error {
	connector := sink.Connector{
		Name: "clickhouse",
		Config: map[string]string{
			"connector.class":                "com.clickhouse.kafka.connect.ClickHouseSinkConnector",
			"database":                       c.config.SinkConfig.Database,
			"errors.retry.timeout":           "30",
			"hostname":                       c.config.SinkConfig.Hostname,
			"port":                           fmt.Sprint(c.config.SinkConfig.Port),
			"ssl":                            fmt.Sprint(c.config.SinkConfig.SSL),
			"username":                       c.config.SinkConfig.Username,
			"password":                       c.config.SinkConfig.Password,
			"key.converter":                  "org.apache.kafka.connect.storage.StringConverter",
			"value.converter":                "org.apache.kafka.connect.json.JsonConverter",
			"value.converter.schemas.enable": "false",
			"schemas.enable":                 "false",
			"topics.regex":                   "^om_[A-Za-z0-9]+(?:_[A-Za-z0-9]+)*_events$",
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
