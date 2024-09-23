// Code generated by ent, DO NOT EDIT.

package billingprofile

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
)

const (
	// Label holds the string label denoting the billingprofile type in the database.
	Label = "billing_profile"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldKey holds the string denoting the key field in the database.
	FieldKey = "key"
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
	// FieldTaxProvider holds the string denoting the tax_provider field in the database.
	FieldTaxProvider = "tax_provider"
	// FieldTaxProviderConfig holds the string denoting the tax_provider_config field in the database.
	FieldTaxProviderConfig = "tax_provider_config"
	// FieldInvoicingProvider holds the string denoting the invoicing_provider field in the database.
	FieldInvoicingProvider = "invoicing_provider"
	// FieldInvoicingProviderConfig holds the string denoting the invoicing_provider_config field in the database.
	FieldInvoicingProviderConfig = "invoicing_provider_config"
	// FieldPaymentProvider holds the string denoting the payment_provider field in the database.
	FieldPaymentProvider = "payment_provider"
	// FieldPaymentProviderConfig holds the string denoting the payment_provider_config field in the database.
	FieldPaymentProviderConfig = "payment_provider_config"
	// FieldWorkflowConfigID holds the string denoting the workflow_config_id field in the database.
	FieldWorkflowConfigID = "workflow_config_id"
	// FieldDefault holds the string denoting the default field in the database.
	FieldDefault = "default"
	// EdgeBillingInvoices holds the string denoting the billing_invoices edge name in mutations.
	EdgeBillingInvoices = "billing_invoices"
	// EdgeWorkflowConfig holds the string denoting the workflow_config edge name in mutations.
	EdgeWorkflowConfig = "workflow_config"
	// Table holds the table name of the billingprofile in the database.
	Table = "billing_profiles"
	// BillingInvoicesTable is the table that holds the billing_invoices relation/edge.
	BillingInvoicesTable = "billing_invoices"
	// BillingInvoicesInverseTable is the table name for the BillingInvoice entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoice" package.
	BillingInvoicesInverseTable = "billing_invoices"
	// BillingInvoicesColumn is the table column denoting the billing_invoices relation/edge.
	BillingInvoicesColumn = "billing_profile_id"
	// WorkflowConfigTable is the table that holds the workflow_config relation/edge.
	WorkflowConfigTable = "billing_profiles"
	// WorkflowConfigInverseTable is the table name for the BillingWorkflowConfig entity.
	// It exists in this package in order to avoid circular dependency with the "billingworkflowconfig" package.
	WorkflowConfigInverseTable = "billing_workflow_configs"
	// WorkflowConfigColumn is the table column denoting the workflow_config relation/edge.
	WorkflowConfigColumn = "workflow_config_id"
)

// Columns holds all SQL columns for billingprofile fields.
var Columns = []string{
	FieldID,
	FieldKey,
	FieldNamespace,
	FieldMetadata,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldTaxProvider,
	FieldTaxProviderConfig,
	FieldInvoicingProvider,
	FieldInvoicingProviderConfig,
	FieldPaymentProvider,
	FieldPaymentProviderConfig,
	FieldWorkflowConfigID,
	FieldDefault,
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
	// KeyValidator is a validator for the "key" field. It is called by the builders before save.
	KeyValidator func(string) error
	// NamespaceValidator is a validator for the "namespace" field. It is called by the builders before save.
	NamespaceValidator func(string) error
	// DefaultCreatedAt holds the default value on creation for the "created_at" field.
	DefaultCreatedAt func() time.Time
	// DefaultUpdatedAt holds the default value on creation for the "updated_at" field.
	DefaultUpdatedAt func() time.Time
	// UpdateDefaultUpdatedAt holds the default value on update for the "updated_at" field.
	UpdateDefaultUpdatedAt func() time.Time
	// WorkflowConfigIDValidator is a validator for the "workflow_config_id" field. It is called by the builders before save.
	WorkflowConfigIDValidator func(string) error
	// DefaultDefault holds the default value on creation for the "default" field.
	DefaultDefault bool
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
	// ValueScanner of all BillingProfile fields.
	ValueScanner struct {
		TaxProviderConfig       field.TypeValueScanner[provider.TaxConfiguration]
		InvoicingProviderConfig field.TypeValueScanner[provider.InvoicingConfiguration]
		PaymentProviderConfig   field.TypeValueScanner[provider.PaymentConfiguration]
	}
)

// TaxProviderValidator is a validator for the "tax_provider" field enum values. It is called by the builders before save.
func TaxProviderValidator(tp provider.TaxProvider) error {
	switch tp {
	case "openmeter_sandbox", "stripe":
		return nil
	default:
		return fmt.Errorf("billingprofile: invalid enum value for tax_provider field: %q", tp)
	}
}

// InvoicingProviderValidator is a validator for the "invoicing_provider" field enum values. It is called by the builders before save.
func InvoicingProviderValidator(ip provider.InvoicingProvider) error {
	switch ip {
	case "openmeter_sandbox", "stripe":
		return nil
	default:
		return fmt.Errorf("billingprofile: invalid enum value for invoicing_provider field: %q", ip)
	}
}

// PaymentProviderValidator is a validator for the "payment_provider" field enum values. It is called by the builders before save.
func PaymentProviderValidator(pp provider.PaymentProvider) error {
	switch pp {
	case "openmeter_sandbox", "stripe_payments":
		return nil
	default:
		return fmt.Errorf("billingprofile: invalid enum value for payment_provider field: %q", pp)
	}
}

// OrderOption defines the ordering options for the BillingProfile queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByKey orders the results by the key field.
func ByKey(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldKey, opts...).ToFunc()
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

// ByTaxProvider orders the results by the tax_provider field.
func ByTaxProvider(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTaxProvider, opts...).ToFunc()
}

// ByTaxProviderConfig orders the results by the tax_provider_config field.
func ByTaxProviderConfig(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTaxProviderConfig, opts...).ToFunc()
}

// ByInvoicingProvider orders the results by the invoicing_provider field.
func ByInvoicingProvider(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoicingProvider, opts...).ToFunc()
}

// ByInvoicingProviderConfig orders the results by the invoicing_provider_config field.
func ByInvoicingProviderConfig(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoicingProviderConfig, opts...).ToFunc()
}

// ByPaymentProvider orders the results by the payment_provider field.
func ByPaymentProvider(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPaymentProvider, opts...).ToFunc()
}

// ByPaymentProviderConfig orders the results by the payment_provider_config field.
func ByPaymentProviderConfig(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPaymentProviderConfig, opts...).ToFunc()
}

// ByWorkflowConfigID orders the results by the workflow_config_id field.
func ByWorkflowConfigID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldWorkflowConfigID, opts...).ToFunc()
}

// ByDefault orders the results by the default field.
func ByDefault(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDefault, opts...).ToFunc()
}

// ByBillingInvoicesCount orders the results by billing_invoices count.
func ByBillingInvoicesCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newBillingInvoicesStep(), opts...)
	}
}

// ByBillingInvoices orders the results by billing_invoices terms.
func ByBillingInvoices(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingInvoicesStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}

// ByWorkflowConfigField orders the results by workflow_config field.
func ByWorkflowConfigField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newWorkflowConfigStep(), sql.OrderByField(field, opts...))
	}
}
func newBillingInvoicesStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingInvoicesInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, BillingInvoicesTable, BillingInvoicesColumn),
	)
}
func newWorkflowConfigStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(WorkflowConfigInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2O, true, WorkflowConfigTable, WorkflowConfigColumn),
	)
}
