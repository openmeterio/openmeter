package billing

import (
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TODO: Rename to DetailedLineCostCategory (separate PR)
type FlatFeeCategory string

const (
	// FlatFeeCategoryRegular is a regular flat fee, that is based on the usage or a subscription.
	FlatFeeCategoryRegular FlatFeeCategory = "regular"
	// FlatFeeCategoryCommitment is a flat fee that is based on a commitment such as min spend.
	FlatFeeCategoryCommitment FlatFeeCategory = "commitment"
)

func (FlatFeeCategory) Values() []string {
	return []string{
		string(FlatFeeCategoryRegular),
		string(FlatFeeCategoryCommitment),
	}
}

var _ models.Validator = (*FlatFeeCategory)(nil)

func (c FlatFeeCategory) Validate() error {
	if !slices.Contains(FlatFeeCategory("").Values(), string(c)) {
		return fmt.Errorf("invalid category %s", c)
	}
	return nil
}

type DetailedLineBase struct {
	models.ManagedResource

	// Relationships
	InvoiceID string `json:"invoiceID"`

	// Line details
	Category               FlatFeeCategory                `json:"category"`
	ChildUniqueReferenceID *string                        `json:"childUniqueReferenceID,omitempty"`
	Index                  *int                           `json:"index,omitempty"`
	PaymentTerm            productcatalog.PaymentTermType `json:"paymentTerm"`
	ServicePeriod          Period                         `json:"servicePeriod"`

	// Line amount
	Currency      currencyx.Code        `json:"currency"`
	PerUnitAmount alpacadecimal.Decimal `json:"perUnitAmount"`
	Quantity      alpacadecimal.Decimal `json:"quantity"`
	Totals        Totals                `json:"totals"`

	// Apps
	TaxConfig   *productcatalog.TaxConfig `json:"taxConfig,omitempty"`
	ExternalIDs LineExternalIDs           `json:"externalIDs,omitempty"`

	// FeeLineConfigID contains the ID of the fee configuration in the DB, this should go away
	// as soon as we split the ubp/flatfee db parts
	FeeLineConfigID string `json:"feeLineConfigID,omitempty"`

	// CreditsApplied is the list of credits that are applied to the line (credits are pre-tax)
	CreditsApplied CreditsApplied `json:"creditsApplied,omitempty"`
}

var _ models.Validator = (*DetailedLineBase)(nil)

func (l DetailedLineBase) Validate() error {
	errs := []error{}

	if l.InvoiceID == "" {
		errs = append(errs, errors.New("invoiceID is required"))
	}

	if err := l.Category.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("category: %w", err))
	}

	if l.PerUnitAmount.IsNegative() {
		errs = append(errs, errors.New("price should be positive or zero"))
	}

	if l.Quantity.IsNegative() {
		errs = append(errs, errors.New("quantity should be positive or zero"))
	}

	if err := l.PaymentTerm.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("payment term: %w", err))
	}

	if err := l.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := l.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if err := l.CreditsApplied.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("credits applied: %w", err))
	}

	return errors.Join(errs...)
}

// TODO: Is this even needed?
func (l DetailedLineBase) Clone() DetailedLineBase {
	if l.TaxConfig != nil {
		taxConfig := *l.TaxConfig
		l.TaxConfig = &taxConfig
	}

	return l
}

func (l DetailedLineBase) Equal(other DetailedLineBase) bool {
	return deriveEqualDetailedLineBase(&l, &other)
}

type DetailedLine struct {
	DetailedLineBase

	// Discounts
	AmountDiscounts AmountLineDiscountsManaged `json:"discounts,omitempty"`
}

var _ models.Validator = (*DetailedLine)(nil)

func (l DetailedLine) Validate() error {
	errs := []error{}

	if err := l.DetailedLineBase.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("detailed line base: %w", err))
	}

	if err := l.AmountDiscounts.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("amount discounts: %w", err))
	}

	return errors.Join(errs...)
}

func (l DetailedLine) Clone() DetailedLine {
	return DetailedLine{
		DetailedLineBase: l.DetailedLineBase.Clone(),
		AmountDiscounts:  l.AmountDiscounts.Clone(),
	}
}

func (l DetailedLine) SetDiscountExternalIDs(externalIDs map[string]string) []string {
	foundIDs := []string{}

	for idx := range l.AmountDiscounts {
		discount := &l.AmountDiscounts[idx]

		if externalID, ok := externalIDs[discount.ID]; ok {
			discount.ExternalIDs.Invoicing = externalID
			foundIDs = append(foundIDs, discount.ID)
		}
	}

	return foundIDs
}

type DetailedLines []DetailedLine

func (l DetailedLines) Map(fn func(DetailedLine) DetailedLine) DetailedLines {
	return lo.Map(l, func(item DetailedLine, _ int) DetailedLine {
		return fn(item)
	})
}

func (l DetailedLines) Clone() DetailedLines {
	return l.Map(func(dl DetailedLine) DetailedLine {
		return dl.Clone()
	})
}

func (l DetailedLines) Validate() error {
	outErr := []error{}

	for idx, dl := range l {
		if err := dl.Validate(); err != nil {
			outErr = append(outErr, fmt.Errorf("[%s/%d]: %w", lo.CoalesceOrEmpty(dl.ID, "NO-ID"), idx, err))
		}
	}

	return errors.Join(outErr...)
}

func (l DetailedLines) GetByChildUniqueReferenceID(id string) *DetailedLine {
	for _, dl := range l {
		if lo.FromPtr(dl.ChildUniqueReferenceID) == id {
			return &dl
		}
	}

	return nil
}
