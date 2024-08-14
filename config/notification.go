package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type NotificationConfiguration struct {
	Enabled  bool
	Consumer NotificationConsumerConfiguration
}

func (c NotificationConfiguration) Validate() error {
	if err := c.Consumer.Validate(); err != nil {
		return fmt.Errorf("consumer: %w", err)
	}
	return nil
}

type NotificationConsumerConfiguration struct {
	DLQ               DLQConfiguration
	Retry             RetryConfiguration
	ConsumerGroupName string
}

func (c NotificationConsumerConfiguration) Validate() error {
	if err := c.DLQ.Validate(); err != nil {
		return fmt.Errorf("poision queue: %w", err)
	}

	if err := c.Retry.Validate(); err != nil {
		return fmt.Errorf("retry: %w", err)
	}

	if c.ConsumerGroupName == "" {
		return errors.New("consumer group name is required")
	}
	return nil
}

func ConfigureNotification(v *viper.Viper) {
	v.SetDefault("notification.consumer.dlq.enabled", true)
	v.SetDefault("notification.consumer.dlq.topic", "om_sys.notification_service_dlq")
	v.SetDefault("notification.consumer.dlq.autoProvision.enabled", true)
	v.SetDefault("notification.consumer.dlq.autoProvision.partitions", 1)

	v.SetDefault("notification.consumer.dlq.throttle.enabled", true)
	// Let's throttle poison queue to 10 messages per second
	v.SetDefault("notification.consumer.dlq.throttle.count", 10)
	v.SetDefault("notification.consumer.dlq.throttle.duration", time.Second)

	v.SetDefault("notification.consumer.retry.maxRetries", 5)
	v.SetDefault("notification.consumer.retry.initialInterval", 100*time.Millisecond)

	v.SetDefault("notification.consumer.consumerGroupName", "om_notification_service")
}
