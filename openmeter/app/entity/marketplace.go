package appentity

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

type MarketplaceListing struct {
	Type         AppType      `json:"type"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	IconURL      string       `json:"iconUrl"`
	Capabilities []Capability `json:"capabilities"`
}

func (p MarketplaceListing) Validate() error {
	if p.Type == "" {
		return errors.New("type is required")
	}

	if p.Name == "" {
		return errors.New("name is required")
	}

	if p.Description == "" {
		return errors.New("description is required")
	}

	if p.IconURL == "" {
		return errors.New("icon url is required")
	}

	for i, capability := range p.Capabilities {
		if err := capability.Validate(); err != nil {
			return fmt.Errorf("error validating capability a position %d: %w", i, err)
		}
	}

	return nil
}

type CapabilityType string

const (
	CapabilityTypeReportUsage      CapabilityType = "reportUsage"
	CapabilityTypeReportEvents     CapabilityType = "reportEvents"
	CapabilityTypeCalculateTax     CapabilityType = "calculateTax"
	CapabilityTypeInvoiceCustomers CapabilityType = "invoiceCustomers"
	CapabilityTypeCollectPayments  CapabilityType = "collectPayments"
)

type Capability struct {
	Type        CapabilityType `json:"type"`
	Key         string         `json:"key"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
}

func (c Capability) Validate() error {
	if c.Key == "" {
		return errors.New("key is required")
	}

	if c.Name == "" {
		return errors.New("name is required")
	}

	if c.Description == "" {
		return errors.New("description is required")
	}

	return nil
}

type MarketplaceListingID struct {
	Type AppType
}

func (i MarketplaceListingID) Validate() error {
	if i.Type == "" {
		return errors.New("type is required")
	}

	return nil
}

type RegisterMarketplaceListingInput = MarketplaceListing

type GetMarketplaceListingInput = MarketplaceListingID

type ListMarketplaceListingInput struct {
	pagination.Page
}

func (i ListMarketplaceListingInput) Validate() error {
	if err := i.Page.Validate(); err != nil {
		return fmt.Errorf("error validating page: %w", err)
	}

	return nil
}

type InstallAppWithAPIKeyInput struct {
	MarketplaceListingID

	APIKey string
}

func (i InstallAppWithAPIKeyInput) Validate() error {
	if err := i.MarketplaceListingID.Validate(); err != nil {
		return fmt.Errorf("error validating marketplace listing id: %w", err)
	}

	return nil
}

type GetOauth2InstallURLInput = MarketplaceListingID

type GetOauth2InstallURLOutput struct {
	URL string
}

type AuthorizeOauth2InstallInput struct {
	MarketplaceListingID

	Code string
	// Success response fields
	State string
	// Error response fields
	Error            string
	ErrorDescription string
	ErrorURI         string
}

func (i AuthorizeOauth2InstallInput) Validate() error {
	if err := i.MarketplaceListingID.Validate(); err != nil {
		return fmt.Errorf("error validating marketplace listing id: %w", err)
	}

	if i.State == "" {
		return errors.New("state is required")
	}

	if i.Error != "" && i.Code != "" {
		return errors.New("code and error cannot be set at the same time")
	}

	return nil
}
