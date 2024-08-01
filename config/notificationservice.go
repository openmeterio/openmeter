package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type NotificationServiceConfiguration struct {
	Consumer NotificationServiceConsumerConfiguration
}

type NotificationServiceConsumerConfiguration struct {
	PoisionQueue      PoisionQueueConfiguration
	Retry             RetryConfiguration
	ConsumerGroupName string
}

func (c NotificationServiceConfiguration) Validate() error {
	if err := c.Consumer.Validate(); err != nil {
		return fmt.Errorf("consumer: %w", err)
	}
	return nil
}

func (c NotificationServiceConsumerConfiguration) Validate() error {
	if err := c.PoisionQueue.Validate(); err != nil {
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

func ConfigureNotificationService(v *viper.Viper) {
	v.SetDefault("notificationService.consumer.poisionQueue.enabled", true)
	v.SetDefault("notificationService.consumer.poisionQueue.topic", "om_sys.notification_service_poision")
	v.SetDefault("notificationService.consumer.poisionQueue.autoProvision.enabled", true)
	v.SetDefault("notificationService.consumer.poisionQueue.autoProvision.partitions", 1)

	v.SetDefault("notificationService.consumer.poisionQueue.throttle.enabled", true)
	// Let's throttle poision queue to 10 messages per second
	v.SetDefault("notificationService.consumer.poisionQueue.throttle.count", 10)
	v.SetDefault("notificationService.consumer.poisionQueue.throttle.duration", time.Second)

	v.SetDefault("notificationService.consumer.retry.maxRetries", 5)
	v.SetDefault("notificationService.consumer.retry.initialInterval", 100*time.Millisecond)

	v.SetDefault("notificationService.consumer.consumerGroupName", "om_notification_service")
}
