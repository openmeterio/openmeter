package webhook

import "github.com/openmeterio/openmeter/internal/notification/webhook"

type (
	Config     = webhook.Config
	SvixConfig = webhook.SvixConfig
	Handler    = webhook.Handler
)

func New(config Config) (Handler, error) {
	return webhook.New(config)
}
