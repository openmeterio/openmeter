package appservice

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/app"
)

var _ app.WebhookURLGenerator = (*baseURLWebhookURLGenerator)(nil)

type baseURLWebhookURLGenerator struct {
	baseURL string
}

func NewBaseURLWebhookURLGenerator(baseURL string) (app.WebhookURLGenerator, error) {
	if baseURL == "" {
		return nil, errors.New("base url is required")
	}

	return &baseURLWebhookURLGenerator{
		baseURL: baseURL,
	}, nil
}

func (g *baseURLWebhookURLGenerator) GetWebhookURL(ctx context.Context, appID app.AppID) (string, error) {
	if err := appID.Validate(); err != nil {
		return "", fmt.Errorf("error validating app id: %w", err)
	}

	return url.JoinPath(g.baseURL, "/api/v1/apps/", appID.ID, "/stripe/webhook")
}

var _ app.WebhookURLGenerator = (*patternWebhookURLGenerator)(nil)

type patternWebhookURLGenerator struct {
	pattern string
}

func NewPatternWebhookURLGenerator(pattern string) (app.WebhookURLGenerator, error) {
	if pattern == "" {
		return nil, errors.New("pattern is required")
	}

	if !strings.Contains(pattern, "%s") {
		return nil, errors.New("pattern must contain %s")
	}

	return &patternWebhookURLGenerator{
		pattern: pattern,
	}, nil
}

func (g *patternWebhookURLGenerator) GetWebhookURL(ctx context.Context, appID app.AppID) (string, error) {
	if appID.ID == "" {
		return "", errors.New("app id is required")
	}

	return fmt.Sprintf(g.pattern, appID.ID), nil
}
