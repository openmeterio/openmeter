package serializer

import (
	_ "embed"
	"encoding/json"

	"github.com/cloudevents/sdk-go/v2/event"
)

type JSONSerializer struct{}

func NewJSONSerializer() JSONSerializer {
	return JSONSerializer{}
}

func (s JSONSerializer) SerializeKey(topic string, ev event.Event) ([]byte, error) {
	return []byte(ev.Subject()), nil
}

func (s JSONSerializer) SerializeValue(topic string, ev event.Event) ([]byte, error) {
	value, err := toCloudEventsKafkaPayload(ev)
	if err != nil {
		return nil, err
	}

	return json.Marshal(value)
}

func (s JSONSerializer) GetKeySchemaId() *int {
	return nil
}

func (s JSONSerializer) GetValueSchemaId() *int {
	return nil
}
