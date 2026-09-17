package kafkaingest

import (
	"context"
	"errors"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

type recordingSerializer struct {
	topics []string
	err    error
}

func (s *recordingSerializer) SerializeKey(topic string, _ string, _ cloudevents.Event) ([]byte, error) {
	s.topics = append(s.topics, topic)

	return nil, s.err
}

func (s *recordingSerializer) SerializeValue(_ string, _ cloudevents.Event) ([]byte, error) {
	return nil, nil
}

func (s *recordingSerializer) GetFormat() string {
	return ""
}

func (s *recordingSerializer) GetKeySchemaId() int {
	return 0
}

func (s *recordingSerializer) GetValueSchemaId() int {
	return 0
}

type recordingTopicProvisioner struct {
	topics []pkgkafka.TopicConfig
}

func (p *recordingTopicProvisioner) Provision(_ context.Context, topics ...pkgkafka.TopicConfig) error {
	p.topics = append(p.topics, topics...)

	return nil
}

func (p *recordingTopicProvisioner) DeProvision(_ context.Context, _ ...string) error {
	return nil
}

func TestCollectorUsesFixedTopic(t *testing.T) {
	serializerError := errors.New("stop before producing")
	serializer := &recordingSerializer{err: serializerError}
	provisioner := &recordingTopicProvisioner{}

	collector, err := NewCollector(
		&kafka.Producer{},
		serializer,
		"om_default_events",
		provisioner,
		1,
		testutils.NewDiscardLogger(t),
		noop.NewTracerProvider().Tracer("test"),
	)
	require.NoError(t, err)

	ev := cloudevents.New()
	ev.SetID("event-id")

	for _, namespace := range []string{"default", "customer"} {
		err = collector.Ingest(t.Context(), namespace, ev)
		require.ErrorIs(t, err, serializerError)
	}

	assert.Equal(t, []string{"om_default_events", "om_default_events"}, serializer.topics)
	assert.Equal(t, []pkgkafka.TopicConfig{
		{Name: "om_default_events", Partitions: 1},
		{Name: "om_default_events", Partitions: 1},
	}, provisioner.topics)
}

func TestNewCollectorRequiresTopic(t *testing.T) {
	collector, err := NewCollector(
		&kafka.Producer{},
		&recordingSerializer{},
		"",
		&recordingTopicProvisioner{},
		1,
		testutils.NewDiscardLogger(t),
		noop.NewTracerProvider().Tracer("test"),
	)

	require.EqualError(t, err, "topic is required")
	assert.Nil(t, collector)
}
