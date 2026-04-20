package externalid

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/samber/lo"
)

type LineMixin struct {
	mixin.Schema
}

func (LineMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("invoicing_app_external_id").
			Optional().
			Nillable(),
	}
}

type LineExternalIDCreator[T any] interface {
	SetNillableInvoicingAppExternalID(invoicingAppExternalID *string) T
}

func CreateLineExternalID[T LineExternalIDCreator[T]](creator LineExternalIDCreator[T], ids LineExternalIDs) T {
	return creator.SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(ids.Invoicing))
}

type LineExternalIDUpdater[T any] interface {
	SetOrClearInvoicingAppExternalID(invoicingAppExternalID *string) T
}

func UpdateLineExternalID[T LineExternalIDUpdater[T]](updater LineExternalIDUpdater[T], ids LineExternalIDs) T {
	return updater.SetOrClearInvoicingAppExternalID(lo.EmptyableToPtr(ids.Invoicing))
}

type LineExternalIDGetter interface {
	GetInvoicingAppExternalID() *string
}

func MapLineExternalIDFromDB(dbEntity LineExternalIDGetter) LineExternalIDs {
	return LineExternalIDs{
		Invoicing: lo.FromPtr(dbEntity.GetInvoicingAppExternalID()),
	}
}

type InvoiceMixin struct {
	mixin.Schema
}

func (InvoiceMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("invoicing_app_external_id").
			Optional().
			Nillable(),
		field.String("payment_app_external_id").
			Optional().
			Nillable(),
		field.String("tax_app_external_id").
			Optional().
			Nillable(),
	}
}

type InvoiceExternalIDCreator[T any] interface {
	SetNillableInvoicingAppExternalID(invoicingAppExternalID *string) T
	SetNillablePaymentAppExternalID(paymentAppExternalID *string) T
}

func CreateInvoiceExternalID[T InvoiceExternalIDCreator[T]](creator InvoiceExternalIDCreator[T], ids InvoiceExternalIDs) T {
	return creator.
		SetNillableInvoicingAppExternalID(lo.EmptyableToPtr(ids.Invoicing)).
		SetNillablePaymentAppExternalID(lo.EmptyableToPtr(ids.Payment))
}

type InvoiceExternalIDUpdater[T any] interface {
	SetOrClearInvoicingAppExternalID(invoicingAppExternalID *string) T
	SetOrClearPaymentAppExternalID(paymentAppExternalID *string) T
}

func UpdateInvoiceExternalID[T InvoiceExternalIDUpdater[T]](updater InvoiceExternalIDUpdater[T], ids InvoiceExternalIDs) T {
	return updater.
		SetOrClearInvoicingAppExternalID(lo.EmptyableToPtr(ids.Invoicing)).
		SetOrClearPaymentAppExternalID(lo.EmptyableToPtr(ids.Payment))
}

type InvoiceExternalIDGetter interface {
	GetInvoicingAppExternalID() *string
	GetPaymentAppExternalID() *string
}

func MapInvoiceExternalIDFromDB(dbEntity InvoiceExternalIDGetter) InvoiceExternalIDs {
	return InvoiceExternalIDs{
		Invoicing: lo.FromPtr(dbEntity.GetInvoicingAppExternalID()),
		Payment:   lo.FromPtr(dbEntity.GetPaymentAppExternalID()),
	}
}
