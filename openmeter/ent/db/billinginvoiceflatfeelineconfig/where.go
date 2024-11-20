// Code generated by ent, DO NOT EDIT.

package billinginvoiceflatfeelineconfig

import (
	"entgo.io/ent/dialect/sql"
	"github.com/alpacahq/alpacadecimal"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldContainsFold(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldEQ(FieldNamespace, v))
}

// PerUnitAmount applies equality check predicate on the "per_unit_amount" field. It's identical to PerUnitAmountEQ.
func PerUnitAmount(v alpacadecimal.Decimal) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldEQ(FieldPerUnitAmount, v))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldContainsFold(FieldNamespace, v))
}

// PerUnitAmountEQ applies the EQ predicate on the "per_unit_amount" field.
func PerUnitAmountEQ(v alpacadecimal.Decimal) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldEQ(FieldPerUnitAmount, v))
}

// PerUnitAmountNEQ applies the NEQ predicate on the "per_unit_amount" field.
func PerUnitAmountNEQ(v alpacadecimal.Decimal) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldNEQ(FieldPerUnitAmount, v))
}

// PerUnitAmountIn applies the In predicate on the "per_unit_amount" field.
func PerUnitAmountIn(vs ...alpacadecimal.Decimal) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldIn(FieldPerUnitAmount, vs...))
}

// PerUnitAmountNotIn applies the NotIn predicate on the "per_unit_amount" field.
func PerUnitAmountNotIn(vs ...alpacadecimal.Decimal) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldNotIn(FieldPerUnitAmount, vs...))
}

// PerUnitAmountGT applies the GT predicate on the "per_unit_amount" field.
func PerUnitAmountGT(v alpacadecimal.Decimal) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldGT(FieldPerUnitAmount, v))
}

// PerUnitAmountGTE applies the GTE predicate on the "per_unit_amount" field.
func PerUnitAmountGTE(v alpacadecimal.Decimal) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldGTE(FieldPerUnitAmount, v))
}

// PerUnitAmountLT applies the LT predicate on the "per_unit_amount" field.
func PerUnitAmountLT(v alpacadecimal.Decimal) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldLT(FieldPerUnitAmount, v))
}

// PerUnitAmountLTE applies the LTE predicate on the "per_unit_amount" field.
func PerUnitAmountLTE(v alpacadecimal.Decimal) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldLTE(FieldPerUnitAmount, v))
}

// CategoryEQ applies the EQ predicate on the "category" field.
func CategoryEQ(v billingentity.FlatFeeCategory) predicate.BillingInvoiceFlatFeeLineConfig {
	vc := v
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldEQ(FieldCategory, vc))
}

// CategoryNEQ applies the NEQ predicate on the "category" field.
func CategoryNEQ(v billingentity.FlatFeeCategory) predicate.BillingInvoiceFlatFeeLineConfig {
	vc := v
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldNEQ(FieldCategory, vc))
}

// CategoryIn applies the In predicate on the "category" field.
func CategoryIn(vs ...billingentity.FlatFeeCategory) predicate.BillingInvoiceFlatFeeLineConfig {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldIn(FieldCategory, v...))
}

// CategoryNotIn applies the NotIn predicate on the "category" field.
func CategoryNotIn(vs ...billingentity.FlatFeeCategory) predicate.BillingInvoiceFlatFeeLineConfig {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.FieldNotIn(FieldCategory, v...))
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.BillingInvoiceFlatFeeLineConfig) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.BillingInvoiceFlatFeeLineConfig) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.BillingInvoiceFlatFeeLineConfig) predicate.BillingInvoiceFlatFeeLineConfig {
	return predicate.BillingInvoiceFlatFeeLineConfig(sql.NotPredicates(p))
}
