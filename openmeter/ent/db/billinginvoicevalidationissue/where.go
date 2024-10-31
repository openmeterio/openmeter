// Code generated by ent, DO NOT EDIT.

package billinginvoicevalidationissue

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContainsFold(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldNamespace, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldUpdatedAt, v))
}

// DeletedAt applies equality check predicate on the "deleted_at" field. It's identical to DeletedAtEQ.
func DeletedAt(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldDeletedAt, v))
}

// InvoiceID applies equality check predicate on the "invoice_id" field. It's identical to InvoiceIDEQ.
func InvoiceID(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldInvoiceID, v))
}

// Code applies equality check predicate on the "code" field. It's identical to CodeEQ.
func Code(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldCode, v))
}

// Message applies equality check predicate on the "message" field. It's identical to MessageEQ.
func Message(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldMessage, v))
}

// Path applies equality check predicate on the "path" field. It's identical to PathEQ.
func Path(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldPath, v))
}

// Component applies equality check predicate on the "component" field. It's identical to ComponentEQ.
func Component(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldComponent, v))
}

// DedupeHash applies equality check predicate on the "dedupe_hash" field. It's identical to DedupeHashEQ.
func DedupeHash(v []byte) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldDedupeHash, v))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContainsFold(FieldNamespace, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLTE(FieldUpdatedAt, v))
}

// DeletedAtEQ applies the EQ predicate on the "deleted_at" field.
func DeletedAtEQ(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldDeletedAt, v))
}

// DeletedAtNEQ applies the NEQ predicate on the "deleted_at" field.
func DeletedAtNEQ(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldDeletedAt, v))
}

// DeletedAtIn applies the In predicate on the "deleted_at" field.
func DeletedAtIn(vs ...time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldDeletedAt, vs...))
}

// DeletedAtNotIn applies the NotIn predicate on the "deleted_at" field.
func DeletedAtNotIn(vs ...time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldDeletedAt, vs...))
}

// DeletedAtGT applies the GT predicate on the "deleted_at" field.
func DeletedAtGT(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGT(FieldDeletedAt, v))
}

// DeletedAtGTE applies the GTE predicate on the "deleted_at" field.
func DeletedAtGTE(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGTE(FieldDeletedAt, v))
}

// DeletedAtLT applies the LT predicate on the "deleted_at" field.
func DeletedAtLT(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLT(FieldDeletedAt, v))
}

// DeletedAtLTE applies the LTE predicate on the "deleted_at" field.
func DeletedAtLTE(v time.Time) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLTE(FieldDeletedAt, v))
}

// DeletedAtIsNil applies the IsNil predicate on the "deleted_at" field.
func DeletedAtIsNil() predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIsNull(FieldDeletedAt))
}

// DeletedAtNotNil applies the NotNil predicate on the "deleted_at" field.
func DeletedAtNotNil() predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotNull(FieldDeletedAt))
}

// InvoiceIDEQ applies the EQ predicate on the "invoice_id" field.
func InvoiceIDEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldInvoiceID, v))
}

// InvoiceIDNEQ applies the NEQ predicate on the "invoice_id" field.
func InvoiceIDNEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldInvoiceID, v))
}

// InvoiceIDIn applies the In predicate on the "invoice_id" field.
func InvoiceIDIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldInvoiceID, vs...))
}

// InvoiceIDNotIn applies the NotIn predicate on the "invoice_id" field.
func InvoiceIDNotIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldInvoiceID, vs...))
}

// InvoiceIDGT applies the GT predicate on the "invoice_id" field.
func InvoiceIDGT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGT(FieldInvoiceID, v))
}

// InvoiceIDGTE applies the GTE predicate on the "invoice_id" field.
func InvoiceIDGTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGTE(FieldInvoiceID, v))
}

// InvoiceIDLT applies the LT predicate on the "invoice_id" field.
func InvoiceIDLT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLT(FieldInvoiceID, v))
}

// InvoiceIDLTE applies the LTE predicate on the "invoice_id" field.
func InvoiceIDLTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLTE(FieldInvoiceID, v))
}

// InvoiceIDContains applies the Contains predicate on the "invoice_id" field.
func InvoiceIDContains(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContains(FieldInvoiceID, v))
}

// InvoiceIDHasPrefix applies the HasPrefix predicate on the "invoice_id" field.
func InvoiceIDHasPrefix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasPrefix(FieldInvoiceID, v))
}

// InvoiceIDHasSuffix applies the HasSuffix predicate on the "invoice_id" field.
func InvoiceIDHasSuffix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasSuffix(FieldInvoiceID, v))
}

// InvoiceIDEqualFold applies the EqualFold predicate on the "invoice_id" field.
func InvoiceIDEqualFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEqualFold(FieldInvoiceID, v))
}

// InvoiceIDContainsFold applies the ContainsFold predicate on the "invoice_id" field.
func InvoiceIDContainsFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContainsFold(FieldInvoiceID, v))
}

// SeverityEQ applies the EQ predicate on the "severity" field.
func SeverityEQ(v billingentity.ValidationIssueSeverity) predicate.BillingInvoiceValidationIssue {
	vc := v
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldSeverity, vc))
}

// SeverityNEQ applies the NEQ predicate on the "severity" field.
func SeverityNEQ(v billingentity.ValidationIssueSeverity) predicate.BillingInvoiceValidationIssue {
	vc := v
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldSeverity, vc))
}

// SeverityIn applies the In predicate on the "severity" field.
func SeverityIn(vs ...billingentity.ValidationIssueSeverity) predicate.BillingInvoiceValidationIssue {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldSeverity, v...))
}

// SeverityNotIn applies the NotIn predicate on the "severity" field.
func SeverityNotIn(vs ...billingentity.ValidationIssueSeverity) predicate.BillingInvoiceValidationIssue {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldSeverity, v...))
}

// CodeEQ applies the EQ predicate on the "code" field.
func CodeEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldCode, v))
}

// CodeNEQ applies the NEQ predicate on the "code" field.
func CodeNEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldCode, v))
}

// CodeIn applies the In predicate on the "code" field.
func CodeIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldCode, vs...))
}

// CodeNotIn applies the NotIn predicate on the "code" field.
func CodeNotIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldCode, vs...))
}

// CodeGT applies the GT predicate on the "code" field.
func CodeGT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGT(FieldCode, v))
}

// CodeGTE applies the GTE predicate on the "code" field.
func CodeGTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGTE(FieldCode, v))
}

// CodeLT applies the LT predicate on the "code" field.
func CodeLT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLT(FieldCode, v))
}

// CodeLTE applies the LTE predicate on the "code" field.
func CodeLTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLTE(FieldCode, v))
}

// CodeContains applies the Contains predicate on the "code" field.
func CodeContains(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContains(FieldCode, v))
}

// CodeHasPrefix applies the HasPrefix predicate on the "code" field.
func CodeHasPrefix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasPrefix(FieldCode, v))
}

// CodeHasSuffix applies the HasSuffix predicate on the "code" field.
func CodeHasSuffix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasSuffix(FieldCode, v))
}

// CodeIsNil applies the IsNil predicate on the "code" field.
func CodeIsNil() predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIsNull(FieldCode))
}

// CodeNotNil applies the NotNil predicate on the "code" field.
func CodeNotNil() predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotNull(FieldCode))
}

// CodeEqualFold applies the EqualFold predicate on the "code" field.
func CodeEqualFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEqualFold(FieldCode, v))
}

// CodeContainsFold applies the ContainsFold predicate on the "code" field.
func CodeContainsFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContainsFold(FieldCode, v))
}

// MessageEQ applies the EQ predicate on the "message" field.
func MessageEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldMessage, v))
}

// MessageNEQ applies the NEQ predicate on the "message" field.
func MessageNEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldMessage, v))
}

// MessageIn applies the In predicate on the "message" field.
func MessageIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldMessage, vs...))
}

// MessageNotIn applies the NotIn predicate on the "message" field.
func MessageNotIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldMessage, vs...))
}

// MessageGT applies the GT predicate on the "message" field.
func MessageGT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGT(FieldMessage, v))
}

// MessageGTE applies the GTE predicate on the "message" field.
func MessageGTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGTE(FieldMessage, v))
}

// MessageLT applies the LT predicate on the "message" field.
func MessageLT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLT(FieldMessage, v))
}

// MessageLTE applies the LTE predicate on the "message" field.
func MessageLTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLTE(FieldMessage, v))
}

// MessageContains applies the Contains predicate on the "message" field.
func MessageContains(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContains(FieldMessage, v))
}

// MessageHasPrefix applies the HasPrefix predicate on the "message" field.
func MessageHasPrefix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasPrefix(FieldMessage, v))
}

// MessageHasSuffix applies the HasSuffix predicate on the "message" field.
func MessageHasSuffix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasSuffix(FieldMessage, v))
}

// MessageEqualFold applies the EqualFold predicate on the "message" field.
func MessageEqualFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEqualFold(FieldMessage, v))
}

// MessageContainsFold applies the ContainsFold predicate on the "message" field.
func MessageContainsFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContainsFold(FieldMessage, v))
}

// PathEQ applies the EQ predicate on the "path" field.
func PathEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldPath, v))
}

// PathNEQ applies the NEQ predicate on the "path" field.
func PathNEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldPath, v))
}

// PathIn applies the In predicate on the "path" field.
func PathIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldPath, vs...))
}

// PathNotIn applies the NotIn predicate on the "path" field.
func PathNotIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldPath, vs...))
}

// PathGT applies the GT predicate on the "path" field.
func PathGT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGT(FieldPath, v))
}

// PathGTE applies the GTE predicate on the "path" field.
func PathGTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGTE(FieldPath, v))
}

// PathLT applies the LT predicate on the "path" field.
func PathLT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLT(FieldPath, v))
}

// PathLTE applies the LTE predicate on the "path" field.
func PathLTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLTE(FieldPath, v))
}

// PathContains applies the Contains predicate on the "path" field.
func PathContains(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContains(FieldPath, v))
}

// PathHasPrefix applies the HasPrefix predicate on the "path" field.
func PathHasPrefix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasPrefix(FieldPath, v))
}

// PathHasSuffix applies the HasSuffix predicate on the "path" field.
func PathHasSuffix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasSuffix(FieldPath, v))
}

// PathIsNil applies the IsNil predicate on the "path" field.
func PathIsNil() predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIsNull(FieldPath))
}

// PathNotNil applies the NotNil predicate on the "path" field.
func PathNotNil() predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotNull(FieldPath))
}

// PathEqualFold applies the EqualFold predicate on the "path" field.
func PathEqualFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEqualFold(FieldPath, v))
}

// PathContainsFold applies the ContainsFold predicate on the "path" field.
func PathContainsFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContainsFold(FieldPath, v))
}

// ComponentEQ applies the EQ predicate on the "component" field.
func ComponentEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldComponent, v))
}

// ComponentNEQ applies the NEQ predicate on the "component" field.
func ComponentNEQ(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldComponent, v))
}

// ComponentIn applies the In predicate on the "component" field.
func ComponentIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldComponent, vs...))
}

// ComponentNotIn applies the NotIn predicate on the "component" field.
func ComponentNotIn(vs ...string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldComponent, vs...))
}

// ComponentGT applies the GT predicate on the "component" field.
func ComponentGT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGT(FieldComponent, v))
}

// ComponentGTE applies the GTE predicate on the "component" field.
func ComponentGTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGTE(FieldComponent, v))
}

// ComponentLT applies the LT predicate on the "component" field.
func ComponentLT(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLT(FieldComponent, v))
}

// ComponentLTE applies the LTE predicate on the "component" field.
func ComponentLTE(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLTE(FieldComponent, v))
}

// ComponentContains applies the Contains predicate on the "component" field.
func ComponentContains(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContains(FieldComponent, v))
}

// ComponentHasPrefix applies the HasPrefix predicate on the "component" field.
func ComponentHasPrefix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasPrefix(FieldComponent, v))
}

// ComponentHasSuffix applies the HasSuffix predicate on the "component" field.
func ComponentHasSuffix(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldHasSuffix(FieldComponent, v))
}

// ComponentEqualFold applies the EqualFold predicate on the "component" field.
func ComponentEqualFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEqualFold(FieldComponent, v))
}

// ComponentContainsFold applies the ContainsFold predicate on the "component" field.
func ComponentContainsFold(v string) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldContainsFold(FieldComponent, v))
}

// DedupeHashEQ applies the EQ predicate on the "dedupe_hash" field.
func DedupeHashEQ(v []byte) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldEQ(FieldDedupeHash, v))
}

// DedupeHashNEQ applies the NEQ predicate on the "dedupe_hash" field.
func DedupeHashNEQ(v []byte) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNEQ(FieldDedupeHash, v))
}

// DedupeHashIn applies the In predicate on the "dedupe_hash" field.
func DedupeHashIn(vs ...[]byte) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldIn(FieldDedupeHash, vs...))
}

// DedupeHashNotIn applies the NotIn predicate on the "dedupe_hash" field.
func DedupeHashNotIn(vs ...[]byte) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldNotIn(FieldDedupeHash, vs...))
}

// DedupeHashGT applies the GT predicate on the "dedupe_hash" field.
func DedupeHashGT(v []byte) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGT(FieldDedupeHash, v))
}

// DedupeHashGTE applies the GTE predicate on the "dedupe_hash" field.
func DedupeHashGTE(v []byte) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldGTE(FieldDedupeHash, v))
}

// DedupeHashLT applies the LT predicate on the "dedupe_hash" field.
func DedupeHashLT(v []byte) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLT(FieldDedupeHash, v))
}

// DedupeHashLTE applies the LTE predicate on the "dedupe_hash" field.
func DedupeHashLTE(v []byte) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.FieldLTE(FieldDedupeHash, v))
}

// HasBillingInvoice applies the HasEdge predicate on the "billing_invoice" edge.
func HasBillingInvoice() predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, BillingInvoiceTable, BillingInvoiceColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasBillingInvoiceWith applies the HasEdge predicate on the "billing_invoice" edge with a given conditions (other predicates).
func HasBillingInvoiceWith(preds ...predicate.BillingInvoice) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(func(s *sql.Selector) {
		step := newBillingInvoiceStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.BillingInvoiceValidationIssue) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.BillingInvoiceValidationIssue) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.BillingInvoiceValidationIssue) predicate.BillingInvoiceValidationIssue {
	return predicate.BillingInvoiceValidationIssue(sql.NotPredicates(p))
}
