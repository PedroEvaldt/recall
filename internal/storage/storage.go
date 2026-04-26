package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dbUrl string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		return nil, errors.New("failed to get pool conection")
	}
	err = pool.Ping(ctx)
	if err != nil {
		return nil, errors.New("failed to ping to pool")
	}
	return pool, nil
}
