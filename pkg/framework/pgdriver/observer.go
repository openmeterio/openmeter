package pgdriver

import "github.com/jackc/pgx/v5/pgxpool"

type Observer interface {
	ObservePool(pool *pgxpool.Pool) error
}
