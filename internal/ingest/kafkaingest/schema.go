package kafkaingest

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/jsonschema"
)

//go:embed schema/event_key.json
var eventKeySchema string

//go:embed schema/event_value.json
var eventValueSchema string

type schema struct {
	keySerializer   *jsonschema.Serializer
	valueSerializer *jsonschema.Serializer
}

// NewSchema initializes a new schema in the registry.
func NewSchema(schemaRegistry schemaregistry.Client, topic string) (Schema, int, int, error) {
	keySerializer, err := getSerializer(schemaRegistry, topic, serde.KeySerde, eventKeySchema)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("init event key serializer: %w", err)
	}

	valueSerializer, err := getSerializer(schemaRegistry, topic, serde.ValueSerde, eventValueSchema)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("init event value serializer: %w", err)
	}

	// TODO: improve schema ID propagation
	return schema{
		keySerializer:   keySerializer,
		valueSerializer: valueSerializer,
	}, keySerializer.Conf.UseSchemaID, valueSerializer.Conf.UseSchemaID, nil
}

func (s schema) SerializeKey(topic string, ev event.Event) ([]byte, error) {
	return s.keySerializer.Serialize(topic, ev.Subject())
}

type cloudEventsKafkaPayload struct {
	Id      string      `json:"ID"`
	Type    string      `json:"TYPE"`
	Source  string      `json:"SOURCE"`
	Subject string      `json:"SUBJECT"`
	Time    string      `json:"TIME"`
	Data    interface{} `json:"DATA"`
}

func toCloudEventsKafkaPayload(ev event.Event) (*cloudEventsKafkaPayload, error) {
	payload := &cloudEventsKafkaPayload{
		Id:      ev.ID(),
		Type:    ev.Type(),
		Source:  ev.Source(),
		Subject: ev.Subject(),
		Time:    ev.Time().String(),
	}

	err := json.Unmarshal(ev.Data(), &payload.Data)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (s schema) SerializeValue(topic string, ev event.Event) ([]byte, error) {
	value, err := toCloudEventsKafkaPayload(ev)
	if err != nil {
		return nil, err
	}

	return s.valueSerializer.Serialize(topic, value)
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
