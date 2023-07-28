package ksqldb_connector

import (
	"context"
	"fmt"

	"github.com/thmeitz/ksqldb-go"
	"golang.org/x/exp/slog"

	ns "github.com/openmeterio/openmeter/internal/namespace"
)

// NamespaceHandler is a namespace handler for Kafka ingest topics.
type NamespaceHandler struct {
	KsqlDBClient *ksqldb.KsqldbClient

	// NamespacedTopicTemplate needs to contain at least one string parameter passed to fmt.Sprintf.
	// For example: "om_%s_events"
	NamespacedEventsTopicTemplate string

	// NamespacedDetectedEventsTopicTemplate needs to contain at least one string parameter passed to fmt.Sprintf.
	// For example: "om_%s_detected_events"
	NamespacedDetectedEventsTopicTemplate string

	Format        string
	KeySchemaID   int
	ValueSchemaID int
	Partitions    int
}

// CreateNamespace implements the namespace handler interface.
func (h NamespaceHandler) CreateNamespace(ctx context.Context, namespace string) error {
	eventsTopic := fmt.Sprintf(h.NamespacedEventsTopicTemplate, ns.DefaultNamespace)
	detectedEventsTopic := fmt.Sprintf(h.NamespacedDetectedEventsTopicTemplate, ns.DefaultNamespace)

	if namespace != "" {
		eventsTopic = fmt.Sprintf(h.NamespacedEventsTopicTemplate, namespace)
		detectedEventsTopic = fmt.Sprintf(h.NamespacedDetectedEventsTopicTemplate, namespace)
	}

	cloudEventsStreamQuery, err := templateQuery(cloudEventsStreamQueryTemplate, cloudEventsStreamQueryData{
		Format:        h.Format,
		Namespace:     namespace,
		Topic:         eventsTopic,
		KeySchemaId:   h.KeySchemaID,
		ValueSchemaId: h.ValueSchemaID,
	})
	if err != nil {
		return fmt.Errorf("template event ksql stream: %w", err)
	}

	detectedEventsTableQuery, err := templateQuery(detectedEventsTableQueryTemplate, detectedEventsTableQueryData{
		Format:     h.Format,
		Namespace:  namespace,
		Topic:      detectedEventsTopic,
		Retention:  32,
		Partitions: h.Partitions,
	})
	if err != nil {
		return fmt.Errorf("template detected events ksql table: %w", err)
	}

	detectedEventsStreamQuery, err := templateQuery(detectedEventsStreamQueryTemplate, detectedEventsStreamQueryData{
		Format:    h.Format,
		Namespace: namespace,
		Topic:     detectedEventsTopic,
	})
	if err != nil {
		return fmt.Errorf("template detected events ksql stream: %w", err)
	}
	slog.Debug("cloudEventsStreamQuery", "query", cloudEventsStreamQuery)

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
