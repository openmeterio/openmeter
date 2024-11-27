package billing

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
)

type CreateInvoiceLinesInput struct {
	CustomerID string
	Namespace  string
	Lines      []billingentity.Line
}

func (c CreateInvoiceLinesInput) Validate() error {
	if c.Namespace == "" {
		return errors.New("namespace is required")
	}

	if c.CustomerID == "" {
		return errors.New("customer key or ID is required")
	}

	for _, line := range c.Lines {
		if err := line.Validate(); err != nil {
			return fmt.Errorf("Line: %w", err)
		}
	}

	return nil
}

type UpsertInvoiceLinesAdapterInput struct {
	Namespace string
	Lines     []*billingentity.Line
}

func (c UpsertInvoiceLinesAdapterInput) Validate() error {
	if c.Namespace == "" {
		return errors.New("namespace is required")
	}

	for i, line := range c.Lines {
		if err := line.Validate(); err != nil {
			return fmt.Errorf("line[%d]: %w", i, err)
		}

		if line.Namespace == "" {
			return fmt.Errorf("line[%d]: namespace is required", i)
		}

		if line.InvoiceID == "" {
			return fmt.Errorf("line[%d]: invoice id is required", i)
		}
	}

	return nil
}

type ListInvoiceLinesAdapterInput struct {
	Namespace string

	CustomerID                 string
	InvoiceStatuses            []billingentity.InvoiceStatus
	InvoiceAtBefore            *time.Time
	IncludeDeleted             bool
	ParentLineIDs              []string
	ParentLineIDsIncludeParent bool
	Statuses                   []billingentity.InvoiceLineStatus

	LineIDs []string
}

func (g ListInvoiceLinesAdapterInput) Validate() error {
	if g.Namespace == "" {
		return errors.New("namespace is required")
	}

	return nil
}

type AssociateLinesToInvoiceAdapterInput struct {
	Invoice billingentity.InvoiceID

	LineIDs []string
}

func (i AssociateLinesToInvoiceAdapterInput) Validate() error {
	if err := i.Invoice.Validate(); err != nil {
		return fmt.Errorf("invoice: %w", err)
	}

	if len(i.LineIDs) == 0 {
		return errors.New("line ids are required")
	}

	return nil
}

type UpdateInvoiceLineAdapterInput billingentity.Line

type UpdateInvoiceLineInput struct {
	// Mandatory fields for update
	Line billingentity.LineID
	Type billingentity.InvoiceLineType

	LineBase   UpdateInvoiceLineBaseInput
	UsageBased UpdateInvoiceLineUsageBasedInput
	FlatFee    UpdateInvoiceLineFlatFeeInput
}

func (u UpdateInvoiceLineInput) Validate() error {
	var outErr error
	if err := u.LineBase.Validate(); err != nil {
		outErr = errors.Join(outErr, err)
	}

	if err := u.Line.Validate(); err != nil {
		outErr = errors.Join(outErr, fmt.Errorf("validating LineID: %w", err))
	}

	if !slices.Contains(u.Type.Values(), string(u.Type)) {
		outErr = errors.Join(outErr, billingentity.ValidationWithFieldPrefix(
			"type", fmt.Errorf("line base: invalid type %s", u.Type),
		))
		return outErr
	}

	switch u.Type {
	case billingentity.InvoiceLineTypeUsageBased:
		if err := u.UsageBased.Validate(); err != nil {
			outErr = errors.Join(outErr, err)
		}
	case billingentity.InvoiceLineTypeFee:
		if err := u.FlatFee.Validate(); err != nil {
			outErr = errors.Join(outErr, err)
		}
	}

	return outErr
}

func (u UpdateInvoiceLineInput) Apply(l *billingentity.Line) (*billingentity.Line, error) {
	oldParentLine := l.ParentLine

	l = l.Clone()

	// Clone doesn't carry over parent line, so that the cloned hierarchy and the new one are disjunct,
	// however in this specific case we don't care about that, so we just copy it over
	l.ParentLine = oldParentLine

	if u.Type != l.Type {
		return l, fmt.Errorf("line type cannot be changed")
	}

	if err := u.LineBase.Apply(l); err != nil {
		return l, err
	}

	switch l.Type {
	case billingentity.InvoiceLineTypeUsageBased:
		if err := u.UsageBased.Apply(&l.UsageBased); err != nil {
			return l, err
		}
	case billingentity.InvoiceLineTypeFee:
		if err := u.FlatFee.Apply(&l.FlatFee); err != nil {
			return l, err
		}
	}

	return l, nil
}

type UpdateInvoiceLineBaseInput struct {
	InvoiceAt mo.Option[time.Time]

	Metadata  mo.Option[map[string]string]
	Name      mo.Option[string]
	Period    mo.Option[billingentity.Period]
	TaxConfig mo.Option[*billingentity.TaxConfig]
}

func (u UpdateInvoiceLineBaseInput) Validate() error {
	var outErr error

	if u.InvoiceAt.IsPresent() {
		invoiceAt := u.InvoiceAt.OrEmpty()

		if invoiceAt.IsZero() {
			outErr = errors.Join(outErr, billingentity.ValidationWithFieldPrefix("invoice_at", billingentity.ErrFieldRequired))
		}
	}

	if u.Name.IsPresent() && u.Name.OrEmpty() == "" {
		outErr = errors.Join(outErr, billingentity.ValidationWithFieldPrefix("name", billingentity.ErrFieldRequired))
	}

	if u.Period.IsPresent() {
		if err := u.Period.OrEmpty().Validate(); err != nil {
			outErr = errors.Join(outErr, billingentity.ValidationWithFieldPrefix("period", err))
		}
	}

	if u.TaxConfig.IsPresent() {
		if err := u.TaxConfig.OrEmpty().Validate(); err != nil {
			outErr = errors.Join(outErr, billingentity.ValidationWithFieldPrefix("tax_config", err))
		}
	}

	return outErr
}

func (u UpdateInvoiceLineBaseInput) Apply(l *billingentity.Line) error {
	if u.InvoiceAt.IsPresent() {
		l.InvoiceAt = u.InvoiceAt.OrEmpty().In(time.UTC)
	}

	if u.Metadata.IsPresent() {
		l.Metadata = u.Metadata.OrEmpty()
	}

	if u.Name.IsPresent() {
		l.Name = u.Name.OrEmpty()
	}

	if u.Period.IsPresent() {
		l.Period = u.Period.OrEmpty()
	}

	if u.TaxConfig.IsPresent() {
		l.TaxConfig = u.TaxConfig.OrEmpty()
	}

	return nil
}

type UpdateInvoiceLineUsageBasedInput struct {
	Price mo.Option[billingentity.Price]
}

func (u UpdateInvoiceLineUsageBasedInput) Validate() error {
	var outErr error

	if u.Price.IsPresent() {
		price := u.Price.OrEmpty()
		if err := price.Validate(); err != nil {
			outErr = errors.Join(outErr, billingentity.ValidationWithFieldPrefix("price", err))
		}
	}

	return outErr
}

func (u UpdateInvoiceLineUsageBasedInput) Apply(l *billingentity.UsageBasedLine) error {
	if u.Price.IsPresent() {
		l.Price = u.Price.OrEmpty()
	}

	return nil
}

type UpdateInvoiceLineFlatFeeInput struct {
	PerUnitAmount mo.Option[alpacadecimal.Decimal]
	Quantity      mo.Option[alpacadecimal.Decimal]
	PaymentTerm   mo.Option[plan.PaymentTermType]
}

func (u UpdateInvoiceLineFlatFeeInput) Validate() error {
	var outErr error

	if u.PerUnitAmount.IsPresent() && !u.PerUnitAmount.OrEmpty().IsPositive() {
		outErr = errors.Join(outErr, billingentity.ValidationWithFieldPrefix("per_unit_amount", billingentity.ErrFieldMustBePositive))
	}

	if u.Quantity.IsPresent() && u.Quantity.OrEmpty().IsNegative() {
		outErr = errors.Join(outErr, billingentity.ValidationWithFieldPrefix("quantity", billingentity.ErrFieldMustBePositiveOrZero))
	}

	if u.PaymentTerm.IsPresent() && !slices.Contains(plan.PaymentTermType("").Values(), u.PaymentTerm.OrEmpty()) {
		outErr = errors.Join(outErr, billingentity.ValidationWithFieldPrefix("payment_term", fmt.Errorf("invalid payment term %s", u.PaymentTerm.OrEmpty())))
	}

	return outErr
}

func (u UpdateInvoiceLineFlatFeeInput) Apply(l *billingentity.FlatFeeLine) error {
	if u.PerUnitAmount.IsPresent() {
		l.PerUnitAmount = u.PerUnitAmount.OrEmpty()
	}

	if u.Quantity.IsPresent() {
		l.Quantity = u.Quantity.OrEmpty()
	}

	if u.PaymentTerm.IsPresent() {
		l.PaymentTerm = u.PaymentTerm.OrEmpty()
	}

	return nil
}

type GetInvoiceLineAdapterInput = billingentity.LineID

type GetInvoiceLineInput = billingentity.LineID

type GetInvoiceLineOwnershipAdapterInput = billingentity.LineID

type ValidateLineOwnershipInput struct {
	Namespace  string
	LineID     string
	InvoiceID  string
	CustomerID string
}

func (v ValidateLineOwnershipInput) Validate() error {
	if v.Namespace == "" {
		return errors.New("namespace is required")
	}

	if v.LineID == "" {
		return errors.New("line id is required")
	}

	if v.InvoiceID == "" {
		return errors.New("invoice id is required")
	}

	if v.CustomerID == "" {
		return errors.New("customer id is required")
	}

	return nil
}

type DeleteInvoiceLineInput = billingentity.LineID
