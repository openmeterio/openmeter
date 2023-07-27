package kafka_connector

import (
	"context"
	"fmt"

	"github.com/thmeitz/ksqldb-go"
)

// NamespaceHandler is a namespace handler for Kafka ingest topics.
type NamespaceHandler struct {
	KsqlDBClient *ksqldb.KsqldbClient

	DefaultEventsTopic string

	// NamespacedTopicTemplate needs to contain at least one string parameter passed to fmt.Sprintf.
	// For example: "om_%s_events"
	NamespacedEventsTopicTemplate string

	DefaultDetectedEventsTopic string

	// NamespacedDetectedEventsTopicTemplate needs to contain at least one string parameter passed to fmt.Sprintf.
	// For example: "om_%s_detected_events"
	NamespacedDetectedEventsTopicTemplate string

	KeySchemaID   int
	ValueSchemaID int
	Partitions    int
}

// CreateNamespace implements the namespace handler interface.
func (h NamespaceHandler) CreateNamespace(ctx context.Context, name string) error {
	eventsTopic := h.DefaultEventsTopic
	detectedEventsTopic := h.DefaultDetectedEventsTopic

	if name != "" {
		eventsTopic = fmt.Sprintf(h.NamespacedEventsTopicTemplate, name)
		detectedEventsTopic = fmt.Sprintf(h.NamespacedDetectedEventsTopicTemplate, name)
	}

	cloudEventsStreamQuery, err := templateQuery(cloudEventsStreamQueryTemplate, cloudEventsStreamQueryData{
		Topic:         eventsTopic,
		KeySchemaId:   h.KeySchemaID,
		ValueSchemaId: h.ValueSchemaID,
	})
	if err != nil {
		return fmt.Errorf("template event ksql stream: %w", err)
	}

	detectedEventsTableQuery, err := templateQuery(detectedEventsTableQueryTemplate, detectedEventsTableQueryData{
		Topic:      detectedEventsTopic,
		Retention:  32,
		Partitions: h.Partitions,
	})
	if err != nil {
		return fmt.Errorf("template detected events ksql table: %w", err)
	}

	detectedEventsStreamQuery, err := templateQuery(detectedEventsStreamQueryTemplate, detectedEventsStreamQueryData{
		Topic: detectedEventsTopic,
	})
	if err != nil {
		return fmt.Errorf("template detected events ksql stream: %w", err)
	}

	_, err = h.KsqlDBClient.Execute(ctx, ksqldb.ExecOptions{
		KSql: cloudEventsStreamQuery,
	})
	if err != nil {
		return fmt.Errorf("init events ksql stream: %w", err)
	}

	_, err = h.KsqlDBClient.Execute(ctx, ksqldb.ExecOptions{
		KSql: detectedEventsTableQuery,
	})
	if err != nil {
		return fmt.Errorf("init detected events ksql table: %w", err)
	}

	_, err = h.KsqlDBClient.Execute(ctx, ksqldb.ExecOptions{
		KSql: detectedEventsStreamQuery,
	})
	if err != nil {
		return fmt.Errorf("init detected event ksql stream: %w", err)
	}

	return nil
}
