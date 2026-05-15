package app

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/pkg/models"
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
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	if i.APIKey == "" {
		return models.NewGenericValidationError(errors.New("api key is required"))
	}

	if i.Name == "" {
		return models.NewGenericValidationError(errors.New("name is required"))
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
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	if i.Name == "" {
		return models.NewGenericValidationError(errors.New("name is required"))
	}

	return nil
}

type RegistryItem struct {
	Listing MarketplaceListing
	Factory AppFactory
}

func (r RegistryItem) Validate() error {
	var errs []error

	if err := r.Listing.Validate(); err != nil {
		errs = append(errs, err)
	}

	if r.Factory == nil {
		errs = append(errs, models.NewGenericValidationError(errors.New("factory is required")))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
