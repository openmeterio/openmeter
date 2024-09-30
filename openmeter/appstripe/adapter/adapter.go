package appstripeadapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/appcustomer"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
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
		db:                 config.Client,
		appService:         config.AppService,
		appCustomerService: config.AppCustomerService,
	}

	return adapter, nil
}

var _ appstripe.Adapter = (*adapter)(nil)

type adapter struct {
	db *entdb.Client
	tx *entdb.Tx

	appService         app.Service
	appCustomerService appcustomer.Service
}

func (r adapter) Commit() error {
	if r.tx != nil {
		return r.tx.Commit()
	}

	return nil
}

func (r adapter) Rollback() error {
	if r.tx != nil {
		return r.tx.Rollback()
	}

	return nil
}

func (r adapter) client() *entdb.Client {
	if r.tx != nil {
		return r.tx.Client()
	}

	return r.db
}

func (r adapter) WithTx(ctx context.Context) (appstripe.TxAdapter, error) {
	if r.tx != nil {
		return r, nil
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &adapter{
		db:                 r.db,
		tx:                 tx,
		appService:         r.appService,
		appCustomerService: r.appCustomerService,
	}, nil
}
