package collector

import (
	"context"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Service interface {
	CollectToAccrued(ctx context.Context, input CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error)
	CorrectCollectedAccrued(ctx context.Context, input CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error)
}

type Config struct {
	Ledger       ledger.Ledger
	Dependencies transactions.ResolverDependencies
}

type CollectToAccruedInput struct {
	Namespace      string
	ChargeID       string
	CustomerID     string
	At             time.Time
	Currency       currencyx.Code
	SettlementMode productcatalog.SettlementMode
	ServicePeriod  timeutil.ClosedPeriod
	Amount         alpacadecimal.Decimal
}

type CorrectCollectedAccruedInput struct {
	Namespace                    string
	ChargeID                     string
	CustomerID                   string
	AllocateAt                   time.Time
	Corrections                  creditrealization.CorrectionRequest
	LineageSegmentsByRealization lineage.ActiveSegmentsByRealizationID
}

type service struct {
	collector *accrualCollector
	corrector *accrualCorrector
}

func NewService(config Config) Service {
	return &service{
		collector: &accrualCollector{
			ledger: config.Ledger,
			deps:   config.Dependencies,
		},
		corrector: &accrualCorrector{
			ledger: config.Ledger,
			deps:   config.Dependencies,
		},
	}
}

func (s *service) CollectToAccrued(ctx context.Context, input CollectToAccruedInput) (creditrealization.CreateAllocationInputs, error) {
	return s.collector.collect(ctx, input)
}

func (s *service) CorrectCollectedAccrued(ctx context.Context, input CorrectCollectedAccruedInput) (creditrealization.CreateCorrectionInputs, error) {
	return s.corrector.correct(ctx, input)
}
