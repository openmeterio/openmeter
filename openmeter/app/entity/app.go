package appentity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// App represents an installed app
type App interface {
	GetID() AppID
	GetType() AppType
	GetName() string
	GetStatus() AppStatus
	GetListing() MarketplaceListing

	// ValidateCapabilities validates if the app can run for the given capabilities
	ValidateCapabilities(capabilities []CapabilityType) error

	// ValidateCustomer validates if the app can run for the given customer
	ValidateCustomer(customer customer.Customer, capabilities []CapabilityType) error
}

// AppType represents the type of an app
type AppType string

const (
	AppTypeStripe AppType = "stripe"
)

// AppStatus represents the status of an app
type AppStatus string

const (
	AppStatusReady        AppStatus = "ready"
	AppStatusUnauthorized AppStatus = "unauthorized"
)

// AppBase represents an abstract with the base fields of an app
type AppBase struct {
	models.ManagedResource

	Type    AppType            `json:"type"`
	Name    string             `json:"name"`
	Status  AppStatus          `json:"status"`
	Listing MarketplaceListing `json:"listing"`
}

func (a AppBase) GetID() AppID {
	return AppID{
		Namespace: a.Namespace,
		ID:        a.ID,
	}
}

func (a AppBase) GetType() AppType {
	return a.Type
}

func (a AppBase) GetName() string {
	return a.Name
}

func (a AppBase) GetStatus() AppStatus {
	return a.Status
}

func (a AppBase) GetListing() MarketplaceListing {
	return a.Listing
}

// ValidateCapabilities validates if the app can run for the given capabilities
func (a AppBase) ValidateCapabilities(capabilities []CapabilityType) error {
	for _, capability := range capabilities {
		found := false

		for _, c := range a.Listing.Capabilities {
			if c.Type == capability {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("capability %s is not supported by %s app type", capability, a.Type)
		}
	}

	return nil
}

// ValidateCustomer validates if the app can run for the given customer
func (a AppBase) ValidateCustomer(c customer.Customer, capabilities []CapabilityType) error {
	return fmt.Errorf("each app must implement its own ValidateCustomer method")
}

// Validate validates the app base
func (a AppBase) Validate() error {
	if err := a.ManagedResource.Validate(); err != nil {
		return fmt.Errorf("error validating managed resource: %w", err)
	}

	if a.ID == "" {
		return errors.New("id is required")
	}

	if a.Namespace == "" {
		return errors.New("namespace is required")
	}

	if a.Name == "" {
		return errors.New("name is required")
	}

	if a.Status == "" {
		return errors.New("status is required")
	}

	if err := a.Listing.Validate(); err != nil {
		return fmt.Errorf("error validating listing: %w", err)
	}

	return nil
}

// AppID represents the unique identifier for an installed app
type AppID struct {
	Namespace string
	ID        string
}

func (i AppID) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.ID == "" {
		return errors.New("id is required")
	}

	return nil
}

// GetAppInput is the input for getting an installed app
type GetAppInput = AppID

// ListAppInput is the input for listing installed apps
type ListAppInput struct {
	AppID
	pagination.Page

	IncludeDeleted bool
}

func (i ListAppInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	return nil
}

type DeleteAppInput = AppID
