package billing

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/pkg/models"
)

type DetailedLineBase struct {
	stddetailedline.Base

	// Relationships
	InvoiceID string `json:"invoiceID"`

	// FeeLineConfigID contains the ID of the fee configuration in the DB, this should go away
	// as soon as we split the ubp/flatfee db parts
	FeeLineConfigID string `json:"feeLineConfigID,omitempty"`
}

var _ models.Validator = (*DetailedLineBase)(nil)

func (l DetailedLineBase) Validate() error {
	errs := []error{}

	if l.InvoiceID == "" {
		errs = append(errs, errors.New("invoiceID is required"))
	}

	if err := l.Base.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("base: %w", err))
	}

	return errors.Join(errs...)
}

// TODO: Is this even needed?
func (l DetailedLineBase) Clone() DetailedLineBase {
	l.Base = l.Base.Clone()

	return l
}

func (l DetailedLineBase) Equal(other DetailedLineBase) bool {
	return l.Base.Equal(other.Base) &&
		l.InvoiceID == other.InvoiceID &&
		l.FeeLineConfigID == other.FeeLineConfigID
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
		if dl.ChildUniqueReferenceID == id {
			return &dl
		}
	}

	return nil
}
