package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/app/config"
)

func TestNewEventTopic(t *testing.T) {
	tests := []struct {
		name            string
		ingestConfig    config.KafkaIngestConfiguration
		namespaceConfig config.NamespaceConfiguration
		expected        EventTopic
	}{
		{
			name: "explicit topic",
			ingestConfig: config.KafkaIngestConfiguration{
				EventsTopic:         "events",
				EventsTopicTemplate: "om_%s_events",
			},
			namespaceConfig: config.NamespaceConfiguration{Default: "default"},
			expected:        EventTopic("events"),
		},
		{
			name: "default namespace fallback",
			ingestConfig: config.KafkaIngestConfiguration{
				EventsTopicTemplate: "om_%s_events",
			},
			namespaceConfig: config.NamespaceConfiguration{Default: "default"},
			expected:        EventTopic("om_default_events"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, NewEventTopic(test.ingestConfig, test.namespaceConfig))
		})
	}
}
