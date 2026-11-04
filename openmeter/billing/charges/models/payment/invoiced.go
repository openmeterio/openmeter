package payment

import (
	"errors"
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type InvoicedMixin = entutils.RecursiveMixin[invoicedMixin]

type invoicedMixin struct {
	ent.Schema
}

func (invoicedMixin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		Mixin{},
	}
}

func (invoicedMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("line_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
		field.String("invoice_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Immutable(),
	}
}

type InvoicedCreate struct {
	Base

	Namespace string `json:"namespace"`
	LineID    string `json:"lineID"`
	InvoiceID string `json:"invoiceID"`
}

func (i InvoicedCreate) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if err := i.Base.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("payment settlement base: %w", err))
	}

	if i.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID is required"))
	}

	if i.InvoiceID == "" {
		errs = append(errs, fmt.Errorf("invoice ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type InvoicedCreator[T any] interface {
	Creator[T]
	SetLineID(lineID string) T
	SetInvoiceID(invoiceID string) T
}

func CreateInvoiced[T InvoicedCreator[T]](creator InvoicedCreator[T], in InvoicedCreate) T {
	creator = Create(creator, in.Namespace, in.Base)
	creator = creator.SetInvoiceID(in.InvoiceID)

	return creator.SetLineID(in.LineID)
}

type InvoicedUpdater[T any] = Updater[T]

func UpdateInvoiced[T InvoicedUpdater[T]](updater InvoicedUpdater[T], in Invoiced) T {
	return Update(updater, in.Payment)
}

type BookingKey struct {
	Namespace  string
	InvoiceID  string
	LineID     string
	FiatAmount alpacadecimal.Decimal
}

// InvoicePayment represents a payment settlement using a standard invoice managed
// by the OpenMeter platform.
type Invoiced struct {
	Payment

	LineID    string `json:"lineID"`
	InvoiceID string `json:"invoiceID"`
}

var _ models.Validator = (*Invoiced)(nil)

func (r Invoiced) Validate() error {
	var errs []error

	if r.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID is required"))
	}

	if err := r.Payment.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("payment: %w", err))
	}

	if r.InvoiceID == "" {
		errs = append(errs, fmt.Errorf("invoice ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Billing retries payment callbacks for every line when any line fails, so a
// retried line can meet its own completed booking. Matching on namespace,
// invoice, line, and amount tells that apart from a booking for another line.
func (r Invoiced) IsBookedFor(key BookingKey) bool {
	return r.DeletedAt == nil &&
		r.Namespace == key.Namespace &&
		r.InvoiceID == key.InvoiceID &&
		r.LineID == key.LineID &&
		r.FiatAmount.Equal(key.FiatAmount)
}

func (r Invoiced) ErrorAttributes() models.Attributes {
	return models.Attributes{
		PaymentSettlementStatusAttributeKey: string(r.Status),
		PaymentSettlementTypeAttributeKey:   string(PaymentSettlementTypeStandardInvoice),
		paymentSettlementIDAttributeKey:     r.ID,
	}
}

type InvoicedGetter interface {
	Getter
	GetLineID() string
	GetInvoiceID() string
}

func MapInvoicedFromDB(dbEntity InvoicedGetter) Invoiced {
	return Invoiced{
		Payment:   mapPaymentFromDB(dbEntity),
		LineID:    dbEntity.GetLineID(),
		InvoiceID: dbEntity.GetInvoiceID(),
	}
}
