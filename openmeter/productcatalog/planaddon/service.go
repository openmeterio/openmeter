package planaddon

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

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
	OrderByCreatedAt OrderBy = "created_at"
	OrderByUpdatedAt OrderBy = "updated_at"
)

type OrderBy string

type Service interface {
	ListPlanAddons(ctx context.Context, params ListPlanAddonsInput) (pagination.Result[PlanAddon], error)
	CreatePlanAddon(ctx context.Context, params CreatePlanAddonInput) (*PlanAddon, error)
	DeletePlanAddon(ctx context.Context, params DeletePlanAddonInput) error
	GetPlanAddon(ctx context.Context, params GetPlanAddonInput) (*PlanAddon, error)
	UpdatePlanAddon(ctx context.Context, params UpdatePlanAddonInput) (*PlanAddon, error)
}

var _ models.Validator = (*ListPlanAddonsInput)(nil)

type ListPlanAddonsInput struct {
	// Page is the pagination parameters.
	pagination.Page

	// OrderBy is the field to order by.
	OrderBy OrderBy

	// Order is the order direction.
	Order sortx.Order

	// Namespaces is the list of namespaces to filter by.
	Namespaces []string

	// IDs is the list of PlanAddonAssignment ids to filter by.
	IDs []string

	// PlanIDs is the list of plan.Plan ids to filter by.
	PlanIDs []string

	// PlanKeys is the list of plan.Plan keys to filter by.
	PlanKeys []string

	// PlanKeyVersions is the map of plan.Plan versioned keys to filter by.
	PlanKeyVersions map[string][]int

	// AddonIDs is the list of addon.Addon ids to filter by.
	AddonIDs []string

	// AddonKeys is the list of addon.Addon keys to filter by.
	AddonKeys []string

	// AddonKeyVersions is the map of addon.Addon versioned keys to filter by.
	AddonKeyVersions map[string][]int

	// IncludeDeleted defines whether to include deleted PlanAddonAssignments.
	IncludeDeleted bool

	// Currencies is the list of currencies to filter by.
	Currencies []string
}

func (i ListPlanAddonsInput) Validate() error {
	return nil
}

var _ models.Validator = (*CreatePlanAddonInput)(nil)

type CreatePlanAddonInput struct {
	models.NamespacedModel

	// Metadata
	Metadata models.Metadata `json:"metadata,omitempty"`

	// Annotations
	Annotations models.Annotations `json:"annotations,omitempty"`

	// PlanID
	PlanID string `json:"planId"`

	// AddonID
	AddonID string `json:"addonId"`

	// FromPhase
	FromPlanPhase string `json:"fromPlanPhase"`

	// MaxQuantity
	MaxQuantity *int `json:"maxQuantity"`
}

func (i CreatePlanAddonInput) Validate() error {
	var errs []error

	if err := i.NamespacedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid namespace: %w", err))
	}

	if i.PlanID == "" {
		errs = append(errs, errors.New("invalid add-on assignment: plan id must be provided"))
	}

	if i.AddonID == "" {
		errs = append(errs, errors.New("invalid add-on assignment: add-on id must be provided"))
	}

	return errors.Join(errs...)
}

var (
	_ models.Validator          = (*UpdatePlanAddonInput)(nil)
	_ models.Equaler[PlanAddon] = (*UpdatePlanAddonInput)(nil)
)

type UpdatePlanAddonInput struct {
	models.NamespacedModel

	// Annotations
	Annotations *models.Annotations `json:"annotations,omitempty"`

	// Annotations
	Metadata *models.Metadata `json:"metadata,omitempty"`

	// ID defines the plan add-on assignment ID
	ID string `json:"id"`

	// PlanID
	PlanID string `json:"planId"`

	// AddonID
	AddonID string `json:"addonId"`

	// FromPhase
	FromPlanPhase *string `json:"fromPlanPhase"`

	// MaxQuantity
	MaxQuantity *int `json:"maxQuantity"`
}

func (i UpdatePlanAddonInput) Equal(p PlanAddon) bool {
	if i.Namespace != p.Namespace {
		return false
	}

	if i.PlanID != p.Plan.ID {
		return false
	}

	if i.AddonID != p.Addon.ID {
		return false
	}

	if i.FromPlanPhase != nil && *i.FromPlanPhase != p.FromPlanPhase {
		return false
	}

	if i.MaxQuantity == nil && p.MaxQuantity != nil {
		return false
	}

	if i.MaxQuantity != nil && p.MaxQuantity == nil {
		return false
	}

	if lo.FromPtr(i.MaxQuantity) != lo.FromPtr(p.MaxQuantity) {
		return false
	}

	// FIXME: annotations

	if i.Metadata != nil {
		if !i.Metadata.Equal(p.Metadata) {
			return false
		}
	}

	return true
}

func (i UpdatePlanAddonInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("invalid Namespace: must not be empty"))
	}

	if i.ID == "" {
		if i.PlanID == "" {
			errs = append(errs, errors.New("plan id must be provided if assignment id is not provided"))
		}

		if i.AddonID == "" {
			errs = append(errs, errors.New("add-on id must be provided if assignment id is not provided"))
		}
	}

	return errors.Join(errs...)
}

// GetPlanAddonInput defines the input parameters for fetching plan add-on assignment either by PlanAddon.ID or
// by the plan and add-on identifiers.
type GetPlanAddonInput struct {
	models.NamespacedModel

	// ID defines the plan add-on assignment ID
	ID string `json:"id"`

	// PlanIDOrKey
	PlanIDOrKey string `json:"planIdOrKey"`

	// AddonIDOrKey
	AddonIDOrKey string `json:"addonIdOrKey"`
}

func (i GetPlanAddonInput) Validate() error {
	var errs []error

	if err := i.NamespacedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid namespace: %w", err))
	}

	if i.ID == "" {
		if i.PlanIDOrKey == "" {
			errs = append(errs, errors.New("plan id or key must be provided if assignment id is not provided"))
		}

		if i.AddonIDOrKey == "" {
			errs = append(errs, errors.New("add-on id or key must be provided if assignment id is not provided"))
		}
	}

	return errors.Join(errs...)
}

type DeletePlanAddonInput struct {
	models.NamespacedModel

	ID string `json:"id"`

	PlanID string `json:"planID"`

	AddonID string `json:"addonID"`
}

func (i DeletePlanAddonInput) Validate() error {
	var errs []error

	if err := i.NamespacedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid namespace: %w", err))
	}

	if i.ID == "" {
		if i.PlanID == "" {
			errs = append(errs, errors.New("plan id must be provided if assignment id is not provided"))
		}

		if i.AddonID == "" {
			errs = append(errs, errors.New("add-on id must be provided if assignment id is not provided"))
		}
	}

	return errors.Join(errs...)
}
