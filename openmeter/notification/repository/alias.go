package repository

import (
	"github.com/openmeterio/openmeter/internal/notification"
	notificationrepository "github.com/openmeterio/openmeter/internal/notification/repository"
)

type (
	Config                  = notificationrepository.Config
	Repository              = notification.Repository
	PostgresAdapterConfig   = notificationrepository.PostgresAdapterConfig
	ClickhouseAdapterConfig = notificationrepository.ClickhouseAdapterConfig
)

func New(config Config) (Repository, error) {
	return notificationrepository.New(config)
}
