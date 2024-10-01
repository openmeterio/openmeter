package entutils

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"sync"

	"entgo.io/ent/dialect"

	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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
	SavePoint(name string) error
	RollbackTo(name string) error
	Release(name string) error
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

type txSavepoint int

const (
	txSavepointNone txSavepoint = 0
)

func (sp txSavepoint) Next() txSavepoint {
	return sp + 1
}

func (sp txSavepoint) Prev() txSavepoint {
	if sp == txSavepointNone {
		return txSavepointNone
	}

	return sp - 1
}

func (sp txSavepoint) String() string {
	return "s" + strconv.Itoa(int(sp))
}

type TxDriver struct {
	driver Transactable
	// db.config is nominally different but structurally identical for all generations of entgo,
	// so we represent it as an interface{} here
	cfg *RawEntConfig

	mu   sync.Mutex
	once sync.Once

	currentSavepoint txSavepoint

	err error
}

var _ transaction.Driver = &TxDriver{}

func (t *TxDriver) GetConfig() *RawEntConfig {
	return t.cfg
}

// Commit commits the (complete) transaction.
func (t *TxDriver) Commit() error {
	// lock so we don't use the driver twice
	t.mu.Lock()
	defer t.mu.Unlock()

	// If there was an error before, we don't do anything
	if t.err != nil {
		return t.err
	}

	if t.currentSavepoint != txSavepointNone {
		// If we're not at the top level, we release the savepoint
		if err := t.driver.Release(t.currentSavepoint.String()); err == nil {
			t.currentSavepoint = t.currentSavepoint.Prev()
		} else {
			t.err = err
		}
	} else {
		// If we're at the top level, we commit the transaction
		t.err = t.driver.Commit()
	}

	return t.err
}

// Rollback rolls back the (complete) transaction.
func (t *TxDriver) Rollback() error {
	// lock so we don't use the driver twice
	t.mu.Lock()
	defer t.mu.Unlock()

	// If there was an error before, we don't do anything
	if t.err != nil {
		return t.err
	}

	if t.currentSavepoint != txSavepointNone {
		// If we're not at the top level, we rollback to the savepoint
		if err := t.driver.RollbackTo(t.currentSavepoint.String()); err == nil {
			t.currentSavepoint = t.currentSavepoint.Prev()
		} else {
			t.err = err
		}
	} else {
		// If we're at the top level, we rollback the transaction
		t.err = t.driver.Rollback()
	}

	return t.err
}

func (t *TxDriver) SavePoint() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	skipSavePoint := false

	t.once.Do(func() {
		// As savePoint() is called each time we use the wrapper (including the first)
		// we don't want to create a savepoint for the first call, otherwise the transaction itself
		// would never be closed.
		skipSavePoint = true
	})

	if !skipSavePoint {
		next := t.currentSavepoint.Next()

		err := t.driver.SavePoint(next.String())
		if err != nil {
			return err
		}

		t.currentSavepoint = next
	}

	return nil
}

// Able to start a new transaction
type TxCreator = transaction.Creator

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

// TransactingRepo is a helper that can be used inside repository methods.
// It uses any preexisting transaction in the context or starts and executes a new one.
func TransactingRepo[R, T any](
	ctx context.Context,
	repo interface {
		TxUser[T]
		TxCreator
	},
	cb func(ctx context.Context, rep T) (R, error),
) (R, error) {
	return transaction.Run(ctx, repo, func(ctx context.Context) (R, error) {
		var def R
		tx, err := GetDriverFromContext(ctx)
		if err != nil {
			return def, err
		}
		return cb(ctx, repo.WithTx(ctx, tx))
	})
}

// Only use for direct interacton with the Ent driver implementation
func GetDriverFromContext(ctx context.Context) (*TxDriver, error) {
	driver, err := transaction.GetDriverFromContext(ctx)
	if err != nil {
		return nil, err
	}
	entTxDriver, ok := driver.(*TxDriver)
	if !ok {
		return nil, fmt.Errorf("tx driver is not ent tx driver")
	}
	return entTxDriver, nil
}
