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
	"encoding/json"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
)

type eventWithSource struct {
	Event `json:",inline"`

	source string `json:"-"`
}

// WithSource can be used to add the CloudEvents source field to an event.
func WithSource(source string, ev Event) Event {
	return &eventWithSource{
		source: source,
		Event:  ev,
	}
}

func (e *eventWithSource) EventMetadata() metadata.EventMetadata {
	metadata := e.Event.EventMetadata()
	metadata.Source = e.source

	return metadata
}

func (e *eventWithSource) Validate() error {
	if err := e.Event.Validate(); err != nil {
		return err
	}

	if e.source == "" {
		return errors.New("source must be set")
	}

	return nil
}

func (e *eventWithSource) EventName() string {
	return e.Event.EventName()
}

// MarshalJSON marshals the event only, as JSON library embeds the Event name into the output,
// if the composed object is a pointer to an interface. (e.g. we would get "Event": {} in the payload)
func (e *eventWithSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Event)
}
