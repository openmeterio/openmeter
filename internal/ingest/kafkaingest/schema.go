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

package kafkaingest

import (
	_ "embed"
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
	Id      string `json:"ID"`
	Type    string `json:"TYPE"`
	Source  string `json:"SOURCE"`
	Subject string `json:"SUBJECT"`
	Time    string `json:"TIME"`
	Data    string `json:"DATA"`
}

func toCloudEventsKafkaPayload(ev event.Event) cloudEventsKafkaPayload {
	return cloudEventsKafkaPayload{
		Id:      ev.ID(),
		Type:    ev.Type(),
		Source:  ev.Source(),
		Subject: ev.Subject(),
		Time:    ev.Time().String(),
		Data:    string(ev.Data()),
	}
}

func (s schema) SerializeValue(topic string, ev event.Event) ([]byte, error) {
	return s.valueSerializer.Serialize(topic, toCloudEventsKafkaPayload(ev))
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
