package app

import (
	"context"
	"errors"
	"fmt"
)

type AppFactory interface {
	NewApp(context.Context, AppBase) (App, error)
	UninstallApp(ctx context.Context, input UninstallAppInput) error
}

type AppFactoryInstallWithAPIKey interface {
	InstallAppWithAPIKey(ctx context.Context, input AppFactoryInstallAppWithAPIKeyInput) (App, error)
}

type UninstallAppInput = AppID

type AppFactoryInstallAppWithAPIKeyInput struct {
	Namespace string
	APIKey    string
	Name      string
}

func (i AppFactoryInstallAppWithAPIKeyInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.APIKey == "" {
		return errors.New("api key is required")
	}

	if i.Name == "" {
		return errors.New("name is required")
	}

	return nil
}

type AppFactoryInstall interface {
	InstallApp(ctx context.Context, input AppFactoryInstallAppInput) (App, error)
}

type AppFactoryInstallAppInput struct {
	Namespace string
	Name      string
}

func (i AppFactoryInstallAppInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Name == "" {
		return errors.New("name is required")
	}

	return nil
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
