package plan

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

	OrderByStartAfter OrderBy = "start_after"
)

type OrderBy string

type Service interface {
	ListPlans(ctx context.Context, params ListPlansInput) (pagination.PagedResponse[Plan], error)
	CreatePlan(ctx context.Context, params CreatePlanInput) (*Plan, error)
	DeletePlan(ctx context.Context, params DeletePlanInput) error
	GetPlan(ctx context.Context, params GetPlanInput) (*Plan, error)
	UpdatePlan(ctx context.Context, params UpdatePlanInput) (*Plan, error)
	PublishPlan(ctx context.Context, params PublishPlanInput) (*Plan, error)
	ArchivePlan(ctx context.Context, params ArchivePlanInput) (*Plan, error)
	NextPlan(ctx context.Context, params NextPlanInput) (*Plan, error)
}

var _ models.Validator = (*ListPlansInput)(nil)

type ListPlansInput struct {
	pagination.Page

	OrderBy OrderBy
	Order   sortx.Order

	Namespaces []string

	IDs []string

	Keys []string

	KeyVersions map[string][]int

	IncludeDeleted bool
}

func (i ListPlansInput) Validate() error {
	// TODO: implement the rest of the validator

	return nil
}

var _ models.Validator = (*CreatePlanInput)(nil)

type CreatePlanInput struct {
	models.NamespacedModel
	productcatalog.Plan
}

func (i CreatePlanInput) Validate() error {
	var errs []error

	if err := i.NamespacedModel.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid Namespace: %w", err))
	}

	if i.Key == "" {
		errs = append(errs, errors.New("invalid Key: must not be empty"))
	}

	if i.Name == "" {
		errs = append(errs, errors.New("invalid Name: must not be empty"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid CurrencyCode: %w", err))
	}

	for _, phase := range i.Phases {
		if err := phase.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid PlanPhase: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

var (
	_ models.Validator     = (*UpdatePlanInput)(nil)
	_ models.Equaler[Plan] = (*UpdatePlanInput)(nil)
)

type UpdatePlanInput struct {
	models.NamespacedID

	// EffectivePeriod
	productcatalog.EffectivePeriod

	// Name
	Name *string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Metadata
	Metadata *models.Metadata `json:"metadata,omitempty"`

	// Phases
	Phases *[]productcatalog.Phase `json:"phases"`
}

func (i UpdatePlanInput) Equal(p Plan) bool {
	if i.Namespace != p.Namespace {
		return false
	}

	if i.ID != p.ID {
		return false
	}

	if i.EffectivePeriod.Status() != p.EffectivePeriod.Status() {
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

	return true
}

func (i UpdatePlanInput) Validate() error {
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

	if i.Phases != nil && len(*i.Phases) > 0 {
		for _, phase := range *i.Phases {
			if err := phase.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("invalid PlanPhase: %w", err))
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type GetPlanInput struct {
	models.NamespacedID

	// Key is the unique key for Plan.
	Key string `json:"key,omitempty"`

	// Version is the version of the Plan.
	// If not set the latest version is assumed.
	Version int `json:"version,omitempty"`

	// IncludeLatest defines whether return the latest version regardless of its PlanStatus or with ActiveStatus only if
	// Version is not set.
	IncludeLatest bool `json:"includeLatest,omitempty"`
}

func (i GetPlanInput) Validate() error {
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

type DeletePlanInput struct {
	models.NamespacedID
}

func (i DeletePlanInput) Validate() error {
	if err := i.NamespacedID.Validate(); err != nil {
		return fmt.Errorf("invalid Namespace: %w", err)
	}

	return nil
}

type PublishPlanInput struct {
	models.NamespacedID

	// EffectivePeriod
	productcatalog.EffectivePeriod
}

func (i PublishPlanInput) Validate() error {
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

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type ArchivePlanInput struct {
	// NamespacedID
	models.NamespacedID

	// EffectiveFrom defines the time from the Plan is going to be unpublished.
	EffectiveTo time.Time `json:"effectiveTo,omitempty"`
}

func (i ArchivePlanInput) Validate() error {
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

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type NextPlanInput struct {
	// NamespacedID
	models.NamespacedID

	// Key is the unique key for Plan.
	Key string `json:"key,omitempty"`

	// Version is the version of the Plan.
	// If not set the latest version is assumed.
	Version int `json:"version,omitempty"`
}

func (i NextPlanInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("invalid Namespace: must not be empty"))
	}

	if i.ID == "" && i.Key == "" {
		errs = append(errs, errors.New("invalid: either ID or Key pair must be provided"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
