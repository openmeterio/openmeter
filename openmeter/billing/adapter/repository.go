package adapter

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

func New(config Config) (billing.Repository, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &repository{
		db: config.Client,
	}, nil
}

var _ billing.Repository = (*repository)(nil)

type repository struct {
	db *entdb.Client
	tx *entdb.Tx
}

func (r repository) Commit() error {
	if r.tx != nil {
		return r.tx.Commit()
	}

	return nil
}

func (r repository) Rollback() error {
	if r.tx != nil {
		return r.tx.Rollback()
	}

	return nil
}

func (r repository) client() *entdb.Client {
	if r.tx != nil {
		return r.tx.Client()
	}

	return r.db
}

func (r repository) WithTx(ctx context.Context) (billing.TxRepository, error) {
	if r.tx != nil {
		return r, nil
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &repository{
		db: r.db,
		tx: tx,
	}, nil
}
