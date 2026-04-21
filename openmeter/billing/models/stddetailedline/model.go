package stddetailedline

import (
	"errors"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/models/creditsapplied"
	"github.com/openmeterio/openmeter/openmeter/billing/models/externalid"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

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
	ChildUniqueReferenceID *string                        `json:"childUniqueReferenceID,omitempty"`
	Index                  *int                           `json:"index,omitempty"`
	PaymentTerm            productcatalog.PaymentTermType `json:"paymentTerm"`
	ServicePeriod          timeutil.ClosedPeriod          `json:"servicePeriod"`

	Currency      currencyx.Code        `json:"currency"`
	PerUnitAmount alpacadecimal.Decimal `json:"perUnitAmount"`
	Quantity      alpacadecimal.Decimal `json:"quantity"`
	Totals        totals.Totals         `json:"totals"`

	TaxConfig      *productcatalog.TaxConfig     `json:"taxConfig,omitempty"`
	ExternalIDs    externalid.LineExternalIDs    `json:"externalIDs,omitempty"`
	CreditsApplied creditsapplied.CreditsApplied `json:"creditsApplied,omitempty"`
}

var _ models.Validator = (*Base)(nil)

func (l Base) Validate() error {
	errs := []error{}

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

func (l Base) Clone() Base {
	if l.TaxConfig != nil {
		taxConfig := *l.TaxConfig
		l.TaxConfig = &taxConfig
	}

	if len(l.CreditsApplied) > 0 {
		l.CreditsApplied = l.CreditsApplied.Clone()
	}

	return l
}

func (l Base) Equal(other Base) bool {
	return deriveEqualBase(&l, &other)
}
