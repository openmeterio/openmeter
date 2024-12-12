package common

import (
	"log/slog"

	"github.com/openmeterio/openmeter/app/config"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
)

func NewSecretService(logger *slog.Logger, db *entdb.Client, appsConfig config.AppsConfiguration) (secret.Service, error) {
	secretAdapter := secretadapter.New()

	return secretservice.New(secretservice.Config{
		Adapter: secretAdapter,
	})
}
