package adapter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ currencies.Adapter = (*adapter)(nil)

// Tx implements entutils.TxCreator interface
func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter {
	txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return &adapter{
		db:     txClient.Client(),
		logger: a.logger,
	}
}

func (a *adapter) Self() *adapter {
	return a
}

func (a *adapter) ListCurrencies(ctx context.Context, params ListCurrenciesInput) (pagination.Result[Currency], error) {
	return pagination.Result[Currency]{}, nil
}
