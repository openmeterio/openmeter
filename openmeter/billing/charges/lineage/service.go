package lineage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Service interface {
	CreateInitialLineages(ctx context.Context, input CreateInitialLineagesInput) error
	LoadActiveSegmentsByRealizationID(ctx context.Context, namespace string, realizationIDs []string) (ActiveSegmentsByRealizationID, error)
	LoadLineagesByCustomer(ctx context.Context, input LoadLineagesByCustomerInput) ([]Lineage, error)
	PersistCorrectionLineageSegments(ctx context.Context, input PersistCorrectionLineageSegmentsInput) error
	BackfillAdvanceLineageSegments(ctx context.Context, input BackfillAdvanceLineageSegmentsInput) error
	CloseSegment(ctx context.Context, segmentID string, closedAt time.Time) error
	CreateSegment(ctx context.Context, input CreateSegmentInput) error
}

type Adapter interface {
	entutils.TxCreator

	CreateLineages(ctx context.Context, input CreateLineagesInput) error
	LoadActiveSegmentsByRealizationID(ctx context.Context, namespace string, realizationIDs []string) (ActiveSegmentsByRealizationID, error)
	LoadLineagesByCustomer(ctx context.Context, input LoadLineagesByCustomerInput) ([]Lineage, error)
	LockCorrectionLineages(ctx context.Context, namespace string, realizationIDs []string) ([]Lineage, error)
	LockAdvanceLineagesForBackfill(ctx context.Context, namespace string, customerID string, currency currencyx.Code) ([]Lineage, error)
	ListActiveSegments(ctx context.Context, input ListActiveSegmentsInput) ([]Segment, error)
	CloseSegment(ctx context.Context, segmentID string, closedAt time.Time) error
	CreateSegment(ctx context.Context, input CreateSegmentInput) error
}

type CreateInitialLineagesInput struct {
	Namespace    string
	ChargeID     string
	CustomerID   string
	Currency     currencyx.Code
	Realizations creditrealization.Realizations
}

func (i CreateInitialLineagesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge id is required"))
	}
	if i.CustomerID == "" {
		errs = append(errs, errors.New("customer id is required"))
	}
	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}
	if err := i.Realizations.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("realizations: %w", err))
	}

	return errors.Join(errs...)
}

type PersistCorrectionLineageSegmentsInput struct {
	Namespace    string
	Realizations creditrealization.Realizations
}

func (i PersistCorrectionLineageSegmentsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for idx, realization := range i.Realizations {
		if realization.Type != creditrealization.TypeCorrection {
			continue
		}

		if realization.CorrectsRealizationID == nil || *realization.CorrectsRealizationID == "" {
			errs = append(errs, fmt.Errorf("realizations[%d]: corrects realization id is required for corrections", idx))
		}
	}

	return errors.Join(errs...)
}

type BackfillAdvanceLineageSegmentsInput struct {
	Namespace                 string
	CustomerID                string
	Currency                  currencyx.Code
	Amount                    alpacadecimal.Decimal
	BackingTransactionGroupID string
}

func (i BackfillAdvanceLineageSegmentsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if i.CustomerID == "" {
		errs = append(errs, errors.New("customer id is required"))
	}
	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}
	if !i.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}
	if i.BackingTransactionGroupID == "" {
		errs = append(errs, errors.New("backing transaction group id is required"))
	}

	return errors.Join(errs...)
}

type LoadLineagesByCustomerInput struct {
	Namespace  string
	CustomerID string
	Currency   currencyx.Code
}

func (i LoadLineagesByCustomerInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}
	if i.CustomerID == "" {
		errs = append(errs, errors.New("customer id is required"))
	}
	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	return errors.Join(errs...)
}

type CreateLineagesInput struct {
	Namespace  string
	ChargeID   string
	CustomerID string
	Currency   currencyx.Code
	Specs      []creditrealization.InitialLineageSpec
}

type ListActiveSegmentsInput struct {
	LineageIDs []string
	State      *creditrealization.LineageSegmentState
}

type CreateSegmentInput struct {
	LineageID                       string
	Amount                          alpacadecimal.Decimal
	State                           creditrealization.LineageSegmentState
	BackingTransactionGroupID       *string
	SourceState                     *creditrealization.LineageSegmentState
	SourceBackingTransactionGroupID *string
}

func (i CreateSegmentInput) Validate() error {
	var errs []error

	if i.LineageID == "" {
		errs = append(errs, errors.New("lineage id is required"))
	}
	if !i.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}
	if err := i.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	switch i.State {
	case creditrealization.LineageSegmentStateAdvanceBackfilled,
		creditrealization.LineageSegmentStateEarningsRecognized:
		if i.BackingTransactionGroupID == nil || *i.BackingTransactionGroupID == "" {
			errs = append(errs, fmt.Errorf("backing transaction group id is required for %s segments", i.State))
		}
	default:
		if i.BackingTransactionGroupID != nil && *i.BackingTransactionGroupID == "" {
			errs = append(errs, errors.New("backing transaction group id must not be empty when provided"))
		}
	}
	switch i.State {
	case creditrealization.LineageSegmentStateEarningsRecognized:
		if i.SourceState == nil {
			errs = append(errs, errors.New("source state is required for earnings_recognized segments"))
		} else {
			if err := i.SourceState.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("source state: %w", err))
			}
			if *i.SourceState == creditrealization.LineageSegmentStateEarningsRecognized {
				errs = append(errs, errors.New("source state cannot be earnings_recognized"))
			}
			if *i.SourceState == creditrealization.LineageSegmentStateAdvanceBackfilled &&
				(i.SourceBackingTransactionGroupID == nil || *i.SourceBackingTransactionGroupID == "") {
				errs = append(errs, errors.New("source backing transaction group id is required when source state is advance_backfilled"))
			}
		}
	default:
		if i.SourceState != nil {
			errs = append(errs, errors.New("source state is only valid for earnings_recognized segments"))
		}
		if i.SourceBackingTransactionGroupID != nil && *i.SourceBackingTransactionGroupID == "" {
			errs = append(errs, errors.New("source backing transaction group id must not be empty when provided"))
		}
	}

	return errors.Join(errs...)
}

type Lineage struct {
	ID                string
	ChargeID          string
	RootRealizationID string
	CustomerID        string
	Currency          currencyx.Code
	OriginKind        creditrealization.LineageOriginKind
	Segments          []Segment
}

type Segment struct {
	ID                              string
	LineageID                       string
	Amount                          alpacadecimal.Decimal
	State                           creditrealization.LineageSegmentState
	BackingTransactionGroupID       *string
	SourceState                     *creditrealization.LineageSegmentState
	SourceBackingTransactionGroupID *string
}

type ActiveSegmentsByRealizationID map[string][]Segment
