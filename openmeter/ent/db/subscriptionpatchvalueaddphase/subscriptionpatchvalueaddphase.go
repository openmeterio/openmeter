// Code generated by ent, DO NOT EDIT.

package subscriptionpatchvalueaddphase

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

const (
	// Label holds the string label denoting the subscriptionpatchvalueaddphase type in the database.
	Label = "subscription_patch_value_add_phase"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldNamespace holds the string denoting the namespace field in the database.
	FieldNamespace = "namespace"
	// FieldSubscriptionPatchID holds the string denoting the subscription_patch_id field in the database.
	FieldSubscriptionPatchID = "subscription_patch_id"
	// FieldPhaseKey holds the string denoting the phase_key field in the database.
	FieldPhaseKey = "phase_key"
	// FieldStartAfterIso holds the string denoting the start_after_iso field in the database.
	FieldStartAfterIso = "start_after_iso"
	// FieldDurationIso holds the string denoting the duration_iso field in the database.
	FieldDurationIso = "duration_iso"
	// FieldCreateDiscount holds the string denoting the create_discount field in the database.
	FieldCreateDiscount = "create_discount"
	// FieldCreateDiscountAppliesTo holds the string denoting the create_discount_applies_to field in the database.
	FieldCreateDiscountAppliesTo = "create_discount_applies_to"
	// EdgeSubscriptionPatch holds the string denoting the subscription_patch edge name in mutations.
	EdgeSubscriptionPatch = "subscription_patch"
	// Table holds the table name of the subscriptionpatchvalueaddphase in the database.
	Table = "subscription_patch_value_add_phases"
	// SubscriptionPatchTable is the table that holds the subscription_patch relation/edge.
	SubscriptionPatchTable = "subscription_patch_value_add_phases"
	// SubscriptionPatchInverseTable is the table name for the SubscriptionPatch entity.
	// It exists in this package in order to avoid circular dependency with the "subscriptionpatch" package.
	SubscriptionPatchInverseTable = "subscription_patches"
	// SubscriptionPatchColumn is the table column denoting the subscription_patch relation/edge.
	SubscriptionPatchColumn = "subscription_patch_id"
)

// Columns holds all SQL columns for subscriptionpatchvalueaddphase fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldSubscriptionPatchID,
	FieldPhaseKey,
	FieldStartAfterIso,
	FieldDurationIso,
	FieldCreateDiscount,
	FieldCreateDiscountAppliesTo,
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

var (
	// NamespaceValidator is a validator for the "namespace" field. It is called by the builders before save.
	NamespaceValidator func(string) error
	// SubscriptionPatchIDValidator is a validator for the "subscription_patch_id" field. It is called by the builders before save.
	SubscriptionPatchIDValidator func(string) error
	// PhaseKeyValidator is a validator for the "phase_key" field. It is called by the builders before save.
	PhaseKeyValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// OrderOption defines the ordering options for the SubscriptionPatchValueAddPhase queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByNamespace orders the results by the namespace field.
func ByNamespace(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldNamespace, opts...).ToFunc()
}

// BySubscriptionPatchID orders the results by the subscription_patch_id field.
func BySubscriptionPatchID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSubscriptionPatchID, opts...).ToFunc()
}

// ByPhaseKey orders the results by the phase_key field.
func ByPhaseKey(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPhaseKey, opts...).ToFunc()
}

// ByStartAfterIso orders the results by the start_after_iso field.
func ByStartAfterIso(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldStartAfterIso, opts...).ToFunc()
}

// ByDurationIso orders the results by the duration_iso field.
func ByDurationIso(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDurationIso, opts...).ToFunc()
}

// ByCreateDiscount orders the results by the create_discount field.
func ByCreateDiscount(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCreateDiscount, opts...).ToFunc()
}

// BySubscriptionPatchField orders the results by subscription_patch field.
func BySubscriptionPatchField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newSubscriptionPatchStep(), sql.OrderByField(field, opts...))
	}
}
func newSubscriptionPatchStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(SubscriptionPatchInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2O, true, SubscriptionPatchTable, SubscriptionPatchColumn),
	)
}
