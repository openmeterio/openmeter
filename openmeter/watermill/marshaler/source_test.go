package marshaler

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/internal/event/metadata"
)

type event struct {
	Value string `json:"value"`
}

func (e *event) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{}
}

func (e *event) Validate() error {
	return nil
}

func (e *event) EventName() string {
	return "event"
}

func TestWithSubject(t *testing.T) {
	marshaler := New(nil)

	ev := &event{
		Value: "value",
	}

	evWithSource := WithSource("source", ev)
	msg, err := marshaler.Marshal(evWithSource)

	// Check if the source is set in the metadata
	assert.NoError(t, err)
	assert.Equal(t, "source", msg.Metadata.Get(CloudEventsHeaderSource))

	// Check if the event can be unmarshaled
	unmarshaledEvent := &event{}
	err = marshaler.Unmarshal(msg, unmarshaledEvent)
	assert.NoError(t, err)

	assert.Equal(t, ev, unmarshaledEvent)
}
