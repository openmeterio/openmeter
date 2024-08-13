package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type NotificationConfiguration struct {
	Enabled  bool
	Consumer ConsumerConfiguration
}

func (c NotificationConfiguration) Validate() error {
	if err := c.Consumer.Validate(); err != nil {
		return fmt.Errorf("consumer: %w", err)
	}
	return nil
}

func ConfigureNotification(v *viper.Viper) {
	ConfigureConsumer(v, "notification.consumer")
	v.SetDefault("notification.consumer.dlq.topic", "om_sys.notification_service_dlq")
	v.SetDefault("notification.consumer.consumerGroupName", "om_notification_service")
}
