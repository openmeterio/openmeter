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
	tx          *entdb.Tx
	marketplace app.MarketplaceAdapter
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

func (r adapter) WithTx(ctx context.Context) (app.TxAdapter, error) {
	if r.tx != nil {
		return r, nil
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &adapter{
		db:          r.db,
		tx:          tx,
		marketplace: r.marketplace,
	}, nil
}
