package common

import (
	"log/slog"

	"github.com/google/wire"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
)

var Secret = wire.NewSet(
	NewSecretService,
)

func NewSecretService(logger *slog.Logger, db *entdb.Client) (secret.Service, error) {
	secretAdapter := secretadapter.New()

	return secretservice.New(secretservice.Config{
		Adapter: secretAdapter,
	})
}
