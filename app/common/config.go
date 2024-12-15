package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
)

// We have configs separatly to be able to reuse wires in other projects
var Config = wire.NewSet(
	// App
	wire.FieldsOf(new(config.Configuration), "Apps"),
	// Aggregation
	wire.FieldsOf(new(config.Configuration), "Aggregation"),
	// Billing
	wire.FieldsOf(new(config.Configuration), "Billing"),
	// ClickHouse
	wire.FieldsOf(new(config.AggregationConfiguration), "ClickHouse"),
	// Database
	wire.FieldsOf(new(config.Configuration), "Postgres"),
	// Entitlement
	wire.FieldsOf(new(config.Configuration), "Entitlements"),
	// Events
	wire.FieldsOf(new(config.Configuration), "Events"),
	// Kafka
	// TODO: refactor to move out Kafka config from ingest and consolidate
	wire.FieldsOf(new(config.KafkaIngestConfiguration), "KafkaConfiguration"),
	wire.FieldsOf(new(config.Configuration), "Ingest"),
	wire.FieldsOf(new(config.IngestConfiguration), "Kafka"),
	wire.FieldsOf(new(config.KafkaIngestConfiguration), "TopicProvisionerConfig"),
	// Namespace
	wire.FieldsOf(new(config.Configuration), "Namespace"),
	// Notification
	wire.FieldsOf(new(config.Configuration), "Notification"),
	// ProductCatalog
	wire.FieldsOf(new(config.Configuration), "ProductCatalog"),
	// Svix
	wire.FieldsOf(new(config.Configuration), "Svix"),
	// Telemetry
	wire.FieldsOf(new(config.Configuration), "Telemetry"),
	wire.FieldsOf(new(config.TelemetryConfig), "Metrics"),
	wire.FieldsOf(new(config.TelemetryConfig), "Trace"),
	wire.FieldsOf(new(config.TelemetryConfig), "Log"),
)
