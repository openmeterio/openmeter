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
	GetDescription() *string
	GetStatus() appentitybase.AppStatus
	GetMetadata() map[string]string
	GetListing() appentitybase.MarketplaceListing

	// ValidateCapabilities validates if the app can run for the given capabilities
	ValidateCapabilities(capabilities ...appentitybase.CapabilityType) error

	// Customer data
	GetCustomerData(ctx context.Context, input GetAppInstanceCustomerDataInput) (CustomerData, error)
	UpsertCustomerData(ctx context.Context, input UpsertAppInstanceCustomerDataInput) error
	DeleteCustomerData(ctx context.Context, input DeleteAppInstanceCustomerDataInput) error
}

type GetAppInstanceCustomerDataInput struct {
	CustomerID customerentity.CustomerID
}

func (i GetAppInstanceCustomerDataInput) Validate() error {
	if err := i.CustomerID.Validate(); err != nil {
		return err
	}

	return nil
}

type UpsertAppInstanceCustomerDataInput struct {
	CustomerID customerentity.CustomerID
	Data       CustomerData
}

func (i UpsertAppInstanceCustomerDataInput) Validate() error {
	if err := i.CustomerID.Validate(); err != nil {
		return err
	}

	if err := i.Data.Validate(); err != nil {
		return err
	}

	return nil
}

type DeleteAppInstanceCustomerDataInput struct {
	CustomerID customerentity.CustomerID
}

func (i DeleteAppInstanceCustomerDataInput) Validate() error {
	if err := i.CustomerID.Validate(); err != nil {
		return err
	}

	return nil
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

	return nil
}

// ListAppInput is the input for listing installed apps
type ListAppInput struct {
	Namespace string
	pagination.Page

	Type           *appentitybase.AppType
	IncludeDeleted bool
	// Only list apps that has data for the given customer
	CustomerID *customerentity.CustomerID
}

func (i ListAppInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if err := i.Page.Validate(); err != nil {
		return fmt.Errorf("error validating page: %w", err)
	}

	if i.CustomerID != nil {
		if err := i.CustomerID.Validate(); err != nil {
			return fmt.Errorf("error validating customer ID: %w", err)
		}

		if i.CustomerID.Namespace != i.Namespace {
			return fmt.Errorf("customer ID namespace %s does not match app namespace %s", i.CustomerID.Namespace, i.Namespace)
		}
	}

	return nil
}
