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

package kafka

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
)

type PublisherOptions struct {
	Broker          BrokerOptions
	ProvisionTopics []AutoProvisionTopic
}

func (o *PublisherOptions) Validate() error {
	if err := o.Broker.Validate(); err != nil {
		return fmt.Errorf("invalid broker options: %w", err)
	}

	return nil
}

func NewPublisher(ctx context.Context, in PublisherOptions) (*kafka.Publisher, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	saramaConfig, err := in.Broker.createKafkaConfig("publisher")
	if err != nil {
		return nil, err
	}

	wmConfig := kafka.PublisherConfig{
		Brokers:   []string{in.Broker.KafkaConfig.Broker},
		Marshaler: marshalerWithPartitionKey{},
		Tracer:    kafka.NewOTELSaramaTracer(), // This relies on the global trace provider
	}

	wmConfig.OverwriteSaramaConfig = saramaConfig

	if err := wmConfig.Validate(); err != nil {
		return nil, err
	}

	if err := provisionTopics(ctx, in.Broker.Logger, in.Broker.KafkaConfig.CreateKafkaConfig(), in.ProvisionTopics); err != nil {
		return nil, err
	}

	return kafka.NewPublisher(wmConfig, watermill.NewSlogLogger(in.Broker.Logger))
}
