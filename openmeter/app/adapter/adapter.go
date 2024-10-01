package appadapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

type Config struct {
	Client      *entdb.Client
	Marketplace app.MarketplaceAdapter
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}

	if c.Marketplace == nil {
		return errors.New("marketplace adapter is required")
	}

	return nil
}

func New(config Config) (app.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	adapter := &adapter{
		db:          config.Client,
		marketplace: config.Marketplace,
	}

	return adapter, nil
}

func NewMarketplaceAdapter() app.MarketplaceAdapter {
	return DefaultMarketplace()
}

var _ app.Adapter = (*adapter)(nil)

type adapter struct {
	db          *entdb.Client
	marketplace app.MarketplaceAdapter
}

func (r adapter) Commit(ctx context.Context) error {
	tx := entdb.TxFromContext(ctx)
	if tx != nil {
		return tx.Commit()
	}

	return nil
}

func (r adapter) Rollback(ctx context.Context) error {
	tx := entdb.TxFromContext(ctx)
	if tx != nil {
		return tx.Rollback()
	}

	return nil
}

func (r adapter) client(ctx context.Context) *entdb.Client {
	client := entdb.FromContext(ctx)
	if client != nil {
		return client
	}

	return r.db
}

func (r adapter) WithTx(ctx context.Context) (context.Context, error) {
	// If there is already a transaction in the context, we don't need to create a new one
	tx := entdb.TxFromContext(ctx)
	if tx != nil {
		return ctx, nil
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	ctx = entdb.NewTxContext(ctx, tx)

	return ctx, nil
}
