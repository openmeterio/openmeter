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

type Serializer interface {
	SerializeKey(topic string, ev event.Event) ([]byte, error)
	SerializeValue(topic string, ev event.Event) ([]byte, error)
	GetFormat() string
	GetKeySchemaId() int
	GetValueSchemaId() int
}

type CloudEventsKafkaPayload struct {
	Id      string `json:"id"`
	Type    string `json:"type"`
	Source  string `json:"source"`
	Subject string `json:"subject"`
	// Note: By converting to unix timestamp we loose timezone information.
	Time int64  `json:"time"`
	Data string `json:"data"`
}

func toCloudEventsKafkaPayload(ev event.Event) (CloudEventsKafkaPayload, error) {
	payload := CloudEventsKafkaPayload{
		Id:      ev.ID(),
		Type:    ev.Type(),
		Source:  ev.Source(),
		Subject: ev.Subject(),
		Time:    ev.Time().Unix(),
	}

	// We try to parse data as JSON.
	// CloudEvents data can be other than JSON but currently we only support JSON data.
	var data interface{}
	err := json.Unmarshal(ev.Data(), &data)
	if err != nil {
		return payload, err
	}

	payloadData, _ := json.Marshal(data)
	payload.Data = string(payloadData)

	return payload, nil
}
