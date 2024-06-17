package entutils

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"entgo.io/ent/dialect"
)

type RawEntConfig struct {
	// driver used for executing database requests.
	Driver dialect.Driver
	// debug enable a debug logging.
	Debug bool
	// log used for logging on debug mode.
	Log func(...any)

	// Hooks and interceptors are excluded in transaction handling
	// due to differing types.
	//
	// TODO: implement them in the templating when creating the new transactional client
	// from this RawEntConfig.

	// // hooks to execute on mutations.
	// hooks *hooks
	// // interceptors to execute on queries.
	// inters *inters
}

type Transactable interface {
	Commit() error
	Rollback() error
}

type TxHijacker interface {
	HijackTx(ctx context.Context, opts *sql.TxOptions) (context.Context, *RawEntConfig, Transactable, error)
}

func NewTxDriver(driver Transactable, cfg *RawEntConfig) *TxDriver {
	return &TxDriver{
		driver: driver,
		cfg:    cfg,
	}
}

type TxDriver struct {
	driver Transactable
	// db.config is nominally different but structurally identical for all generations of entgo,
	// so we represent it as an interface{} here
	cfg *RawEntConfig

	mu        sync.Mutex
	endTxOnce sync.Once

	err error
}

func (t *TxDriver) GetConfig() *RawEntConfig {
	return t.cfg
}

func (t *TxDriver) Commit() error {
	// lock so we don't use the driver twice
	t.mu.Lock()
	defer t.mu.Unlock()

	// you can end a transaction only once
	t.endTxOnce.Do(func() {
		t.err = t.driver.Commit()
	})

	return t.err
}

func (t *TxDriver) Rollback() error {
	// lock so we don't use the driver twice
	t.mu.Lock()
	defer t.mu.Unlock()

	// you can end a transaction only once
	t.endTxOnce.Do(func() {
		t.err = t.driver.Rollback()
	})

	return t.err
}

// Able to start a new transaction
type TxCreator interface {
	// Creates a TxDriver from a hijacked ent transaction (the driver of it).
	// Example:
	//
	// type dbAdapter struct {
	// 	db *db.Client
	// }
	//
	// // we have to implement the TxCreator interface
	// func (d *dbAdapter) Tx(ctx context.Context) (context.Context, *entutils.TxDriver, error) {
	//     // HijackTx gets generated when using expose.tpl
	// 	txCtx, rawConfig, eDriver, err := d.db.HijackTX(ctx, &sql.TxOptions{
	// 		ReadOnly: false,
	// 	})
	//
	// 	if err != nil {
	// 		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	// 	}
	// 	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
	// }
	Tx(ctx context.Context) (context.Context, *TxDriver, error)
}

// Able to use an existing transaction
type TxUser[T any] interface {
	// Creates a new instance of the adapter using the provided transaction.
	// Example:
	//
	// type dbAdapter struct {
	//     db *db.Client
	// }
	//
	// func (d *dbAdapter) WithTx(ctx context.Context, tx *entutils.TxDriver) SomeDB[db1.Example1] {
	//     // NewTxClientFromRawConfig gets generated when using expose.tpl
	//     txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	//     res := &db1Adapter{db: txClient.Client()}
	//     return res
	// }
	WithTx(ctx context.Context, tx *TxDriver) T
}

func StartAndRunTx[R any](ctx context.Context, src TxCreator, cb func(ctx context.Context, tx *TxDriver) (*R, error)) (*R, error) {
	txCtx, txDriver, err := src.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	return RunInTransaction(txCtx, txDriver, cb)
}

func RunInTransaction[R any](txCtx context.Context, txDriver *TxDriver, cb func(ctx context.Context, tx *TxDriver) (*R, error)) (*R, error) {
	defer func() {
		if r := recover(); r != nil {
			// roll back the tx for all downstream (WithTx) clients
			_ = txDriver.Rollback()
			panic(r)
		}
	}()

	result, err := cb(txCtx, txDriver)
	if err != nil {
		// roll back the tx for all downstream (WithTx) clients
		if rerr := txDriver.Rollback(); rerr != nil {
			err = fmt.Errorf("%w: %v", err, rerr)
		}
		return nil, err
	}

	// commit the transaction
	err = txDriver.Commit()
	if err != nil {
		return nil, err
	}

	return result, nil
}
