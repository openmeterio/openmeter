// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	"github.com/openmeterio/openmeter/openmeter/ent/db/planphase"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

// PlanPhase is the model entity for the PlanPhase schema.
type PlanPhase struct {
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
	// The plan identifier the phase is assigned to.
	PlanID string `json:"plan_id,omitempty"`
	// The index of the phase in the plan.
	Index uint8 `json:"index,omitempty"`
	// The duration of the phase.
	Duration *isodate.String `json:"duration,omitempty"`
	// Discounts holds the value of the "discounts" field.
	//
	// Deprecated: Use ratecards.discounts instead
	Discounts productcatalog.Discounts `json:"discounts,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the PlanPhaseQuery when eager-loading is set.
	Edges        PlanPhaseEdges `json:"edges"`
	selectValues sql.SelectValues
}

// PlanPhaseEdges holds the relations/edges for other nodes in the graph.
type PlanPhaseEdges struct {
	// Plan holds the value of the plan edge.
	Plan *Plan `json:"plan,omitempty"`
	// Ratecards holds the value of the ratecards edge.
	Ratecards []*PlanRateCard `json:"ratecards,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [2]bool
}

// PlanOrErr returns the Plan value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e PlanPhaseEdges) PlanOrErr() (*Plan, error) {
	if e.Plan != nil {
		return e.Plan, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: plan.Label}
	}
	return nil, &NotLoadedError{edge: "plan"}
}

// RatecardsOrErr returns the Ratecards value or an error if the edge
// was not loaded in eager-loading.
func (e PlanPhaseEdges) RatecardsOrErr() ([]*PlanRateCard, error) {
	if e.loadedTypes[1] {
		return e.Ratecards, nil
	}
	return nil, &NotLoadedError{edge: "ratecards"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*PlanPhase) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case planphase.FieldMetadata:
			values[i] = new([]byte)
		case planphase.FieldIndex:
			values[i] = new(sql.NullInt64)
		case planphase.FieldID, planphase.FieldNamespace, planphase.FieldName, planphase.FieldDescription, planphase.FieldKey, planphase.FieldPlanID, planphase.FieldDuration:
			values[i] = new(sql.NullString)
		case planphase.FieldCreatedAt, planphase.FieldUpdatedAt, planphase.FieldDeletedAt:
			values[i] = new(sql.NullTime)
		case planphase.FieldDiscounts:
			values[i] = planphase.ValueScanner.Discounts.ScanValue()
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the PlanPhase fields.
func (pp *PlanPhase) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case planphase.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				pp.ID = value.String
			}
		case planphase.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				pp.Namespace = value.String
			}
		case planphase.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &pp.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case planphase.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				pp.CreatedAt = value.Time
			}
		case planphase.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				pp.UpdatedAt = value.Time
			}
		case planphase.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				pp.DeletedAt = new(time.Time)
				*pp.DeletedAt = value.Time
			}
		case planphase.FieldName:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field name", values[i])
			} else if value.Valid {
				pp.Name = value.String
			}
		case planphase.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				pp.Description = new(string)
				*pp.Description = value.String
			}
		case planphase.FieldKey:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field key", values[i])
			} else if value.Valid {
				pp.Key = value.String
			}
		case planphase.FieldPlanID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field plan_id", values[i])
			} else if value.Valid {
				pp.PlanID = value.String
			}
		case planphase.FieldIndex:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field index", values[i])
			} else if value.Valid {
				pp.Index = uint8(value.Int64)
			}
		case planphase.FieldDuration:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field duration", values[i])
			} else if value.Valid {
				pp.Duration = new(isodate.String)
				*pp.Duration = isodate.String(value.String)
			}
		case planphase.FieldDiscounts:
			if value, err := planphase.ValueScanner.Discounts.FromValue(values[i]); err != nil {
				return err
			} else {
				pp.Discounts = value
			}
		default:
			pp.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the PlanPhase.
// This includes values selected through modifiers, order, etc.
func (pp *PlanPhase) Value(name string) (ent.Value, error) {
	return pp.selectValues.Get(name)
}

// QueryPlan queries the "plan" edge of the PlanPhase entity.
func (pp *PlanPhase) QueryPlan() *PlanQuery {
	return NewPlanPhaseClient(pp.config).QueryPlan(pp)
}

// QueryRatecards queries the "ratecards" edge of the PlanPhase entity.
func (pp *PlanPhase) QueryRatecards() *PlanRateCardQuery {
	return NewPlanPhaseClient(pp.config).QueryRatecards(pp)
}

// Update returns a builder for updating this PlanPhase.
// Note that you need to call PlanPhase.Unwrap() before calling this method if this PlanPhase
// was returned from a transaction, and the transaction was committed or rolled back.
func (pp *PlanPhase) Update() *PlanPhaseUpdateOne {
	return NewPlanPhaseClient(pp.config).UpdateOne(pp)
}

// Unwrap unwraps the PlanPhase entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (pp *PlanPhase) Unwrap() *PlanPhase {
	_tx, ok := pp.config.driver.(*txDriver)
	if !ok {
		panic("db: PlanPhase is not a transactional entity")
	}
	pp.config.driver = _tx.drv
	return pp
}

// String implements the fmt.Stringer.
func (pp *PlanPhase) String() string {
	var builder strings.Builder
	builder.WriteString("PlanPhase(")
	builder.WriteString(fmt.Sprintf("id=%v, ", pp.ID))
	builder.WriteString("namespace=")
	builder.WriteString(pp.Namespace)
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", pp.Metadata))
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(pp.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(pp.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := pp.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("name=")
	builder.WriteString(pp.Name)
	builder.WriteString(", ")
	if v := pp.Description; v != nil {
		builder.WriteString("description=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	builder.WriteString("key=")
	builder.WriteString(pp.Key)
	builder.WriteString(", ")
	builder.WriteString("plan_id=")
	builder.WriteString(pp.PlanID)
	builder.WriteString(", ")
	builder.WriteString("index=")
	builder.WriteString(fmt.Sprintf("%v", pp.Index))
	builder.WriteString(", ")
	if v := pp.Duration; v != nil {
		builder.WriteString("duration=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("discounts=")
	builder.WriteString(fmt.Sprintf("%v", pp.Discounts))
	builder.WriteByte(')')
	return builder.String()
}

// PlanPhases is a parsable slice of PlanPhase.
type PlanPhases []*PlanPhase
