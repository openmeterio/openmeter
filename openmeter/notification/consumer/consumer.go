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

package consumer

import (
	"context"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/watermill/grouphandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/openmeter/watermill/router"
)

type Options struct {
	SystemEventsTopic string

	Router router.Options

	Notification notification.Service

	Marshaler marshaler.Marshaler

	Logger *slog.Logger
}

type Consumer struct {
	opts   Options
	router *message.Router

	balanceThresholdHandler *BalanceThresholdEventHandler
}

func New(opts Options) (*Consumer, error) {
	balanceThresholdEventHandler := &BalanceThresholdEventHandler{
		Notification: opts.Notification,
		Logger:       opts.Logger.WithGroup("balance_threshold_event_handler"),
	}

	router, err := router.NewDefaultRouter(opts.Router)
	if err != nil {
		return nil, err
	}

	consumer := &Consumer{
		opts:   opts,
		router: router,

		balanceThresholdHandler: balanceThresholdEventHandler,
	}

	handler, err := grouphandler.NewNoPublishingHandler(opts.Marshaler, opts.Router.MetricMeter,
		grouphandler.NewGroupEventHandler(func(ctx context.Context, event *snapshot.SnapshotEvent) error {
			if event == nil {
				return nil
			}

			return consumer.balanceThresholdHandler.Handle(ctx, *event)
		}),
	)
	if err != nil {
		return nil, err
	}

	_ = router.AddNoPublisherHandler(
		"balance_consumer_system_events",
		opts.SystemEventsTopic,
		opts.Router.Subscriber,
		handler,
	)

	return consumer, nil
}

func (c *Consumer) Handle(ctx context.Context, event snapshot.SnapshotEvent) error {
	return c.balanceThresholdHandler.Handle(ctx, event)
}

func (c *Consumer) Run(ctx context.Context) error {
	return c.router.Run(ctx)
}

func (c *Consumer) Close() error {
	return c.router.Close()
}
