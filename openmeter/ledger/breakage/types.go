package breakage

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

type SourceKind = ledger.BreakageSourceKind

const (
	SourceKindCreditPurchase  SourceKind = ledger.BreakageSourceKindCreditPurchase
	SourceKindUsage           SourceKind = ledger.BreakageSourceKindUsage
	SourceKindUsageCorrection SourceKind = ledger.BreakageSourceKindUsageCorrection
	// SourceKindCreditPurchaseCorrection is reserved for a future credit-purchase
	// correction flow. The breakage primitive is clear, but the charge domain does
	// not yet define correction/delete semantics, source-specific removable amount
	// checks, or policy for already-consumed purchased credit.
	SourceKindCreditPurchaseCorrection SourceKind = ledger.BreakageSourceKindCreditPurchaseCorrection
	SourceKindAdvanceBackfill          SourceKind = ledger.BreakageSourceKindAdvanceBackfill
)

// Record is the durable bookkeeping row for one breakage ledger transaction.
//
// It intentionally repeats the fields the allocator needs to find and lock open
// plans without joining back through ledger transactions or parsing annotations.
// The ledger entry remains the accounting source of truth; the record is the
// allocation/index layer that ties plan, release, and reopen transactions
// together.
type Record struct {
	ID        models.NamespacedID
	Kind      ledger.BreakageKind
	Amount    alpacadecimal.Decimal
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	CustomerID customer.CustomerID
	Currency   currencyx.Code

	CreditPriority int
	ExpiresAt      time.Time

	SourceKind               SourceKind
	SourceTransactionGroupID *string
	SourceTransactionID      *string
	SourceEntryID            *string

	BreakageTransactionGroupID string
	BreakageTransactionID      string

	FBOSubAccountID      string
	BreakageSubAccountID string

	PlanID    *string
	ReleaseID *string

	Annotations models.Annotations
}

// Plan is a planned breakage row with its current unreleased amount and the
// posting addresses needed to book a release. It is calculated from plan rows
// minus releases plus reopens.
type Plan struct {
	Record

	OpenAmount      alpacadecimal.Decimal
	FBOAddress      ledger.PostingAddress
	BreakageAddress ledger.PostingAddress
}

// Release is a breakage release row with the amount that can still be reopened
// by corrections.
type Release struct {
	Record

	OpenAmount      alpacadecimal.Decimal
	FBOAddress      ledger.PostingAddress
	BreakageAddress ledger.PostingAddress
}

// PendingRecord is a record row that has been planned before the ledger commit.
// PersistCommittedRecords fills in committed ledger ids after CommitGroup
// succeeds. SourceEntryIdentityKey is transient; it lets usage releases attach
// to the committed FBO source entry without knowing the entry id before commit.
type PendingRecord struct {
	Record

	SourceEntryIdentityKey string
}

// ListPlansInput selects expiring credit that can still produce breakage as of
// the collector's AsOf time. Plans expiring at or before AsOf are not
// candidates because those credits are already expired from the collector's
// perspective.
type ListPlansInput struct {
	CustomerID customer.CustomerID
	Currency   currencyx.Code
	AsOf       time.Time
}

// ListReleasesInput selects usage release rows that may need to be reopened by
// a correction of the original FBO collection source entries or by unwinding an
// advance backfill.
type ListReleasesInput struct {
	CustomerID               customer.CustomerID
	SourceEntryID            []string
	SourceTransactionGroupID []string
	ReleaseSourceKind        []SourceKind
}

// ListExpiredRecordsInput selects breakage rows that have reached their
// expiration timestamp and can be presented as customer-visible expired credit.
type ListExpiredRecordsInput struct {
	CustomerID customer.CustomerID
	Currency   *currencyx.Code
	AsOf       time.Time
}

// ListExpiredBreakageImpactsInput selects customer-visible breakage impact rows.
// Impacts are derived by netting expired plan/release/reopen records by expiry.
type ListExpiredBreakageImpactsInput struct {
	CustomerID customer.CustomerID
	Currency   *currencyx.Code
	AsOf       time.Time
	After      *ledger.TransactionCursor
	Before     *ledger.TransactionCursor
	Limit      int
}

// ListExpiredBreakageImpactsResult contains breakage impacts ordered by
// ledger cursor descending, matching customer transaction listing semantics.
type ListExpiredBreakageImpactsResult struct {
	Items   []BreakageImpact
	HasMore bool
}

// BreakageImpact is the customer-facing effect of breakage that has reached its
// expiration timestamp. Amount is negative because expiry reduces FBO balance.
type BreakageImpact struct {
	ID          models.NamespacedID
	CreatedAt   time.Time
	BookedAt    time.Time
	CustomerID  customer.CustomerID
	Currency    currencyx.Code
	Amount      alpacadecimal.Decimal
	Annotations models.Annotations
}

func (i BreakageImpact) Cursor() ledger.TransactionCursor {
	return ledger.TransactionCursor{
		BookedAt:  i.BookedAt,
		CreatedAt: i.CreatedAt,
		ID:        i.ID,
	}
}

type ListBreakageTransactionCursorsInput struct {
	Namespace     string
	TransactionID []string
}

// CreateRecordsInput persists record rows for already committed
// breakage ledger transactions.
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

func (c Record) Validate() error {
	var errs []error

	if err := c.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("id: %w", err))
	}

	switch c.Kind {
	case ledger.BreakageKindPlan, ledger.BreakageKindRelease, ledger.BreakageKindReopen:
	default:
		errs = append(errs, fmt.Errorf("invalid kind: %s", c.Kind))
	}

	if !c.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}

	if err := c.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer id: %w", err))
	}

	if err := c.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if err := ledger.ValidateCreditPriority(c.CreditPriority); err != nil {
		errs = append(errs, fmt.Errorf("credit priority: %w", err))
	}

	if c.ExpiresAt.IsZero() {
		errs = append(errs, errors.New("expires at is required"))
	}

	switch c.SourceKind {
	case SourceKindCreditPurchase, SourceKindUsage, SourceKindUsageCorrection, SourceKindCreditPurchaseCorrection, SourceKindAdvanceBackfill:
	default:
		errs = append(errs, fmt.Errorf("invalid source kind: %s", c.SourceKind))
	}

	if c.BreakageTransactionGroupID == "" {
		errs = append(errs, errors.New("breakage transaction group id is required"))
	}

	if c.BreakageTransactionID == "" {
		errs = append(errs, errors.New("breakage transaction id is required"))
	}

	if c.FBOSubAccountID == "" {
		errs = append(errs, errors.New("FBO sub-account id is required"))
	}

	if c.BreakageSubAccountID == "" {
		errs = append(errs, errors.New("breakage sub-account id is required"))
	}

	if c.Kind != ledger.BreakageKindPlan && (c.PlanID == nil || *c.PlanID == "") {
		errs = append(errs, errors.New("plan id is required"))
	}

	return errors.Join(errs...)
}

type Adapter interface {
	// CreateRecords persists planned/released/reopened breakage rows.
	CreateRecords(ctx context.Context, input CreateRecordsInput) error

	// ListCandidateRecords returns plan and adjustment rows needed to compute
	// open plans. Implementations should lock returned rows when the caller is in
	// a transaction so concurrent collectors cannot release the same open amount.
	ListCandidateRecords(ctx context.Context, input ListPlansInput) ([]Record, error)

	// ListReleaseRecords returns release and reopen rows for the given source
	// entries. Implementations should lock returned rows when the caller is in a
	// transaction so concurrent corrections cannot reopen the same release amount.
	ListReleaseRecords(ctx context.Context, input ListReleasesInput) ([]Record, error)

	// ListExpiredRecords returns breakage rows whose expiry is visible as of the
	// query time. The caller owns netting plan/release/reopen rows into a
	// customer-facing expired transaction.
	ListExpiredRecords(ctx context.Context, input ListExpiredRecordsInput) ([]Record, error)

	// ListBreakageTransactionCursors returns ledger cursors for committed
	// breakage transaction ids. This keeps read-model projection in breakage
	// from loading transaction groups one by one through the ledger API.
	ListBreakageTransactionCursors(ctx context.Context, input ListBreakageTransactionCursorsInput) (map[string]ledger.TransactionCursor, error)
}
