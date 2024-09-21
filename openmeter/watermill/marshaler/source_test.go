// Copyright © 2024 Tailfin Cloud Inc.
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

package marshaler

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
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
