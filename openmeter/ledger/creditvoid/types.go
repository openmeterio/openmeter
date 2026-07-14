package creditvoid

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Record struct {
	ID        models.NamespacedID
	Amount    alpacadecimal.Decimal
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	CustomerID customer.CustomerID
	Currency   currencyx.Code
	VoidedAt   time.Time

	SourceChargeID string

	VoidTransactionGroupID string
	VoidTransactionID      string

	FBOSubAccountID        string
	ReceivableSubAccountID string

	Annotations models.Annotations
}

// pendingVoidRecord is a pre-commit record plan. The committed ledger
// transaction IDs are not known until after CommitGroup returns.
type pendingVoidRecord struct {
	ID         models.NamespacedID
	Amount     alpacadecimal.Decimal
	CustomerID customer.CustomerID
	Currency   currencyx.Code
	VoidedAt   time.Time

	SourceChargeID string

	FBOSubAccountID        string
	ReceivableSubAccountID string
}

func (r pendingVoidRecord) committed(groupID, transactionID string, annotations models.Annotations) Record {
	return Record{
		ID:                     r.ID,
		Amount:                 r.Amount,
		CustomerID:             r.CustomerID,
		Currency:               r.Currency,
		VoidedAt:               r.VoidedAt,
		SourceChargeID:         r.SourceChargeID,
		VoidTransactionGroupID: groupID,
		VoidTransactionID:      transactionID,
		FBOSubAccountID:        r.FBOSubAccountID,
		ReceivableSubAccountID: r.ReceivableSubAccountID,
		Annotations:            annotations,
	}
}

type CreateRecordsInput struct {
	Records []Record
}

func (i CreateRecordsInput) Validate() error {
	for idx, record := range i.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("records[%d]: %w", idx, err)
		}
	}

	return nil
}

func (r Record) Validate() error {
	var errs []error

	if err := r.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("id: %w", err))
	}
	if !r.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}
	if err := r.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer id: %w", err))
	}
	if err := r.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}
	if r.VoidedAt.IsZero() {
		errs = append(errs, errors.New("voided at is required"))
	}
	if r.SourceChargeID == "" {
		errs = append(errs, errors.New("source charge id is required"))
	}
	if r.VoidTransactionGroupID == "" {
		errs = append(errs, errors.New("void transaction group id is required"))
	}
	if r.VoidTransactionID == "" {
		errs = append(errs, errors.New("void transaction id is required"))
	}
	if r.FBOSubAccountID == "" {
		errs = append(errs, errors.New("FBO sub-account id is required"))
	}
	if r.ReceivableSubAccountID == "" {
		errs = append(errs, errors.New("receivable sub-account id is required"))
	}

	return errors.Join(errs...)
}

type ListRecordsInput struct {
	CustomerID customer.CustomerID
	Currency   *currencyx.Code
	AsOf       time.Time
	Route      ledger.RouteFilter
}

type Adapter interface {
	CreateRecords(ctx context.Context, input CreateRecordsInput) error
	ListRecords(ctx context.Context, input ListRecordsInput) ([]Record, error)
}
