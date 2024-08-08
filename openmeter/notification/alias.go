package notification

import "github.com/openmeterio/openmeter/internal/notification"

type (
	Config  = notification.Config
	Service = notification.Service
)

func New(config Config) (Service, error) {
	return notification.New(config)
}
