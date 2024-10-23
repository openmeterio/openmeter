package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/openmeter/testutils"
)

var _ AdminClient = (*mockTopicProvisioner)(nil)

type mockTopicProvisioner struct {
	added   []string
	removed []string
}

func (m *mockTopicProvisioner) CreateTopics(_ context.Context, topics []kafka.TopicSpecification, _ ...kafka.CreateTopicsAdminOption) ([]kafka.TopicResult, error) {
	result := make([]kafka.TopicResult, 0, len(topics))

	for _, topic := range topics {
		m.added = append(m.added, topic.Topic)

		result = append(result, kafka.TopicResult{
			Topic: topic.Topic,
			Error: kafka.NewError(kafka.ErrNoError, "", false),
		})
	}

	return result, nil
}

func (m *mockTopicProvisioner) DeleteTopics(_ context.Context, topics []string, _ ...kafka.DeleteTopicsAdminOption) ([]kafka.TopicResult, error) {
	result := make([]kafka.TopicResult, 0, len(topics))

	for _, topic := range topics {
		m.removed = append(m.removed, topic)

		result = append(result, kafka.TopicResult{
			Topic: topic,
			Error: kafka.NewError(kafka.ErrNoError, "", false),
		})
	}

	return result, nil
}

func (m *mockTopicProvisioner) reset() {
	m.added, m.removed = []string{}, []string{}
}

func TestTopicProvisioner(t *testing.T) {
	tests := []struct {
		Name string

		AddTopics    []TopicConfig
		RemoveTopics []string

		ExpectedError         error
		ExpectedAddedTopics   []string
		ExpectedRemovedTopics []string
	}{
		{
			Name: "Add topics",
			AddTopics: []TopicConfig{
				{
					Name:       "topic-1",
					Partitions: 1,
				},

				{
					Name:       "topic-2",
					Partitions: 1,
				},
			},
			ExpectedError:         nil,
			ExpectedAddedTopics:   []string{"topic-1", "topic-2"},
			ExpectedRemovedTopics: []string{},
		},
		{
			Name:                  "Remove topics",
			RemoveTopics:          []string{"topic-1", "topic-2"},
			ExpectedError:         nil,
			ExpectedAddedTopics:   []string{},
			ExpectedRemovedTopics: []string{"topic-1", "topic-2"},
		},
		{
			Name:                  "Remove protected topics",
			RemoveTopics:          []string{"protected-topic-1", "protected-topic-2"},
			ExpectedError:         nil,
			ExpectedAddedTopics:   []string{},
			ExpectedRemovedTopics: []string{},
		},
	}

	adminClient := &mockTopicProvisioner{}
	meter := noop.NewMeterProvider().Meter("test")
	logger := testutils.NewDiscardLogger(t)

	provisioner, err := NewTopicProvisioner(TopicProvisionerConfig{
		AdminClient: adminClient,
		Logger:      logger,
		Meter:       meter,
		CacheSize:   200,
		CacheTTL:    5 * time.Second,
		ProtectedTopics: []string{
			"protected-topic-1",
			"protected-topic-2",
		},
	})
	require.NoError(t, err, "initializing new topic provisioner should not fail")

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			adminClient.reset()

			ctx := context.TODO()

			err = provisioner.Provision(ctx, test.AddTopics...)
			assert.NoError(t, err, "provisioning topics must not fail")

			assert.ElementsMatch(t, test.ExpectedAddedTopics, adminClient.added, "provisioned topics must match")

			err = provisioner.DeProvision(ctx, test.RemoveTopics...)
			assert.NoError(t, err, "de-provisioning topics must not fail")

			assert.ElementsMatch(t, test.ExpectedRemovedTopics, adminClient.removed, "de-provisioned topics must match")
		})
	}
}
