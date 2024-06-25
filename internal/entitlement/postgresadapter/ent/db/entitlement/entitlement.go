// Code generated by ent, DO NOT EDIT.

package entitlement

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
)

const (
	// Label holds the string label denoting the entitlement type in the database.
	Label = "entitlement"
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
	// FieldEntitlementType holds the string denoting the entitlement_type field in the database.
	FieldEntitlementType = "entitlement_type"
	// FieldFeatureID holds the string denoting the feature_id field in the database.
	FieldFeatureID = "feature_id"
	// FieldSubjectKey holds the string denoting the subject_key field in the database.
	FieldSubjectKey = "subject_key"
	// FieldMeasureUsageFrom holds the string denoting the measure_usage_from field in the database.
	FieldMeasureUsageFrom = "measure_usage_from"
	// FieldIssueAfterReset holds the string denoting the issue_after_reset field in the database.
	FieldIssueAfterReset = "issue_after_reset"
	// FieldIsSoftLimit holds the string denoting the is_soft_limit field in the database.
	FieldIsSoftLimit = "is_soft_limit"
	// FieldConfig holds the string denoting the config field in the database.
	FieldConfig = "config"
	// FieldUsagePeriodInterval holds the string denoting the usage_period_interval field in the database.
	FieldUsagePeriodInterval = "usage_period_interval"
	// FieldUsagePeriodAnchor holds the string denoting the usage_period_anchor field in the database.
	FieldUsagePeriodAnchor = "usage_period_anchor"
	// EdgeUsageReset holds the string denoting the usage_reset edge name in mutations.
	EdgeUsageReset = "usage_reset"
	// Table holds the table name of the entitlement in the database.
	Table = "entitlements"
	// UsageResetTable is the table that holds the usage_reset relation/edge.
	UsageResetTable = "usage_resets"
	// UsageResetInverseTable is the table name for the UsageReset entity.
	// It exists in this package in order to avoid circular dependency with the "usagereset" package.
	UsageResetInverseTable = "usage_resets"
	// UsageResetColumn is the table column denoting the usage_reset relation/edge.
	UsageResetColumn = "entitlement_id"
)

// Columns holds all SQL columns for entitlement fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldMetadata,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldEntitlementType,
	FieldFeatureID,
	FieldSubjectKey,
	FieldMeasureUsageFrom,
	FieldIssueAfterReset,
	FieldIsSoftLimit,
	FieldConfig,
	FieldUsagePeriodInterval,
	FieldUsagePeriodAnchor,
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
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// EntitlementType defines the type for the "entitlement_type" enum field.
type EntitlementType string

// EntitlementType values.
const (
	EntitlementTypeMetered EntitlementType = "metered"
	EntitlementTypeStatic  EntitlementType = "static"
	EntitlementTypeBoolean EntitlementType = "boolean"
)

func (et EntitlementType) String() string {
	return string(et)
}

// EntitlementTypeValidator is a validator for the "entitlement_type" field enum values. It is called by the builders before save.
func EntitlementTypeValidator(et EntitlementType) error {
	switch et {
	case EntitlementTypeMetered, EntitlementTypeStatic, EntitlementTypeBoolean:
		return nil
	default:
		return fmt.Errorf("entitlement: invalid enum value for entitlement_type field: %q", et)
	}
}

// UsagePeriodInterval defines the type for the "usage_period_interval" enum field.
type UsagePeriodInterval string

// UsagePeriodInterval values.
const (
	UsagePeriodIntervalDAY   UsagePeriodInterval = "DAY"
	UsagePeriodIntervalWEEK  UsagePeriodInterval = "WEEK"
	UsagePeriodIntervalMONTH UsagePeriodInterval = "MONTH"
	UsagePeriodIntervalYEAR  UsagePeriodInterval = "YEAR"
)

func (upi UsagePeriodInterval) String() string {
	return string(upi)
}

// UsagePeriodIntervalValidator is a validator for the "usage_period_interval" field enum values. It is called by the builders before save.
func UsagePeriodIntervalValidator(upi UsagePeriodInterval) error {
	switch upi {
	case UsagePeriodIntervalDAY, UsagePeriodIntervalWEEK, UsagePeriodIntervalMONTH, UsagePeriodIntervalYEAR:
		return nil
	default:
		return fmt.Errorf("entitlement: invalid enum value for usage_period_interval field: %q", upi)
	}
}

// OrderOption defines the ordering options for the Entitlement queries.
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

// ByEntitlementType orders the results by the entitlement_type field.
func ByEntitlementType(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldEntitlementType, opts...).ToFunc()
}

// ByFeatureID orders the results by the feature_id field.
func ByFeatureID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldFeatureID, opts...).ToFunc()
}

// BySubjectKey orders the results by the subject_key field.
func BySubjectKey(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSubjectKey, opts...).ToFunc()
}

// ByMeasureUsageFrom orders the results by the measure_usage_from field.
func ByMeasureUsageFrom(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldMeasureUsageFrom, opts...).ToFunc()
}

// ByIssueAfterReset orders the results by the issue_after_reset field.
func ByIssueAfterReset(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldIssueAfterReset, opts...).ToFunc()
}

// ByIsSoftLimit orders the results by the is_soft_limit field.
func ByIsSoftLimit(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldIsSoftLimit, opts...).ToFunc()
}

// ByConfig orders the results by the config field.
func ByConfig(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldConfig, opts...).ToFunc()
}

// ByUsagePeriodInterval orders the results by the usage_period_interval field.
func ByUsagePeriodInterval(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUsagePeriodInterval, opts...).ToFunc()
}

// ByUsagePeriodAnchor orders the results by the usage_period_anchor field.
func ByUsagePeriodAnchor(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldUsagePeriodAnchor, opts...).ToFunc()
}

// ByUsageResetCount orders the results by usage_reset count.
func ByUsageResetCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newUsageResetStep(), opts...)
	}
}

// ByUsageReset orders the results by usage_reset terms.
func ByUsageReset(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newUsageResetStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newUsageResetStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(UsageResetInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, UsageResetTable, UsageResetColumn),
	)
}
