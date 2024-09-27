package apputils

import (
	"context"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
)

type AppGetter interface {
	GetApp(ctx context.Context, input appentity.AppID) (appentity.App, error)
}

type IntegrationGetter[T appentity.App] struct {
	Getter AppGetter
}

func (g IntegrationGetter[T]) Get(ctx context.Context, appID appentity.AppID) (T, error) {
	var empty T
	app, err := g.Getter.GetApp(ctx, appID)
	if err != nil {
		return empty, err
	}

	integration, ok := app.(T)
	if !ok {
		return empty, appentity.ErrIntegrationNotSupported
	}

	return integration, nil
}
