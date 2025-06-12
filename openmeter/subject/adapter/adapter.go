package adapter

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func New(db *entdb.Client) (subject.Adapter, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}

	return &adapter{
		db: db,
	}, nil
}

var (
	_ subject.Adapter           = (*adapter)(nil)
	_ entutils.TxUser[*adapter] = (*adapter)(nil)
)

type adapter struct {
	db *entdb.Client
}

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
	txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig())

	return &adapter{
		db: txDb.Client(),
	}
}

func (a *adapter) Self() *adapter {
	return a
}
