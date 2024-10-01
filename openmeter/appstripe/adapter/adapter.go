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

	appService         app.Service
	appCustomerService appcustomer.Service
}

func (r adapter) Commit(ctx context.Context) error {
	tx := entdb.TxFromContext(ctx)
	if tx != nil {
		return tx.Commit()
	}

	return nil
}

func (r adapter) Rollback(ctx context.Context) error {
	tx := entdb.TxFromContext(ctx)
	if tx != nil {
		return tx.Rollback()
	}

	return nil
}

func (r adapter) client(ctx context.Context) *entdb.Client {
	client := entdb.FromContext(ctx)
	if client != nil {
		return client
	}

	return r.db
}

func (r adapter) WithTx(ctx context.Context) (context.Context, error) {
	// If there is already a transaction in the context, we don't need to create a new one
	tx := entdb.TxFromContext(ctx)
	if tx != nil {
		return ctx, nil
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	ctx = entdb.NewTxContext(ctx, tx)

	return ctx, nil
}
