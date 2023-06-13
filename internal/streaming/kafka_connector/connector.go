// Copyright © 2023 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafka_connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/thmeitz/ksqldb-go"
	"github.com/thmeitz/ksqldb-go/net"
	"golang.org/x/exp/slog"

	"github.com/openmeterio/openmeter/internal/models"
	. "github.com/openmeterio/openmeter/internal/streaming"
)

type KafkaConnector struct {
	config         *KafkaConnectorConfig
	KafkaProducer  *kafka.Producer
	KsqlDBClient   *ksqldb.KsqldbClient
	SchemaRegistry *schemaregistry.Client
	Schema         *Schema
}

type KafkaConnectorConfig struct {
	Kafka          *kafka.ConfigMap
	KsqlDB         *net.Options
	SchemaRegistry *schemaregistry.Config
	EventsTopic    string
	Partitions     int
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

	// Initialize schema
	schemaRegistry, err := schemaregistry.NewClient(config.SchemaRegistry)
	if err != nil {
		return nil, fmt.Errorf("init schema registry client: %w", err)
	}

	schemaConfig := SchemaConfig{
		SchemaRegistry:      schemaRegistry,
		EventsTopic:         config.EventsTopic,
		DetectedEventsTopic: "om_detected_events",
	}
	schema, err := NewSchema(SchemaConfig{
		SchemaRegistry:      schemaRegistry,
		EventsTopic:         config.EventsTopic,
		DetectedEventsTopic: "om_detected_events",
	})
	if err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	// Create KSQL Entities (tables, streams)
	cloudEventsStreamQuery, err := templateQuery(cloudEventsStreamQueryTemplate, cloudEventsStreamQueryData{
		Topic:         schemaConfig.EventsTopic,
		Partitions:    int(config.Partitions),
		KeySchemaId:   int(schema.EventKeySerializer.Conf.UseSchemaID),
		ValueSchemaId: int(schema.EventValueSerializer.Conf.UseSchemaID),
	})
	if err != nil {
		return nil, fmt.Errorf("template event ksql stream: %w", err)
	}
	slog.Debug("ksqlDB create events stream query", "query", cloudEventsStreamQuery)

	detectedEventsTableQuery, err := templateQuery(detectedEventsTableQueryTemplate, detectedEventsTableQueryData{
		Topic:      schemaConfig.DetectedEventsTopic,
		Retention:  32,
		Partitions: int(config.Partitions),
	})
	if err != nil {
		return nil, fmt.Errorf("template detected events ksql table: %w", err)
	}
	slog.Debug("ksqlDB create detected table query", "query", detectedEventsTableQuery)

	detectedEventsStreamQuery, err := templateQuery(detectedEventsStreamQueryTemplate, detectedEventsStreamQueryData{
		Topic: schemaConfig.DetectedEventsTopic,
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

	// Initialize Kafka Producer
	producer, err := kafka.NewProducer(config.Kafka)
	if err != nil {
		return nil, err
	}

	slog.Debug("connected to Kafka")

	// TODO: move to main
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				// The message delivery report, indicating success or
				// permanent failure after retries have been exhausted.
				// Application level retries won't help since the client
				// is already configured to do that.
				m := ev
				if m.TopicPartition.Error != nil {
					slog.Error("kafka delivery failed", "error", m.TopicPartition.Error)
				} else {
					slog.Debug("kafka message delivered", "topic", *m.TopicPartition.Topic, "partition", m.TopicPartition.Partition, "offset", m.TopicPartition.Offset)
				}
			case kafka.Error:
				// Generic client instance-level errors, such as
				// broker connection failures, authentication issues, etc.
				//
				// These errors should generally be considered informational
				// as the underlying client will automatically try to
				// recover from any errors encountered, the application
				// does not need to take action on them.
				slog.Error("kafka error", "error", ev)
			}
		}
	}()

	connector := &KafkaConnector{
		config:         config,
		KafkaProducer:  producer,
		KsqlDBClient:   &ksqldbClient,
		SchemaRegistry: &schemaRegistry,
		Schema:         schema,
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
	if c.KafkaProducer != nil {
		c.KafkaProducer.Flush(30 * 1000)
		c.KafkaProducer.Close()
	}
	if c.KsqlDBClient != nil {
		c.KsqlDBClient.Close()
	}
	return nil
}

func (c *KafkaConnector) Publish(event event.Event) error {
	key, err := c.Schema.EventKeySerializer.Serialize(c.config.EventsTopic, event.Subject())
	if err != nil {
		return fmt.Errorf("serialize event key: %w", err)
	}

	ce := ToCloudEventsKafkaPayload(event)
	value, err := c.Schema.EventValueSerializer.Serialize(c.config.EventsTopic, &ce)
	if err != nil {
		return fmt.Errorf("serialize event value: %w", err)
	}

	err = c.KafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &c.config.EventsTopic, Partition: kafka.PartitionAny},
		Timestamp:      event.Time(),
		Headers: []kafka.Header{
			{Key: "specversion", Value: []byte(event.SpecVersion())},
		},
		Key:   key,
		Value: value,
	}, nil)

	return err
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
