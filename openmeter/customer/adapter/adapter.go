package adapter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
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

func New(config Config) (customer.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db:        config.Client,
		logger:    config.Logger,
		observers: &[]appobserver.Observer[customerentity.Customer]{},
	}, nil
}

var (
	_ customer.Adapter                               = (*adapter)(nil)
	_ appobserver.Publisher[customerentity.Customer] = (*adapter)(nil)
)

type adapter struct {
	db *entdb.Client
	tx *entdb.Tx
	// It is a reference so we can pass it down in WithTx
	observers *[]appobserver.Observer[customerentity.Customer]

	logger *slog.Logger
}

func (r *adapter) Commit() error {
	if r.tx != nil {
		return r.tx.Commit()
	}

	return nil
}

func (r *adapter) Rollback() error {
	if r.tx != nil {
		return r.tx.Rollback()
	}

	return nil
}

func (r *adapter) client() *entdb.Client {
	if r.tx != nil {
		return r.tx.Client()
	}

	return r.db
}

func (r *adapter) WithTx(ctx context.Context) (customer.TxAdapter, error) {
	if r.tx != nil {
		return r, nil
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &adapter{
		db:        r.db,
		tx:        tx,
		logger:    r.logger,
		observers: r.observers,
	}, nil
}
