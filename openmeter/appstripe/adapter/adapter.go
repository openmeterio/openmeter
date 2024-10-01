package appstripeadapter

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/appcustomer"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	entcontext "github.com/openmeterio/openmeter/pkg/framework/entutils/context"
)

type Config struct {
	Client             *entdb.Client
	AppService         app.Service
	AppCustomerService appcustomer.Service
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}

	if c.AppService == nil {
		return errors.New("app service is required")
	}

	if c.AppCustomerService == nil {
		return errors.New("app customer service is required")
	}

	return nil
}

func New(config Config) (appstripe.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	adapter := &adapter{
		db:                 entcontext.NewClient(config.Client),
		appService:         config.AppService,
		appCustomerService: config.AppCustomerService,
	}

	return adapter, nil
}

var _ appstripe.Adapter = (*adapter)(nil)

type adapter struct {
	db entcontext.DB

	appService         app.Service
	appCustomerService appcustomer.Service
}

func (a *adapter) DB() entcontext.DB {
	return a.db
}
