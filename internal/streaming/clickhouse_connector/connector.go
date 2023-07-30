package clickhouse_connector

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/internal/sink"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

var prefix = "om"
var eventsTableName = "events"

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
	Logger       *slog.Logger
	KafkaConnect sink.KafkaConnect
	ClickHouse   clickhouse.Conn
	Database     string
	SinkConfig   SinkConfig
}

func NewClickhouseConnector(config ClickhouseConnectorConfig) (*ClickhouseConnector, error) {
	connector := &ClickhouseConnector{
		config: config,
	}

	return connector, nil
}

func (c *ClickhouseConnector) Init(meter *models.Meter, namespace string) error {
	// TODO: pass context to Init, also consider renaming it to CreateMeter
	ctx := context.TODO()

	err := c.createMeterView(ctx, namespace, meter)
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) GetValues(meter *models.Meter, params *streaming.GetValuesParams, namespace string) ([]*models.MeterValue, error) {
	// TODO: pass context to GetValues, also consider renaming it to QueryMeter
	ctx := context.TODO()

	values, err := c.queryMeterView(ctx, namespace, meter, params)
	if err != nil {
		return values, fmt.Errorf("get values: %w", err)
	}

	// TODO: aggregate windows in query
	return meter.AggregateMeterValues(values, params.WindowSize)
}

func (c *ClickhouseConnector) CreateNamespace(ctx context.Context, namespace string) error {
	err := c.createEventsTable(ctx, namespace)
	if err != nil {
		return fmt.Errorf("create namespace in clickhouse: %w", err)
	}

	err = c.createSinkConnector(ctx, namespace)
	if err != nil {
		return fmt.Errorf("create namespace in clickhouse: %w", err)
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
		MeterViewName:   getMeterViewName(namespace, meter),
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

func (c *ClickhouseConnector) queryMeterView(ctx context.Context, namespace string, meter *models.Meter, params *streaming.GetValuesParams) ([]*models.MeterValue, error) {
	values := []*models.MeterValue{}

	groupBy := make([]string, 0, len(meter.GroupBy))
	for key := range meter.GroupBy {
		groupBy = append(groupBy, key)
	}

	query, err := streaming.TemplateQuery(queryMeterViewTemplate, queryMeterViewData{
		Database:      c.config.Database,
		MeterViewName: getMeterViewName(namespace, meter),
		Subject:       params.Subject,
		From:          params.From,
		To:            params.To,
		GroupBy:       groupBy,
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
		for range groupBy {
			tmp := ""
			args = append(args, &tmp)
		}

		if err := rows.Scan(args...); err != nil {
			return values, fmt.Errorf("query meter view row scan: %w", err)
		}

		for i, key := range groupBy {
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
			"topics.regex":                   "om_.+_events",
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

func getMeterViewName(namespace string, meter *models.Meter) string {
	return fmt.Sprintf("%s_%s_%s", prefix, namespace, meter.Slug)
}
