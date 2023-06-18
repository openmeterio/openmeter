package kafka_connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/thmeitz/ksqldb-go"
	"github.com/thmeitz/ksqldb-go/net"
	"golang.org/x/exp/slog"

	. "github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

type KafkaConnector struct {
	config       *KafkaConnectorConfig
	KsqlDBClient *ksqldb.KsqldbClient
}

type KafkaConnectorConfig struct {
	KsqlDB      *net.Options
	EventsTopic string
	Partitions  int

	KeySchemaID   int
	ValueSchemaID int
}

func NewKafkaConnector(config *KafkaConnectorConfig) (Connector, error) {
	// Initialize KSQLDB Client
	ksqldbClient, err := ksqldb.NewClientWithOptions(*config.KsqlDB)
	if err != nil {
		return nil, fmt.Errorf("init ksqldb client: %w", err)
	}

	i, err := ksqldbClient.GetServerInfo()
	if err != nil {
		return nil, fmt.Errorf("get ksqldb server info: %w", err)
	}

	slog.Debug(
		"connected to ksqlDB",
		"cluster", i.KafkaClusterID,
		"service", i.KsqlServiceID,
		"version", i.Version,
		"status", i.ServerStatus,
	)

	const detectedEventsTopic = "om_detected_events"

	// Create KSQL Entities (tables, streams)
	cloudEventsStreamQuery, err := templateQuery(cloudEventsStreamQueryTemplate, cloudEventsStreamQueryData{
		Topic:         config.EventsTopic,
		Partitions:    int(config.Partitions),
		KeySchemaId:   config.KeySchemaID,
		ValueSchemaId: config.ValueSchemaID,
	})
	if err != nil {
		return nil, fmt.Errorf("template event ksql stream: %w", err)
	}
	slog.Debug("ksqlDB create events stream query", "query", cloudEventsStreamQuery)

	detectedEventsTableQuery, err := templateQuery(detectedEventsTableQueryTemplate, detectedEventsTableQueryData{
		Topic:      detectedEventsTopic,
		Retention:  32,
		Partitions: int(config.Partitions),
	})
	if err != nil {
		return nil, fmt.Errorf("template detected events ksql table: %w", err)
	}
	slog.Debug("ksqlDB create detected table query", "query", detectedEventsTableQuery)

	detectedEventsStreamQuery, err := templateQuery(detectedEventsStreamQueryTemplate, detectedEventsStreamQueryData{
		Topic: detectedEventsTopic,
	})
	if err != nil {
		return nil, fmt.Errorf("template detected events ksql stream: %w", err)
	}
	slog.Debug("ksqlDB create detected stream query", "query", detectedEventsStreamQuery)

	resp, err := ksqldbClient.Execute(ksqldb.ExecOptions{
		KSql: cloudEventsStreamQuery,
	})
	if err != nil {
		return nil, fmt.Errorf("init events ksql stream: %w", err)
	}
	slog.Debug("ksqlDB create event stream response", "response", resp)

	resp, err = ksqldbClient.Execute(ksqldb.ExecOptions{
		KSql: detectedEventsTableQuery,
	})
	if err != nil {
		return nil, fmt.Errorf("init detected events ksql table: %w", err)
	}
	slog.Debug("ksqlDB create detected table response", "response", resp)

	resp, err = ksqldbClient.Execute(ksqldb.ExecOptions{
		KSql: detectedEventsStreamQuery,
	})
	if err != nil {
		return nil, fmt.Errorf("init detected event ksql stream: %w", err)
	}
	slog.Debug("ksqlDB create detected stream response", "response", resp)

	connector := &KafkaConnector{
		config:       config,
		KsqlDBClient: &ksqldbClient,
	}

	return connector, nil
}

func (c *KafkaConnector) Init(meter *models.Meter) error {
	queryData := meterTableQueryData{
		Meter:           meter,
		WindowRetention: "36500 DAYS",
		Partitions:      c.config.Partitions,
	}

	err := c.MeterAssert(queryData)
	if err != nil {
		return err
	}

	q, err := GetTableQuery(queryData)
	if err != nil {
		return fmt.Errorf("get table query for meter: %w", err)
	}
	slog.Debug("ksqlDB create table query", "query", q)

	resp, err := c.KsqlDBClient.Execute(ksqldb.ExecOptions{
		KSql: q,
	})
	if err != nil {
		return fmt.Errorf("create ksql table for meter: %w", err)
	}
	slog.Debug("ksqlDB response", "response", resp)

	return nil
}

// MeterAssert ensures meter table immutability by checking that existing meter table is the same as new
func (c *KafkaConnector) MeterAssert(data meterTableQueryData) error {
	q, err := GetTableDescribeQuery(data.Meter)
	if err != nil {
		return fmt.Errorf("get table describe query: %w", err)
	}

	resp, err := c.KsqlDBClient.Execute(ksqldb.ExecOptions{
		KSql: q,
	})
	if err != nil {
		// It's not an issue if the table doesn't exist yet
		// If the table we want to describe does not exist yet ksqldb returns a 40001 error code (bad statement)
		// which is not specific enough to check here.
		if strings.HasPrefix(err.Error(), "Could not find") {
			return nil
		}

		return fmt.Errorf("describe table: %w", err)
	}

	sourceDescription := (*resp)[0]

	if len(sourceDescription.SourceDescription.WriteQueries) > 0 {
		slog.Debug("ksqlDB meter assert", "exists", true)

		query := sourceDescription.SourceDescription.WriteQueries[0].QueryString

		err = MeterQueryAssert(query, data)
		if err != nil {
			return err
		}

		slog.Debug("ksqlDB meter assert", "equals", true)
	} else {
		slog.Debug("ksqlDB meter assert", "exists", false)
	}

	return nil
}

func (c *KafkaConnector) Close() error {
	if c.KsqlDBClient != nil {
		c.KsqlDBClient.Close()
	}
	return nil
}

func (c *KafkaConnector) GetValues(meter *models.Meter, params *GetValuesParams) ([]*models.MeterValue, error) {
	q, err := GetTableValuesQuery(meter, params)
	if err != nil {
		return nil, err
	}

	header, payload, err := c.KsqlDBClient.Pull(context.TODO(), ksqldb.QueryOptions{
		Sql: q,
	})
	if err != nil {
		return nil, err
	}

	slog.Debug("ksqlDB response", "header", header, "payload", payload)
	values, err := NewMeterValues(header, payload)
	if err != nil {
		return nil, fmt.Errorf("get meter values: %w", err)
	}

	return meter.AggregateMeterValues(values, params.WindowSize)
}
