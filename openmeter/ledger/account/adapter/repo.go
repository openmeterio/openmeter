package adapter

import (
	"context"
	"database/sql"
	"fmt"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type repo struct {
	db *entdb.Client
}

var _ ledgeraccount.Repo = (*repo)(nil)

func NewRepo(db *entdb.Client) ledgeraccount.Repo {
	return &repo{
		db: db,
	}
}

func (r *repo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}

	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

var _ entutils.TxUser[*repo] = (*repo)(nil)

func (r *repo) WithTx(ctx context.Context, tx *entutils.TxDriver) *repo {
	return &repo{
		db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client(),
	}
}

func (r *repo) Self() *repo {
	return r
}
