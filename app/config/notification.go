package config

import (
	"errors"
	"time"

	"github.com/spf13/viper"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/errorsx"
)

type WebhookConfiguration struct {
	// Timeout for registering event types in webhook provider
	EventTypeRegistrationTimeout time.Duration
	// Skip registering event types on unsuccessful attempt instead of returning with error
	SkipEventTypeRegistrationOnError bool
}

type NotificationConfiguration struct {
	Consumer ConsumerConfiguration

	Webhook WebhookConfiguration

	ReconcileInterval time.Duration
	SendingTimeout    time.Duration
	PendingTimeout    time.Duration
}

func (c NotificationConfiguration) Validate() error {
	var errs []error

	if err := c.Consumer.Validate(); err != nil {
		errs = append(errs, errorsx.WithPrefix(err, "consumer"))
	}

	return errors.Join(errs...)
}

func ConfigureNotification(v *viper.Viper) {
	ConfigureConsumer(v, "notification.consumer")
	v.SetDefault("notification.consumer.dlq.topic", "om_sys.notification_service_dlq")
	v.SetDefault("notification.consumer.consumerGroupName", "om_notification_service")
	v.SetDefault("notification.webhook.eventTypeRegistrationTimeout", webhook.DefaultRegistrationTimeout)
	v.SetDefault("notification.webhook.skipEventTypeRegistrationOnError", false)
	v.SetDefault("notification.reconcileInterval", notification.DefaultReconcileInterval)
	v.SetDefault("notification.sendingTimeout", notification.DefaultDeliveryStateSendingTimeout)
	v.SetDefault("notification.pendingTimeout", notification.DefaultDeliveryStatePendingTimeout)
}
