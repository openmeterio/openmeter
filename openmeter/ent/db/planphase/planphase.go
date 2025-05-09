// Code generated by ent, DO NOT EDIT.

package planphase

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

const (
	// Label holds the string label denoting the planphase type in the database.
	Label = "plan_phase"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldNamespace holds the string denoting the namespace field in the database.
	FieldNamespace = "namespace"
	// FieldMetadata holds the string denoting the metadata field in the database.
	FieldMetadata = "metadata"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// FieldUpdatedAt holds the string denoting the updated_at field in the database.
	FieldUpdatedAt = "updated_at"
	// FieldDeletedAt holds the string denoting the deleted_at field in the database.
	FieldDeletedAt = "deleted_at"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldKey holds the string denoting the key field in the database.
	FieldKey = "key"
	// FieldPlanID holds the string denoting the plan_id field in the database.
	FieldPlanID = "plan_id"
	// FieldIndex holds the string denoting the index field in the database.
	FieldIndex = "index"
	// FieldDuration holds the string denoting the duration field in the database.
	FieldDuration = "duration"
	// EdgePlan holds the string denoting the plan edge name in mutations.
	EdgePlan = "plan"
	// EdgeRatecards holds the string denoting the ratecards edge name in mutations.
	EdgeRatecards = "ratecards"
	// Table holds the table name of the planphase in the database.
	Table = "plan_phases"
	// PlanTable is the table that holds the plan relation/edge.
	PlanTable = "plan_phases"
	// PlanInverseTable is the table name for the Plan entity.
	// It exists in this package in order to avoid circular dependency with the "plan" package.
	PlanInverseTable = "plans"
	// PlanColumn is the table column denoting the plan relation/edge.
	PlanColumn = "plan_id"
	// RatecardsTable is the table that holds the ratecards relation/edge.
	RatecardsTable = "plan_rate_cards"
	// RatecardsInverseTable is the table name for the PlanRateCard entity.
	// It exists in this package in order to avoid circular dependency with the "planratecard" package.
	RatecardsInverseTable = "plan_rate_cards"
	// RatecardsColumn is the table column denoting the ratecards relation/edge.
	RatecardsColumn = "phase_id"
)

// Columns holds all SQL columns for planphase fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldMetadata,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldName,
	FieldDescription,
	FieldKey,
	FieldPlanID,
	FieldIndex,
	FieldDuration,
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
	// KeyValidator is a validator for the "key" field. It is called by the builders before save.
	KeyValidator func(string) error
	// PlanIDValidator is a validator for the "plan_id" field. It is called by the builders before save.
	PlanIDValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// OrderOption defines the ordering options for the PlanPhase queries.
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

// ByName orders the results by the name field.
func ByName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldName, opts...).ToFunc()
}

// ByDescription orders the results by the description field.
func ByDescription(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDescription, opts...).ToFunc()
}

// ByKey orders the results by the key field.
func ByKey(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldKey, opts...).ToFunc()
}

// ByPlanID orders the results by the plan_id field.
func ByPlanID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPlanID, opts...).ToFunc()
}

// ByIndex orders the results by the index field.
func ByIndex(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldIndex, opts...).ToFunc()
}

// ByDuration orders the results by the duration field.
func ByDuration(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDuration, opts...).ToFunc()
}

// ByPlanField orders the results by plan field.
func ByPlanField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newPlanStep(), sql.OrderByField(field, opts...))
	}
}

// ByRatecardsCount orders the results by ratecards count.
func ByRatecardsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newRatecardsStep(), opts...)
	}
}

// ByRatecards orders the results by ratecards terms.
func ByRatecards(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newRatecardsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newPlanStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(PlanInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, PlanTable, PlanColumn),
	)
}
func newRatecardsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(RatecardsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, RatecardsTable, RatecardsColumn),
	)
}
