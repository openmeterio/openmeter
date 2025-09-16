package addon

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// TODO: add bulk api

type Repository interface {
	entutils.TxCreator

	ListAddons(ctx context.Context, params ListAddonsInput) (pagination.Result[Addon], error)
	CreateAddon(ctx context.Context, params CreateAddonInput) (*Addon, error)
	DeleteAddon(ctx context.Context, params DeleteAddonInput) error
	GetAddon(ctx context.Context, params GetAddonInput) (*Addon, error)
	UpdateAddon(ctx context.Context, params UpdateAddonInput) (*Addon, error)
}
