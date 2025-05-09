// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	dbmeter "github.com/openmeterio/openmeter/openmeter/ent/db/meter"
	"github.com/openmeterio/openmeter/openmeter/meter"
)

// Meter is the model entity for the Meter schema.
type Meter struct {
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
	// Key holds the value of the "key" field.
	Key string `json:"key,omitempty"`
	// EventType holds the value of the "event_type" field.
	EventType string `json:"event_type,omitempty"`
	// ValueProperty holds the value of the "value_property" field.
	ValueProperty *string `json:"value_property,omitempty"`
	// GroupBy holds the value of the "group_by" field.
	GroupBy map[string]string `json:"group_by,omitempty"`
	// Aggregation holds the value of the "aggregation" field.
	Aggregation meter.MeterAggregation `json:"aggregation,omitempty"`
	// EventFrom holds the value of the "event_from" field.
	EventFrom    *time.Time `json:"event_from,omitempty"`
	selectValues sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Meter) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case dbmeter.FieldMetadata, dbmeter.FieldGroupBy:
			values[i] = new([]byte)
		case dbmeter.FieldID, dbmeter.FieldNamespace, dbmeter.FieldName, dbmeter.FieldDescription, dbmeter.FieldKey, dbmeter.FieldEventType, dbmeter.FieldValueProperty, dbmeter.FieldAggregation:
			values[i] = new(sql.NullString)
		case dbmeter.FieldCreatedAt, dbmeter.FieldUpdatedAt, dbmeter.FieldDeletedAt, dbmeter.FieldEventFrom:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Meter fields.
func (_m *Meter) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case dbmeter.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				_m.ID = value.String
			}
		case dbmeter.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				_m.Namespace = value.String
			}
		case dbmeter.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &_m.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case dbmeter.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				_m.CreatedAt = value.Time
			}
		case dbmeter.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				_m.UpdatedAt = value.Time
			}
		case dbmeter.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				_m.DeletedAt = new(time.Time)
				*_m.DeletedAt = value.Time
			}
		case dbmeter.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				_m.Name = value.String
			}
		case dbmeter.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				_m.Description = new(string)
				*_m.Description = value.String
			}
		case dbmeter.FieldKey:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field key", values[i])
			} else if value.Valid {
				_m.Key = value.String
			}
		case dbmeter.FieldEventType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field event_type", values[i])
			} else if value.Valid {
				_m.EventType = value.String
			}
		case dbmeter.FieldValueProperty:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field value_property", values[i])
			} else if value.Valid {
				_m.ValueProperty = new(string)
				*_m.ValueProperty = value.String
			}
		case dbmeter.FieldGroupBy:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field group_by", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &_m.GroupBy); err != nil {
					return fmt.Errorf("unmarshal field group_by: %w", err)
				}
			}
		case dbmeter.FieldAggregation:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field aggregation", values[i])
			} else if value.Valid {
				_m.Aggregation = meter.MeterAggregation(value.String)
			}
		case dbmeter.FieldEventFrom:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field event_from", values[i])
			} else if value.Valid {
				_m.EventFrom = new(time.Time)
				*_m.EventFrom = value.Time
			}
		default:
			_m.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Meter.
// This includes values selected through modifiers, order, etc.
func (_m *Meter) Value(name string) (ent.Value, error) {
	return _m.selectValues.Get(name)
}

// Update returns a builder for updating this Meter.
// Note that you need to call Meter.Unwrap() before calling this method if this Meter
// was returned from a transaction, and the transaction was committed or rolled back.
func (_m *Meter) Update() *MeterUpdateOne {
	return NewMeterClient(_m.config).UpdateOne(_m)
}

// Unwrap unwraps the Meter entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (_m *Meter) Unwrap() *Meter {
	_tx, ok := _m.config.driver.(*txDriver)
	if !ok {
		panic("db: Meter is not a transactional entity")
	}
	_m.config.driver = _tx.drv
	return _m
}

// String implements the fmt.Stringer.
func (_m *Meter) String() string {
	var builder strings.Builder
	builder.WriteString("Meter(")
	builder.WriteString(fmt.Sprintf("id=%v, ", _m.ID))
	builder.WriteString("namespace=")
	builder.WriteString(_m.Namespace)
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", _m.Metadata))
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
	builder.WriteString("name=")
	builder.WriteString(_m.Name)
	builder.WriteString(", ")
	if v := _m.Description; v != nil {
		builder.WriteString("description=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("key=")
	builder.WriteString(_m.Key)
	builder.WriteString(", ")
	builder.WriteString("event_type=")
	builder.WriteString(_m.EventType)
	builder.WriteString(", ")
	if v := _m.ValueProperty; v != nil {
		builder.WriteString("value_property=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("group_by=")
	builder.WriteString(fmt.Sprintf("%v", _m.GroupBy))
	builder.WriteString(", ")
	builder.WriteString("aggregation=")
	builder.WriteString(fmt.Sprintf("%v", _m.Aggregation))
	builder.WriteString(", ")
	if v := _m.EventFrom; v != nil {
		builder.WriteString("event_from=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteByte(')')
	return builder.String()
}

// Meters is a parsable slice of Meter.
type Meters []*Meter
