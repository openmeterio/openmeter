package fbo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	NewPlan(input NewPlanInput) (*Plan, error)
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

type NewPlanInput struct {
	CustomerID customer.CustomerID
	GroupID    string
}

var _ models.Validator = NewPlanInput{}

func (i NewPlanInput) Validate() error {
	var errs []error
	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	if err := ledger.ValidateAssignedID(i.GroupID); err != nil {
		errs = append(errs, fmt.Errorf("group id: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Plan owns the postings and source reservations for one customer's ledger
// group. Reservations prevent reuse regardless of posting time, but do
// not calculate the minimum balance after the source-selection boundary or
// project pending journal entries. Pending issuance/restoration are not available
// as new sources in this consumption-only plan.
type Plan struct {
	owner      *service
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

func (s *service) NewPlan(input NewPlanInput) (*Plan, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	groupID := input.GroupID
	if groupID == "" {
		groupID = ulid.Make().String()
	}

	return &Plan{
		owner:      s,
		customerID: input.CustomerID,
		groupID:    groupID,
		reserved:   make(map[sourceKey]alpacadecimal.Decimal),
	}, nil
}

func (p *Plan) GroupID() string { return p.groupID }

// Finalize seals the plan and returns every consumption with its companion
// postings. A failed plan requires rollback; individual consumptions cannot
// be discarded.
func (p *Plan) Finalize(annotations models.Annotations, additional ...ledger.TransactionInput) (ledger.TransactionGroupInput, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}

	inputs := append(slices.Clone(p.inputs), additional...)
	p.sealed = true

	return ledger.WithGroupID(transactions.GroupInputs(p.customerID.Namespace, annotations, inputs...), p.groupID), nil
}

func (p *Plan) validate() error {
	if p == nil || p.owner == nil {
		return errors.New("FBO plan is missing")
	}

	if p.failure != nil {
		return fmt.Errorf("FBO plan failed: %w", p.failure)
	}

	if p.sealed {
		return errors.New("FBO plan is finalized")
	}

	return nil
}
