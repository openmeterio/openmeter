package adapter

import (
	"context"
	stdsql "database/sql"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type repo struct {
	db *db.Client
}

var _ ledgerhistorical.Repo = (*repo)(nil)

func NewRepo(dbClient *db.Client) ledgerhistorical.Repo {
	return &repo{
		db: dbClient,
	}
}

func (r *repo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &stdsql.TxOptions{ReadOnly: false})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}

	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

var _ entutils.TxUser[*repo] = (*repo)(nil)

func (r *repo) WithTx(ctx context.Context, tx *entutils.TxDriver) *repo {
	return &repo{
		db: db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client(),
	}
}

func (r *repo) Self() *repo {
	return r
}
