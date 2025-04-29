package adapter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
)

type Config struct {
	Client *entdb.Client
	Logger *slog.Logger
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("postgres client is required")
	}

	if c.Logger == nil {
		return errors.New("logger must not be nil")
	}

	return nil
}

func New(config Config) (notification.Repository, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &repository{
		db:     config.Client,
		logger: config.Logger,
	}, nil
}

var _ notification.Repository = (*repository)(nil)

type repository struct {
	db *entdb.Client
	tx *entdb.Tx

	logger *slog.Logger
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

func (r repository) WithTx(ctx context.Context) (notification.TxRepository, error) {
	if r.tx != nil {
		return r, nil
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &repository{
		db:     r.db,
		tx:     tx,
		logger: r.logger,
	}, nil
}
