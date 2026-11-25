package ledger

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// JournalReader verifies when a balance projection can reuse earlier ledger reads.
type JournalReader interface {
	// CountAccountEntries includes postings at all booking times, even future ones.
	CountAccountEntries(ctx context.Context, scope AccountEntryScope) (int, error)
	GetAccountBookingCoverage(ctx context.Context, input AccountBookingCoverageInput) (AccountBookingCoverage, error)
	SumAccountMovement(ctx context.Context, input AccountMovementInput) (Balance, error)
	ListSubAccountsBookedAfter(ctx context.Context, input SubAccountsBookedAfterInput) ([]string, error)
}

type SubAccountsBookedAfterInput struct {
	AccountEntryScope
	After time.Time
	Limit int
}

func (i SubAccountsBookedAfterInput) Validate() error {
	var errs []error
	errs = append(errs, i.AccountEntryScope.Validate())
	if i.After.IsZero() {
		errs = append(errs, errors.New("booking boundary is required"))
	}

	if i.Limit <= 0 {
		errs = append(errs, errors.New("limit must be positive"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// AccountBookingCoverage counts effective postings at several boundaries in one
// read. Earlier counts correspond to Earlier in the input.
type AccountBookingCoverage struct {
	EntryCount       int
	EffectiveAtAsOf  int
	EffectiveEarlier []int
	NextBooking      *time.Time
}

type AccountBookingCoverageInput struct {
	AccountEntryScope
	AsOf    time.Time
	Earlier []time.Time
}

func (i AccountBookingCoverageInput) Validate() error {
	var errs []error
	errs = append(errs, i.AccountEntryScope.Validate())
	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}

	for _, at := range i.Earlier {
		if at.IsZero() || at.After(i.AsOf) {
			errs = append(errs, errors.New("earlier boundary must be at or before as of"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// AccountMovementInput selects (After, Through]. A posting exactly at After
// is already included in the opening account balance.
type AccountMovementInput struct {
	AccountID models.NamespacedID
	Route     RouteFilter
	After     time.Time
	Through   time.Time
}

func (i AccountMovementInput) Validate() error {
	var errs []error
	errs = append(errs, i.AccountID.Validate())
	if i.After.IsZero() || i.Through.IsZero() || i.After.After(i.Through) {
		errs = append(errs, errors.New("a valid movement interval is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// AccountEntryScope selects the journals whose postings can affect a projection.
type AccountEntryScope struct {
	Namespace  string
	AccountIDs []string
}

func (s AccountEntryScope) Validate() error {
	var errs []error
	if s.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if len(s.AccountIDs) == 0 {
		errs = append(errs, errors.New("account IDs are required"))
	}

	for _, id := range s.AccountIDs {
		if id == "" {
			errs = append(errs, errors.New("account ID must not be empty"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
