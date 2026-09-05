package collector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Service interface {
	CollectToAccrued(ctx context.Context, input CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error)
	CollectToReceivable(ctx context.Context, input CollectToReceivableInput) (creditrealization.CreateAllocationInputs, error)
	CorrectCollectedAccrued(ctx context.Context, input CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error)
	CorrectCollectedReceivable(ctx context.Context, input CorrectCollectedReceivableInput) (creditrealization.CreateCorrectionInputs, error)
}

type Config struct {
	Ledger        ledger.Ledger
	Dependencies  transactions.ResolverDependencies
	Breakage      breakage.Service
	AccountLocker ledger.AccountLocker
	// TransactionManager wraps the full collection flow so source selection,
	// ledger commit, and follow-up bookkeeping share one DB transaction.
	TransactionManager transaction.Creator
}

func (c Config) Validate() error {
	var errs []error

	if c.Ledger == nil {
		errs = append(errs, fmt.Errorf("ledger is required"))
	}
	if c.Dependencies.AccountService == nil {
		errs = append(errs, fmt.Errorf("account service is required"))
	}
	if c.Dependencies.AccountCatalog == nil {
		errs = append(errs, fmt.Errorf("account catalog is required"))
	}
	if c.Dependencies.BalanceQuerier == nil {
		errs = append(errs, fmt.Errorf("balance querier is required"))
	}
	if c.AccountLocker == nil {
		errs = append(errs, fmt.Errorf("account locker is required"))
	}
	if c.TransactionManager == nil {
		errs = append(errs, fmt.Errorf("transaction manager is required"))
	}

	return errors.Join(errs...)
}

type CollectToAccruedInput struct {
	Namespace         string
	ChargeID          string
	CustomerID        string
	Annotations       models.Annotations
	BookedAt          time.Time
	SourceBalanceAsOf time.Time
	Currency          currencies.CurrencyReference
	FeatureKey        string
	SettlementMode    productcatalog.SettlementMode
	ServicePeriod     timeutil.ClosedPeriod
	Amount            alpacadecimal.Decimal
	TaxCode           *string
	TaxBehavior       *ledger.TaxBehavior
}

type CorrectCollectedAccruedInput struct {
	Namespace                    string
	ChargeID                     string
	CustomerID                   string
	Annotations                  models.Annotations
	AllocateAt                   time.Time
	Corrections                  creditrealization.CorrectionRequest
	LineageSegmentsByRealization lineage.ActiveSegmentsByRealizationID
}

func (i CorrectCollectedAccruedInput) Validate() error {
	var errs []error
	if err := (models.NamespacedID{Namespace: i.Namespace, ID: i.ChargeID}).Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}
	if err := (customer.CustomerID{Namespace: i.Namespace, ID: i.CustomerID}).Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}
	if i.AllocateAt.IsZero() {
		errs = append(errs, errors.New("allocate at is required"))
	}
	seen := make(map[string]bool)
	type allocationSource struct {
		groupID  string
		sortHint int
	}
	seenSources := make(map[allocationSource]bool)
	for idx, correction := range i.Corrections {
		if err := correction.Allocation.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("corrections[%d].allocation: %w", idx, err))
		}
		if correction.Amount.IsPositive() {
			errs = append(errs, fmt.Errorf("corrections[%d]: amount must be non-positive", idx))
		}
		if seen[correction.Allocation.ID] {
			errs = append(errs, errors.New("a correction batch cannot repeat an allocation"))
		}
		seen[correction.Allocation.ID] = true
		source := allocationSource{groupID: correction.Allocation.LedgerTransaction.TransactionGroupID, sortHint: correction.Allocation.SortHint}
		if seenSources[source] {
			errs = append(errs, errors.New("a correction batch cannot repeat an original collection source"))
		}
		seenSources[source] = true
		if correction.Allocation.Namespace != i.Namespace || correction.Allocation.Type != creditrealization.TypeAllocation {
			errs = append(errs, fmt.Errorf("corrections[%d]: allocation must belong to the correction namespace", idx))
		}
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CollectToReceivableInput struct {
	Namespace         string
	ChargeID          string
	CustomerID        string
	Annotations       models.Annotations
	BookedAt          time.Time
	SourceBalanceAsOf time.Time
	Currency          currencies.CurrencyReference
	FeatureKey        string
	ServicePeriod     timeutil.ClosedPeriod
	Amount            alpacadecimal.Decimal
}

func (i CollectToReceivableInput) Validate() error {
	var errs []error

	if err := (models.NamespacedID{Namespace: i.Namespace, ID: i.ChargeID}).Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}
	if err := (customer.CustomerID{Namespace: i.Namespace, ID: i.CustomerID}).Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}
	if i.BookedAt.IsZero() {
		errs = append(errs, errors.New("booked at is required"))
	}
	if i.SourceBalanceAsOf.IsZero() {
		errs = append(errs, errors.New("source balance as of is required"))
	}
	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	} else if !i.Currency.IsFiat() {
		errs = append(errs, errors.New("currency must be fiat"))
	}
	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}
	if i.Amount.IsNegative() {
		errs = append(errs, errors.New("amount cannot be negative"))
	} else if i.Amount.IsPositive() {
		if err := ledger.ValidateTransactionAmount(i.Amount); err != nil {
			errs = append(errs, fmt.Errorf("amount: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CorrectCollectedReceivableInput struct {
	Namespace   string
	ChargeID    string
	CustomerID  string
	Annotations models.Annotations
	AllocateAt  time.Time
	Corrections creditrealization.CorrectionRequest
}

func (i CorrectCollectedReceivableInput) Validate() error {
	var errs []error

	if err := (models.NamespacedID{Namespace: i.Namespace, ID: i.ChargeID}).Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}
	if err := (customer.CustomerID{Namespace: i.Namespace, ID: i.CustomerID}).Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}
	if i.AllocateAt.IsZero() {
		errs = append(errs, errors.New("allocate at is required"))
	}
	for idx, correction := range i.Corrections {
		if err := correction.Allocation.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("corrections[%d].allocation: %w", idx, err))
		}
		if correction.Amount.IsPositive() {
			errs = append(errs, fmt.Errorf("corrections[%d].amount must not be positive", idx))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type service struct {
	collector *accrualCollector
	corrector *accrualCorrector
}

func NewService(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		collector: &accrualCollector{
			ledger:             config.Ledger,
			deps:               config.Dependencies,
			breakage:           config.Breakage,
			accountLocker:      config.AccountLocker,
			transactionManager: config.TransactionManager,
		},
		corrector: &accrualCorrector{
			ledger:             config.Ledger,
			deps:               config.Dependencies,
			breakage:           config.Breakage,
			transactionManager: config.TransactionManager,
		},
	}, nil
}

func (s *service) CollectToAccrued(ctx context.Context, input CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	if input.BookedAt.IsZero() {
		return nil, fmt.Errorf("booked at is required")
	}
	if input.SourceBalanceAsOf.IsZero() {
		return nil, fmt.Errorf("source balance as of is required")
	}

	return s.collector.collect(ctx, input)
}

func (s *service) CollectToReceivable(ctx context.Context, input CollectToReceivableInput) (creditrealization.CreateAllocationInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return s.collector.collectToReceivable(ctx, input)
}

func (s *service) CorrectCollectedAccrued(ctx context.Context, input CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error) {
	return s.corrector.correct(ctx, input)
}

func (s *service) CorrectCollectedReceivable(ctx context.Context, input CorrectCollectedReceivableInput) (creditrealization.CreateCorrectionInputs, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return s.corrector.correct(ctx, CorrectCollectedAccruedInput{
		Namespace:   input.Namespace,
		ChargeID:    input.ChargeID,
		CustomerID:  input.CustomerID,
		Annotations: input.Annotations,
		AllocateAt:  input.AllocateAt,
		Corrections: input.Corrections,
	})
}
