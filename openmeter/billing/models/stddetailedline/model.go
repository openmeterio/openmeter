package stddetailedline

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/models/creditsapplied"
	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// ErrCreditsNotConsumedFully indicates that the detailed lines cannot absorb
// all supplied credit allocations.
var ErrCreditsNotConsumedFully = errors.New("credits not consumed fully")

type Category string

const (
	// CategoryRegular is a regular flat fee, that is based on the usage or a subscription.
	CategoryRegular Category = "regular"
	// CategoryCommitment is a flat fee that is based on a commitment such as min spend.
	CategoryCommitment Category = "commitment"
)

func (Category) Values() []string {
	return []string{
		string(CategoryRegular),
		string(CategoryCommitment),
	}
}

var _ models.Validator = (*Category)(nil)

func (c Category) Validate() error {
	if !slices.Contains(Category("").Values(), string(c)) {
		return fmt.Errorf("invalid category %s", c)
	}

	return nil
}

type Base struct {
	models.ManagedResource

	Category               Category                       `json:"category"`
	ChildUniqueReferenceID string                         `json:"childUniqueReferenceID"`
	Index                  *int                           `json:"index,omitempty"`
	PaymentTerm            productcatalog.PaymentTermType `json:"paymentTerm"`
	ServicePeriod          timeutil.ClosedPeriod          `json:"servicePeriod"`

	PerUnitAmount alpacadecimal.Decimal `json:"perUnitAmount"`
	Quantity      alpacadecimal.Decimal `json:"quantity"`
	Totals        totals.Totals         `json:"totals"`

	ExternalIDs    externalid.LineExternalIDs    `json:"externalIDs,omitempty"`
	CreditsApplied creditsapplied.CreditsApplied `json:"creditsApplied,omitempty"`
}

type validateOptions struct {
	ignoreQuantityChecks bool
}

type ValidateOption func(*validateOptions)

func IgnoreQuantityChecks() ValidateOption {
	return func(o *validateOptions) {
		o.ignoreQuantityChecks = true
	}
}

func (l Base) Validate(opts ...ValidateOption) error {
	options := validateOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	errs := []error{}

	if err := l.Category.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("category: %w", err))
	}

	if l.ChildUniqueReferenceID == "" {
		errs = append(errs, errors.New("child unique reference id is required"))
	}

	if l.PerUnitAmount.IsNegative() {
		errs = append(errs, errors.New("price should be positive or zero"))
	}

	if !options.ignoreQuantityChecks && l.Quantity.IsNegative() {
		errs = append(errs, errors.New("quantity should be positive or zero"))
	}

	if err := l.PaymentTerm.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("payment term: %w", err))
	}

	if err := l.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := l.CreditsApplied.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credits applied: %w", err))
	}

	return errors.Join(errs...)
}

func (l Base) Clone() Base {
	if len(l.CreditsApplied) > 0 {
		l.CreditsApplied = l.CreditsApplied.Clone()
	}

	return l
}

func (l Base) Equal(other Base) bool {
	return deriveEqualBase(&l, &other)
}

func (l Base) GetIndex() *int {
	return l.Index
}

func (l Base) GetCreatedAt() time.Time {
	return l.CreatedAt
}

func (l Base) GetID() string {
	return l.ID
}

type Comparable interface {
	GetIndex() *int
	GetCreatedAt() time.Time
	GetID() string
}

func Compare[T Comparable](a, b T) int {
	if a.GetIndex() == nil && b.GetIndex() != nil {
		return 1
	}
	if a.GetIndex() != nil && b.GetIndex() == nil {
		return -1
	}
	if a.GetIndex() != nil && b.GetIndex() != nil {
		if c := cmp.Compare(*a.GetIndex(), *b.GetIndex()); c != 0 {
			return c
		}
	}
	if c := a.GetCreatedAt().Compare(b.GetCreatedAt()); c != 0 {
		return c
	}
	return cmp.Compare(a.GetID(), b.GetID())
}

type Bases []Base

func (l Bases) Clone() Bases {
	out := make(Bases, len(l))
	for idx, line := range l {
		out[idx] = line.Clone()
	}

	return out
}

func (l Bases) SumTotals() totals.Totals {
	out := totals.Totals{}
	for _, line := range l {
		out = out.Add(line.Totals)
	}

	return out
}

// WithReversedCredits returns cloned detailed-line bases with credit applications
// removed and totals recalculated as if no credits had been applied.
func (l Bases) WithReversedCredits() Bases {
	out := l.Clone()
	for idx := range out {
		out[idx].CreditsApplied = nil
		out[idx].Totals.CreditsTotal = alpacadecimal.Zero
		out[idx].Totals.Total = out[idx].Totals.CalculateTotal()
	}

	return out
}

func (l Bases) WithCreditsApplied(
	creditsApplied creditsapplied.CreditsApplied,
	currency currencyx.Currency,
) (Bases, error) {
	if currency == nil {
		return nil, errors.New("currency is required")
	}

	detailedLines := l.WithReversedCredits()

	for _, creditToApply := range creditsApplied {
		creditValueRemaining := currency.RoundToPrecision(creditToApply.Amount)

		for idx := range detailedLines {
			if creditValueRemaining.IsZero() {
				break
			}

			totalAmount := currency.RoundToPrecision(detailedLines[idx].Totals.Total)
			if !totalAmount.IsPositive() {
				continue
			}

			amountToApply := creditValueRemaining
			if totalAmount.LessThan(creditValueRemaining) {
				amountToApply = totalAmount
			}
			detailedLines[idx].CreditsApplied = append(detailedLines[idx].CreditsApplied, creditToApply.CloneWithAmount(amountToApply))
			detailedLines[idx].Totals.CreditsTotal = currency.RoundToPrecision(detailedLines[idx].Totals.CreditsTotal.Add(amountToApply))
			detailedLines[idx].Totals.Total = currency.RoundToPrecision(detailedLines[idx].Totals.Total.Sub(amountToApply))
			creditValueRemaining = currency.RoundToPrecision(creditValueRemaining.Sub(amountToApply))
		}

		if creditValueRemaining.IsPositive() {
			return nil, ErrCreditsNotConsumedFully
		}
	}

	return detailedLines, nil
}

func (l Bases) Validate() error {
	var errs []error

	for idx, line := range l {
		if err := line.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("[%d]: %w", idx, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
