package app

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

type TxAdapter interface {
	AppAdapter
	MarketplaceAdapter

	Commit() error
	Rollback() error
}

type Adapter interface {
	AppAdapter
	MarketplaceAdapter

	WithTx(context.Context) (TxAdapter, error)
}

type MarketplaceAdapter interface {
	GetListing(ctx context.Context, input GetMarketplaceListingInput) (MarketplaceListing, error)
	ListListings(ctx context.Context, input ListMarketplaceListingInput) (pagination.PagedResponse[MarketplaceListing], error)
	InstallAppWithAPIKey(ctx context.Context, input InstallAppWithAPIKeyInput) (App, error)
	GetOauth2InstallURL(ctx context.Context, input GetOauth2InstallURLInput) (GetOauth2InstallURLOutput, error)
	AuthorizeOauth2Install(ctx context.Context, input AuthorizeOauth2InstallInput) error
}

type AppAdapter interface {
	GetApp(ctx context.Context, input GetAppInput) (App, error)
	ListApps(ctx context.Context, input ListAppInput) (pagination.PagedResponse[App], error)
	UninstallApp(ctx context.Context, input DeleteAppInput) error
}

func WithTxNoValue(ctx context.Context, repo Adapter, fn func(ctx context.Context, repo TxAdapter) error) error {
	var err error

	wrapped := func(ctx context.Context, repo TxAdapter) (interface{}, error) {
		if err = fn(ctx, repo); err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err = WithTx(ctx, repo, wrapped)

	return err
}

func WithTx[T any](ctx context.Context, repo Adapter, fn func(ctx context.Context, repo TxAdapter) (T, error)) (resp T, err error) {
	var txRepo TxAdapter

	txRepo, err = repo.WithTx(ctx)
	if err != nil {
		return resp, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v: %w", r, err)

			if e := txRepo.Rollback(); e != nil {
				err = fmt.Errorf("failed to rollback transaction: %w: %w", e, err)
			}

			return
		}

		if err != nil {
			if e := txRepo.Rollback(); e != nil {
				err = fmt.Errorf("failed to rollback transaction: %w: %w", e, err)
			}

			return
		}

		if e := txRepo.Commit(); e != nil {
			err = fmt.Errorf("failed to commit transaction: %w", e)
		}
	}()

	resp, err = fn(ctx, txRepo)
	if err != nil {
		err = fmt.Errorf("failed to execute transaction: %w", err)
		return
	}

	return
}
