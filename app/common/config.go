// Copyright 2022 The OpenMeter Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	// Credit
	wire.FieldsOf(new(config.Configuration), "Credits"),
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
