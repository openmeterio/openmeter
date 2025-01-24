// Code generated by ent, DO NOT EDIT.

package db

import (
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingsequencenumbers"
)

// BillingSequenceNumbers is the model entity for the BillingSequenceNumbers schema.
type BillingSequenceNumbers struct {
	config `json:"-"`
	// ID of the ent.
	ID int `json:"id,omitempty"`
	// Namespace holds the value of the "namespace" field.
	Namespace string `json:"namespace,omitempty"`
	// Scope holds the value of the "scope" field.
	Scope string `json:"scope,omitempty"`
	// Last holds the value of the "last" field.
	Last         alpacadecimal.Decimal `json:"last,omitempty"`
	selectValues sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*BillingSequenceNumbers) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case billingsequencenumbers.FieldLast:
			values[i] = new(alpacadecimal.Decimal)
		case billingsequencenumbers.FieldID:
			values[i] = new(sql.NullInt64)
		case billingsequencenumbers.FieldNamespace, billingsequencenumbers.FieldScope:
			values[i] = new(sql.NullString)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the BillingSequenceNumbers fields.
func (bsn *BillingSequenceNumbers) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case billingsequencenumbers.FieldID:
			value, ok := values[i].(*sql.NullInt64)
			if !ok {
				return fmt.Errorf("unexpected type %T for field id", value)
			}
			bsn.ID = int(value.Int64)
		case billingsequencenumbers.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				bsn.Namespace = value.String
			}
		case billingsequencenumbers.FieldScope:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field scope", values[i])
			} else if value.Valid {
				bsn.Scope = value.String
			}
		case billingsequencenumbers.FieldLast:
			if value, ok := values[i].(*alpacadecimal.Decimal); !ok {
				return fmt.Errorf("unexpected type %T for field last", values[i])
			} else if value != nil {
				bsn.Last = *value
			}
		default:
			bsn.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the BillingSequenceNumbers.
// This includes values selected through modifiers, order, etc.
func (bsn *BillingSequenceNumbers) Value(name string) (ent.Value, error) {
	return bsn.selectValues.Get(name)
}

// Update returns a builder for updating this BillingSequenceNumbers.
// Note that you need to call BillingSequenceNumbers.Unwrap() before calling this method if this BillingSequenceNumbers
// was returned from a transaction, and the transaction was committed or rolled back.
func (bsn *BillingSequenceNumbers) Update() *BillingSequenceNumbersUpdateOne {
	return NewBillingSequenceNumbersClient(bsn.config).UpdateOne(bsn)
}

// Unwrap unwraps the BillingSequenceNumbers entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (bsn *BillingSequenceNumbers) Unwrap() *BillingSequenceNumbers {
	_tx, ok := bsn.config.driver.(*txDriver)
	if !ok {
		panic("db: BillingSequenceNumbers is not a transactional entity")
	}
	bsn.config.driver = _tx.drv
	return bsn
}

// String implements the fmt.Stringer.
func (bsn *BillingSequenceNumbers) String() string {
	var builder strings.Builder
	builder.WriteString("BillingSequenceNumbers(")
	builder.WriteString(fmt.Sprintf("id=%v, ", bsn.ID))
	builder.WriteString("namespace=")
	builder.WriteString(bsn.Namespace)
	builder.WriteString(", ")
	builder.WriteString("scope=")
	builder.WriteString(bsn.Scope)
	builder.WriteString(", ")
	builder.WriteString("last=")
	builder.WriteString(fmt.Sprintf("%v", bsn.Last))
	builder.WriteByte(')')
	return builder.String()
}

// BillingSequenceNumbersSlice is a parsable slice of BillingSequenceNumbers.
type BillingSequenceNumbersSlice []*BillingSequenceNumbers