// Code generated by ent, DO NOT EDIT.

package db

import (
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/openmeter/ent/db/usagereset"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

// UsageReset is the model entity for the UsageReset schema.
type UsageReset struct {
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
	// EntitlementID holds the value of the "entitlement_id" field.
	EntitlementID string `json:"entitlement_id,omitempty"`
	// ResetTime holds the value of the "reset_time" field.
	ResetTime time.Time `json:"reset_time,omitempty"`
	// Anchor holds the value of the "anchor" field.
	Anchor time.Time `json:"anchor,omitempty"`
	// UsagePeriodInterval holds the value of the "usage_period_interval" field.
	UsagePeriodInterval datetime.ISODurationString `json:"usage_period_interval,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the UsageResetQuery when eager-loading is set.
	Edges        UsageResetEdges `json:"edges"`
	selectValues sql.SelectValues
}

// UsageResetEdges holds the relations/edges for other nodes in the graph.
type UsageResetEdges struct {
	// Entitlement holds the value of the entitlement edge.
	Entitlement *Entitlement `json:"entitlement,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [1]bool
}

// EntitlementOrErr returns the Entitlement value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e UsageResetEdges) EntitlementOrErr() (*Entitlement, error) {
	if e.Entitlement != nil {
		return e.Entitlement, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: entitlement.Label}
	}
	return nil, &NotLoadedError{edge: "entitlement"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*UsageReset) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case usagereset.FieldID, usagereset.FieldNamespace, usagereset.FieldEntitlementID, usagereset.FieldUsagePeriodInterval:
			values[i] = new(sql.NullString)
		case usagereset.FieldCreatedAt, usagereset.FieldUpdatedAt, usagereset.FieldDeletedAt, usagereset.FieldResetTime, usagereset.FieldAnchor:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the UsageReset fields.
func (_m *UsageReset) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case usagereset.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				_m.ID = value.String
			}
		case usagereset.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				_m.Namespace = value.String
			}
		case usagereset.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				_m.CreatedAt = value.Time
			}
		case usagereset.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				_m.UpdatedAt = value.Time
			}
		case usagereset.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				_m.DeletedAt = new(time.Time)
				*_m.DeletedAt = value.Time
			}
		case usagereset.FieldEntitlementID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field entitlement_id", values[i])
			} else if value.Valid {
				_m.EntitlementID = value.String
			}
		case usagereset.FieldResetTime:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field reset_time", values[i])
			} else if value.Valid {
				_m.ResetTime = value.Time
			}
		case usagereset.FieldAnchor:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field anchor", values[i])
			} else if value.Valid {
				_m.Anchor = value.Time
			}
		case usagereset.FieldUsagePeriodInterval:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field usage_period_interval", values[i])
			} else if value.Valid {
				_m.UsagePeriodInterval = datetime.ISODurationString(value.String)
			}
		default:
			_m.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the UsageReset.
// This includes values selected through modifiers, order, etc.
func (_m *UsageReset) Value(name string) (ent.Value, error) {
	return _m.selectValues.Get(name)
}

// QueryEntitlement queries the "entitlement" edge of the UsageReset entity.
func (_m *UsageReset) QueryEntitlement() *EntitlementQuery {
	return NewUsageResetClient(_m.config).QueryEntitlement(_m)
}

// Update returns a builder for updating this UsageReset.
// Note that you need to call UsageReset.Unwrap() before calling this method if this UsageReset
// was returned from a transaction, and the transaction was committed or rolled back.
func (_m *UsageReset) Update() *UsageResetUpdateOne {
	return NewUsageResetClient(_m.config).UpdateOne(_m)
}

// Unwrap unwraps the UsageReset entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (_m *UsageReset) Unwrap() *UsageReset {
	_tx, ok := _m.config.driver.(*txDriver)
	if !ok {
		panic("db: UsageReset is not a transactional entity")
	}
	_m.config.driver = _tx.drv
	return _m
}

// String implements the fmt.Stringer.
func (_m *UsageReset) String() string {
	var builder strings.Builder
	builder.WriteString("UsageReset(")
	builder.WriteString(fmt.Sprintf("id=%v, ", _m.ID))
	builder.WriteString("namespace=")
	builder.WriteString(_m.Namespace)
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(_m.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(_m.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := _m.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("entitlement_id=")
	builder.WriteString(_m.EntitlementID)
	builder.WriteString(", ")
	builder.WriteString("reset_time=")
	builder.WriteString(_m.ResetTime.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("anchor=")
	builder.WriteString(_m.Anchor.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("usage_period_interval=")
	builder.WriteString(fmt.Sprintf("%v", _m.UsagePeriodInterval))
	builder.WriteByte(')')
	return builder.String()
}

// UsageResets is a parsable slice of UsageReset.
type UsageResets []*UsageReset
