package appcustomeradapter

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/appcustomer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	entcontext "github.com/openmeterio/openmeter/pkg/framework/entutils/context"
)

type Config struct {
	Client *entdb.Client
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}

	return nil
}

func New(config Config) (appcustomer.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	adapter := &adapter{
		db: entcontext.NewClient(config.Client),
	}

	return adapter, nil
}

var _ appcustomer.Adapter = (*adapter)(nil)

type adapter struct {
	db entcontext.DB
}

func (a *adapter) DB() entcontext.DB {
	return a.db
}
