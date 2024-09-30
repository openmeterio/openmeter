package appentity

import (
	"context"
	"errors"
	"fmt"
)

type AppFactory interface {
	NewApp(context.Context, AppBase) (App, error)
}

type RegistryItem struct {
	Listing MarketplaceListing
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
