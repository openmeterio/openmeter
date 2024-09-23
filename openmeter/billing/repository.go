package billing

import (
	"context"
	"errors"
	"fmt"
)

type TxRepository interface {
	ProfileRepository

	Commit() error
	Rollback() error
}

type Repository interface {
	ProfileRepository

	WithTx(context.Context) (TxRepository, error)
}

type ProfileRepository interface {
	CreateProfile(ctx context.Context, params CreateProfileInput) (*Profile, error)
	GetProfileByKey(ctx context.Context, params RepoGetProfileByKeyInput) (*Profile, error)
	GetDefaultProfile(ctx context.Context, params RepoGetDefaultProfileInput) (*Profile, error)
}

type RepoGetProfileByKeyInput struct {
	Namespace string
	Key       string
}

func (i RepoGetProfileByKeyInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Key == "" {
		return errors.New("key is required")
	}

	return nil
}

type RepoGetDefaultProfileInput struct {
	Namespace string
}

func (i RepoGetDefaultProfileInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	return nil
}

func WithTxNoValue(ctx context.Context, repo Repository, fn func(ctx context.Context, repo TxRepository) error) error {
	var err error

	wrapped := func(ctx context.Context, repo TxRepository) (interface{}, error) {
		if err = fn(ctx, repo); err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err = WithTx[any](ctx, repo, wrapped)

	return err
}

func WithTx[T any](ctx context.Context, repo Repository, fn func(ctx context.Context, repo TxRepository) (T, error)) (resp T, err error) {
	var txRepo TxRepository

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
