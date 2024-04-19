// Code generated by ent, DO NOT EDIT.

package product

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/predicate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.Product {
	return predicate.Product(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.Product {
	return predicate.Product(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.Product {
	return predicate.Product(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.Product {
	return predicate.Product(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.Product {
	return predicate.Product(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.Product {
	return predicate.Product(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.Product {
	return predicate.Product(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.Product {
	return predicate.Product(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.Product {
	return predicate.Product(sql.FieldContainsFold(FieldID, id))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldUpdatedAt, v))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldNamespace, v))
}

// Name applies equality check predicate on the "name" field. It's identical to NameEQ.
func Name(v string) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldName, v))
}

// MeterSlug applies equality check predicate on the "meter_slug" field. It's identical to MeterSlugEQ.
func MeterSlug(v string) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldMeterSlug, v))
}

// Archived applies equality check predicate on the "archived" field. It's identical to ArchivedEQ.
func Archived(v bool) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldArchived, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.Product {
	return predicate.Product(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.Product {
	return predicate.Product(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.Product {
	return predicate.Product(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.Product {
	return predicate.Product(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.Product {
	return predicate.Product(sql.FieldLTE(FieldUpdatedAt, v))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.Product {
	return predicate.Product(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.Product {
	return predicate.Product(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.Product {
	return predicate.Product(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.Product {
	return predicate.Product(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.Product {
	return predicate.Product(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.Product {
	return predicate.Product(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.Product {
	return predicate.Product(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.Product {
	return predicate.Product(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.Product {
	return predicate.Product(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.Product {
	return predicate.Product(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.Product {
	return predicate.Product(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.Product {
	return predicate.Product(sql.FieldContainsFold(FieldNamespace, v))
}

// NameEQ applies the EQ predicate on the "name" field.
func NameEQ(v string) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldName, v))
}

// NameNEQ applies the NEQ predicate on the "name" field.
func NameNEQ(v string) predicate.Product {
	return predicate.Product(sql.FieldNEQ(FieldName, v))
}

// NameIn applies the In predicate on the "name" field.
func NameIn(vs ...string) predicate.Product {
	return predicate.Product(sql.FieldIn(FieldName, vs...))
}

// NameNotIn applies the NotIn predicate on the "name" field.
func NameNotIn(vs ...string) predicate.Product {
	return predicate.Product(sql.FieldNotIn(FieldName, vs...))
}

// NameGT applies the GT predicate on the "name" field.
func NameGT(v string) predicate.Product {
	return predicate.Product(sql.FieldGT(FieldName, v))
}

// NameGTE applies the GTE predicate on the "name" field.
func NameGTE(v string) predicate.Product {
	return predicate.Product(sql.FieldGTE(FieldName, v))
}

// NameLT applies the LT predicate on the "name" field.
func NameLT(v string) predicate.Product {
	return predicate.Product(sql.FieldLT(FieldName, v))
}

// NameLTE applies the LTE predicate on the "name" field.
func NameLTE(v string) predicate.Product {
	return predicate.Product(sql.FieldLTE(FieldName, v))
}

// NameContains applies the Contains predicate on the "name" field.
func NameContains(v string) predicate.Product {
	return predicate.Product(sql.FieldContains(FieldName, v))
}

// NameHasPrefix applies the HasPrefix predicate on the "name" field.
func NameHasPrefix(v string) predicate.Product {
	return predicate.Product(sql.FieldHasPrefix(FieldName, v))
}

// NameHasSuffix applies the HasSuffix predicate on the "name" field.
func NameHasSuffix(v string) predicate.Product {
	return predicate.Product(sql.FieldHasSuffix(FieldName, v))
}

// NameEqualFold applies the EqualFold predicate on the "name" field.
func NameEqualFold(v string) predicate.Product {
	return predicate.Product(sql.FieldEqualFold(FieldName, v))
}

// NameContainsFold applies the ContainsFold predicate on the "name" field.
func NameContainsFold(v string) predicate.Product {
	return predicate.Product(sql.FieldContainsFold(FieldName, v))
}

// MeterSlugEQ applies the EQ predicate on the "meter_slug" field.
func MeterSlugEQ(v string) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldMeterSlug, v))
}

// MeterSlugNEQ applies the NEQ predicate on the "meter_slug" field.
func MeterSlugNEQ(v string) predicate.Product {
	return predicate.Product(sql.FieldNEQ(FieldMeterSlug, v))
}

// MeterSlugIn applies the In predicate on the "meter_slug" field.
func MeterSlugIn(vs ...string) predicate.Product {
	return predicate.Product(sql.FieldIn(FieldMeterSlug, vs...))
}

// MeterSlugNotIn applies the NotIn predicate on the "meter_slug" field.
func MeterSlugNotIn(vs ...string) predicate.Product {
	return predicate.Product(sql.FieldNotIn(FieldMeterSlug, vs...))
}

// MeterSlugGT applies the GT predicate on the "meter_slug" field.
func MeterSlugGT(v string) predicate.Product {
	return predicate.Product(sql.FieldGT(FieldMeterSlug, v))
}

// MeterSlugGTE applies the GTE predicate on the "meter_slug" field.
func MeterSlugGTE(v string) predicate.Product {
	return predicate.Product(sql.FieldGTE(FieldMeterSlug, v))
}

// MeterSlugLT applies the LT predicate on the "meter_slug" field.
func MeterSlugLT(v string) predicate.Product {
	return predicate.Product(sql.FieldLT(FieldMeterSlug, v))
}

// MeterSlugLTE applies the LTE predicate on the "meter_slug" field.
func MeterSlugLTE(v string) predicate.Product {
	return predicate.Product(sql.FieldLTE(FieldMeterSlug, v))
}

// MeterSlugContains applies the Contains predicate on the "meter_slug" field.
func MeterSlugContains(v string) predicate.Product {
	return predicate.Product(sql.FieldContains(FieldMeterSlug, v))
}

// MeterSlugHasPrefix applies the HasPrefix predicate on the "meter_slug" field.
func MeterSlugHasPrefix(v string) predicate.Product {
	return predicate.Product(sql.FieldHasPrefix(FieldMeterSlug, v))
}

// MeterSlugHasSuffix applies the HasSuffix predicate on the "meter_slug" field.
func MeterSlugHasSuffix(v string) predicate.Product {
	return predicate.Product(sql.FieldHasSuffix(FieldMeterSlug, v))
}

// MeterSlugEqualFold applies the EqualFold predicate on the "meter_slug" field.
func MeterSlugEqualFold(v string) predicate.Product {
	return predicate.Product(sql.FieldEqualFold(FieldMeterSlug, v))
}

// MeterSlugContainsFold applies the ContainsFold predicate on the "meter_slug" field.
func MeterSlugContainsFold(v string) predicate.Product {
	return predicate.Product(sql.FieldContainsFold(FieldMeterSlug, v))
}

// MeterGroupByFiltersIsNil applies the IsNil predicate on the "meter_group_by_filters" field.
func MeterGroupByFiltersIsNil() predicate.Product {
	return predicate.Product(sql.FieldIsNull(FieldMeterGroupByFilters))
}

// MeterGroupByFiltersNotNil applies the NotNil predicate on the "meter_group_by_filters" field.
func MeterGroupByFiltersNotNil() predicate.Product {
	return predicate.Product(sql.FieldNotNull(FieldMeterGroupByFilters))
}

// ArchivedEQ applies the EQ predicate on the "archived" field.
func ArchivedEQ(v bool) predicate.Product {
	return predicate.Product(sql.FieldEQ(FieldArchived, v))
}

// ArchivedNEQ applies the NEQ predicate on the "archived" field.
func ArchivedNEQ(v bool) predicate.Product {
	return predicate.Product(sql.FieldNEQ(FieldArchived, v))
}

// HasCreditGrants applies the HasEdge predicate on the "credit_grants" edge.
func HasCreditGrants() predicate.Product {
	return predicate.Product(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.O2M, false, CreditGrantsTable, CreditGrantsColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasCreditGrantsWith applies the HasEdge predicate on the "credit_grants" edge with a given conditions (other predicates).
func HasCreditGrantsWith(preds ...predicate.CreditEntry) predicate.Product {
	return predicate.Product(func(s *sql.Selector) {
		step := newCreditGrantsStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.Product) predicate.Product {
	return predicate.Product(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.Product) predicate.Product {
	return predicate.Product(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.Product) predicate.Product {
	return predicate.Product(sql.NotPredicates(p))
}
