package plan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
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
	ListPlans(ctx context.Context, params ListPlansInput) (pagination.Result[Plan], error)
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

	// IncludeDeleted defines whether to include deleted Plans.
	IncludeDeleted bool

	// Status filter
	Status []productcatalog.PlanStatus

	// Currencies is the list of currencies to filter by.
	Currencies []string
}

func (i ListPlansInput) Validate() error {
	return nil
}

type ListPlansStatusFilter struct {
	// Active signals that the active plans should be returned.
	Active bool

	// Draft signals that the draft plans should be returned.
	Draft bool

	// Archived signals that the archived plans should be returned.
	Archived bool
}

type inputOptions struct {
	// ignoreNonCriticalIssues makes Validate() return errors with critical severity or higher.
	// This allows creating resource with expected validation issues.
	IgnoreNonCriticalIssues bool
}

var _ models.Validator = (*CreatePlanInput)(nil)

type CreatePlanInput struct {
	models.NamespacedModel
	productcatalog.Plan

	inputOptions
}

func (i CreatePlanInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, productcatalog.ErrNamespaceEmpty)
	}

	if err := i.Plan.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid plan: %w", err))
	}

	issues, err := models.AsValidationIssues(errors.Join(errs...))
	if err != nil {
		return models.NewGenericValidationError(err)
	}

	if i.IgnoreNonCriticalIssues {
		issues = issues.WithSeverityOrHigher(models.ErrorSeverityCritical)
	}

	return models.NewNillableGenericValidationError(issues.AsError())
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

	// BillingCadence is the default billing cadence for subscriptions using this plan
	BillingCadence *datetime.ISODuration `json:"billingCadence,omitempty"`

	// ProRatingConfig is the default pro-rating configuration for subscriptions using this plan
	ProRatingConfig *productcatalog.ProRatingConfig `json:"proRatingConfig,omitempty"`

	// Phases
	Phases *[]productcatalog.Phase `json:"phases"`

	inputOptions
}

func (i UpdatePlanInput) Equal(p Plan) bool {
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

	if i.Description != nil && lo.FromPtr(i.Description) != lo.FromPtr(p.Description) {
		return false
	}

	if i.Metadata != nil && !i.Metadata.Equal(p.Metadata) {
		return false
	}

	if i.BillingCadence != nil && i.BillingCadence.String() != p.BillingCadence.String() {
		return false
	}

	if i.ProRatingConfig != nil && !i.ProRatingConfig.Equal(p.ProRatingConfig) {
		return false
	}

	if i.Phases != nil {
		if len(*i.Phases) != len(p.Phases) {
			return false
		}

		for idx, phase := range *i.Phases {
			if !phase.Equal(p.Phases[idx].Phase) {
				return false
			}
		}
	}

	return true
}

func (i UpdatePlanInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, productcatalog.ErrNamespaceEmpty)
	}

	if i.ID == "" {
		errs = append(errs, productcatalog.ErrIDEmpty)
	}

	if i.Name != nil && *i.Name == "" {
		errs = append(errs, productcatalog.ErrResourceNameEmpty)
	}

	if i.EffectiveFrom != nil || i.EffectiveTo != nil {
		if err := i.EffectivePeriod.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid effective period: %w", err))
		}
	}

	if i.Phases != nil {
		for _, phase := range *i.Phases {
			if err := phase.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("invalid plan phase: %w", err))
			}
		}
	}

	issues, err := models.AsValidationIssues(errors.Join(errs...))
	if err != nil {
		return models.NewGenericValidationError(err)
	}

	if i.IgnoreNonCriticalIssues {
		issues = issues.WithSeverityOrHigher(models.ErrorSeverityCritical)
	}

	return models.NewNillableGenericValidationError(issues.AsError())
}

func (i UpdatePlanInput) ValidateWithPlan(p productcatalog.Plan) error {
	var errs []error

	if i.Name != nil {
		p.Name = *i.Name
	}

	if i.Description != nil {
		p.Description = i.Description
	}

	if i.Metadata != nil {
		p.Metadata = *i.Metadata
	}

	if i.BillingCadence != nil {
		p.BillingCadence = *i.BillingCadence
	}

	if i.ProRatingConfig != nil {
		p.ProRatingConfig = *i.ProRatingConfig
	}

	if i.Phases != nil {
		p.Phases = *i.Phases
	}

	if err := p.Validate(); err != nil {
		errs = append(errs, err)
	}

	issues, err := models.AsValidationIssues(errors.Join(errs...))
	if err != nil {
		return models.NewGenericValidationError(err)
	}

	if i.IgnoreNonCriticalIssues {
		issues = issues.WithSeverityOrHigher(models.ErrorSeverityCritical)
	}

	return models.NewNillableGenericValidationError(issues.AsError())
}

// ExpandFields defines which fields to expand when returning the Plan.
type ExpandFields struct {
	PlanAddons bool `json:"addons,omitempty"`
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

	Expand ExpandFields `json:"expand,omitempty"`
}

func (i GetPlanInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, productcatalog.ErrNamespaceEmpty)
	}

	if i.ID == "" && i.Key == "" {
		errs = append(errs, errors.New("either plan id or key must be provided"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type DeletePlanInput struct {
	models.NamespacedID
}

func (i DeletePlanInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, productcatalog.ErrNamespaceEmpty)
	}

	if i.ID == "" {
		errs = append(errs, productcatalog.ErrIDEmpty)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type PublishPlanInput struct {
	models.NamespacedID

	// EffectivePeriod
	productcatalog.EffectivePeriod
}

func (i PublishPlanInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, productcatalog.ErrNamespaceEmpty)
	}

	if i.ID == "" {
		errs = append(errs, productcatalog.ErrIDEmpty)
	}

	now := clock.Now()

	from := lo.FromPtr(i.EffectiveFrom)

	if from.IsZero() {
		errs = append(errs, errors.New("invalid EffectiveFrom: must not be empty"))
	}

	if !from.IsZero() && from.Before(now.Add(-timeJitter)) {
		errs = append(errs, errors.New("invalid EffectiveFrom: period start must not be in the past"))
	}

	to := lo.FromPtr(i.EffectiveTo)

	if !to.IsZero() && from.IsZero() {
		errs = append(errs, errors.New("invalid EffectiveFrom: must not be empty if EffectiveTo is also set"))
	}

	if !to.IsZero() && to.Before(now.Add(timeJitter)) {
		errs = append(errs, errors.New("invalid EffectiveTo: period end must not be in the past"))
	}

	if !from.IsZero() && !to.IsZero() && from.After(to) {
		errs = append(errs, errors.New("invalid EffectivePeriod: period start must not be later than period end"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ArchivePlanInput struct {
	// NamespacedID
	models.NamespacedID

	// EffectiveFrom defines the time from the Plan is going to be unpublished.
	EffectiveTo time.Time `json:"effectiveTo,omitempty"`
}

func (i ArchivePlanInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, productcatalog.ErrNamespaceEmpty)
	}

	if i.ID == "" {
		errs = append(errs, productcatalog.ErrIDEmpty)
	}
	if i.EffectiveTo.IsZero() {
		errs = append(errs, errors.New("invalid EffectiveTo: must not be empty"))
	}

	now := clock.Now()

	if i.EffectiveTo.Before(now.Add(-timeJitter)) {
		errs = append(errs, errors.New("invalid EffectiveTo: period end must not be in the past"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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
		errs = append(errs, productcatalog.ErrNamespaceEmpty)
	}

	if i.ID == "" && i.Key == "" {
		errs = append(errs, errors.New("invalid: either ID or Key pair must be provided"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
