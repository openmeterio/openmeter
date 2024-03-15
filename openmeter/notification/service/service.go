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

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/eventhandler"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

const (
	ChannelIDMetadataKey = "om-channel-id"
)

var _ notification.Service = (*Service)(nil)

type Service struct {
	feature productcatalog.FeatureConnector

	repo    notification.Repository
	webhook webhook.Handler

	eventHandler notification.EventHandler

	logger *slog.Logger
}

func (s Service) Close() error {
	return s.eventHandler.Close()
}

type Config struct {
	FeatureConnector productcatalog.FeatureConnector

	Repository notification.Repository
	Webhook    webhook.Handler

	Logger *slog.Logger
}

func New(config Config) (*Service, error) {
	if config.Repository == nil {
		return nil, errors.New("missing repository")
	}

	if config.FeatureConnector == nil {
		return nil, errors.New("missing feature connector")
	}

	if config.Webhook == nil {
		return nil, errors.New("missing webhook handler")
	}

	if config.Logger == nil {
		return nil, errors.New("missing logger")
	}
	config.Logger = config.Logger.WithGroup("notification")

	eventHandler, err := eventhandler.New(eventhandler.Config{
		Repository: config.Repository,
		Webhook:    config.Webhook,
		Logger:     config.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize notification event handler: %w", err)
	}

	if err = eventHandler.Start(); err != nil {
		return nil, fmt.Errorf("failed to initialize notification event handler: %w", err)
	}

	return &Service{
		repo:         config.Repository,
		feature:      config.FeatureConnector,
		webhook:      config.Webhook,
		eventHandler: eventHandler,
		logger:       config.Logger,
	}, nil
}

func (s Service) ListFeature(ctx context.Context, namespace string, features ...string) ([]productcatalog.Feature, error) {
	resp, err := s.feature.ListFeatures(ctx, productcatalog.ListFeaturesParams{
		IDsOrKeys:       features,
		Namespace:       namespace,
		MeterSlugs:      nil,
		IncludeArchived: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get features: %w", err)
	}

	return resp.Items, nil
}
