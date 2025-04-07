// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicediscount"
)

// BillingInvoiceDiscount is the model entity for the BillingInvoiceDiscount schema.
type BillingInvoiceDiscount struct {
	config `json:"-"`
	// ID of the ent.
	ID string `json:"id,omitempty"`
	// Namespace holds the value of the "namespace" field.
	Namespace string `json:"namespace,omitempty"`
	// Metadata holds the value of the "metadata" field.
	Metadata map[string]string `json:"metadata,omitempty"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// UpdatedAt holds the value of the "updated_at" field.
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	// DeletedAt holds the value of the "deleted_at" field.
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	// Name holds the value of the "name" field.
	Name string `json:"name,omitempty"`
	// Description holds the value of the "description" field.
	Description *string `json:"description,omitempty"`
	// InvoiceID holds the value of the "invoice_id" field.
	InvoiceID string `json:"invoice_id,omitempty"`
	// Type holds the value of the "type" field.
	Type billing.LineDiscountType `json:"type,omitempty"`
	// Amount holds the value of the "amount" field.
	Amount alpacadecimal.Decimal `json:"amount,omitempty"`
	// LineIds holds the value of the "line_ids" field.
	LineIds      []string `json:"line_ids,omitempty"`
	selectValues sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*BillingInvoiceDiscount) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case billinginvoicediscount.FieldMetadata, billinginvoicediscount.FieldLineIds:
			values[i] = new([]byte)
		case billinginvoicediscount.FieldAmount:
			values[i] = new(alpacadecimal.Decimal)
		case billinginvoicediscount.FieldID, billinginvoicediscount.FieldNamespace, billinginvoicediscount.FieldName, billinginvoicediscount.FieldDescription, billinginvoicediscount.FieldInvoiceID, billinginvoicediscount.FieldType:
			values[i] = new(sql.NullString)
		case billinginvoicediscount.FieldCreatedAt, billinginvoicediscount.FieldUpdatedAt, billinginvoicediscount.FieldDeletedAt:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the BillingInvoiceDiscount fields.
func (bid *BillingInvoiceDiscount) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case billinginvoicediscount.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				bid.ID = value.String
			}
		case billinginvoicediscount.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				bid.Namespace = value.String
			}
		case billinginvoicediscount.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &bid.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case billinginvoicediscount.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				bid.CreatedAt = value.Time
			}
		case billinginvoicediscount.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				bid.UpdatedAt = value.Time
			}
		case billinginvoicediscount.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				bid.DeletedAt = new(time.Time)
				*bid.DeletedAt = value.Time
			}
		case billinginvoicediscount.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				bid.Name = value.String
			}
		case billinginvoicediscount.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				bid.Description = new(string)
				*bid.Description = value.String
			}
		case billinginvoicediscount.FieldInvoiceID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field invoice_id", values[i])
			} else if value.Valid {
				bid.InvoiceID = value.String
			}
		case billinginvoicediscount.FieldType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field type", values[i])
			} else if value.Valid {
				bid.Type = billing.LineDiscountType(value.String)
			}
		case billinginvoicediscount.FieldAmount:
			if value, ok := values[i].(*alpacadecimal.Decimal); !ok {
				return fmt.Errorf("unexpected type %T for field amount", values[i])
			} else if value != nil {
				bid.Amount = *value
			}
		case billinginvoicediscount.FieldLineIds:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field line_ids", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &bid.LineIds); err != nil {
					return fmt.Errorf("unmarshal field line_ids: %w", err)
				}
			}
		default:
			bid.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the BillingInvoiceDiscount.
// This includes values selected through modifiers, order, etc.
func (bid *BillingInvoiceDiscount) Value(name string) (ent.Value, error) {
	return bid.selectValues.Get(name)
}

// Update returns a builder for updating this BillingInvoiceDiscount.
// Note that you need to call BillingInvoiceDiscount.Unwrap() before calling this method if this BillingInvoiceDiscount
// was returned from a transaction, and the transaction was committed or rolled back.
func (bid *BillingInvoiceDiscount) Update() *BillingInvoiceDiscountUpdateOne {
	return NewBillingInvoiceDiscountClient(bid.config).UpdateOne(bid)
}

// Unwrap unwraps the BillingInvoiceDiscount entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (bid *BillingInvoiceDiscount) Unwrap() *BillingInvoiceDiscount {
	_tx, ok := bid.config.driver.(*txDriver)
	if !ok {
		panic("db: BillingInvoiceDiscount is not a transactional entity")
	}
	bid.config.driver = _tx.drv
	return bid
}

// String implements the fmt.Stringer.
func (bid *BillingInvoiceDiscount) String() string {
	var builder strings.Builder
	builder.WriteString("BillingInvoiceDiscount(")
	builder.WriteString(fmt.Sprintf("id=%v, ", bid.ID))
	builder.WriteString("namespace=")
	builder.WriteString(bid.Namespace)
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", bid.Metadata))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(bid.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(bid.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := bid.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(bid.Name)
	builder.WriteString(", ")
	if v := bid.Description; v != nil {
		builder.WriteString("description=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("invoice_id=")
	builder.WriteString(bid.InvoiceID)
	builder.WriteString(", ")
	builder.WriteString("type=")
	builder.WriteString(fmt.Sprintf("%v", bid.Type))
	builder.WriteString(", ")
	builder.WriteString("amount=")
	builder.WriteString(fmt.Sprintf("%v", bid.Amount))
	builder.WriteString(", ")
	builder.WriteString("line_ids=")
	builder.WriteString(fmt.Sprintf("%v", bid.LineIds))
	builder.WriteByte(')')
	return builder.String()
}

// BillingInvoiceDiscounts is a parsable slice of BillingInvoiceDiscount.
type BillingInvoiceDiscounts []*BillingInvoiceDiscount
