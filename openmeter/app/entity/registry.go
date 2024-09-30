package appentity

import (
	"context"
	"errors"
	"fmt"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

type AppFactory interface {
	NewApp(context.Context, appentitybase.AppBase) (App, error)
}

type RegistryItem struct {
	Listing appentitybase.MarketplaceListing
	Factory AppFactory
}

func (r RegistryItem) Validate() error {
	if err := r.Listing.Validate(); err != nil {
		return fmt.Errorf("error validating registry item: %w", err)
	}

	if r.Factory == nil {
		return errors.New("factory is required")
	}

	return nil
}
