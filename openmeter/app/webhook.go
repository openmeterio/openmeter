package app

import (
	"context"
)

type WebhookURLGenerator interface {
	GetWebhookURL(ctx context.Context, appID AppID) (string, error)
}
