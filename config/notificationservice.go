// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	DLQ               DLQConfiguration
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

func ConfigureNotificationService(v *viper.Viper) {
	v.SetDefault("notificationService.consumer.dlq.enabled", true)
	v.SetDefault("notificationService.consumer.dlq.topic", "om_sys.notification_service_dlq")
	v.SetDefault("notificationService.consumer.dlq.autoProvision.enabled", true)
	v.SetDefault("notificationService.consumer.dlq.autoProvision.partitions", 1)

	v.SetDefault("notificationService.consumer.dlq.throttle.enabled", true)
	// Let's throttle poision queue to 10 messages per second
	v.SetDefault("notificationService.consumer.dlq.throttle.count", 10)
	v.SetDefault("notificationService.consumer.dlq.throttle.duration", time.Second)

	v.SetDefault("notificationService.consumer.retry.maxRetries", 5)
	v.SetDefault("notificationService.consumer.retry.initialInterval", 100*time.Millisecond)

	v.SetDefault("notificationService.consumer.consumerGroupName", "om_notification_service")
}
