package billingadapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

type Config struct {
	Client *entdb.Client
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}

	return nil
}

func New(config Config) (billing.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db: config.Client,
	}, nil
}

var _ billing.Adapter = (*adapter)(nil)

type adapter struct {
	db *entdb.Client
	tx *entdb.Tx
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

func (r adapter) WithTx(ctx context.Context) (billing.TxAdapter, error) {
	if r.tx != nil {
		return r, nil
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &adapter{
		db: r.db,
		tx: tx,
	}, nil
}
