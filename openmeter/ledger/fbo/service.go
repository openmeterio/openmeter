package fbo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	NewScope(ctx context.Context, input ScopeInput) (*Scope, error)
	PlanConsume(ctx context.Context, scope *Scope, input ConsumeInput) (ConsumePlan, error)
	ListSources(ctx context.Context, input SourceQuery) ([]AvailableSource, error)
}

type Config struct {
	Logger        *slog.Logger
	Dependencies  transactions.ResolverDependencies
	Advance       advance.Service
	Breakage      breakage.Service
	AccountLocker ledger.AccountLocker
}

var _ models.Validator = Config{}

func (c Config) Validate() error {
	var errs []error
	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if c.Dependencies.AccountService == nil {
		errs = append(errs, errors.New("account service is required"))
	}

	if c.Dependencies.AccountCatalog == nil {
		errs = append(errs, errors.New("account catalog is required"))
	}

	if c.Dependencies.BalanceQuerier == nil {
		errs = append(errs, errors.New("balance querier is required"))
	}

	if c.Advance == nil {
		errs = append(errs, errors.New("advance service is required"))
	}

	if c.Breakage == nil {
		errs = append(errs, errors.New("breakage service is required"))
	}

	if c.AccountLocker == nil {
		errs = append(errs, errors.New("account locker is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type service struct {
	dependencies  transactions.ResolverDependencies
	advance       advance.Service
	breakage      breakage.Service
	accountLocker ledger.AccountLocker
}

var _ Service = (*service)(nil)

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		dependencies:  config.Dependencies,
		advance:       config.Advance,
		breakage:      config.Breakage,
		accountLocker: config.AccountLocker,
	}, nil
}

type ScopeInput struct {
	CustomerID customer.CustomerID
	GroupID    string
}

var _ models.Validator = ScopeInput{}

func (i ScopeInput) Validate() error {
	var errs []error
	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := ledger.ValidateAssignedID(i.GroupID); err != nil {
		errs = append(errs, fmt.Errorf("group id: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Scope reserves sources across consume plans in one customer transaction and
// ledger group. Reservations apply regardless of posting time; they are not a
// projection of pending journal entries at a historical balance boundary.
type Scope struct {
	owner      *service
	driver     transaction.Driver
	customerID customer.CustomerID
	groupID    string
	reserved   map[sourceKey]alpacadecimal.Decimal
	inputs     []ledger.TransactionInput
	failure    error
	sealed     bool
}

type sourceKey struct {
	subAccountID    string
	sourceChargeID  string
	hasSourceCharge bool
}

func keyForSource(address ledger.PostingAddress, chargeID *string) sourceKey {
	key := sourceKey{subAccountID: address.SubAccountID()}
	if chargeID != nil {
		key.sourceChargeID = *chargeID
		key.hasSourceCharge = true
	}

	return key
}

func (s *service) NewScope(ctx context.Context, input ScopeInput) (*Scope, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	driver, err := transaction.GetDriverFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("FBO planning requires the caller's transaction: %w", err)
	}

	if !reflect.TypeOf(driver).Comparable() {
		return nil, errors.New("FBO planning requires a comparable transaction driver")
	}

	groupID := input.GroupID
	if groupID == "" {
		groupID = ulid.Make().String()
	}

	return &Scope{
		owner:      s,
		driver:     driver,
		customerID: input.CustomerID,
		groupID:    groupID,
		reserved:   make(map[sourceKey]alpacadecimal.Decimal),
	}, nil
}

func (s *Scope) GroupID() string { return s.groupID }

// GroupInput includes every successful consume plan. A failed scope cannot be
// committed by dropping the failing plan; the caller must roll back its transaction.
func (s *Scope) GroupInput(ctx context.Context, annotations models.Annotations, additional ...ledger.TransactionInput) (ledger.TransactionGroupInput, error) {
	if s == nil || s.owner == nil {
		return nil, errors.New("FBO planning scope is missing")
	}

	if err := s.owner.validateScope(ctx, s); err != nil {
		return nil, err
	}

	inputs := append(slices.Clone(s.inputs), additional...)
	s.sealed = true

	return ledger.WithGroupID(transactions.GroupInputs(s.customerID.Namespace, annotations, inputs...), s.groupID), nil
}

func (s *service) validateScope(ctx context.Context, scope *Scope) error {
	if scope == nil || scope.owner != s {
		return errors.New("FBO planning scope belongs to another service or is missing")
	}

	if scope.failure != nil {
		return fmt.Errorf("FBO planning scope failed: %w", scope.failure)
	}

	if scope.sealed {
		return errors.New("FBO planning scope is finalized")
	}

	driver, err := transaction.GetDriverFromContext(ctx)
	if err != nil {
		return fmt.Errorf("FBO planning requires the caller's transaction: %w", err)
	}

	if !reflect.TypeOf(driver).Comparable() || driver != scope.driver {
		return errors.New("FBO planning scope belongs to another transaction")
	}

	return nil
}
