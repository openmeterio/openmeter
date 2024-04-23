package postgres_connector

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db"
)

// we can use this interface if we want to generalize it

// type transactionStarter interface {
//	startTransaction(ctx context.Context) (*db.Tx, error)
//	upsertSubjectLedger(ctx context.Context, namespace, subject string) error
//	getHighWatermarkWithTrnsLock(ctx context.Context, namespace, subject string) (time.Time, error)
// }

func (c *PostgresConnector) startTransaction(ctx context.Context) (*db.Tx, error) {
	return c.db.Tx(ctx)
}

func transaction[R any](ctx context.Context, connector *PostgresConnector, callback func(tx *db.Tx) (*R, error)) (*R, error) {
	tx, err := connector.startTransaction(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		// This prevents pgsql connections being stuck in a transaction if a panic occurs
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(err)
		}
	}()

	result, err := callback(tx)
	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			err = fmt.Errorf("%w: %v", err, rerr)
		}
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func syncronizedTransaction[R any](ctx context.Context, connector *PostgresConnector, namespace, subject string, callback func(tx *db.Tx, highWatermark time.Time) (*R, error)) (*R, error) {
	err := connector.upsertSubjectLedger(ctx, namespace, subject)
	if err != nil {
		return nil, err
	}

	return transaction(ctx, connector, func(tx *db.Tx) (*R, error) {
		highWatermark, err := connector.getHighWatermarkWithTrnsLock(ctx, namespace, subject)
		if err != nil {
			return nil, err
		}

		return callback(tx, highWatermark)

	})
}
