package collector

import (
	"context"
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
	Ledger       ledger.Ledger
	Dependencies transactions.ResolverDependencies
	Breakage     breakage.Service
	// TransactionManager wraps the full collection flow so source selection,
	// ledger commit, and follow-up bookkeeping share one DB transaction.
	TransactionManager transaction.Creator
}

type CollectToAccruedInput struct {
	Namespace         string
	ChargeID          string
	CustomerID        string
	Annotations       models.Annotations
	BookedAt          time.Time
	SourceBalanceAsOf time.Time
	Currency          currencyx.Code
	SettlementMode    productcatalog.SettlementMode
	ServicePeriod     timeutil.ClosedPeriod
	Amount            alpacadecimal.Decimal
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

func NewService(config Config) Service {
	if config.TransactionManager == nil {
		panic("collector transaction manager is required")
	}

	return &service{
		collector: &accrualCollector{
			ledger:             config.Ledger,
			deps:               config.Dependencies,
			breakage:           config.Breakage,
			transactionManager: config.TransactionManager,
		},
		corrector: &accrualCorrector{
			ledger:             config.Ledger,
			deps:               config.Dependencies,
			breakage:           config.Breakage,
			transactionManager: config.TransactionManager,
		},
	}
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
