// Code generated by ent, DO NOT EDIT.

package subscriptionaddonquantity

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldContainsFold(FieldID, id))
}

// Namespace applies equality check predicate on the "namespace" field. It's identical to NamespaceEQ.
func Namespace(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldNamespace, v))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldUpdatedAt, v))
}

// DeletedAt applies equality check predicate on the "deleted_at" field. It's identical to DeletedAtEQ.
func DeletedAt(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldDeletedAt, v))
}

// ActiveFrom applies equality check predicate on the "active_from" field. It's identical to ActiveFromEQ.
func ActiveFrom(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldActiveFrom, v))
}

// Quantity applies equality check predicate on the "quantity" field. It's identical to QuantityEQ.
func Quantity(v int) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldQuantity, v))
}

// SubscriptionAddonID applies equality check predicate on the "subscription_addon_id" field. It's identical to SubscriptionAddonIDEQ.
func SubscriptionAddonID(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldSubscriptionAddonID, v))
}

// NamespaceEQ applies the EQ predicate on the "namespace" field.
func NamespaceEQ(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldNamespace, v))
}

// NamespaceNEQ applies the NEQ predicate on the "namespace" field.
func NamespaceNEQ(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNEQ(FieldNamespace, v))
}

// NamespaceIn applies the In predicate on the "namespace" field.
func NamespaceIn(vs ...string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldIn(FieldNamespace, vs...))
}

// NamespaceNotIn applies the NotIn predicate on the "namespace" field.
func NamespaceNotIn(vs ...string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNotIn(FieldNamespace, vs...))
}

// NamespaceGT applies the GT predicate on the "namespace" field.
func NamespaceGT(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGT(FieldNamespace, v))
}

// NamespaceGTE applies the GTE predicate on the "namespace" field.
func NamespaceGTE(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGTE(FieldNamespace, v))
}

// NamespaceLT applies the LT predicate on the "namespace" field.
func NamespaceLT(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLT(FieldNamespace, v))
}

// NamespaceLTE applies the LTE predicate on the "namespace" field.
func NamespaceLTE(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLTE(FieldNamespace, v))
}

// NamespaceContains applies the Contains predicate on the "namespace" field.
func NamespaceContains(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldContains(FieldNamespace, v))
}

// NamespaceHasPrefix applies the HasPrefix predicate on the "namespace" field.
func NamespaceHasPrefix(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldHasPrefix(FieldNamespace, v))
}

// NamespaceHasSuffix applies the HasSuffix predicate on the "namespace" field.
func NamespaceHasSuffix(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldHasSuffix(FieldNamespace, v))
}

// NamespaceEqualFold applies the EqualFold predicate on the "namespace" field.
func NamespaceEqualFold(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEqualFold(FieldNamespace, v))
}

// NamespaceContainsFold applies the ContainsFold predicate on the "namespace" field.
func NamespaceContainsFold(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldContainsFold(FieldNamespace, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLTE(FieldUpdatedAt, v))
}

// DeletedAtEQ applies the EQ predicate on the "deleted_at" field.
func DeletedAtEQ(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldDeletedAt, v))
}

// DeletedAtNEQ applies the NEQ predicate on the "deleted_at" field.
func DeletedAtNEQ(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNEQ(FieldDeletedAt, v))
}

// DeletedAtIn applies the In predicate on the "deleted_at" field.
func DeletedAtIn(vs ...time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldIn(FieldDeletedAt, vs...))
}

// DeletedAtNotIn applies the NotIn predicate on the "deleted_at" field.
func DeletedAtNotIn(vs ...time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNotIn(FieldDeletedAt, vs...))
}

// DeletedAtGT applies the GT predicate on the "deleted_at" field.
func DeletedAtGT(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGT(FieldDeletedAt, v))
}

// DeletedAtGTE applies the GTE predicate on the "deleted_at" field.
func DeletedAtGTE(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGTE(FieldDeletedAt, v))
}

// DeletedAtLT applies the LT predicate on the "deleted_at" field.
func DeletedAtLT(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLT(FieldDeletedAt, v))
}

// DeletedAtLTE applies the LTE predicate on the "deleted_at" field.
func DeletedAtLTE(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLTE(FieldDeletedAt, v))
}

// DeletedAtIsNil applies the IsNil predicate on the "deleted_at" field.
func DeletedAtIsNil() predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldIsNull(FieldDeletedAt))
}

// DeletedAtNotNil applies the NotNil predicate on the "deleted_at" field.
func DeletedAtNotNil() predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNotNull(FieldDeletedAt))
}

// ActiveFromEQ applies the EQ predicate on the "active_from" field.
func ActiveFromEQ(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldActiveFrom, v))
}

// ActiveFromNEQ applies the NEQ predicate on the "active_from" field.
func ActiveFromNEQ(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNEQ(FieldActiveFrom, v))
}

// ActiveFromIn applies the In predicate on the "active_from" field.
func ActiveFromIn(vs ...time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldIn(FieldActiveFrom, vs...))
}

// ActiveFromNotIn applies the NotIn predicate on the "active_from" field.
func ActiveFromNotIn(vs ...time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNotIn(FieldActiveFrom, vs...))
}

// ActiveFromGT applies the GT predicate on the "active_from" field.
func ActiveFromGT(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGT(FieldActiveFrom, v))
}

// ActiveFromGTE applies the GTE predicate on the "active_from" field.
func ActiveFromGTE(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGTE(FieldActiveFrom, v))
}

// ActiveFromLT applies the LT predicate on the "active_from" field.
func ActiveFromLT(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLT(FieldActiveFrom, v))
}

// ActiveFromLTE applies the LTE predicate on the "active_from" field.
func ActiveFromLTE(v time.Time) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLTE(FieldActiveFrom, v))
}

// QuantityEQ applies the EQ predicate on the "quantity" field.
func QuantityEQ(v int) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldQuantity, v))
}

// QuantityNEQ applies the NEQ predicate on the "quantity" field.
func QuantityNEQ(v int) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNEQ(FieldQuantity, v))
}

// QuantityIn applies the In predicate on the "quantity" field.
func QuantityIn(vs ...int) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldIn(FieldQuantity, vs...))
}

// QuantityNotIn applies the NotIn predicate on the "quantity" field.
func QuantityNotIn(vs ...int) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNotIn(FieldQuantity, vs...))
}

// QuantityGT applies the GT predicate on the "quantity" field.
func QuantityGT(v int) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGT(FieldQuantity, v))
}

// QuantityGTE applies the GTE predicate on the "quantity" field.
func QuantityGTE(v int) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGTE(FieldQuantity, v))
}

// QuantityLT applies the LT predicate on the "quantity" field.
func QuantityLT(v int) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLT(FieldQuantity, v))
}

// QuantityLTE applies the LTE predicate on the "quantity" field.
func QuantityLTE(v int) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLTE(FieldQuantity, v))
}

// SubscriptionAddonIDEQ applies the EQ predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDEQ(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEQ(FieldSubscriptionAddonID, v))
}

// SubscriptionAddonIDNEQ applies the NEQ predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDNEQ(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNEQ(FieldSubscriptionAddonID, v))
}

// SubscriptionAddonIDIn applies the In predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDIn(vs ...string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldIn(FieldSubscriptionAddonID, vs...))
}

// SubscriptionAddonIDNotIn applies the NotIn predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDNotIn(vs ...string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldNotIn(FieldSubscriptionAddonID, vs...))
}

// SubscriptionAddonIDGT applies the GT predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDGT(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGT(FieldSubscriptionAddonID, v))
}

// SubscriptionAddonIDGTE applies the GTE predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDGTE(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldGTE(FieldSubscriptionAddonID, v))
}

// SubscriptionAddonIDLT applies the LT predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDLT(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLT(FieldSubscriptionAddonID, v))
}

// SubscriptionAddonIDLTE applies the LTE predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDLTE(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldLTE(FieldSubscriptionAddonID, v))
}

// SubscriptionAddonIDContains applies the Contains predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDContains(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldContains(FieldSubscriptionAddonID, v))
}

// SubscriptionAddonIDHasPrefix applies the HasPrefix predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDHasPrefix(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldHasPrefix(FieldSubscriptionAddonID, v))
}

// SubscriptionAddonIDHasSuffix applies the HasSuffix predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDHasSuffix(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldHasSuffix(FieldSubscriptionAddonID, v))
}

// SubscriptionAddonIDEqualFold applies the EqualFold predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDEqualFold(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldEqualFold(FieldSubscriptionAddonID, v))
}

// SubscriptionAddonIDContainsFold applies the ContainsFold predicate on the "subscription_addon_id" field.
func SubscriptionAddonIDContainsFold(v string) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.FieldContainsFold(FieldSubscriptionAddonID, v))
}

// HasSubscriptionAddon applies the HasEdge predicate on the "subscription_addon" edge.
func HasSubscriptionAddon() predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, SubscriptionAddonTable, SubscriptionAddonColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasSubscriptionAddonWith applies the HasEdge predicate on the "subscription_addon" edge with a given conditions (other predicates).
func HasSubscriptionAddonWith(preds ...predicate.SubscriptionAddon) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(func(s *sql.Selector) {
		step := newSubscriptionAddonStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.SubscriptionAddonQuantity) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.SubscriptionAddonQuantity) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.SubscriptionAddonQuantity) predicate.SubscriptionAddonQuantity {
	return predicate.SubscriptionAddonQuantity(sql.NotPredicates(p))
}
