// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingprofile"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingworkflowconfig"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

// BillingWorkflowConfig is the model entity for the BillingWorkflowConfig schema.
type BillingWorkflowConfig struct {
	config `json:"-"`
	// ID of the ent.
	ID string `json:"id,omitempty"`
	// Namespace holds the value of the "namespace" field.
	Namespace string `json:"namespace,omitempty"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// UpdatedAt holds the value of the "updated_at" field.
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	// DeletedAt holds the value of the "deleted_at" field.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	// CollectionAlignment holds the value of the "collection_alignment" field.
	CollectionAlignment billing.AlignmentKind `json:"collection_alignment,omitempty"`
	// LineCollectionPeriod holds the value of the "line_collection_period" field.
	LineCollectionPeriod isodate.String `json:"line_collection_period,omitempty"`
	// InvoiceAutoAdvance holds the value of the "invoice_auto_advance" field.
	InvoiceAutoAdvance bool `json:"invoice_auto_advance,omitempty"`
	// InvoiceDraftPeriod holds the value of the "invoice_draft_period" field.
	InvoiceDraftPeriod isodate.String `json:"invoice_draft_period,omitempty"`
	// InvoiceDueAfter holds the value of the "invoice_due_after" field.
	InvoiceDueAfter isodate.String `json:"invoice_due_after,omitempty"`
	// InvoiceCollectionMethod holds the value of the "invoice_collection_method" field.
	InvoiceCollectionMethod billing.CollectionMethod `json:"invoice_collection_method,omitempty"`
	// InvoiceProgressiveBilling holds the value of the "invoice_progressive_billing" field.
	InvoiceProgressiveBilling bool `json:"invoice_progressive_billing,omitempty"`
	// InvoiceDefaultTaxSettings holds the value of the "invoice_default_tax_settings" field.
	InvoiceDefaultTaxSettings productcatalog.TaxConfig `json:"invoice_default_tax_settings,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the BillingWorkflowConfigQuery when eager-loading is set.
	Edges        BillingWorkflowConfigEdges `json:"edges"`
	selectValues sql.SelectValues
}

// BillingWorkflowConfigEdges holds the relations/edges for other nodes in the graph.
type BillingWorkflowConfigEdges struct {
	// BillingInvoices holds the value of the billing_invoices edge.
	BillingInvoices *BillingInvoice `json:"billing_invoices,omitempty"`
	// BillingProfile holds the value of the billing_profile edge.
	BillingProfile *BillingProfile `json:"billing_profile,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// BillingInvoicesOrErr returns the BillingInvoices value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BillingWorkflowConfigEdges) BillingInvoicesOrErr() (*BillingInvoice, error) {
	if e.BillingInvoices != nil {
		return e.BillingInvoices, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: billinginvoice.Label}
	}
	return nil, &NotLoadedError{edge: "billing_invoices"}
}

// BillingProfileOrErr returns the BillingProfile value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e BillingWorkflowConfigEdges) BillingProfileOrErr() (*BillingProfile, error) {
	if e.BillingProfile != nil {
		return e.BillingProfile, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: billingprofile.Label}
	}
	return nil, &NotLoadedError{edge: "billing_profile"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*BillingWorkflowConfig) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case billingworkflowconfig.FieldInvoiceDefaultTaxSettings:
			values[i] = new([]byte)
		case billingworkflowconfig.FieldInvoiceAutoAdvance, billingworkflowconfig.FieldInvoiceProgressiveBilling:
			values[i] = new(sql.NullBool)
		case billingworkflowconfig.FieldID, billingworkflowconfig.FieldNamespace, billingworkflowconfig.FieldCollectionAlignment, billingworkflowconfig.FieldLineCollectionPeriod, billingworkflowconfig.FieldInvoiceDraftPeriod, billingworkflowconfig.FieldInvoiceDueAfter, billingworkflowconfig.FieldInvoiceCollectionMethod:
			values[i] = new(sql.NullString)
		case billingworkflowconfig.FieldCreatedAt, billingworkflowconfig.FieldUpdatedAt, billingworkflowconfig.FieldDeletedAt:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the BillingWorkflowConfig fields.
func (bwc *BillingWorkflowConfig) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case billingworkflowconfig.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				bwc.ID = value.String
			}
		case billingworkflowconfig.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				bwc.Namespace = value.String
			}
		case billingworkflowconfig.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				bwc.CreatedAt = value.Time
			}
		case billingworkflowconfig.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				bwc.UpdatedAt = value.Time
			}
		case billingworkflowconfig.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				bwc.DeletedAt = new(time.Time)
				*bwc.DeletedAt = value.Time
			}
		case billingworkflowconfig.FieldCollectionAlignment:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field collection_alignment", values[i])
			} else if value.Valid {
				bwc.CollectionAlignment = billing.AlignmentKind(value.String)
			}
		case billingworkflowconfig.FieldLineCollectionPeriod:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field line_collection_period", values[i])
			} else if value.Valid {
				bwc.LineCollectionPeriod = isodate.String(value.String)
			}
		case billingworkflowconfig.FieldInvoiceAutoAdvance:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_auto_advance", values[i])
			} else if value.Valid {
				bwc.InvoiceAutoAdvance = value.Bool
			}
		case billingworkflowconfig.FieldInvoiceDraftPeriod:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_draft_period", values[i])
			} else if value.Valid {
				bwc.InvoiceDraftPeriod = isodate.String(value.String)
			}
		case billingworkflowconfig.FieldInvoiceDueAfter:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_due_after", values[i])
			} else if value.Valid {
				bwc.InvoiceDueAfter = isodate.String(value.String)
			}
		case billingworkflowconfig.FieldInvoiceCollectionMethod:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_collection_method", values[i])
			} else if value.Valid {
				bwc.InvoiceCollectionMethod = billing.CollectionMethod(value.String)
			}
		case billingworkflowconfig.FieldInvoiceProgressiveBilling:
			if value, ok := values[i].(*sql.NullBool); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_progressive_billing", values[i])
			} else if value.Valid {
				bwc.InvoiceProgressiveBilling = value.Bool
			}
		case billingworkflowconfig.FieldInvoiceDefaultTaxSettings:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_default_tax_settings", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &bwc.InvoiceDefaultTaxSettings); err != nil {
					return fmt.Errorf("unmarshal field invoice_default_tax_settings: %w", err)
				}
			}
		default:
			bwc.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the BillingWorkflowConfig.
// This includes values selected through modifiers, order, etc.
func (bwc *BillingWorkflowConfig) Value(name string) (ent.Value, error) {
	return bwc.selectValues.Get(name)
}

// QueryBillingInvoices queries the "billing_invoices" edge of the BillingWorkflowConfig entity.
func (bwc *BillingWorkflowConfig) QueryBillingInvoices() *BillingInvoiceQuery {
	return NewBillingWorkflowConfigClient(bwc.config).QueryBillingInvoices(bwc)
}

// QueryBillingProfile queries the "billing_profile" edge of the BillingWorkflowConfig entity.
func (bwc *BillingWorkflowConfig) QueryBillingProfile() *BillingProfileQuery {
	return NewBillingWorkflowConfigClient(bwc.config).QueryBillingProfile(bwc)
}

// Update returns a builder for updating this BillingWorkflowConfig.
// Note that you need to call BillingWorkflowConfig.Unwrap() before calling this method if this BillingWorkflowConfig
// was returned from a transaction, and the transaction was committed or rolled back.
func (bwc *BillingWorkflowConfig) Update() *BillingWorkflowConfigUpdateOne {
	return NewBillingWorkflowConfigClient(bwc.config).UpdateOne(bwc)
}

// Unwrap unwraps the BillingWorkflowConfig entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (bwc *BillingWorkflowConfig) Unwrap() *BillingWorkflowConfig {
	_tx, ok := bwc.config.driver.(*txDriver)
	if !ok {
		panic("db: BillingWorkflowConfig is not a transactional entity")
	}
	bwc.config.driver = _tx.drv
	return bwc
}

// String implements the fmt.Stringer.
func (bwc *BillingWorkflowConfig) String() string {
	var builder strings.Builder
	builder.WriteString("BillingWorkflowConfig(")
	builder.WriteString(fmt.Sprintf("id=%v, ", bwc.ID))
	builder.WriteString("namespace=")
	builder.WriteString(bwc.Namespace)
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(bwc.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(bwc.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := bwc.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("collection_alignment=")
	builder.WriteString(fmt.Sprintf("%v", bwc.CollectionAlignment))
	builder.WriteString(", ")
	builder.WriteString("line_collection_period=")
	builder.WriteString(fmt.Sprintf("%v", bwc.LineCollectionPeriod))
	builder.WriteString(", ")
	builder.WriteString("invoice_auto_advance=")
	builder.WriteString(fmt.Sprintf("%v", bwc.InvoiceAutoAdvance))
	builder.WriteString(", ")
	builder.WriteString("invoice_draft_period=")
	builder.WriteString(fmt.Sprintf("%v", bwc.InvoiceDraftPeriod))
	builder.WriteString(", ")
	builder.WriteString("invoice_due_after=")
	builder.WriteString(fmt.Sprintf("%v", bwc.InvoiceDueAfter))
	builder.WriteString(", ")
	builder.WriteString("invoice_collection_method=")
	builder.WriteString(fmt.Sprintf("%v", bwc.InvoiceCollectionMethod))
	builder.WriteString(", ")
	builder.WriteString("invoice_progressive_billing=")
	builder.WriteString(fmt.Sprintf("%v", bwc.InvoiceProgressiveBilling))
	builder.WriteString(", ")
	builder.WriteString("invoice_default_tax_settings=")
	builder.WriteString(fmt.Sprintf("%v", bwc.InvoiceDefaultTaxSettings))
	builder.WriteByte(')')
	return builder.String()
}

// BillingWorkflowConfigs is a parsable slice of BillingWorkflowConfig.
type BillingWorkflowConfigs []*BillingWorkflowConfig
