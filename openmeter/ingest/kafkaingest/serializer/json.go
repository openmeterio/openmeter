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

func (s JSONSerializer) GetFormat() string {
	return "JSON"
}

func (s JSONSerializer) GetKeySchemaId() int {
	return -1
}

func (s JSONSerializer) GetValueSchemaId() int {
	return -1
}
