// Code generated by ent, DO NOT EDIT.

package billinginvoice

import (
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
)

const (
	// Label holds the string label denoting the billinginvoice type in the database.
	Label = "billing_invoice"
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
	// FieldSeries holds the string denoting the series field in the database.
	FieldSeries = "series"
	// FieldCode holds the string denoting the code field in the database.
	FieldCode = "code"
	// FieldCustomerID holds the string denoting the customer_id field in the database.
	FieldCustomerID = "customer_id"
	// FieldBillingProfileID holds the string denoting the billing_profile_id field in the database.
	FieldBillingProfileID = "billing_profile_id"
	// FieldVoidedAt holds the string denoting the voided_at field in the database.
	FieldVoidedAt = "voided_at"
	// FieldCurrency holds the string denoting the currency field in the database.
	FieldCurrency = "currency"
	// FieldDueDate holds the string denoting the due_date field in the database.
	FieldDueDate = "due_date"
	// FieldStatus holds the string denoting the status field in the database.
	FieldStatus = "status"
	// FieldTaxProvider holds the string denoting the tax_provider field in the database.
	FieldTaxProvider = "tax_provider"
	// FieldInvoicingProvider holds the string denoting the invoicing_provider field in the database.
	FieldInvoicingProvider = "invoicing_provider"
	// FieldPaymentProvider holds the string denoting the payment_provider field in the database.
	FieldPaymentProvider = "payment_provider"
	// FieldWorkflowConfigID holds the string denoting the workflow_config_id field in the database.
	FieldWorkflowConfigID = "workflow_config_id"
	// FieldPeriodStart holds the string denoting the period_start field in the database.
	FieldPeriodStart = "period_start"
	// FieldPeriodEnd holds the string denoting the period_end field in the database.
	FieldPeriodEnd = "period_end"
	// EdgeBillingProfile holds the string denoting the billing_profile edge name in mutations.
	EdgeBillingProfile = "billing_profile"
	// EdgeBillingWorkflowConfig holds the string denoting the billing_workflow_config edge name in mutations.
	EdgeBillingWorkflowConfig = "billing_workflow_config"
	// EdgeBillingInvoiceItems holds the string denoting the billing_invoice_items edge name in mutations.
	EdgeBillingInvoiceItems = "billing_invoice_items"
	// Table holds the table name of the billinginvoice in the database.
	Table = "billing_invoices"
	// BillingProfileTable is the table that holds the billing_profile relation/edge.
	BillingProfileTable = "billing_invoices"
	// BillingProfileInverseTable is the table name for the BillingProfile entity.
	// It exists in this package in order to avoid circular dependency with the "billingprofile" package.
	BillingProfileInverseTable = "billing_profiles"
	// BillingProfileColumn is the table column denoting the billing_profile relation/edge.
	BillingProfileColumn = "billing_profile_id"
	// BillingWorkflowConfigTable is the table that holds the billing_workflow_config relation/edge.
	BillingWorkflowConfigTable = "billing_invoices"
	// BillingWorkflowConfigInverseTable is the table name for the BillingWorkflowConfig entity.
	// It exists in this package in order to avoid circular dependency with the "billingworkflowconfig" package.
	BillingWorkflowConfigInverseTable = "billing_workflow_configs"
	// BillingWorkflowConfigColumn is the table column denoting the billing_workflow_config relation/edge.
	BillingWorkflowConfigColumn = "workflow_config_id"
	// BillingInvoiceItemsTable is the table that holds the billing_invoice_items relation/edge.
	BillingInvoiceItemsTable = "billing_invoice_items"
	// BillingInvoiceItemsInverseTable is the table name for the BillingInvoiceItem entity.
	// It exists in this package in order to avoid circular dependency with the "billinginvoiceitem" package.
	BillingInvoiceItemsInverseTable = "billing_invoice_items"
	// BillingInvoiceItemsColumn is the table column denoting the billing_invoice_items relation/edge.
	BillingInvoiceItemsColumn = "invoice_id"
)

// Columns holds all SQL columns for billinginvoice fields.
var Columns = []string{
	FieldID,
	FieldNamespace,
	FieldCreatedAt,
	FieldUpdatedAt,
	FieldDeletedAt,
	FieldMetadata,
	FieldSeries,
	FieldCode,
	FieldCustomerID,
	FieldBillingProfileID,
	FieldVoidedAt,
	FieldCurrency,
	FieldDueDate,
	FieldStatus,
	FieldTaxProvider,
	FieldInvoicingProvider,
	FieldPaymentProvider,
	FieldWorkflowConfigID,
	FieldPeriodStart,
	FieldPeriodEnd,
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
	// CustomerIDValidator is a validator for the "customer_id" field. It is called by the builders before save.
	CustomerIDValidator func(string) error
	// BillingProfileIDValidator is a validator for the "billing_profile_id" field. It is called by the builders before save.
	BillingProfileIDValidator func(string) error
	// CurrencyValidator is a validator for the "currency" field. It is called by the builders before save.
	CurrencyValidator func(string) error
	// DefaultID holds the default value on creation for the "id" field.
	DefaultID func() string
)

// StatusValidator is a validator for the "status" field enum values. It is called by the builders before save.
func StatusValidator(s billing.InvoiceStatus) error {
	switch s {
	case "created", "draft", "draft_sync", "draft_sync_failed", "issuing", "issued", "issuing_failed", "manual_approval_needed":
		return nil
	default:
		return fmt.Errorf("billinginvoice: invalid enum value for status field: %q", s)
	}
}

// TaxProviderValidator is a validator for the "tax_provider" field enum values. It is called by the builders before save.
func TaxProviderValidator(tp provider.TaxProvider) error {
	switch tp {
	case "openmeter_sandbox", "stripe":
		return nil
	default:
		return fmt.Errorf("billinginvoice: invalid enum value for tax_provider field: %q", tp)
	}
}

// InvoicingProviderValidator is a validator for the "invoicing_provider" field enum values. It is called by the builders before save.
func InvoicingProviderValidator(ip provider.InvoicingProvider) error {
	switch ip {
	case "openmeter_sandbox", "stripe":
		return nil
	default:
		return fmt.Errorf("billinginvoice: invalid enum value for invoicing_provider field: %q", ip)
	}
}

// PaymentProviderValidator is a validator for the "payment_provider" field enum values. It is called by the builders before save.
func PaymentProviderValidator(pp provider.PaymentProvider) error {
	switch pp {
	case "openmeter_sandbox", "stripe_payments":
		return nil
	default:
		return fmt.Errorf("billinginvoice: invalid enum value for payment_provider field: %q", pp)
	}
}

// OrderOption defines the ordering options for the BillingInvoice queries.
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

// BySeries orders the results by the series field.
func BySeries(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldSeries, opts...).ToFunc()
}

// ByCode orders the results by the code field.
func ByCode(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCode, opts...).ToFunc()
}

// ByCustomerID orders the results by the customer_id field.
func ByCustomerID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCustomerID, opts...).ToFunc()
}

// ByBillingProfileID orders the results by the billing_profile_id field.
func ByBillingProfileID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldBillingProfileID, opts...).ToFunc()
}

// ByVoidedAt orders the results by the voided_at field.
func ByVoidedAt(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldVoidedAt, opts...).ToFunc()
}

// ByCurrency orders the results by the currency field.
func ByCurrency(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldCurrency, opts...).ToFunc()
}

// ByDueDate orders the results by the due_date field.
func ByDueDate(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDueDate, opts...).ToFunc()
}

// ByStatus orders the results by the status field.
func ByStatus(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldStatus, opts...).ToFunc()
}

// ByTaxProvider orders the results by the tax_provider field.
func ByTaxProvider(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldTaxProvider, opts...).ToFunc()
}

// ByInvoicingProvider orders the results by the invoicing_provider field.
func ByInvoicingProvider(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldInvoicingProvider, opts...).ToFunc()
}

// ByPaymentProvider orders the results by the payment_provider field.
func ByPaymentProvider(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPaymentProvider, opts...).ToFunc()
}

// ByWorkflowConfigID orders the results by the workflow_config_id field.
func ByWorkflowConfigID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldWorkflowConfigID, opts...).ToFunc()
}

// ByPeriodStart orders the results by the period_start field.
func ByPeriodStart(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPeriodStart, opts...).ToFunc()
}

// ByPeriodEnd orders the results by the period_end field.
func ByPeriodEnd(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPeriodEnd, opts...).ToFunc()
}

// ByBillingProfileField orders the results by billing_profile field.
func ByBillingProfileField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingProfileStep(), sql.OrderByField(field, opts...))
	}
}

// ByBillingWorkflowConfigField orders the results by billing_workflow_config field.
func ByBillingWorkflowConfigField(field string, opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingWorkflowConfigStep(), sql.OrderByField(field, opts...))
	}
}

// ByBillingInvoiceItemsCount orders the results by billing_invoice_items count.
func ByBillingInvoiceItemsCount(opts ...sql.OrderTermOption) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborsCount(s, newBillingInvoiceItemsStep(), opts...)
	}
}

// ByBillingInvoiceItems orders the results by billing_invoice_items terms.
func ByBillingInvoiceItems(term sql.OrderTerm, terms ...sql.OrderTerm) OrderOption {
	return func(s *sql.Selector) {
		sqlgraph.OrderByNeighborTerms(s, newBillingInvoiceItemsStep(), append([]sql.OrderTerm{term}, terms...)...)
	}
}
func newBillingProfileStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingProfileInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.M2O, true, BillingProfileTable, BillingProfileColumn),
	)
}
func newBillingWorkflowConfigStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingWorkflowConfigInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2O, true, BillingWorkflowConfigTable, BillingWorkflowConfigColumn),
	)
}
func newBillingInvoiceItemsStep() *sqlgraph.Step {
	return sqlgraph.NewStep(
		sqlgraph.From(Table, FieldID),
		sqlgraph.To(BillingInvoiceItemsInverseTable, FieldID),
		sqlgraph.Edge(sqlgraph.O2M, false, BillingInvoiceItemsTable, BillingInvoiceItemsColumn),
	)
}
