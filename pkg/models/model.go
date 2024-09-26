package models

import (
	"errors"
	"fmt"
	"time"
)

type ManagedUniqueResource struct {
	NamespacedModel
	ManagedModel

	// ID is the unique identifier for Resource.
	ID string `json:"id"`

	// Key is the unique key for Resource.
	Key string `json:"key"`
}

type ManagedResource struct {
	NamespacedModel
	ManagedModel

	// ID is the unique identifier for Resource.
	ID string `json:"id"`
}

func (r ManagedResource) Validate() error {
	if err := r.NamespacedModel.Validate(); err != nil {
		return fmt.Errorf("error validating namespaced model: %w", err)
	}

	if err := r.ManagedModel.Validate(); err != nil {
		return fmt.Errorf("error validating managed model: %w", err)
	}

	if r.ID == "" {
		return errors.New("id is required")
	}

	return nil
}

type ManagedModel struct {
	CreatedAt time.Time `json:"createdAt"`
	// After creation the entity is considered updated.
	UpdatedAt time.Time `json:"updatedAt"`
	// Time of soft delete. If not null, the entity is considered deleted.
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

func (m ManagedModel) Validate() error {
	if m.CreatedAt.IsZero() {
		return errors.New("created at is required")
	}

	if m.UpdatedAt.IsZero() {
		return errors.New("updated at is required")
	}

	return nil
}

type NamespacedModel struct {
	Namespace string `json:"-" yaml:"-"`
}

func (m NamespacedModel) Validate() error {
	if m.Namespace == "" {
		return errors.New("namespace is required")
	}

	return nil
}

type Address struct {
	Country     *CountryCode `json:"country,omitempty"`
	PostalCode  *string      `json:"postalCode,omitempty"`
	State       *string      `json:"state,omitempty"`
	City        *string      `json:"city,omitempty"`
	Line1       *string      `json:"line1,omitempty"`
	Line2       *string      `json:"line2,omitempty"`
	PhoneNumber *string      `json:"phoneNumber,omitempty"`
}

// [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.
type CountryCode string
