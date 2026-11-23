package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
)

func TestRTEProvisionTopicsHonorsDLQSettings(t *testing.T) {
	var conf config.RTEConfiguration
	conf.DLQ.Topic = "rte-dlq"
	conf.DLQ.AutoProvision.Enabled = true

	var events config.EventsConfiguration
	events.SystemEvents.Topic = "system"
	events.SystemEvents.AutoProvision.Enabled = true

	topics := RTEProvisionTopics(conf, events)
	require.Len(t, topics, 1)
	require.Equal(t, "system", topics[0].Name)

	conf.DLQ.Enabled = true
	topics = RTEProvisionTopics(conf, events)
	require.Len(t, topics, 2)
	require.Equal(t, "rte-dlq", topics[0].Name)

	conf.DLQ.AutoProvision.Enabled = false
	topics = RTEProvisionTopics(conf, events)
	require.Len(t, topics, 1)
	require.Equal(t, "system", topics[0].Name)
}
