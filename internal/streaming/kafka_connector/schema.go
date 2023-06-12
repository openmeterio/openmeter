package kafka_connector

import (
	_ "embed"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/jsonschema"
	"golang.org/x/exp/slog"
)

//go:embed schema/event_key.json
var eventKeySchema string

//go:embed schema/event_value.json
var eventValueSchema string

type SchemaConfig struct {
	SchemaRegistry schemaregistry.Client
	EventsTopic    string
}

type Schema struct {
	EventKeySerializer   *jsonschema.Serializer
	EventValueSerializer *jsonschema.Serializer
}

func NewSchema(config SchemaConfig) (*Schema, error) {
	// Event Key Serializer
	eventsSchemaKeySubject := fmt.Sprintf("%s-key", config.EventsTopic)
	eventsSchemaKeyId, err := config.SchemaRegistry.Register(eventsSchemaKeySubject, schemaregistry.SchemaInfo{
		Schema:     eventKeySchema,
		SchemaType: "JSON",
	}, true)
	if err != nil {
		slog.Error("Schema Registry failed to register event value schema", "topic", config.EventsTopic, "error", err)
		return nil, err
	}

	eventKeySerializerConfig := jsonschema.NewSerializerConfig()
	eventKeySerializerConfig.AutoRegisterSchemas = false
	eventKeySerializerConfig.UseSchemaID = eventsSchemaKeyId
	eventKeySerializer, err := jsonschema.NewSerializer(config.SchemaRegistry, serde.KeySerde, eventKeySerializerConfig)
	if err != nil {
		slog.Error("Schema Registry failed to create event key serializer", "error", err)
		return nil, err
	}

	// Event Value Serializer
	eventsSchemaValueSubject := fmt.Sprintf("%s-value", config.EventsTopic)
	eventsSchemaValueId, err := config.SchemaRegistry.Register(eventsSchemaValueSubject, schemaregistry.SchemaInfo{
		Schema:     eventValueSchema,
		SchemaType: "JSON",
	}, true)
	if err != nil {
		slog.Error("Schema Registry failed to register event key schema", "topic", config.EventsTopic, "error", err)
		return nil, err
	}

	eventValueSerializerConfig := jsonschema.NewSerializerConfig()
	eventValueSerializerConfig.AutoRegisterSchemas = false
	eventValueSerializerConfig.UseSchemaID = eventsSchemaValueId
	eventValueSerializer, err := jsonschema.NewSerializer(config.SchemaRegistry, serde.ValueSerde, eventValueSerializerConfig)
	if err != nil {
		slog.Error("Schema Registry failed to create event key serializer", "error", err)
		return nil, err
	}

	// Serializers
	serializer := &Schema{
		EventKeySerializer:   eventKeySerializer,
		EventValueSerializer: eventValueSerializer,
	}

	return serializer, nil
}
