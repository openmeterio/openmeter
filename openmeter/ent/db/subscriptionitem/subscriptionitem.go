// Code generated by ent, DO NOT EDIT.

package subscriptionitem

import (
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

const (
	// Label holds the string label denoting the subscriptionitem type in the database.
	Label = "subscription_item"
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
	// FieldAnnotations holds the string denoting the annotations field in the database.
	FieldAnnotations = "annotations"
	// FieldActiveFrom holds the string denoting the active_from field in the database.
	FieldActiveFrom = "active_from"
	// FieldActiveTo holds the string denoting the active_to field in the database.
	FieldActiveTo = "active_to"
	// FieldPhaseID holds the string denoting the phase_id field in the database.
	FieldPhaseID = "phase_id"
	// FieldKey holds the string denoting the key field in the database.
	FieldKey = "key"
	// FieldEntitlementID holds the string denoting the entitlement_id field in the database.
	FieldEntitlementID = "entitlement_id"
	// FieldRestartsBillingPeriod holds the string denoting the restarts_billing_period field in the database.
	FieldRestartsBillingPeriod = "restarts_billing_period"
	// FieldActiveFromOverrideRelativeToPhaseStart holds the string denoting the active_from_override_relative_to_phase_start field in the database.
	FieldActiveFromOverrideRelativeToPhaseStart = "active_from_override_relative_to_phase_start"
	// FieldActiveToOverrideRelativeToPhaseStart holds the string denoting the active_to_override_relative_to_phase_start field in the database.
	FieldActiveToOverrideRelativeToPhaseStart = "active_to_override_relative_to_phase_start"
	// FieldName holds the string denoting the name field in the database.
	FieldName = "name"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldFeatureKey holds the string denoting the feature_key field in the database.
	FieldFeatureKey = "feature_key"
	// FieldEntitlementTemplate holds the string denoting the entitlement_template field in the database.
	FieldEntitlementTemplate = "entitlement_template"
	// FieldTaxConfig holds the string denoting the tax_config field in the database.
	FieldTaxConfig = "tax_config"
	// FieldBillingCadence holds the string denoting the billing_cadence field in the database.
	FieldBillingCadence = "billing_cadence"
	// FieldPrice holds the string denoting the price field in the database.
	FieldPrice = "price"
	// FieldDiscounts holds the string denoting the discounts field in the database.
	FieldDiscounts = "discounts"
	// EdgePhase holds the string denoting the phase edge name in mutations.
	EdgePhase = "phase"
	// EdgeEntitlement holds the string denoting the entitlement edge name in mutations.
	EdgeEntitlement = "entitlement"
	// EdgeBillingLines holds the string denoting the billing_lines edge name in mutations.
	EdgeBillingLines = "billing_lines"
	// Table holds the table name of the subscriptionitem in the database.
	Table = "subscription_items"
	// PhaseTable is the table that holds the phase relation/edge.
	PhaseTable = "subscription_items"
	// PhaseInverseTable is the table name for the SubscriptionPhase entity.
	// It exists in this package in order to avoid circular dependency with the "subscriptionphase" package.
	PhaseInverseTable = "subscription_phases"
	// PhaseColumn is the table column denoting the phase relation/edge.
	PhaseColumn = "phase_id"
	// EntitlementTable is the table that holds the entitlement relation/edge.
	EntitlementTable = "subscription_items"
	// EntitlementInverseTable is the table name for the Entitlement entity.
	// It exists in this package in order to avoid circular dependency with the "entitlement" package.
	EntitlementInverseTable = "entitlements"
	// EntitlementColumn is the table column denoting the entitlement relation/edge.
	EntitlementColumn = "entitlement_id"
	// BillingLinesTable is the table that holds the billing_lines relation/edge.
	BillingLinesTable = "billing_invoice_lines"
	// BillingLinesInverseTable is the table name for the BillingInvoiceLine entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoiceline" package.
	BillingLinesInverseTable = "billing_invoice_lines"
	// BillingLinesColumn is the table column denoting the billing_lines relation/edge.
	BillingLinesColumn = "subscription_item_id"
)

// Columns holds all SQL columns for subscriptionitem fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldMetadata,
	FieldAnnotations,
	FieldActiveFrom,
	FieldActiveTo,
	FieldPhaseID,
	FieldKey,
	FieldEntitlementID,
	FieldRestartsBillingPeriod,
	FieldActiveFromOverrideRelativeToPhaseStart,
	FieldActiveToOverrideRelativeToPhaseStart,
	FieldName,
	FieldDescription,
	FieldFeatureKey,
	FieldEntitlementTemplate,
	FieldTaxConfig,
	FieldBillingCadence,
	FieldPrice,
	FieldDiscounts,
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
	// PhaseIDValidator is a validator for the "phase_id" field. It is called by the builders before save.
	PhaseIDValidator func(string) error
	// KeyValidator is a validator for the "key" field. It is called by the builders before save.
	KeyValidator func(string) error
	// NameValidator is a validator for the "name" field. It is called by the builders before save.
	NameValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
	// ValueScanner of all SubscriptionItem fields.
	ValueScanner struct {
		Annotations         field.TypeValueScanner[map[string]interface{}]
		EntitlementTemplate field.TypeValueScanner[*productcatalog.EntitlementTemplate]
		TaxConfig           field.TypeValueScanner[*productcatalog.TaxConfig]
		Price               field.TypeValueScanner[*productcatalog.Price]
		Discounts           field.TypeValueScanner[*productcatalog.Discounts]
	}
)

// OrderOption defines the ordering options for the SubscriptionItem queries.
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

// ByAnnotations orders the results by the annotations field.
func ByAnnotations(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldAnnotations, opts...).ToFunc()
}

// ByActiveFrom orders the results by the active_from field.
func ByActiveFrom(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldActiveFrom, opts...).ToFunc()
}

// ByActiveTo orders the results by the active_to field.
func ByActiveTo(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldActiveTo, opts...).ToFunc()
}

// ByPhaseID orders the results by the phase_id field.
func ByPhaseID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPhaseID, opts...).ToFunc()
}

// ByKey orders the results by the key field.
func ByKey(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldKey, opts...).ToFunc()
}

// ByEntitlementID orders the results by the entitlement_id field.
func ByEntitlementID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldEntitlementID, opts...).ToFunc()
}

// ByRestartsBillingPeriod orders the results by the restarts_billing_period field.
func ByRestartsBillingPeriod(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldRestartsBillingPeriod, opts...).ToFunc()
}

// ByActiveFromOverrideRelativeToPhaseStart orders the results by the active_from_override_relative_to_phase_start field.
func ByActiveFromOverrideRelativeToPhaseStart(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldActiveFromOverrideRelativeToPhaseStart, opts...).ToFunc()
}

// ByActiveToOverrideRelativeToPhaseStart orders the results by the active_to_override_relative_to_phase_start field.
func ByActiveToOverrideRelativeToPhaseStart(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldActiveToOverrideRelativeToPhaseStart, opts...).ToFunc()
}

// ByName orders the results by the name field.
func ByName(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldName, opts...).ToFunc()
}

// ByDescription orders the results by the description field.
func ByDescription(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDescription, opts...).ToFunc()
}

// ByFeatureKey orders the results by the feature_key field.
func ByFeatureKey(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldFeatureKey, opts...).ToFunc()
}

// ByEntitlementTemplate orders the results by the entitlement_template field.
func ByEntitlementTemplate(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldEntitlementTemplate, opts...).ToFunc()
}

// ByTaxConfig orders the results by the tax_config field.
func ByTaxConfig(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTaxConfig, opts...).ToFunc()
}

// ByBillingCadence orders the results by the billing_cadence field.
func ByBillingCadence(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldBillingCadence, opts...).ToFunc()
}

// ByPrice orders the results by the price field.
func ByPrice(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPrice, opts...).ToFunc()
}

// ByDiscounts orders the results by the discounts field.
func ByDiscounts(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDiscounts, opts...).ToFunc()
}

// ByPhaseField orders the results by phase field.
func ByPhaseField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newPhaseStep(), sql.OrderByField(field, opts...))
	}
}

// ByEntitlementField orders the results by entitlement field.
func ByEntitlementField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newEntitlementStep(), sql.OrderByField(field, opts...))
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
func newPhaseStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(PhaseInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, PhaseTable, PhaseColumn),
	)
}
func newEntitlementStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(EntitlementInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, EntitlementTable, EntitlementColumn),
	)
}
func newBillingLinesStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingLinesInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, BillingLinesTable, BillingLinesColumn),
	)
}
