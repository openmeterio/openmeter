package common

import (
	"github.com/openmeterio/openmeter/app/config"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

// TODO: create a separate file or package for each application instead

func BalanceWorkerProvisionTopics(conf config.BalanceWorkerConfiguration) []pkgkafka.TopicConfig {
	var provisionTopics []pkgkafka.TopicConfig

	if conf.DLQ.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, pkgkafka.TopicConfig{
			Name:          conf.DLQ.Topic,
			Partitions:    conf.DLQ.AutoProvision.Partitions,
			RetentionTime: pkgkafka.TimeDurationMilliSeconds(conf.DLQ.AutoProvision.Retention),
		})
	}

	return provisionTopics
}

func NotificationServiceProvisionTopics(conf config.NotificationConfiguration) []pkgkafka.TopicConfig {
	var provisionTopics []pkgkafka.TopicConfig

	if conf.Consumer.DLQ.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, pkgkafka.TopicConfig{
			Name:          conf.Consumer.DLQ.Topic,
			Partitions:    conf.Consumer.DLQ.AutoProvision.Partitions,
			RetentionTime: pkgkafka.TimeDurationMilliSeconds(conf.Consumer.DLQ.AutoProvision.Retention),
		})
	}

	return provisionTopics
}

func ServerProvisionTopics(conf config.EventsConfiguration) []pkgkafka.TopicConfig {
	var provisionTopics []pkgkafka.TopicConfig

	if conf.SystemEvents.AutoProvision.Enabled {
		provisionTopics = append(provisionTopics, pkgkafka.TopicConfig{
			Name:       conf.SystemEvents.Topic,
			Partitions: conf.SystemEvents.AutoProvision.Partitions,
		})
	}

	return provisionTopics
}

func SinkWorkerProvisionTopics(conf config.EventsConfiguration) []pkgkafka.TopicConfig {
	return []pkgkafka.TopicConfig{
		{
			Name:       conf.IngestEvents.Topic,
			Partitions: conf.IngestEvents.AutoProvision.Partitions,
		},
	}
}
