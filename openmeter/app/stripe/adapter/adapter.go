package appstripeadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/secret"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Client              *entdb.Client
	AppService          app.Service
	CustomerService     customer.Service
	SecretService       secret.Service
	StripeClientFactory stripeclient.StripeClientFactory
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}

	if c.AppService == nil {
		return errors.New("app service is required")
	}

	if c.CustomerService == nil {
		return errors.New("customer service is required")
	}

	if c.SecretService == nil {
		return errors.New("secret service is required")
	}

	return nil
}

func New(config Config) (appstripe.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	// Create stripe app factory
	stripeClientFactory := config.StripeClientFactory
	if stripeClientFactory == nil {
		stripeClientFactory = stripeclient.NewStripeClient
	}

	// Create app stripe adapter
	adapter := &adapter{
		db:                  config.Client,
		appService:          config.AppService,
		customerService:     config.CustomerService,
		secretService:       config.SecretService,
		stripeClientFactory: stripeClientFactory,
	}

	return adapter, nil
}

var _ appstripe.Adapter = (*adapter)(nil)

type adapter struct {
	db *entdb.Client

	appService          app.Service
	customerService     customer.Service
	secretService       secret.Service
	stripeClientFactory stripeclient.StripeClientFactory
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

func (a adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return &adapter{
		db:                  txClient.Client(),
		appService:          a.appService,
		customerService:     a.customerService,
		secretService:       a.secretService,
		stripeClientFactory: a.stripeClientFactory,
	}
}
