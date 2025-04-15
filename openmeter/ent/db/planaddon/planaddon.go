// Code generated by ent, DO NOT EDIT.

package planaddon

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

const (
	// Label holds the string label denoting the planaddon type in the database.
	Label = "plan_addon"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldNamespace holds the string denoting the namespace field in the database.
	FieldNamespace = "namespace"
	// FieldMetadata holds the string denoting the metadata field in the database.
	FieldMetadata = "metadata"
	// FieldAnnotations holds the string denoting the annotations field in the database.
	FieldAnnotations = "annotations"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// FieldUpdatedAt holds the string denoting the updated_at field in the database.
	FieldUpdatedAt = "updated_at"
	// FieldDeletedAt holds the string denoting the deleted_at field in the database.
	FieldDeletedAt = "deleted_at"
	// FieldPlanID holds the string denoting the plan_id field in the database.
	FieldPlanID = "plan_id"
	// FieldAddonID holds the string denoting the addon_id field in the database.
	FieldAddonID = "addon_id"
	// FieldFromPlanPhase holds the string denoting the from_plan_phase field in the database.
	FieldFromPlanPhase = "from_plan_phase"
	// FieldMaxQuantity holds the string denoting the max_quantity field in the database.
	FieldMaxQuantity = "max_quantity"
	// EdgePlan holds the string denoting the plan edge name in mutations.
	EdgePlan = "plan"
	// EdgeAddon holds the string denoting the addon edge name in mutations.
	EdgeAddon = "addon"
	// Table holds the table name of the planaddon in the database.
	Table = "plan_addons"
	// PlanTable is the table that holds the plan relation/edge.
	PlanTable = "plan_addons"
	// PlanInverseTable is the table name for the Plan entity.
	// It exists in this package in order to avoid circular dependency with the "plan" package.
	PlanInverseTable = "plans"
	// PlanColumn is the table column denoting the plan relation/edge.
	PlanColumn = "plan_id"
	// AddonTable is the table that holds the addon relation/edge.
	AddonTable = "plan_addons"
	// AddonInverseTable is the table name for the Addon entity.
	// It exists in this package in order to avoid circular dependency with the "addon" package.
	AddonInverseTable = "addons"
	// AddonColumn is the table column denoting the addon relation/edge.
	AddonColumn = "addon_id"
)

// Columns holds all SQL columns for planaddon fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldMetadata,
	FieldAnnotations,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldPlanID,
	FieldAddonID,
	FieldFromPlanPhase,
	FieldMaxQuantity,
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
	// DefaultCreatedAt holds the default value on creation for the "created_at" field.
	DefaultCreatedAt func() time.Time
	// DefaultUpdatedAt holds the default value on creation for the "updated_at" field.
	DefaultUpdatedAt func() time.Time
	// UpdateDefaultUpdatedAt holds the default value on update for the "updated_at" field.
	UpdateDefaultUpdatedAt func() time.Time
	// PlanIDValidator is a validator for the "plan_id" field. It is called by the builders before save.
	PlanIDValidator func(string) error
	// AddonIDValidator is a validator for the "addon_id" field. It is called by the builders before save.
	AddonIDValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// OrderOption defines the ordering options for the PlanAddon queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByNamespace orders the results by the namespace field.
func ByNamespace(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldNamespace, opts...).ToFunc()
}

// ByCreatedAt orders the results by the created_at field.
func ByCreatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCreatedAt, opts...).ToFunc()
}

// ByUpdatedAt orders the results by the updated_at field.
func ByUpdatedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUpdatedAt, opts...).ToFunc()
}

// ByDeletedAt orders the results by the deleted_at field.
func ByDeletedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDeletedAt, opts...).ToFunc()
}

// ByPlanID orders the results by the plan_id field.
func ByPlanID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPlanID, opts...).ToFunc()
}

// ByAddonID orders the results by the addon_id field.
func ByAddonID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldAddonID, opts...).ToFunc()
}

// ByFromPlanPhase orders the results by the from_plan_phase field.
func ByFromPlanPhase(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldFromPlanPhase, opts...).ToFunc()
}

// ByMaxQuantity orders the results by the max_quantity field.
func ByMaxQuantity(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldMaxQuantity, opts...).ToFunc()
}

// ByPlanField orders the results by plan field.
func ByPlanField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newPlanStep(), sql.OrderByField(field, opts...))
	}
}

// ByAddonField orders the results by addon field.
func ByAddonField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newAddonStep(), sql.OrderByField(field, opts...))
	}
}
func newPlanStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(PlanInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, PlanTable, PlanColumn),
	)
}
func newAddonStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(AddonInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, AddonTable, AddonColumn),
	)
}
