package models

import (
	"time"

	"github.com/invopop/gobl/l10n"
)

type ManagedResource struct {
	NamespacedModel
	ManagedModel

	// ID is the unique identifier for Resource.
	ID string `json:"id"`
	// Key is the unique key for Resource.
	Key string `json:"key"`
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
	Country     *CountryCode `json:"country"`
	PostalCode  *string      `json:"postalCode"`
	State       *string      `json:"state"`
	City        *string      `json:"city"`
	Line1       *string      `json:"line1"`
	Line2       *string      `json:"line2"`
	PhoneNumber *string      `json:"phoneNumber"`
}

// [ISO 3166-1](https://www.iso.org/iso-3166-country-codes.html) alpha-2 country code.
type CountryCode = l10n.ISOCountryCode
