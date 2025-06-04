package serializer

import (
	"encoding/json"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

type JSONSerializer struct{}

func NewJSONSerializer() JSONSerializer {
	return JSONSerializer{}
}

func (s JSONSerializer) SerializeKey(topic string, namespace string, ev event.Event) ([]byte, error) {
	dedupeItem := dedupe.Item{
		Namespace: namespace,
		ID:        ev.ID(),
		Source:    ev.Source(),
	}

	return []byte(dedupeItem.Key()), nil
}

func (s JSONSerializer) SerializeValue(topic string, ev event.Event) ([]byte, error) {
	value, err := toCloudEventsKafkaPayload(ev)
	if err != nil {
		return nil, err
	}

	return json.Marshal(value)
}

func (s JSONSerializer) GetFormat() string {
	return "JSON"
}

func (s JSONSerializer) GetKeySchemaId() int {
	return -1
}

func (s JSONSerializer) GetValueSchemaId() int {
	return -1
}
