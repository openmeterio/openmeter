package pglockx

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"cirello.io/pglock"
)

const (
	lockTable                = "distributed_locks"
	DefaultHeartbeatInterval = 3 * time.Second
	DefaultLeaseTime         = time.Minute
)

type Config struct {
	LeaseTime         time.Duration
	HeartbeatInterval time.Duration
	Owner             string
}

func (c Config) Validate() error {
	var errs []error

	if c.LeaseTime/2 < c.HeartbeatInterval {
		errs = append(errs, errors.New("lease time must be at least twice as long as heartbeat interval"))
	}

	if c.Owner == "" {
		errs = append(errs, errors.New("lock owner is required"))
	}

	return errors.Join(errs...)
}

func New(db *sql.DB, config Config) (*pglock.Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid lock configuration: %w", err)
	}

	client, err := pglock.UnsafeNew(db,
		pglock.WithCustomTable(lockTable),
		pglock.WithLeaseDuration(config.LeaseTime),
		pglock.WithHeartbeatFrequency(config.HeartbeatInterval),
		pglock.WithOwner(config.Owner),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize lock client: %w", err)
	}

	return client, nil
}
