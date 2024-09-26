package app

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type AppType string

const (
	AppTypeStripe AppType = "stripe"
)

type AppStatus string

const (
	AppStatusReady        AppStatus = "ready"
	AppStatusUnauthorized AppStatus = "unauthorized"
)

type App struct {
	models.ManagedResource

	Type       AppType   `json:"type"`
	Name       string    `json:"name"`
	Status     AppStatus `json:"status"`
	ListingKey string    `json:"listingKey"`
}

func (p App) Validate() error {
	if err := p.ManagedResource.Validate(); err != nil {
		return fmt.Errorf("error validating managed resource: %w", err)
	}

	if p.ID == "" {
		return errors.New("id is required")
	}

	if p.Namespace == "" {
		return errors.New("namespace is required")
	}

	if p.Name == "" {
		return errors.New("name is required")
	}

	if p.Status == "" {
		return errors.New("status is required")
	}

	if p.ListingKey == "" {
		return errors.New("listing key is required")
	}

	return nil
}

type StripeApp struct {
	App

	StripeAccountId string `json:"stripeAccountId"`
	Livemode        bool   `json:"livemode"`
}

func (p StripeApp) Validate() error {
	if p.Type != AppTypeStripe {
		return errors.New("app type must be stripe")
	}

	if p.StripeAccountId == "" {
		return errors.New("stripe account id is required")
	}

	return p.App.Validate()
}

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

type GetAppInput = AppID

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
