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
