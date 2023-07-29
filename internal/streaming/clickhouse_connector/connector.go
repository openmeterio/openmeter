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

// ClickhouseConnector implements `ingest.Connector“ and `namespace.Handler interfaces.
type ClickhouseConnector struct {
	config *ClickhouseConnectorConfig
}

type ClickhouseConnectorConfig struct {
	Logger       *slog.Logger
	KafkaConnect *sink.KafkaConnect
	ClickHouse   clickhouse.Conn
	Database     string
}

func NewClickhouseConnector(config *ClickhouseConnectorConfig) (*ClickhouseConnector, error) {
	connector := &ClickhouseConnector{
		config: config,
	}

	return connector, nil
}

func (c *ClickhouseConnector) Init(meter *models.Meter, namespace string) error {
	// TODO: bass context to Init, also consider renaming it to CreateMeter
	ctx := context.Background()

	err := c.createMetersTable(ctx, namespace, meter)
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}

	return nil
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

func (c *ClickhouseConnector) GetValues(meter *models.Meter, params *streaming.GetValuesParams, namespace string) ([]*models.MeterValue, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *ClickhouseConnector) createEventsTable(ctx context.Context, namespace string) error {
	query, err := templateQuery(createEventsTableTemplate, createEventsTableData{
		Database:        c.config.Database,
		EventsTableName: getEventsTableName(namespace),
	})
	if err != nil {
		return fmt.Errorf("create events table: %w", err)
	}

	return c.config.ClickHouse.Exec(ctx, query)
}

func (c *ClickhouseConnector) createMetersTable(ctx context.Context, namespace string, meter *models.Meter) error {
	query, err := templateQuery(createMeterViewTemplate, createMeterViewData{
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
		return fmt.Errorf("create meter table: %w", err)
	}

	return nil
}

func (c *ClickhouseConnector) createSinkConnector(ctx context.Context, namespace string) error {
	connector := &sink.Connector{
		Name: "clickhouse",
		Config: map[string]string{
			"connector.class":                "com.clickhouse.kafka.connect.ClickHouseSinkConnector",
			"database":                       "default",
			"errors.retry.timeout":           "30",
			"hostname":                       "clickhouse",
			"port":                           "8123",
			"ssl":                            "false",
			"username":                       "default",
			"password":                       "",
			"key.converter":                  "org.apache.kafka.connect.storage.StringConverter",
			"value.converter":                "org.apache.kafka.connect.json.JsonConverter",
			"value.converter.schemas.enable": "false",
			"schemas.enable":                 "false",
			"topics":                         getEventsTableName(namespace),
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
