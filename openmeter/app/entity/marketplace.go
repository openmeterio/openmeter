package appentity

import (
	"errors"
	"fmt"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type MarketplaceListingID struct {
	Type appentitybase.AppType
}

func (i MarketplaceListingID) Validate() error {
	if i.Type == "" {
		return errors.New("type is required")
	}

	return nil
}

type RegisterMarketplaceListingInput = RegistryItem

type MarketplaceGetInput = MarketplaceListingID

type MarketplaceListInput struct {
	pagination.Page
}

func (i MarketplaceListInput) Validate() error {
	if err := i.Page.Validate(); err != nil {
		return fmt.Errorf("error validating page: %w", err)
	}

	return nil
}

type InstallAppWithAPIKeyInput struct {
	MarketplaceListingID

	Namespace string
	APIKey    string
	Name      string
}

func (i InstallAppWithAPIKeyInput) Validate() error {
	if err := i.MarketplaceListingID.Validate(); err != nil {
		return fmt.Errorf("error validating marketplace listing id: %w", err)
	}

	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.APIKey == "" {
		return errors.New("api key is required")
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
