package kafka_connector

import (
	_ "embed"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/jsonschema"
)

//go:embed schema/event_key.json
var eventKeySchema string

//go:embed schema/event_value.json
var eventValueSchema string

type SchemaConfig struct {
	SchemaRegistry      schemaregistry.Client
	EventsTopic         string
	DetectedEventsTopic string
}

type Schema struct {
	EventKeySerializer   *jsonschema.Serializer
	EventValueSerializer *jsonschema.Serializer
}

func NewSchema(config SchemaConfig) (*Schema, error) {
	// Event Key Serializer
	eventKeySerializer, err := getSerializer(config.SchemaRegistry, config.EventsTopic, serde.KeySerde, eventKeySchema)
	if err != nil {
		return nil, fmt.Errorf("init event key: %w", err)
	}

	// Event Value Serializer
	eventValueSerializer, err := getSerializer(config.SchemaRegistry, config.EventsTopic, serde.ValueSerde, eventValueSchema)
	if err != nil {
		return nil, fmt.Errorf("init event value: %w", err)
	}

	// Serializers
	serializer := &Schema{
		EventKeySerializer:   eventKeySerializer,
		EventValueSerializer: eventValueSerializer,
	}

	return serializer, nil
}

// Registers schema with Registry and returns configured serializer
func getSerializer(registry schemaregistry.Client, topic string, serdeType serde.Type, schema string) (*jsonschema.Serializer, error) {
	// Event Key Serializer
	suffix := "key"
	if serdeType == serde.ValueSerde {
		suffix = "value"
	}

	schemaSubject := fmt.Sprintf("%s-%s", topic, suffix)
	schemaId, err := registry.Register(schemaSubject, schemaregistry.SchemaInfo{
		Schema:     schema,
		SchemaType: "JSON",
	}, true)
	if err != nil {
		return nil, fmt.Errorf("register schema: %w", err)
	}

	serializerConfig := jsonschema.NewSerializerConfig()
	serializerConfig.AutoRegisterSchemas = false
	serializerConfig.UseSchemaID = schemaId
	serializer, err := jsonschema.NewSerializer(registry, serdeType, serializerConfig)
	if err != nil {
		return nil, fmt.Errorf("init serializer: %w", err)
	}

	return serializer, nil
}
