// Code generated by ent, DO NOT EDIT.

package subscriptionaddonratecarditemlink

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldContainsFold(FieldID, id))
}

// CreatedAt applies equality check predicate on the "created_at" field. It's identical to CreatedAtEQ.
func CreatedAt(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldCreatedAt, v))
}

// UpdatedAt applies equality check predicate on the "updated_at" field. It's identical to UpdatedAtEQ.
func UpdatedAt(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldUpdatedAt, v))
}

// DeletedAt applies equality check predicate on the "deleted_at" field. It's identical to DeletedAtEQ.
func DeletedAt(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldDeletedAt, v))
}

// SubscriptionAddonRateCardID applies equality check predicate on the "subscription_addon_rate_card_id" field. It's identical to SubscriptionAddonRateCardIDEQ.
func SubscriptionAddonRateCardID(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionItemID applies equality check predicate on the "subscription_item_id" field. It's identical to SubscriptionItemIDEQ.
func SubscriptionItemID(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldSubscriptionItemID, v))
}

// CreatedAtEQ applies the EQ predicate on the "created_at" field.
func CreatedAtEQ(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldCreatedAt, v))
}

// CreatedAtNEQ applies the NEQ predicate on the "created_at" field.
func CreatedAtNEQ(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNEQ(FieldCreatedAt, v))
}

// CreatedAtIn applies the In predicate on the "created_at" field.
func CreatedAtIn(vs ...time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldIn(FieldCreatedAt, vs...))
}

// CreatedAtNotIn applies the NotIn predicate on the "created_at" field.
func CreatedAtNotIn(vs ...time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNotIn(FieldCreatedAt, vs...))
}

// CreatedAtGT applies the GT predicate on the "created_at" field.
func CreatedAtGT(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGT(FieldCreatedAt, v))
}

// CreatedAtGTE applies the GTE predicate on the "created_at" field.
func CreatedAtGTE(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGTE(FieldCreatedAt, v))
}

// CreatedAtLT applies the LT predicate on the "created_at" field.
func CreatedAtLT(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLT(FieldCreatedAt, v))
}

// CreatedAtLTE applies the LTE predicate on the "created_at" field.
func CreatedAtLTE(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLTE(FieldCreatedAt, v))
}

// UpdatedAtEQ applies the EQ predicate on the "updated_at" field.
func UpdatedAtEQ(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldUpdatedAt, v))
}

// UpdatedAtNEQ applies the NEQ predicate on the "updated_at" field.
func UpdatedAtNEQ(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNEQ(FieldUpdatedAt, v))
}

// UpdatedAtIn applies the In predicate on the "updated_at" field.
func UpdatedAtIn(vs ...time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldIn(FieldUpdatedAt, vs...))
}

// UpdatedAtNotIn applies the NotIn predicate on the "updated_at" field.
func UpdatedAtNotIn(vs ...time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNotIn(FieldUpdatedAt, vs...))
}

// UpdatedAtGT applies the GT predicate on the "updated_at" field.
func UpdatedAtGT(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGT(FieldUpdatedAt, v))
}

// UpdatedAtGTE applies the GTE predicate on the "updated_at" field.
func UpdatedAtGTE(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGTE(FieldUpdatedAt, v))
}

// UpdatedAtLT applies the LT predicate on the "updated_at" field.
func UpdatedAtLT(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLT(FieldUpdatedAt, v))
}

// UpdatedAtLTE applies the LTE predicate on the "updated_at" field.
func UpdatedAtLTE(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLTE(FieldUpdatedAt, v))
}

// DeletedAtEQ applies the EQ predicate on the "deleted_at" field.
func DeletedAtEQ(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldDeletedAt, v))
}

// DeletedAtNEQ applies the NEQ predicate on the "deleted_at" field.
func DeletedAtNEQ(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNEQ(FieldDeletedAt, v))
}

// DeletedAtIn applies the In predicate on the "deleted_at" field.
func DeletedAtIn(vs ...time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldIn(FieldDeletedAt, vs...))
}

// DeletedAtNotIn applies the NotIn predicate on the "deleted_at" field.
func DeletedAtNotIn(vs ...time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNotIn(FieldDeletedAt, vs...))
}

// DeletedAtGT applies the GT predicate on the "deleted_at" field.
func DeletedAtGT(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGT(FieldDeletedAt, v))
}

// DeletedAtGTE applies the GTE predicate on the "deleted_at" field.
func DeletedAtGTE(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGTE(FieldDeletedAt, v))
}

// DeletedAtLT applies the LT predicate on the "deleted_at" field.
func DeletedAtLT(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLT(FieldDeletedAt, v))
}

// DeletedAtLTE applies the LTE predicate on the "deleted_at" field.
func DeletedAtLTE(v time.Time) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLTE(FieldDeletedAt, v))
}

// DeletedAtIsNil applies the IsNil predicate on the "deleted_at" field.
func DeletedAtIsNil() predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldIsNull(FieldDeletedAt))
}

// DeletedAtNotNil applies the NotNil predicate on the "deleted_at" field.
func DeletedAtNotNil() predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNotNull(FieldDeletedAt))
}

// SubscriptionAddonRateCardIDEQ applies the EQ predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDEQ(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionAddonRateCardIDNEQ applies the NEQ predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDNEQ(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNEQ(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionAddonRateCardIDIn applies the In predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDIn(vs ...string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldIn(FieldSubscriptionAddonRateCardID, vs...))
}

// SubscriptionAddonRateCardIDNotIn applies the NotIn predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDNotIn(vs ...string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNotIn(FieldSubscriptionAddonRateCardID, vs...))
}

// SubscriptionAddonRateCardIDGT applies the GT predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDGT(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGT(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionAddonRateCardIDGTE applies the GTE predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDGTE(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGTE(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionAddonRateCardIDLT applies the LT predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDLT(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLT(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionAddonRateCardIDLTE applies the LTE predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDLTE(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLTE(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionAddonRateCardIDContains applies the Contains predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDContains(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldContains(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionAddonRateCardIDHasPrefix applies the HasPrefix predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDHasPrefix(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldHasPrefix(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionAddonRateCardIDHasSuffix applies the HasSuffix predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDHasSuffix(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldHasSuffix(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionAddonRateCardIDEqualFold applies the EqualFold predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDEqualFold(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEqualFold(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionAddonRateCardIDContainsFold applies the ContainsFold predicate on the "subscription_addon_rate_card_id" field.
func SubscriptionAddonRateCardIDContainsFold(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldContainsFold(FieldSubscriptionAddonRateCardID, v))
}

// SubscriptionItemIDEQ applies the EQ predicate on the "subscription_item_id" field.
func SubscriptionItemIDEQ(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEQ(FieldSubscriptionItemID, v))
}

// SubscriptionItemIDNEQ applies the NEQ predicate on the "subscription_item_id" field.
func SubscriptionItemIDNEQ(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNEQ(FieldSubscriptionItemID, v))
}

// SubscriptionItemIDIn applies the In predicate on the "subscription_item_id" field.
func SubscriptionItemIDIn(vs ...string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldIn(FieldSubscriptionItemID, vs...))
}

// SubscriptionItemIDNotIn applies the NotIn predicate on the "subscription_item_id" field.
func SubscriptionItemIDNotIn(vs ...string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldNotIn(FieldSubscriptionItemID, vs...))
}

// SubscriptionItemIDGT applies the GT predicate on the "subscription_item_id" field.
func SubscriptionItemIDGT(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGT(FieldSubscriptionItemID, v))
}

// SubscriptionItemIDGTE applies the GTE predicate on the "subscription_item_id" field.
func SubscriptionItemIDGTE(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldGTE(FieldSubscriptionItemID, v))
}

// SubscriptionItemIDLT applies the LT predicate on the "subscription_item_id" field.
func SubscriptionItemIDLT(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLT(FieldSubscriptionItemID, v))
}

// SubscriptionItemIDLTE applies the LTE predicate on the "subscription_item_id" field.
func SubscriptionItemIDLTE(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldLTE(FieldSubscriptionItemID, v))
}

// SubscriptionItemIDContains applies the Contains predicate on the "subscription_item_id" field.
func SubscriptionItemIDContains(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldContains(FieldSubscriptionItemID, v))
}

// SubscriptionItemIDHasPrefix applies the HasPrefix predicate on the "subscription_item_id" field.
func SubscriptionItemIDHasPrefix(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldHasPrefix(FieldSubscriptionItemID, v))
}

// SubscriptionItemIDHasSuffix applies the HasSuffix predicate on the "subscription_item_id" field.
func SubscriptionItemIDHasSuffix(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldHasSuffix(FieldSubscriptionItemID, v))
}

// SubscriptionItemIDEqualFold applies the EqualFold predicate on the "subscription_item_id" field.
func SubscriptionItemIDEqualFold(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldEqualFold(FieldSubscriptionItemID, v))
}

// SubscriptionItemIDContainsFold applies the ContainsFold predicate on the "subscription_item_id" field.
func SubscriptionItemIDContainsFold(v string) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.FieldContainsFold(FieldSubscriptionItemID, v))
}

// HasSubscriptionAddonRateCard applies the HasEdge predicate on the "subscription_addon_rate_card" edge.
func HasSubscriptionAddonRateCard() predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, SubscriptionAddonRateCardTable, SubscriptionAddonRateCardColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasSubscriptionAddonRateCardWith applies the HasEdge predicate on the "subscription_addon_rate_card" edge with a given conditions (other predicates).
func HasSubscriptionAddonRateCardWith(preds ...predicate.SubscriptionAddonRateCard) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(func(s *sql.Selector) {
		step := newSubscriptionAddonRateCardStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// HasSubscriptionItem applies the HasEdge predicate on the "subscription_item" edge.
func HasSubscriptionItem() predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(func(s *sql.Selector) {
		step := sqlgraph.NewStep(
			sqlgraph.From(Table, FieldID),
			sqlgraph.Edge(sqlgraph.M2O, true, SubscriptionItemTable, SubscriptionItemColumn),
		)
		sqlgraph.HasNeighbors(s, step)
	})
}

// HasSubscriptionItemWith applies the HasEdge predicate on the "subscription_item" edge with a given conditions (other predicates).
func HasSubscriptionItemWith(preds ...predicate.SubscriptionItem) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(func(s *sql.Selector) {
		step := newSubscriptionItemStep()
		sqlgraph.HasNeighborsWith(s, step, func(s *sql.Selector) {
			for _, p := range preds {
				p(s)
			}
		})
	})
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.SubscriptionAddonRateCardItemLink) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.SubscriptionAddonRateCardItemLink) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.SubscriptionAddonRateCardItemLink) predicate.SubscriptionAddonRateCardItemLink {
	return predicate.SubscriptionAddonRateCardItemLink(sql.NotPredicates(p))
}
