// Code generated by ent, DO NOT EDIT.

package db

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customersubjects"
)

// CustomerSubjects is the model entity for the CustomerSubjects schema.
type CustomerSubjects struct {
	config `json:"-"`
	// ID of the ent.
	ID int `json:"id,omitempty"`
	// CustomerID holds the value of the "customer_id" field.
	CustomerID string `json:"customer_id,omitempty"`
	// SubjectKey holds the value of the "subject_key" field.
	SubjectKey string `json:"subject_key,omitempty"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the CustomerSubjectsQuery when eager-loading is set.
	Edges        CustomerSubjectsEdges `json:"edges"`
	selectValues sql.SelectValues
}

// CustomerSubjectsEdges holds the relations/edges for other nodes in the graph.
type CustomerSubjectsEdges struct {
	// Customer holds the value of the customer edge.
	Customer *Customer `json:"customer,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [1]bool
}

// CustomerOrErr returns the Customer value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e CustomerSubjectsEdges) CustomerOrErr() (*Customer, error) {
	if e.Customer != nil {
		return e.Customer, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: customer.Label}
	}
	return nil, &NotLoadedError{edge: "customer"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*CustomerSubjects) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case customersubjects.FieldID:
			values[i] = new(sql.NullInt64)
		case customersubjects.FieldCustomerID, customersubjects.FieldSubjectKey:
			values[i] = new(sql.NullString)
		case customersubjects.FieldCreatedAt:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the CustomerSubjects fields.
func (cs *CustomerSubjects) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case customersubjects.FieldID:
			value, ok := values[i].(*sql.NullInt64)
			if !ok {
				return fmt.Errorf("unexpected type %T for field id", value)
			}
			cs.ID = int(value.Int64)
		case customersubjects.FieldCustomerID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field customer_id", values[i])
			} else if value.Valid {
				cs.CustomerID = value.String
			}
		case customersubjects.FieldSubjectKey:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field subject_key", values[i])
			} else if value.Valid {
				cs.SubjectKey = value.String
			}
		case customersubjects.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				cs.CreatedAt = value.Time
			}
		default:
			cs.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the CustomerSubjects.
// This includes values selected through modifiers, order, etc.
func (cs *CustomerSubjects) Value(name string) (ent.Value, error) {
	return cs.selectValues.Get(name)
}

// QueryCustomer queries the "customer" edge of the CustomerSubjects entity.
func (cs *CustomerSubjects) QueryCustomer() *CustomerQuery {
	return NewCustomerSubjectsClient(cs.config).QueryCustomer(cs)
}

// Update returns a builder for updating this CustomerSubjects.
// Note that you need to call CustomerSubjects.Unwrap() before calling this method if this CustomerSubjects
// was returned from a transaction, and the transaction was committed or rolled back.
func (cs *CustomerSubjects) Update() *CustomerSubjectsUpdateOne {
	return NewCustomerSubjectsClient(cs.config).UpdateOne(cs)
}

// Unwrap unwraps the CustomerSubjects entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (cs *CustomerSubjects) Unwrap() *CustomerSubjects {
	_tx, ok := cs.config.driver.(*txDriver)
	if !ok {
		panic("db: CustomerSubjects is not a transactional entity")
	}
	cs.config.driver = _tx.drv
	return cs
}

// String implements the fmt.Stringer.
func (cs *CustomerSubjects) String() string {
	var builder strings.Builder
	builder.WriteString("CustomerSubjects(")
	builder.WriteString(fmt.Sprintf("id=%v, ", cs.ID))
	builder.WriteString("customer_id=")
	builder.WriteString(cs.CustomerID)
	builder.WriteString(", ")
	builder.WriteString("subject_key=")
	builder.WriteString(cs.SubjectKey)
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(cs.CreatedAt.Format(time.ANSIC))
	builder.WriteByte(')')
	return builder.String()
}

// CustomerSubjectsSlice is a parsable slice of CustomerSubjects.
type CustomerSubjectsSlice []*CustomerSubjects
