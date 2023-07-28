package serializer

import (
	_ "embed"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/jsonschema"
)

//go:embed event_key.json
var eventKeySchema string

//go:embed event_value.json
var eventValueSchema string

type JSONSchemaSerializer struct {
	keySerializer   *jsonschema.Serializer
	valueSerializer *jsonschema.Serializer
}

// NewJSONSchemaSerializer initializes a new schema in the registry.
func NewJSONSchemaSerializer(schemaRegistry schemaregistry.Client) (*JSONSchemaSerializer, error) {
	keySerializer, err := getSerializer(schemaRegistry, serde.KeySerde, eventKeySchema)
	if err != nil {
		return nil, fmt.Errorf("init event key serializer: %w", err)
	}

	valueSerializer, err := getSerializer(schemaRegistry, serde.ValueSerde, eventValueSchema)
	if err != nil {
		return nil, fmt.Errorf("init event value serializer: %w", err)
	}

	return &JSONSchemaSerializer{
		keySerializer:   keySerializer,
		valueSerializer: valueSerializer,
	}, nil
}

func (s JSONSchemaSerializer) SerializeKey(topic string, ev event.Event) ([]byte, error) {
	return s.keySerializer.Serialize(topic, ev.Subject())
}

func (s JSONSchemaSerializer) SerializeValue(topic string, ev event.Event) ([]byte, error) {
	value, err := toCloudEventsKafkaPayload(ev)
	if err != nil {
		return nil, err
	}

	return s.valueSerializer.Serialize(topic, value)
}

func (s JSONSchemaSerializer) GetKeySchemaId() *int {
	return &s.keySerializer.Conf.UseSchemaID
}

func (s JSONSchemaSerializer) GetValueSchemaId() *int {
	return &s.valueSerializer.Conf.UseSchemaID
}

// Registers schema with Registry and returns configured serializer
func getSerializer(registry schemaregistry.Client, serdeType serde.Type, schema string) (*jsonschema.Serializer, error) {
	// Event Key Serializer
	suffix := "key"
	if serdeType == serde.ValueSerde {
		suffix = "value"
	}

	schemaSubject := fmt.Sprintf("om-cloudevents-%s", suffix)
	schemaID, err := registry.Register(schemaSubject, schemaregistry.SchemaInfo{
		Schema:     schema,
		SchemaType: "JSON",
	}, true)
	if err != nil {
		return nil, fmt.Errorf("register schema: %w", err)
	}

	serializerConfig := jsonschema.NewSerializerConfig()
	serializerConfig.AutoRegisterSchemas = false
	serializerConfig.UseSchemaID = schemaID
	serializer, err := jsonschema.NewSerializer(registry, serdeType, serializerConfig)
	if err != nil {
		return nil, fmt.Errorf("init serializer: %w", err)
	}

	return serializer, nil
}
