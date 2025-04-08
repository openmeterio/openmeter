// Code generated by ent, DO NOT EDIT.

package billinginvoiceusagebasedlineconfig

import (
	"entgo.io/ent/dialect/sql"
	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldContainsFold(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEQ(FieldNamespace, v))
}

// FeatureKey applies equality check predicate on the "feature_key" field. It's identical to FeatureKeyEQ.
func FeatureKey(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEQ(FieldFeatureKey, v))
}

// PreLinePeriodQuantity applies equality check predicate on the "pre_line_period_quantity" field. It's identical to PreLinePeriodQuantityEQ.
func PreLinePeriodQuantity(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEQ(FieldPreLinePeriodQuantity, v))
}

// MeteredQuantity applies equality check predicate on the "metered_quantity" field. It's identical to MeteredQuantityEQ.
func MeteredQuantity(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEQ(FieldMeteredQuantity, v))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldContainsFold(FieldNamespace, v))
}

// PriceTypeEQ applies the EQ predicate on the "price_type" field.
func PriceTypeEQ(v productcatalog.PriceType) predicate.BillingInvoiceUsageBasedLineConfig {
	vc := v
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEQ(FieldPriceType, vc))
}

// PriceTypeNEQ applies the NEQ predicate on the "price_type" field.
func PriceTypeNEQ(v productcatalog.PriceType) predicate.BillingInvoiceUsageBasedLineConfig {
	vc := v
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNEQ(FieldPriceType, vc))
}

// PriceTypeIn applies the In predicate on the "price_type" field.
func PriceTypeIn(vs ...productcatalog.PriceType) predicate.BillingInvoiceUsageBasedLineConfig {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldIn(FieldPriceType, v...))
}

// PriceTypeNotIn applies the NotIn predicate on the "price_type" field.
func PriceTypeNotIn(vs ...productcatalog.PriceType) predicate.BillingInvoiceUsageBasedLineConfig {
	v := make([]any, len(vs))
	for i := range v {
		v[i] = vs[i]
	}
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNotIn(FieldPriceType, v...))
}

// FeatureKeyEQ applies the EQ predicate on the "feature_key" field.
func FeatureKeyEQ(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEQ(FieldFeatureKey, v))
}

// FeatureKeyNEQ applies the NEQ predicate on the "feature_key" field.
func FeatureKeyNEQ(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNEQ(FieldFeatureKey, v))
}

// FeatureKeyIn applies the In predicate on the "feature_key" field.
func FeatureKeyIn(vs ...string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldIn(FieldFeatureKey, vs...))
}

// FeatureKeyNotIn applies the NotIn predicate on the "feature_key" field.
func FeatureKeyNotIn(vs ...string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNotIn(FieldFeatureKey, vs...))
}

// FeatureKeyGT applies the GT predicate on the "feature_key" field.
func FeatureKeyGT(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldGT(FieldFeatureKey, v))
}

// FeatureKeyGTE applies the GTE predicate on the "feature_key" field.
func FeatureKeyGTE(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldGTE(FieldFeatureKey, v))
}

// FeatureKeyLT applies the LT predicate on the "feature_key" field.
func FeatureKeyLT(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldLT(FieldFeatureKey, v))
}

// FeatureKeyLTE applies the LTE predicate on the "feature_key" field.
func FeatureKeyLTE(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldLTE(FieldFeatureKey, v))
}

// FeatureKeyContains applies the Contains predicate on the "feature_key" field.
func FeatureKeyContains(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldContains(FieldFeatureKey, v))
}

// FeatureKeyHasPrefix applies the HasPrefix predicate on the "feature_key" field.
func FeatureKeyHasPrefix(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldHasPrefix(FieldFeatureKey, v))
}

// FeatureKeyHasSuffix applies the HasSuffix predicate on the "feature_key" field.
func FeatureKeyHasSuffix(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldHasSuffix(FieldFeatureKey, v))
}

// FeatureKeyEqualFold applies the EqualFold predicate on the "feature_key" field.
func FeatureKeyEqualFold(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEqualFold(FieldFeatureKey, v))
}

// FeatureKeyContainsFold applies the ContainsFold predicate on the "feature_key" field.
func FeatureKeyContainsFold(v string) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldContainsFold(FieldFeatureKey, v))
}

// PreLinePeriodQuantityEQ applies the EQ predicate on the "pre_line_period_quantity" field.
func PreLinePeriodQuantityEQ(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEQ(FieldPreLinePeriodQuantity, v))
}

// PreLinePeriodQuantityNEQ applies the NEQ predicate on the "pre_line_period_quantity" field.
func PreLinePeriodQuantityNEQ(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNEQ(FieldPreLinePeriodQuantity, v))
}

// PreLinePeriodQuantityIn applies the In predicate on the "pre_line_period_quantity" field.
func PreLinePeriodQuantityIn(vs ...alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldIn(FieldPreLinePeriodQuantity, vs...))
}

// PreLinePeriodQuantityNotIn applies the NotIn predicate on the "pre_line_period_quantity" field.
func PreLinePeriodQuantityNotIn(vs ...alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNotIn(FieldPreLinePeriodQuantity, vs...))
}

// PreLinePeriodQuantityGT applies the GT predicate on the "pre_line_period_quantity" field.
func PreLinePeriodQuantityGT(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldGT(FieldPreLinePeriodQuantity, v))
}

// PreLinePeriodQuantityGTE applies the GTE predicate on the "pre_line_period_quantity" field.
func PreLinePeriodQuantityGTE(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldGTE(FieldPreLinePeriodQuantity, v))
}

// PreLinePeriodQuantityLT applies the LT predicate on the "pre_line_period_quantity" field.
func PreLinePeriodQuantityLT(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldLT(FieldPreLinePeriodQuantity, v))
}

// PreLinePeriodQuantityLTE applies the LTE predicate on the "pre_line_period_quantity" field.
func PreLinePeriodQuantityLTE(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldLTE(FieldPreLinePeriodQuantity, v))
}

// PreLinePeriodQuantityIsNil applies the IsNil predicate on the "pre_line_period_quantity" field.
func PreLinePeriodQuantityIsNil() predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldIsNull(FieldPreLinePeriodQuantity))
}

// PreLinePeriodQuantityNotNil applies the NotNil predicate on the "pre_line_period_quantity" field.
func PreLinePeriodQuantityNotNil() predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNotNull(FieldPreLinePeriodQuantity))
}

// MeteredQuantityEQ applies the EQ predicate on the "metered_quantity" field.
func MeteredQuantityEQ(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldEQ(FieldMeteredQuantity, v))
}

// MeteredQuantityNEQ applies the NEQ predicate on the "metered_quantity" field.
func MeteredQuantityNEQ(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNEQ(FieldMeteredQuantity, v))
}

// MeteredQuantityIn applies the In predicate on the "metered_quantity" field.
func MeteredQuantityIn(vs ...alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldIn(FieldMeteredQuantity, vs...))
}

// MeteredQuantityNotIn applies the NotIn predicate on the "metered_quantity" field.
func MeteredQuantityNotIn(vs ...alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNotIn(FieldMeteredQuantity, vs...))
}

// MeteredQuantityGT applies the GT predicate on the "metered_quantity" field.
func MeteredQuantityGT(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldGT(FieldMeteredQuantity, v))
}

// MeteredQuantityGTE applies the GTE predicate on the "metered_quantity" field.
func MeteredQuantityGTE(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldGTE(FieldMeteredQuantity, v))
}

// MeteredQuantityLT applies the LT predicate on the "metered_quantity" field.
func MeteredQuantityLT(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldLT(FieldMeteredQuantity, v))
}

// MeteredQuantityLTE applies the LTE predicate on the "metered_quantity" field.
func MeteredQuantityLTE(v alpacadecimal.Decimal) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldLTE(FieldMeteredQuantity, v))
}

// MeteredQuantityIsNil applies the IsNil predicate on the "metered_quantity" field.
func MeteredQuantityIsNil() predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldIsNull(FieldMeteredQuantity))
}

// MeteredQuantityNotNil applies the NotNil predicate on the "metered_quantity" field.
func MeteredQuantityNotNil() predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.FieldNotNull(FieldMeteredQuantity))
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.BillingInvoiceUsageBasedLineConfig) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.BillingInvoiceUsageBasedLineConfig) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.BillingInvoiceUsageBasedLineConfig) predicate.BillingInvoiceUsageBasedLineConfig {
	return predicate.BillingInvoiceUsageBasedLineConfig(sql.NotPredicates(p))
}
