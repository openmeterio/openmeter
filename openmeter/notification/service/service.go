package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/notification"
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

	eventHandler, err := notification.NewEventHandler(notification.EventHandlerConfig{
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
