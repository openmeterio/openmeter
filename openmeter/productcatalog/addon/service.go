package addon

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const timeJitter = 30 * time.Second

const (
	OrderAsc  = sortx.OrderAsc
	OrderDesc = sortx.OrderDesc
)

const (
	OrderByID        OrderBy = "id"
	OrderByKey       OrderBy = "key"
	OrderByVersion   OrderBy = "version"
	OrderByCreatedAt OrderBy = "created_at"
	OrderByUpdatedAt OrderBy = "updated_at"
)

type OrderBy string

type Service interface {
	ListAddons(ctx context.Context, params ListAddonsInput) (pagination.PagedResponse[Addon], error)
	CreateAddon(ctx context.Context, params CreateAddonInput) (*Addon, error)
	DeleteAddon(ctx context.Context, params DeleteAddonInput) error
	GetAddon(ctx context.Context, params GetAddonInput) (*Addon, error)
	UpdateAddon(ctx context.Context, params UpdateAddonInput) (*Addon, error)
	PublishAddon(ctx context.Context, params PublishAddonInput) (*Addon, error)
	ArchiveAddon(ctx context.Context, params ArchiveAddonInput) (*Addon, error)
	NextAddon(ctx context.Context, params NextAddonInput) (*Addon, error)
}

var _ models.Validator = (*ListAddonsInput)(nil)

type ListAddonsInput struct {
	// Page is the pagination parameters.
	// TODO: make it optional.
	pagination.Page

	// OrderBy is the field to order by.
	OrderBy OrderBy

	// Order is the order direction.
	Order sortx.Order

	// Namespaces is the list of namespaces to filter by.
	Namespaces []string

	// IDs is the list of IDs to filter by.
	IDs []string

	// Keys is the list of keys to filter by.
	Keys []string

	// KeyVersions is the map of keys to versions to filter by.
	KeyVersions map[string][]int

	// IncludeDeleted defines whether to include deleted Addons.
	IncludeDeleted bool

	// Status filter
	Status []productcatalog.AddonStatus

	// Currencies is the list of currencies to filter by.
	Currencies []string
}

func (i ListAddonsInput) Validate() error {
	return nil
}

type ListAddonsStatusFilter struct {
	// Active signals that the active Addons should be returned.
	Active bool

	// Draft signals that the draft Addons should be returned.
	Draft bool

	// Archived signals that the archived Addons should be returned.
	Archived bool
}

var _ models.Validator = (*CreateAddonInput)(nil)

type CreateAddonInput struct {
	models.NamespacedModel
	productcatalog.Addon
}

func (i CreateAddonInput) Validate() error {
	var errs []error

	if err := i.NamespacedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid namespace: %w", err))
	}

	if err := i.Addon.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid add-on: %w", err))
	}

	return errors.Join(errs...)
}

var (
	_ models.Validator      = (*UpdateAddonInput)(nil)
	_ models.Equaler[Addon] = (*UpdateAddonInput)(nil)
)

type UpdateAddonInput struct {
	models.NamespacedID

	// EffectivePeriod
	productcatalog.EffectivePeriod

	// Name
	Name *string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Metadata
	Metadata *models.Metadata `json:"metadata,omitempty"`

	// Metadata
	Annotations *models.Annotations `json:"annotations,omitempty"`

	// RateCards
	RateCards *productcatalog.RateCards `json:"rateCards,omitempty"`
}

func (i UpdateAddonInput) Equal(p Addon) bool {
	if i.Namespace != p.Namespace {
		return false
	}

	if i.ID != p.ID {
		return false
	}

	if !i.EffectivePeriod.Equal(p.EffectivePeriod) {
		return false
	}

	if i.Name != nil && *i.Name != p.Name {
		return false
	}

	if i.Description != nil && lo.FromPtrOr(i.Description, "") != lo.FromPtrOr(p.Description, "") {
		return false
	}

	if i.Metadata != nil && !i.Metadata.Equal(p.Metadata) {
		return false
	}

	if i.RateCards != nil && !i.RateCards.Equal(p.RateCards) {
		return false
	}

	return true
}

func (i UpdateAddonInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("invalid Namespace: must not be empty"))
	}

	if i.Name != nil && *i.Name == "" {
		return errors.New("invalid Name: must not be empty")
	}

	if i.EffectiveFrom != nil || i.EffectiveTo != nil {
		if err := i.EffectivePeriod.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid EffectivePeriod: %w", err))
		}
	}

	// FIXME: validate ratecards

	return errors.Join(errs...)
}

type GetAddonInput struct {
	models.NamespacedID

	// Key is the unique key for Addon.
	Key string `json:"key,omitempty"`

	// Version is the version of the Addon.
	// If not set the latest version is assumed.
	Version int `json:"version,omitempty"`

	// IncludeLatest defines whether return the latest version regardless of its AddonStatus or with ActiveStatus only if
	// Version is not set.
	IncludeLatest bool `json:"includeLatest,omitempty"`
}

func (i GetAddonInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("invalid Namespace: must not be empty"))
	}

	if i.ID == "" && i.Key == "" {
		errs = append(errs, errors.New("either ID or Key must be provided"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type DeleteAddonInput struct {
	models.NamespacedID
}

func (i DeleteAddonInput) Validate() error {
	var errs []error

	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid Namespace: %w", err))
	}

	return errors.Join(errs...)
}

type PublishAddonInput struct {
	models.NamespacedID

	// AddonEffectivePeriod
	productcatalog.EffectivePeriod
}

func (i PublishAddonInput) Validate() error {
	var errs []error

	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	now := clock.Now()

	from := lo.FromPtrOr(i.EffectiveFrom, time.Time{})

	if from.IsZero() {
		errs = append(errs, errors.New("invalid EffectiveFrom: must not be empty"))
	}

	if !from.IsZero() && from.Before(now.Add(-timeJitter)) {
		errs = append(errs, errors.New("invalid EffectiveFrom: period start must not be in the past"))
	}

	to := lo.FromPtrOr(i.EffectiveTo, time.Time{})

	if !to.IsZero() && from.IsZero() {
		errs = append(errs, errors.New("invalid EffectiveFrom: must not be empty if EffectiveTo is also set"))
	}

	if !to.IsZero() && to.Before(now.Add(timeJitter)) {
		errs = append(errs, errors.New("invalid EffectiveTo: period end must not be in the past"))
	}

	if !from.IsZero() && !to.IsZero() && from.After(to) {
		errs = append(errs, errors.New("invalid EffectivePeriod: period start must not be later than period end"))
	}

	return errors.Join(errs...)
}

type ArchiveAddonInput struct {
	// NamespacedID
	models.NamespacedID

	// EffectiveFrom defines the time from the Addon is going to be unpublished.
	EffectiveTo time.Time `json:"effectiveTo,omitempty"`
}

func (i ArchiveAddonInput) Validate() error {
	var errs []error

	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.EffectiveTo.IsZero() {
		errs = append(errs, errors.New("invalid EffectiveTo: must not be empty"))
	}

	now := clock.Now()

	if i.EffectiveTo.Before(now.Add(-timeJitter)) {
		errs = append(errs, errors.New("invalid EffectiveTo: period end must not be in the past"))
	}

	return errors.Join(errs...)
}

type NextAddonInput struct {
	// NamespacedID
	models.NamespacedID

	// Key is the unique key for Addon.
	Key string `json:"key,omitempty"`

	// Version is the version of the Addon.
	// If not set the latest version is assumed.
	Version int `json:"version,omitempty"`
}

func (i NextAddonInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("invalid Namespace: must not be empty"))
	}

	if i.ID == "" && i.Key == "" {
		errs = append(errs, errors.New("invalid: either ID or Key pair must be provided"))
	}

	return errors.Join(errs...)
}
