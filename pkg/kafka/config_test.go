package kafka

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumerConfig(t *testing.T) {
	tests := []struct {
		Name string

		Params ConsumerConfig

		ExpectedError           error
		ExpectedValidationError error
		ExpectedConfigMap       kafka.ConfigMap
	}{
		{
			Name: "Valid",
			Params: ConsumerConfig{
				CommonConfigParams{
					Brokers:                      "broker-1:9092,broker-2:9092",
					SecurityProtocol:             "SASL_SSL",
					SaslMechanisms:               "PLAIN",
					SaslUsername:                 "user",
					SaslPassword:                 "pass",
					StatsInterval:                TimeDurationMilliSeconds(5 * time.Second),
					BrokerAddressFamily:          BrokerAddressFamilyAny,
					SocketKeepAliveEnabled:       true,
					TopicMetadataRefreshInterval: TimeDurationMilliSeconds(30 * time.Second),
					DebugContexts: DebugContexts{
						DebugContextAdmin,
						DebugContextBroker,
					},
					ClientID: "client-id-1",
				},
				ConsumerConfigParams{
					ConsumerGroupID:             "consumer-group",
					ConsumerGroupInstanceID:     "consumer-group-1",
					SessionTimeout:              TimeDurationMilliSeconds(5 * time.Minute),
					HeartbeatInterval:           TimeDurationMilliSeconds(5 * time.Second),
					EnableAutoCommit:            true,
					EnableAutoOffsetStore:       true,
					AutoOffsetReset:             AutoOffsetResetLatest,
					PartitionAssignmentStrategy: PartitionAssignmentStrategies{PartitionAssignmentStrategyCooperativeSticky},
				},
			},
			ExpectedError:           nil,
			ExpectedValidationError: nil,
			ExpectedConfigMap: kafka.ConfigMap{
				"bootstrap.servers":                  "broker-1:9092,broker-2:9092",
				"broker.address.family":              BrokerAddressFamilyAny,
				"security.protocol":                  "SASL_SSL",
				"sasl.mechanism":                     "PLAIN",
				"sasl.username":                      "user",
				"sasl.password":                      "pass",
				"statistics.interval.ms":             TimeDurationMilliSeconds(5 * time.Second),
				"socket.keepalive.enable":            true,
				"topic.metadata.refresh.interval.ms": TimeDurationMilliSeconds(30 * time.Second),
				"metadata.max.age.ms":                3 * TimeDurationMilliSeconds(30*time.Second),
				"debug":                              "admin,broker",
				"client.id":                          "client-id-1",
				"group.id":                           "consumer-group",
				"group.instance.id":                  "consumer-group-1",
				"session.timeout.ms":                 TimeDurationMilliSeconds(5 * time.Minute),
				"heartbeat.interval.ms":              TimeDurationMilliSeconds(5 * time.Second),
				"enable.auto.commit":                 true,
				"enable.auto.offset.store":           true,
				"auto.offset.reset":                  AutoOffsetResetLatest,
				"partition.assignment.strategy":      PartitionAssignmentStrategies{PartitionAssignmentStrategyCooperativeSticky},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			err := test.Params.Validate()
			require.Equal(t, test.ExpectedValidationError, err)

			m, err := test.Params.AsConfigMap()
			require.Equal(t, test.ExpectedError, err)

			for k, v := range test.ExpectedConfigMap {
				assert.Equal(t, v, m[k], fmt.Sprintf("expected %s got %s", v, m[k]))
			}
		})
	}
}

func TestProducerConfig(t *testing.T) {
	tests := []struct {
		Name string

		Params ProducerConfig

		ExpectedError           error
		ExpectedValidationError error
		ExpectedConfigMap       kafka.ConfigMap
	}{
		{
			Name: "Valid",
			Params: ProducerConfig{
				CommonConfigParams{
					Brokers:                      "broker-1:9092,broker-2:9092",
					SecurityProtocol:             "SASL_SSL",
					SaslMechanisms:               "PLAIN",
					SaslUsername:                 "user",
					SaslPassword:                 "pass",
					StatsInterval:                TimeDurationMilliSeconds(5 * time.Second),
					BrokerAddressFamily:          BrokerAddressFamilyAny,
					SocketKeepAliveEnabled:       true,
					TopicMetadataRefreshInterval: TimeDurationMilliSeconds(30 * time.Second),
					DebugContexts: DebugContexts{
						DebugContextAdmin,
						DebugContextBroker,
					},
					ClientID: "client-id-1",
				},
				ProducerConfigParams{
					Partitioner: PartitionerRandom,
				},
			},
			ExpectedError:           nil,
			ExpectedValidationError: nil,
			ExpectedConfigMap: kafka.ConfigMap{
				"bootstrap.servers":                  "broker-1:9092,broker-2:9092",
				"broker.address.family":              BrokerAddressFamilyAny,
				"security.protocol":                  "SASL_SSL",
				"sasl.mechanism":                     "PLAIN",
				"sasl.username":                      "user",
				"sasl.password":                      "pass",
				"statistics.interval.ms":             TimeDurationMilliSeconds(5 * time.Second),
				"socket.keepalive.enable":            true,
				"topic.metadata.refresh.interval.ms": TimeDurationMilliSeconds(30 * time.Second),
				"metadata.max.age.ms":                3 * TimeDurationMilliSeconds(30*time.Second),
				"debug":                              "admin,broker",
				"partitioner":                        PartitionerRandom,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			err := test.Params.Validate()
			require.Equal(t, test.ExpectedValidationError, err)

			m, err := test.Params.AsConfigMap()
			require.Equal(t, test.ExpectedError, err)

			for k, v := range test.ExpectedConfigMap {
				assert.Equal(t, v, m[k], fmt.Sprintf("expected %s got %s", v, m[k]))
			}
		})
	}
}

func TestBrokerAddressFamily(t *testing.T) {
	tests := []struct {
		Name string

		Value          string
		ExpectedError  error
		ExplectedValue BrokerAddressFamily
	}{
		{
			Name:           "Any",
			Value:          "any",
			ExpectedError:  nil,
			ExplectedValue: BrokerAddressFamilyAny,
		},
		{
			Name:           "IPv4",
			Value:          "v4",
			ExpectedError:  nil,
			ExplectedValue: BrokerAddressFamilyIPv4,
		},
		{
			Name:           "IPv6",
			Value:          "v6",
			ExpectedError:  nil,
			ExplectedValue: BrokerAddressFamilyIPv6,
		},
		{
			Name:          "Invalid",
			Value:         "invalid",
			ExpectedError: errors.New("invalid value broker family address: invalid"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var family BrokerAddressFamily

			err := family.UnmarshalText([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExplectedValue, family)
			}

			err = family.UnmarshalJSON([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExplectedValue, family)
			}
		})
	}
}

func TestTimeDurationMilliSeconds(t *testing.T) {
	tests := []struct {
		Name string

		Value            string
		ExpectedError    error
		ExpectedValue    TimeDurationMilliSeconds
		ExpectedString   string
		ExpectedDuration time.Duration
	}{
		{
			Name:             "Duration",
			Value:            "6s",
			ExpectedError:    nil,
			ExpectedValue:    TimeDurationMilliSeconds(6 * time.Second),
			ExpectedString:   "6000",
			ExpectedDuration: 6 * time.Second,
		},
		{
			Name:          "Invalid",
			Value:         "10000",
			ExpectedError: fmt.Errorf("failed to parse time duration: %w", errors.New("time: missing unit in duration \"10000\"")),
		},
		{
			Name:          "Invalid",
			Value:         "invalid",
			ExpectedError: fmt.Errorf("failed to parse time duration: %w", errors.New("time: invalid duration \"invalid\"")),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var timeMs TimeDurationMilliSeconds

			err := timeMs.UnmarshalText([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExpectedValue, timeMs)
				assert.Equal(t, test.ExpectedString, timeMs.String())
				assert.Equal(t, test.ExpectedDuration, timeMs.Duration())
			}

			err = timeMs.UnmarshalJSON([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExpectedValue, timeMs)
				assert.Equal(t, test.ExpectedString, timeMs.String())
				assert.Equal(t, test.ExpectedDuration, timeMs.Duration())
			}
		})
	}
}

func TestPartitioner(t *testing.T) {
	tests := []struct {
		Name string

		Value          string
		ExpectedError  error
		ExplectedValue Partitioner
	}{
		{
			Name:           "Random",
			Value:          "random",
			ExpectedError:  nil,
			ExplectedValue: PartitionerRandom,
		},
		{
			Name:           "Consistent",
			Value:          "consistent",
			ExpectedError:  nil,
			ExplectedValue: PartitionerConsistent,
		},
		{
			Name:           "ConsistentRandom",
			Value:          "consistent_random",
			ExpectedError:  nil,
			ExplectedValue: PartitionerConsistentRandom,
		},
		{
			Name:           "Murmur2",
			Value:          "murmur2",
			ExpectedError:  nil,
			ExplectedValue: PartitionerMurmur2,
		},
		{
			Name:           "Murmur2Random",
			Value:          "murmur2_random",
			ExpectedError:  nil,
			ExplectedValue: PartitionerMurmur2Random,
		},
		{
			Name:           "Fnv1a",
			Value:          "fnv1a",
			ExpectedError:  nil,
			ExplectedValue: PartitionerFnv1a,
		},
		{
			Name:           "Fnv1aRandom",
			Value:          "fnv1a_random",
			ExpectedError:  nil,
			ExplectedValue: PartitionerFnv1aRandom,
		},
		{
			Name:          "Invalid",
			Value:         "invalid",
			ExpectedError: errors.New("invalid partitioner: invalid"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var partitioner Partitioner

			err := partitioner.UnmarshalText([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExplectedValue, partitioner)
			}

			err = partitioner.UnmarshalJSON([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExplectedValue, partitioner)
			}
		})
	}
}

func TestAutoOffsetReset(t *testing.T) {
	tests := []struct {
		Name string

		Value          string
		ExpectedError  error
		ExplectedValue AutoOffsetReset
	}{
		{
			Name:           "Smallest",
			Value:          "smallest",
			ExpectedError:  nil,
			ExplectedValue: AutoOffsetResetSmallest,
		},
		{
			Name:           "Earliest",
			Value:          "earliest",
			ExpectedError:  nil,
			ExplectedValue: AutoOffsetResetEarliest,
		},
		{
			Name:           "Beginning",
			Value:          "beginning",
			ExpectedError:  nil,
			ExplectedValue: AutoOffsetResetBeginning,
		},
		{
			Name:           "Largest",
			Value:          "largest",
			ExpectedError:  nil,
			ExplectedValue: AutoOffsetResetLargest,
		},
		{
			Name:           "Latest",
			Value:          "latest",
			ExpectedError:  nil,
			ExplectedValue: AutoOffsetResetLatest,
		},
		{
			Name:           "End",
			Value:          "end",
			ExpectedError:  nil,
			ExplectedValue: AutoOffsetResetEnd,
		},
		{
			Name:           "Error",
			Value:          "error",
			ExpectedError:  nil,
			ExplectedValue: AutoOffsetResetError,
		},
		{
			Name:          "Invalid",
			Value:         "invalid",
			ExpectedError: errors.New("invalid auto offset reset strategy: invalid"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var autoOffsetReset AutoOffsetReset

			err := autoOffsetReset.UnmarshalText([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExplectedValue, autoOffsetReset)
			}

			err = autoOffsetReset.UnmarshalJSON([]byte(test.Value))
			assert.Equal(t, test.ExpectedError, err)
			if err == nil {
				assert.Equal(t, test.ExplectedValue, autoOffsetReset)
			}
		})
	}
}
