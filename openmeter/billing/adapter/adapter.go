package billingadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Client *entdb.Client
	Logger *slog.Logger
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}

	if c.Logger == nil {
		return errors.New("logger is required")
	}

	return nil
}

func New(config Config) (billing.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	cache, err := lru.New[string, *db.BillingCustomerOverride](defaultCustomerOverrideCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer override cache: %w", err)
	}

	return &adapter{
		db:                          config.Client,
		dbWithoutTrns:               config.Client,
		logger:                      config.Logger,
		upsertCustomerOverrideCache: cache,
	}, nil
}

var _ billing.Adapter = (*adapter)(nil)

type adapter struct {
	db *entdb.Client
	// dbWithoutTrns is used to execute any upsert operations outside of ctx driven transactions
	dbWithoutTrns               *entdb.Client
	logger                      *slog.Logger
	upsertCustomerOverrideCache *lru.Cache[string, *db.BillingCustomerOverride]
}

func (a adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) billing.Adapter {
	txDb := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())

	return &adapter{
		db:                          txDb.Client(),
		dbWithoutTrns:               a.dbWithoutTrns,
		logger:                      a.logger,
		upsertCustomerOverrideCache: a.upsertCustomerOverrideCache,
	}
}
