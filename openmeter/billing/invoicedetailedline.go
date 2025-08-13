package billing

import (
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

type DetailedLineCategory string

const (
	// DetailedLineCategoryRegular is a regular flat fee, that is based on the usage or a subscription.
	DetailedLineCategoryRegular DetailedLineCategory = "regular"
	// DetailedLineCategoryCommitment is a flat fee that is based on a commitment such as min spend.
	DetailedLineCategoryCommitment DetailedLineCategory = "commitment"
)

func (DetailedLineCategory) Values() []string {
	return []string{
		string(DetailedLineCategoryRegular),
		string(DetailedLineCategoryCommitment),
	}
}

var _ models.Validator = (*DetailedLineCategory)(nil)

func (c DetailedLineCategory) Validate() error {
	if !slices.Contains(DetailedLineCategory("").Values(), string(c)) {
		return fmt.Errorf("invalid category %s", c)
	}
	return nil
}

type DetailedLine struct {
	models.Annotations
	models.ManagedResource

	// Relationships
	InvoiceID    string `json:"invoiceID"`
	ParentLineID string `json:"parentLineID"`

	// Line details
	Category               DetailedLineCategory           `json:"category"`
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
	TaxConfig              *productcatalog.TaxConfig `json:"taxConfig,omitempty"`
	InvoicingAppExternalID *string                   `json:"invoicingAppExternalID,omitempty"`

	// Discounts
	AmountDiscounts AmountLineDiscountsManaged `json:"discounts,omitempty"`
}

var _ models.Validator = (*DetailedLine)(nil)

func (l DetailedLine) Validate() error {
	errs := []error{}

	if l.InvoiceID == "" {
		errs = append(errs, errors.New("invoiceID is required"))
	}

	if l.ParentLineID == "" {
		errs = append(errs, errors.New("parentLineID is required"))
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

	if err := l.AmountDiscounts.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("amount discounts: %w", err))
	}

	return errors.Join(errs...)
}

// TODO: Is this even needed?
func (l DetailedLine) Clone() DetailedLine {
	if l.TaxConfig != nil {
		taxConfig := *l.TaxConfig
		l.TaxConfig = &taxConfig
	}

	return l
}

func (l DetailedLine) Equal(other *DetailedLine) bool {
	return reflect.DeepEqual(l, *other)
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
