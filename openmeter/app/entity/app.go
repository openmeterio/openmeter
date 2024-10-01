package appentity

import (
	"context"
	"errors"
	"fmt"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// App represents an installed app
type App interface {
	GetAppBase() appentitybase.AppBase
	GetID() appentitybase.AppID
	GetType() appentitybase.AppType
	GetName() string
	GetStatus() appentitybase.AppStatus
	Get() appentitybase.MarketplaceListing

	// ValidateCapabilities validates if the app can run for the given capabilities
	ValidateCapabilities(capabilities []appentitybase.CapabilityType) error

	// // ValidateCustomer validates if the app can run for the given customer
	ValidateCustomer(ctx context.Context, customer *customerentity.Customer, capabilities []appentitybase.CapabilityType) error
}

// GetAppInput is the input for getting an installed app
type GetAppInput = appentitybase.AppID

type GetDefaultAppInput struct {
	Namespace string
	Type      appentitybase.AppType
}

func (i GetDefaultAppInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Type == "" {
		return errors.New("type is required")
	}

	return nil
}

// CreateAppInput is the input for creating an app
type CreateAppInput struct {
	Namespace   string
	Name        string
	Description string
	Type        appentitybase.AppType
}

func (i CreateAppInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Name == "" {
		return errors.New("name is required")
	}

	if i.Description == "" {
		return errors.New("description is required")
	}

	return nil
}

// ListAppInput is the input for listing installed apps
type ListAppInput struct {
	Namespace string
	pagination.Page

	Type           *appentitybase.AppType
	IncludeDeleted bool
}

func (i ListAppInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := i.Page.Validate(); err != nil {
		return fmt.Errorf("error validating page: %w", err)
	}

	return nil
}

type DeleteAppInput = appentitybase.AppID