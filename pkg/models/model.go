package models

import "time"

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

type ManagedModel struct {
	CreatedAt time.Time `json:"createdAt"`
	// After creation the entity is considered updated.
	UpdatedAt time.Time `json:"updatedAt"`
	// Time of soft delete. If not null, the entity is considered deleted.
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

type NamespacedModel struct {
	Namespace string `json:"-" yaml:"-"`
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

// Three-letter [ISO4217](https://www.iso.org/iso-4217-currency-codes.html) currency code.
type CurrencyCode string

// [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.
type CountryCode string
