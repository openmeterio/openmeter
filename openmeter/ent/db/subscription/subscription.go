// Code generated by ent, DO NOT EDIT.

package subscription

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

const (
	// Label holds the string label denoting the subscription type in the database.
	Label = "subscription"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldNamespace holds the string denoting the namespace field in the database.
	FieldNamespace = "namespace"
	// FieldCreatedAt holds the string denoting the created_at field in the database.
	FieldCreatedAt = "created_at"
	// FieldUpdatedAt holds the string denoting the updated_at field in the database.
	FieldUpdatedAt = "updated_at"
	// FieldDeletedAt holds the string denoting the deleted_at field in the database.
	FieldDeletedAt = "deleted_at"
	// FieldMetadata holds the string denoting the metadata field in the database.
	FieldMetadata = "metadata"
	// FieldActiveFrom holds the string denoting the active_from field in the database.
	FieldActiveFrom = "active_from"
	// FieldActiveTo holds the string denoting the active_to field in the database.
	FieldActiveTo = "active_to"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldPlanID holds the string denoting the plan_id field in the database.
	FieldPlanID = "plan_id"
	// FieldCustomerID holds the string denoting the customer_id field in the database.
	FieldCustomerID = "customer_id"
	// FieldCurrency holds the string denoting the currency field in the database.
	FieldCurrency = "currency"
	// EdgePlan holds the string denoting the plan edge name in mutations.
	EdgePlan = "plan"
	// EdgeCustomer holds the string denoting the customer edge name in mutations.
	EdgeCustomer = "customer"
	// EdgePhases holds the string denoting the phases edge name in mutations.
	EdgePhases = "phases"
	// EdgeBillingLines holds the string denoting the billing_lines edge name in mutations.
	EdgeBillingLines = "billing_lines"
	// Table holds the table name of the subscription in the database.
	Table = "subscriptions"
	// PlanTable is the table that holds the plan relation/edge.
	PlanTable = "subscriptions"
	// PlanInverseTable is the table name for the Plan entity.
	// It exists in this package in order to avoid circular dependency with the "plan" package.
	PlanInverseTable = "plans"
	// PlanColumn is the table column denoting the plan relation/edge.
	PlanColumn = "plan_id"
	// CustomerTable is the table that holds the customer relation/edge.
	CustomerTable = "subscriptions"
	// CustomerInverseTable is the table name for the Customer entity.
	// It exists in this package in order to avoid circular dependency with the "customer" package.
	CustomerInverseTable = "customers"
	// CustomerColumn is the table column denoting the customer relation/edge.
	CustomerColumn = "customer_id"
	// PhasesTable is the table that holds the phases relation/edge.
	PhasesTable = "subscription_phases"
	// PhasesInverseTable is the table name for the SubscriptionPhase entity.
	// It exists in this package in order to avoid circular dependency with the "subscriptionphase" package.
	PhasesInverseTable = "subscription_phases"
	// PhasesColumn is the table column denoting the phases relation/edge.
	PhasesColumn = "subscription_id"
	// BillingLinesTable is the table that holds the billing_lines relation/edge.
	BillingLinesTable = "billing_invoice_lines"
	// BillingLinesInverseTable is the table name for the BillingInvoiceLine entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoiceline" package.
	BillingLinesInverseTable = "billing_invoice_lines"
	// BillingLinesColumn is the table column denoting the billing_lines relation/edge.
	BillingLinesColumn = "subscription_id"
)

// Columns holds all SQL columns for subscription fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldMetadata,
	FieldActiveFrom,
	FieldActiveTo,
	FieldName,
	FieldDescription,
	FieldPlanID,
	FieldCustomerID,
	FieldCurrency,
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
	// DefaultName holds the default value on creation for the "name" field.
	DefaultName string
	// NameValidator is a validator for the "name" field. It is called by the builders before save.
	NameValidator func(string) error
	// CustomerIDValidator is a validator for the "customer_id" field. It is called by the builders before save.
	CustomerIDValidator func(string) error
	// CurrencyValidator is a validator for the "currency" field. It is called by the builders before save.
	CurrencyValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// OrderOption defines the ordering options for the Subscription queries.
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

// ByActiveFrom orders the results by the active_from field.
func ByActiveFrom(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldActiveFrom, opts...).ToFunc()
}

// ByActiveTo orders the results by the active_to field.
func ByActiveTo(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldActiveTo, opts...).ToFunc()
}

// ByName orders the results by the name field.
func ByName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldName, opts...).ToFunc()
}

// ByDescription orders the results by the description field.
func ByDescription(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDescription, opts...).ToFunc()
}

// ByPlanID orders the results by the plan_id field.
func ByPlanID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPlanID, opts...).ToFunc()
}

// ByCustomerID orders the results by the customer_id field.
func ByCustomerID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerID, opts...).ToFunc()
}

// ByCurrency orders the results by the currency field.
func ByCurrency(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCurrency, opts...).ToFunc()
}

// ByPlanField orders the results by plan field.
func ByPlanField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newPlanStep(), sql.OrderByField(field, opts...))
	}
}

// ByCustomerField orders the results by customer field.
func ByCustomerField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newCustomerStep(), sql.OrderByField(field, opts...))
	}
}

// ByPhasesCount orders the results by phases count.
func ByPhasesCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newPhasesStep(), opts...)
	}
}

// ByPhases orders the results by phases terms.
func ByPhases(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newPhasesStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByBillingLinesCount orders the results by billing_lines count.
func ByBillingLinesCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newBillingLinesStep(), opts...)
	}
}

// ByBillingLines orders the results by billing_lines terms.
func ByBillingLines(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingLinesStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newPlanStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(PlanInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, PlanTable, PlanColumn),
	)
}
func newCustomerStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(CustomerInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, CustomerTable, CustomerColumn),
	)
}
func newPhasesStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(PhasesInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, PhasesTable, PhasesColumn),
	)
}
func newBillingLinesStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingLinesInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, BillingLinesTable, BillingLinesColumn),
	)
}
