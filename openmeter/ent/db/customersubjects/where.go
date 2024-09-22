// Code generated by ent, DO NOT EDIT.

package customersubjects

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// ID filters vertices based on their ID field.
func ID(id int) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id int) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id int) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...int) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...int) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id int) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id int) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id int) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id int) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldLTE(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEQ(FieldNamespace, v))
}

// CustomerID applies equality check predicate on the "customer_id" field. It's identical to CustomerIDEQ.
func CustomerID(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEQ(FieldCustomerID, v))
}

// SubjectKey applies equality check predicate on the "subject_key" field. It's identical to SubjectKeyEQ.
func SubjectKey(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEQ(FieldSubjectKey, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEQ(FieldCreatedAt, v))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldContainsFold(FieldNamespace, v))
}

// CustomerIDEQ applies the EQ predicate on the "customer_id" field.
func CustomerIDEQ(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEQ(FieldCustomerID, v))
}

// CustomerIDNEQ applies the NEQ predicate on the "customer_id" field.
func CustomerIDNEQ(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldNEQ(FieldCustomerID, v))
}

// CustomerIDIn applies the In predicate on the "customer_id" field.
func CustomerIDIn(vs ...string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldIn(FieldCustomerID, vs...))
}

// CustomerIDNotIn applies the NotIn predicate on the "customer_id" field.
func CustomerIDNotIn(vs ...string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldNotIn(FieldCustomerID, vs...))
}

// CustomerIDGT applies the GT predicate on the "customer_id" field.
func CustomerIDGT(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldGT(FieldCustomerID, v))
}

// CustomerIDGTE applies the GTE predicate on the "customer_id" field.
func CustomerIDGTE(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldGTE(FieldCustomerID, v))
}

// CustomerIDLT applies the LT predicate on the "customer_id" field.
func CustomerIDLT(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldLT(FieldCustomerID, v))
}

// CustomerIDLTE applies the LTE predicate on the "customer_id" field.
func CustomerIDLTE(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldLTE(FieldCustomerID, v))
}

// CustomerIDContains applies the Contains predicate on the "customer_id" field.
func CustomerIDContains(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldContains(FieldCustomerID, v))
}

// CustomerIDHasPrefix applies the HasPrefix predicate on the "customer_id" field.
func CustomerIDHasPrefix(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldHasPrefix(FieldCustomerID, v))
}

// CustomerIDHasSuffix applies the HasSuffix predicate on the "customer_id" field.
func CustomerIDHasSuffix(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldHasSuffix(FieldCustomerID, v))
}

// CustomerIDEqualFold applies the EqualFold predicate on the "customer_id" field.
func CustomerIDEqualFold(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEqualFold(FieldCustomerID, v))
}

// CustomerIDContainsFold applies the ContainsFold predicate on the "customer_id" field.
func CustomerIDContainsFold(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldContainsFold(FieldCustomerID, v))
}

// SubjectKeyEQ applies the EQ predicate on the "subject_key" field.
func SubjectKeyEQ(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEQ(FieldSubjectKey, v))
}

// SubjectKeyNEQ applies the NEQ predicate on the "subject_key" field.
func SubjectKeyNEQ(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldNEQ(FieldSubjectKey, v))
}

// SubjectKeyIn applies the In predicate on the "subject_key" field.
func SubjectKeyIn(vs ...string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldIn(FieldSubjectKey, vs...))
}

// SubjectKeyNotIn applies the NotIn predicate on the "subject_key" field.
func SubjectKeyNotIn(vs ...string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldNotIn(FieldSubjectKey, vs...))
}

// SubjectKeyGT applies the GT predicate on the "subject_key" field.
func SubjectKeyGT(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldGT(FieldSubjectKey, v))
}

// SubjectKeyGTE applies the GTE predicate on the "subject_key" field.
func SubjectKeyGTE(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldGTE(FieldSubjectKey, v))
}

// SubjectKeyLT applies the LT predicate on the "subject_key" field.
func SubjectKeyLT(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldLT(FieldSubjectKey, v))
}

// SubjectKeyLTE applies the LTE predicate on the "subject_key" field.
func SubjectKeyLTE(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldLTE(FieldSubjectKey, v))
}

// SubjectKeyContains applies the Contains predicate on the "subject_key" field.
func SubjectKeyContains(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldContains(FieldSubjectKey, v))
}

// SubjectKeyHasPrefix applies the HasPrefix predicate on the "subject_key" field.
func SubjectKeyHasPrefix(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldHasPrefix(FieldSubjectKey, v))
}

// SubjectKeyHasSuffix applies the HasSuffix predicate on the "subject_key" field.
func SubjectKeyHasSuffix(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldHasSuffix(FieldSubjectKey, v))
}

// SubjectKeyEqualFold applies the EqualFold predicate on the "subject_key" field.
func SubjectKeyEqualFold(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEqualFold(FieldSubjectKey, v))
}

// SubjectKeyContainsFold applies the ContainsFold predicate on the "subject_key" field.
func SubjectKeyContainsFold(v string) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldContainsFold(FieldSubjectKey, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.FieldLTE(FieldCreatedAt, v))
}

// HasCustomer applies the HasEdge predicate on the "customer" edge.
func HasCustomer() predicate.CustomerSubjects {
	return predicate.CustomerSubjects(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, CustomerTable, CustomerColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasCustomerWith applies the HasEdge predicate on the "customer" edge with a given conditions (other predicates).
func HasCustomerWith(preds ...predicate.Customer) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(func(s *sql.Selector) {
		step := newCustomerStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.CustomerSubjects) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.CustomerSubjects) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.CustomerSubjects) predicate.CustomerSubjects {
	return predicate.CustomerSubjects(sql.NotPredicates(p))
}
