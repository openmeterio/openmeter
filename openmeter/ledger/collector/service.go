package collector

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Service interface {
	CollectToAccrued(ctx context.Context, input CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error)
	CorrectCollectedAccrued(ctx context.Context, input CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error)
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
	Currency          currencyx.Code
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

func (s *service) CorrectCollectedAccrued(ctx context.Context, input CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error) {
	return s.corrector.correct(ctx, input)
}
