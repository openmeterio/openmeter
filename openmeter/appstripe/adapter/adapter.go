package appstripeadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/appcustomer"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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

// Tx implements entutils.TxCreator interface
func (a adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}
