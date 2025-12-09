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
	wire.FieldsOf(new(config.BillingConfiguration), "AdvancementStrategy"),
	wire.FieldsOf(new(config.BillingConfiguration), "FeatureSwitches"),
	// ClickHouse
	wire.FieldsOf(new(config.AggregationConfiguration), "ClickHouse"),
	// Customer
	wire.FieldsOf(new(config.Configuration), "Customer"),
	// Database
	wire.FieldsOf(new(config.Configuration), "Postgres"),
	// Entitlement
	wire.FieldsOf(new(config.Configuration), "Entitlements"),
	// Events
	wire.FieldsOf(new(config.Configuration), "Events"),
	wire.FieldsOf(new(config.Configuration), "Dedupe"),
	// Kafka
	// TODO: refactor to move out Kafka config from ingest and consolidate
	wire.FieldsOf(new(config.KafkaIngestConfiguration), "KafkaConfiguration"),
	wire.FieldsOf(new(config.Configuration), "Ingest"),
	wire.FieldsOf(new(config.IngestConfiguration), "Kafka"),
	wire.FieldsOf(new(config.KafkaIngestConfiguration), "TopicProvisioner"),
	// Namespace
	wire.FieldsOf(new(config.Configuration), "Namespace"),
	// Notification
	wire.FieldsOf(new(config.Configuration), "Notification"),
	wire.FieldsOf(new(config.NotificationConfiguration), "Webhook"),
	// Subscription
	wire.FieldsOf(new(config.ProductCatalogConfiguration), "Subscription"),
	// Portal
	wire.FieldsOf(new(config.Configuration), "Portal"),
	// ProductCatalog
	wire.FieldsOf(new(config.Configuration), "ProductCatalog"),
	// ProgressManager
	wire.FieldsOf(new(config.Configuration), "ProgressManager"),
	// Reserved Event Types
	wire.FieldsOf(new(config.Configuration), "ReservedEventTypes"),
	// Svix
	wire.FieldsOf(new(config.Configuration), "Svix"),
	// Telemetry
	wire.FieldsOf(new(config.Configuration), "Telemetry"),
	wire.FieldsOf(new(config.TelemetryConfig), "Metrics"),
	wire.FieldsOf(new(config.TelemetryConfig), "Trace"),
	wire.FieldsOf(new(config.TelemetryConfig), "Log"),
	// Termination
	wire.FieldsOf(new(config.Configuration), "Termination"),
)
