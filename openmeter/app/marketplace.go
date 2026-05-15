package app

import (
	"errors"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type InstallMethod string

const (
	InstallMethodOAuth2        InstallMethod = "with_oauth2"
	InstallMethodAPIKey        InstallMethod = "with_api_key"
	InstallMethodNoCredentials InstallMethod = "no_credentials_required"
)

func (i InstallMethod) Validate() error {
	if i == "" {
		return models.NewGenericValidationError(errors.New("install method is required"))
	}

	if !slices.Contains([]InstallMethod{
		InstallMethodOAuth2,
		InstallMethodAPIKey,
		InstallMethodNoCredentials,
	}, i) {
		return models.NewGenericValidationError(fmt.Errorf("invalid install method: %s", i))
	}

	return nil
}

type MarketplaceListing struct {
	Type           AppType         `json:"type"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Capabilities   []Capability    `json:"capabilities"`
	InstallMethods []InstallMethod `json:"installMethods"`
}

func (p MarketplaceListing) Validate() error {
	if p.Type == "" {
		return models.NewGenericValidationError(errors.New("type is required"))
	}

	if p.Name == "" {
		return models.NewGenericValidationError(errors.New("name is required"))
	}

	if p.Description == "" {
		return models.NewGenericValidationError(errors.New("description is required"))
	}

	for i, capability := range p.Capabilities {
		if err := capability.Validate(); err != nil {
			return fmt.Errorf("error validating capability at position %d: %w", i, err)
		}
	}

	for i, installMethod := range p.InstallMethods {
		if err := installMethod.Validate(); err != nil {
			return fmt.Errorf("error validating install method at position %d: %w", i, err)
		}
	}

	return nil
}

type Capability struct {
	Type        CapabilityType `json:"type"`
	Key         string         `json:"key"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
}

func (c Capability) Validate() error {
	if c.Key == "" {
		return models.NewGenericValidationError(errors.New("key is required"))
	}

	if c.Name == "" {
		return models.NewGenericValidationError(errors.New("name is required"))
	}

	if c.Description == "" {
		return models.NewGenericValidationError(errors.New("description is required"))
	}

	return nil
}

type MarketplaceListingID struct {
	Type AppType
}

func (i MarketplaceListingID) Validate() error {
	if i.Type == "" {
		return models.NewGenericValidationError(errors.New("type is required"))
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
		return models.NewGenericValidationError(fmt.Errorf("error validating page: %w", err))
	}

	return nil
}

type InstallAppWithAPIKeyInput struct {
	InstallAppInput

	APIKey string
}

func (i InstallAppWithAPIKeyInput) Validate() error {
	var errs []error

	if err := i.InstallAppInput.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.APIKey == "" {
		errs = append(errs, models.NewGenericValidationError(errors.New("api key is required")))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type InstallAppInput struct {
	MarketplaceListingID

	Namespace string
	Name      string
}

func (i InstallAppInput) Validate() error {
	var errs []error

	if err := i.MarketplaceListingID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Namespace == "" {
		errs = append(errs, models.NewGenericValidationError(errors.New("namespace is required")))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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
	var errs []error

	if err := i.MarketplaceListingID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.State == "" {
		errs = append(errs, models.NewGenericValidationError(errors.New("state is required")))
	}

	if i.Error != "" && i.Code != "" {
		errs = append(errs, models.NewGenericValidationError(errors.New("code and error cannot be set at the same time")))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
