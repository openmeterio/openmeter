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

package config

import (
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"

	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

func TestKafkaIngestConfiguration(t *testing.T) {
	tests := []struct {
		Name string

		KafkaConfig            KafkaConfiguration
		ExpectedKafkaConfigMap kafka.ConfigMap
	}{
		{
			Name: "All",
			KafkaConfig: KafkaConfiguration{
				Broker:                       "127.0.0.1:29092",
				SecurityProtocol:             "SASL_SSL",
				SaslMechanisms:               "PLAIN",
				SaslUsername:                 "user",
				SaslPassword:                 "pass",
				StatsInterval:                pkgkafka.TimeDurationMilliSeconds(10 * time.Second),
				BrokerAddressFamily:          "v6",
				SocketKeepAliveEnabled:       true,
				TopicMetadataRefreshInterval: pkgkafka.TimeDurationMilliSeconds(time.Minute),
				DebugContexts: pkgkafka.DebugContexts{
					"broker",
					"topic",
					"consumer",
				},
			},
			ExpectedKafkaConfigMap: kafka.ConfigMap{
				"bootstrap.servers":                  "127.0.0.1:29092",
				"broker.address.family":              pkgkafka.BrokerAddressFamilyIPv6,
				"go.logs.channel.enable":             true,
				"metadata.max.age.ms":                pkgkafka.TimeDurationMilliSeconds(3 * time.Minute),
				"sasl.mechanism":                     "PLAIN",
				"sasl.password":                      "pass",
				"sasl.username":                      "user",
				"security.protocol":                  "SASL_SSL",
				"socket.keepalive.enable":            true,
				"statistics.interval.ms":             pkgkafka.TimeDurationMilliSeconds(10 * time.Second),
				"topic.metadata.refresh.interval.ms": pkgkafka.TimeDurationMilliSeconds(time.Minute),
				"debug":                              "broker,topic,consumer",
			},
		},
		{
			Name: "Basic",
			KafkaConfig: KafkaConfiguration{
				Broker: "127.0.0.1:29092",
			},
			ExpectedKafkaConfigMap: kafka.ConfigMap{
				"bootstrap.servers":      "127.0.0.1:29092",
				"broker.address.family":  pkgkafka.BrokerAddressFamilyIPv4,
				"go.logs.channel.enable": true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			config := test.KafkaConfig.CreateKafkaConfig()
			assert.Equal(t, test.ExpectedKafkaConfigMap, config)
		})
	}
}
