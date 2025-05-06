package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/eventhandler"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

const (
	ChannelIDMetadataKey = "om-channel-id"
)

var _ notification.Service = (*Service)(nil)

type Service struct {
	feature feature.FeatureConnector

	adapter notification.Repository
	webhook webhook.Handler

	eventHandler notification.EventHandler

	logger *slog.Logger
}

func (s Service) Close() error {
	return s.eventHandler.Close()
}

type Config struct {
	FeatureConnector feature.FeatureConnector

	Adapter notification.Repository
	Webhook webhook.Handler

	Logger *slog.Logger
}

func New(config Config) (*Service, error) {
	if config.Adapter == nil {
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

	eventHandler, err := eventhandler.New(eventhandler.Config{
		Repository: config.Adapter,
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
		adapter:      config.Adapter,
		feature:      config.FeatureConnector,
		webhook:      config.Webhook,
		eventHandler: eventHandler,
		logger:       config.Logger,
	}, nil
}

func (s Service) ListFeature(ctx context.Context, namespace string, features ...string) ([]feature.Feature, error) {
	resp, err := s.feature.ListFeatures(ctx, feature.ListFeaturesParams{
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
