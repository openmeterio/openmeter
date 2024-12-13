package common

import (
	"github.com/openmeterio/openmeter/app/config"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

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
