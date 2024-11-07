package plan

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

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
	// Plans

	ListPlans(ctx context.Context, params ListPlansInput) (pagination.PagedResponse[Plan], error)
	CreatePlan(ctx context.Context, params CreatePlanInput) (*Plan, error)
	DeletePlan(ctx context.Context, params DeletePlanInput) error
	GetPlan(ctx context.Context, params GetPlanInput) (*Plan, error)
	UpdatePlan(ctx context.Context, params UpdatePlanInput) (*Plan, error)
	PublishPlan(ctx context.Context, params PublishPlanInput) (*Plan, error)
	ArchivePlan(ctx context.Context, params ArchivePlanInput) (*Plan, error)
	NextPlan(ctx context.Context, params NextPlanInput) (*Plan, error)

	// Phases

	ListPhases(ctx context.Context, params ListPhasesInput) (pagination.PagedResponse[Phase], error)
	CreatePhase(ctx context.Context, params CreatePhaseInput) (*Phase, error)
	DeletePhase(ctx context.Context, params DeletePhaseInput) error
	GetPhase(ctx context.Context, params GetPhaseInput) (*Phase, error)
	UpdatePhase(ctx context.Context, params UpdatePhaseInput) (*Phase, error)
}

var _ Validator = (*ListPlansInput)(nil)

type ListPlansInput struct {
	pagination.Page

	OrderBy OrderBy
	Order   sortx.Order

	Namespaces []string

	IDs []string

	Keys []string

	KeyVersions map[string]int

	IncludeDeleted bool
}

func (i ListPlansInput) Validate() error {
	// TODO: implement the rest of the validator

	return nil
}

var _ Validator = (*CreatePlanInput)(nil)

type CreatePlanInput struct {
	models.NamespacedModel

	// Key is the unique key for Plan.
	Key string `json:"key"`

	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`

	// Currency
	Currency currency.Code `json:"currency"`

	// Phases
	Phases []Phase `json:"phases"`
}

func (i CreatePlanInput) Validate() error {
	if err := i.NamespacedModel.Validate(); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	if i.Key == "" {
		return errors.New("plan key is required")
	}

	if i.Name == "" {
		return errors.New("plan name is required")
	}

	if err := i.Currency.Validate(); err != nil {
		return fmt.Errorf("invalid currency code: %w", err)
	}

	for _, phase := range i.Phases {
		if err := phase.Validate(); err != nil {
			return fmt.Errorf("invalid plan phase: %w", err)
		}
	}

	return nil
}

var (
	_ Validator     = (*UpdatePlanInput)(nil)
	_ Equaler[Plan] = (*UpdatePlanInput)(nil)
)

type UpdatePlanInput struct {
	models.NamespacedID

	// EffectivePeriod
	*EffectivePeriod

	// Name
	Name *string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Metadata
	Metadata *map[string]string `json:"metadata,omitempty"`

	// Phases
	Phases *[]Phase `json:"phases"`
}

func (i UpdatePlanInput) StrictEqual(p Plan) bool {
	return i.Equal(p)
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

	if lo.FromPtrOr(i.Description, "") != lo.FromPtrOr(p.Description, "") {
		return false
	}

	if !MetadataEqual(lo.FromPtrOr(i.Metadata, nil), p.Metadata) {
		return false
	}

	return true
}

func (i UpdatePlanInput) Validate() error {
	var errs []error

	if err := i.NamespacedID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid namespace or id: %w", err))
	}

	if i.Name != nil && *i.Name == "" {
		return errors.New("plan name is required")
	}

	if err := i.EffectivePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid effective period: %w", err))
	}

	if i.Phases != nil && len(*i.Phases) > 0 {
		for _, phase := range *i.Phases {
			if err := phase.Validate(); err != nil {
				return fmt.Errorf("invalid plan phase: %w", err)
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

	// AllowLatest defines whether return the latest version regardless of its PlanStatus or with ActiveStatus only if
	// Version is not set.
	IncludeLatest bool `json:"includeLatest,omitempty"`
}

func (i GetPlanInput) Validate() error {
	if err := i.NamespacedID.Validate(); err != nil {
		return err
	}

	if i.ID == "" && i.Key == "" {
		return errors.New("either plan id or key must be set")
	}

	return nil
}

type DeletePlanInput struct {
	models.NamespacedID

	// SkipSoftDelete defines whether plan needs to be permanently or soft deleted.
	SkipSoftDelete bool `json:"skipSoftDelete,omitempty"`
}

func (i DeletePlanInput) Validate() error {
	if err := i.NamespacedID.Validate(); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	return nil
}

type PublishPlanInput struct {
	models.NamespacedID

	// EffectivePeriod
	EffectivePeriod
}

func (i PublishPlanInput) Validate() error {
	if err := i.NamespacedID.Validate(); err != nil {
		return err
	}

	if lo.FromPtrOr(i.EffectiveFrom, time.Time{}).IsZero() {
		return errors.New("effectiveFrom must be set")
	}

	if !lo.FromPtrOr(i.EffectiveTo, time.Time{}).IsZero() && lo.FromPtrOr(i.EffectiveFrom, time.Time{}).IsZero() {
		return errors.New("effectiveFrom must be set if effectiveTo is set")
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
	if err := i.NamespacedID.Validate(); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	if i.EffectiveTo.IsZero() {
		return errors.New("invalid input: effectiveTo must be set")
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
	if err := i.NamespacedID.Validate(); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}

	return nil
}

type ListPhasesInput struct {
	pagination.Page

	OrderBy OrderBy
	Order   sortx.Order

	Namespaces []string

	IDs []string

	Keys []string

	PlanIDs []string

	IncludeDeleted bool
}

type CreatePhaseInput struct {
	models.NamespacedModel

	// Key is the unique key for Phase
	Key string `json:"key"`

	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`

	// StartAfter
	StartAfter datex.Period `json:"interval,omitempty"`

	// PlanID
	PlanID string `json:"planId"`

	// RateCards
	RateCards []RateCard `json:"rateCards,omitempty"`
}

type DeletePhaseInput struct {
	// NamespacedID
	models.NamespacedID

	// Key is the unique key for Phase. Can be used as an alternative way to identify a Phase in Plan
	// without providing/knowing its unique ID. Use it with PlanID in order to identify a Phase in Plan.
	Key string `json:"key"`

	// PlanID identifies the Plan the Phase belongs to. See Key.
	PlanID string `json:"planId"`

	// SkipSoftDelete defines whether plan phase needs to be permanently or soft deleted.
	SkipSoftDelete bool `json:"skipSoftDelete,omitempty"`
}

func (i DeletePhaseInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace must be set"))
	}

	if i.ID == "" && (i.Key == "" || i.PlanID == "") {
		errs = append(errs, errors.New("either phase id or key/planId pair must be set"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type GetPhaseInput struct {
	models.NamespacedID

	// Key is the unique key for Phase. Can be used as an alternative way to identify a Phase in Plan
	// without providing/knowing its unique ID. Use it with PlanID in order to identify a Phase in Plan.
	Key string `json:"key"`

	// PlanID identifies the Plan the Phase belongs to. See Key.
	PlanID string `json:"planId"`
}

func (i GetPhaseInput) Validate() error {
	if err := i.NamespacedID.Validate(); err != nil {
		return err
	}

	if i.ID == "" && (i.Key == "" || i.PlanID == "") {
		return errors.New("either phase id or key/planId pair must be set")
	}

	return nil
}

var (
	_ Validator      = (*UpdatePhaseInput)(nil)
	_ Equaler[Phase] = (*UpdatePhaseInput)(nil)
)

type UpdatePhaseInput struct {
	models.NamespacedID

	// Key is the unique key for Resource.
	Key string `json:"key"`

	// Name
	Name *string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Metadata
	Metadata *map[string]string `json:"metadata,omitempty"`

	// StartAfter
	StartAfter *datex.Period `json:"interval,omitempty"`

	// PlanID
	PlanID string `json:"planId"`

	// RateCards
	RateCards *[]RateCard `json:"rateCards,omitempty"`
}

// StrictEqual implements the Equaler interface.
func (i UpdatePhaseInput) StrictEqual(p Phase) bool {
	return i.Equal(p)
}

// Equal implements the Equaler interface.
func (i UpdatePhaseInput) Equal(p Phase) bool {
	if i.Namespace != p.Namespace {
		return false
	}

	if i.Key != p.Key {
		return false
	}

	if i.Name != nil && *i.Name == p.Name {
		return false
	}

	if len(lo.FromPtrOr(i.Metadata, nil)) != len(p.Metadata) {
		return false
	}

	if i.Metadata != nil && !MetadataEqual(*i.Metadata, p.Metadata) {
		return false
	}

	if i.PlanID != p.PlanID {
		return false
	}

	return true
}

func (i UpdatePhaseInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace must be set"))
	}

	if i.ID == "" && (i.Key == "" || i.PlanID == "") {
		return errors.New("either phase id or key/planId pair must be set")
	}

	if i.Name != nil && *i.Name == "" {
		return errors.New("phase name must not be empty")
	}

	if i.RateCards != nil && len(*i.RateCards) > 0 {
		for _, rateCards := range *i.RateCards {
			if err := rateCards.Validate(); err != nil {
				return fmt.Errorf("invalid RateCard: %w", err)
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type Config struct {
	Feature feature.FeatureConnector

	Adapter Repository
	Logger  *slog.Logger
}

func New(config Config) (Service, error) {
	if config.Feature == nil {
		return nil, errors.New("feature connector is required")
	}

	if config.Adapter == nil {
		return nil, errors.New("plan adapter is required")
	}

	if config.Logger == nil {
		return nil, errors.New("logger is required")
	}

	return &service{
		feature: config.Feature,
		adapter: config.Adapter,
		logger:  config.Logger,
	}, nil
}

var _ Service = (*service)(nil)

type service struct {
	feature feature.FeatureConnector

	adapter Repository

	logger *slog.Logger
}

func (s service) ListPlans(ctx context.Context, params ListPlansInput) (pagination.PagedResponse[Plan], error) {
	// TODO(chrisgacsal): implement operation
	return pagination.PagedResponse[Plan]{}, nil
}

func (s service) CreatePlan(ctx context.Context, params CreatePlanInput) (*Plan, error) {
	// TODO(chrisgacsal): implement operation
	return nil, nil
}

func (s service) DeletePlan(ctx context.Context, params DeletePlanInput) error {
	// TODO(chrisgacsal): implement operation
	return nil
}

func (s service) GetPlan(ctx context.Context, params GetPlanInput) (*Plan, error) {
	// TODO(chrisgacsal): implement operation
	return nil, nil
}

func (s service) UpdatePlan(ctx context.Context, params UpdatePlanInput) (*Plan, error) {
	// TODO(chrisgacsal): implement operation
	return nil, nil
}

func (s service) PublishPlan(ctx context.Context, params PublishPlanInput) (*Plan, error) {
	// TODO(chrisgacsal): implement operation
	return nil, nil
}

func (s service) ArchivePlan(ctx context.Context, params ArchivePlanInput) (*Plan, error) {
	// TODO(chrisgacsal): implement operation
	return nil, nil
}

func (s service) NextPlan(ctx context.Context, params NextPlanInput) (*Plan, error) {
	// TODO(chrisgacsal): implement operation
	return nil, nil
}

func (s service) ListPhases(ctx context.Context, params ListPhasesInput) (pagination.PagedResponse[Phase], error) {
	// TODO(chrisgacsal): implement operation
	return pagination.PagedResponse[Phase]{}, nil
}

func (s service) CreatePhase(ctx context.Context, params CreatePhaseInput) (*Phase, error) {
	// TODO(chrisgacsal): implement operation
	return nil, nil
}

func (s service) DeletePhase(ctx context.Context, params DeletePhaseInput) error {
	// TODO(chrisgacsal): implement operation
	return nil
}

func (s service) GetPhase(ctx context.Context, params GetPhaseInput) (*Phase, error) {
	// TODO(chrisgacsal): implement operation
	return nil, nil
}

func (s service) UpdatePhase(ctx context.Context, params UpdatePhaseInput) (*Phase, error) {
	// TODO(chrisgacsal): implement operation
	return nil, nil
}
