package app

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

// AppType represents the type of an app
type AppType string

const (
	AppTypeStripe          AppType = "stripe"
	AppTypeSandbox         AppType = "sandbox"
	AppTypeCustomInvoicing AppType = "custom_invoicing"
)

// AppStatus represents the status of an app
type AppStatus string

const (
	AppStatusReady        AppStatus = "ready"
	AppStatusUnauthorized AppStatus = "unauthorized"
)

type CapabilityType string

const (
	CapabilityTypeReportUsage      CapabilityType = "reportUsage"
	CapabilityTypeReportEvents     CapabilityType = "reportEvents"
	CapabilityTypeCalculateTax     CapabilityType = "calculateTax"
	CapabilityTypeInvoiceCustomers CapabilityType = "invoiceCustomers"
	CapabilityTypeCollectPayments  CapabilityType = "collectPayments"
)

// AppBase represents an abstract with the base fields of an app
type AppBase struct {
	models.ManagedResource

	Type     AppType            `json:"type"`
	Status   AppStatus          `json:"status"`
	Listing  MarketplaceListing `json:"listing"`
	Metadata models.Metadata    `json:"metadata,omitempty"`
}

func (a AppBase) GetAppBase() AppBase {
	return a
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

func (a AppBase) GetDescription() *string {
	return a.Description
}

func (a AppBase) GetStatus() AppStatus {
	return a.Status
}

func (a AppBase) GetListing() MarketplaceListing {
	return a.Listing
}

func (a AppBase) GetMetadata() models.Metadata {
	return a.Metadata
}

// ValidateCapabilities validates if the app can run for the given capabilities
func (a AppBase) ValidateCapabilities(capabilities ...CapabilityType) error {
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
// func (a AppBase) ValidateCustomer(c customerentity.Customer, capabilities []CapabilityType) error {
// 	return fmt.Errorf("each app must implement its own ValidateCustomer method")
// }

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
