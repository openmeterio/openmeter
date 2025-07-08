package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/notification"
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

type Config struct {
	FeatureConnector feature.FeatureConnector

	Adapter      notification.Repository
	Webhook      webhook.Handler
	EventHandler notification.EventHandler

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

	if config.EventHandler == nil {
		return nil, errors.New("missing event handler")
	}

	return &Service{
		adapter:      config.Adapter,
		feature:      config.FeatureConnector,
		webhook:      config.Webhook,
		eventHandler: config.EventHandler,
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
