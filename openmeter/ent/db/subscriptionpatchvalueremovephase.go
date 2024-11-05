// Code generated by ent, DO NOT EDIT.

package db

import (
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionpatch"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionpatchvalueremovephase"
)

// SubscriptionPatchValueRemovePhase is the model entity for the SubscriptionPatchValueRemovePhase schema.
type SubscriptionPatchValueRemovePhase struct {
	config `json:"-"`
	// ID of the ent.
	ID string `json:"id,omitempty"`
	// Namespace holds the value of the "namespace" field.
	Namespace string `json:"namespace,omitempty"`
	// SubscriptionPatchID holds the value of the "subscription_patch_id" field.
	SubscriptionPatchID string `json:"subscription_patch_id,omitempty"`
	// PhaseKey holds the value of the "phase_key" field.
	PhaseKey string `json:"phase_key,omitempty"`
	// ShiftBehavior holds the value of the "shift_behavior" field.
	ShiftBehavior int `json:"shift_behavior,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the SubscriptionPatchValueRemovePhaseQuery when eager-loading is set.
	Edges        SubscriptionPatchValueRemovePhaseEdges `json:"edges"`
	selectValues sql.SelectValues
}

// SubscriptionPatchValueRemovePhaseEdges holds the relations/edges for other nodes in the graph.
type SubscriptionPatchValueRemovePhaseEdges struct {
	// SubscriptionPatch holds the value of the subscription_patch edge.
	SubscriptionPatch *SubscriptionPatch `json:"subscription_patch,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [1]bool
}

// SubscriptionPatchOrErr returns the SubscriptionPatch value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e SubscriptionPatchValueRemovePhaseEdges) SubscriptionPatchOrErr() (*SubscriptionPatch, error) {
	if e.SubscriptionPatch != nil {
		return e.SubscriptionPatch, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: subscriptionpatch.Label}
	}
	return nil, &NotLoadedError{edge: "subscription_patch"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*SubscriptionPatchValueRemovePhase) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case subscriptionpatchvalueremovephase.FieldShiftBehavior:
			values[i] = new(sql.NullInt64)
		case subscriptionpatchvalueremovephase.FieldID, subscriptionpatchvalueremovephase.FieldNamespace, subscriptionpatchvalueremovephase.FieldSubscriptionPatchID, subscriptionpatchvalueremovephase.FieldPhaseKey:
			values[i] = new(sql.NullString)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the SubscriptionPatchValueRemovePhase fields.
func (spvrp *SubscriptionPatchValueRemovePhase) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case subscriptionpatchvalueremovephase.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				spvrp.ID = value.String
			}
		case subscriptionpatchvalueremovephase.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				spvrp.Namespace = value.String
			}
		case subscriptionpatchvalueremovephase.FieldSubscriptionPatchID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field subscription_patch_id", values[i])
			} else if value.Valid {
				spvrp.SubscriptionPatchID = value.String
			}
		case subscriptionpatchvalueremovephase.FieldPhaseKey:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field phase_key", values[i])
			} else if value.Valid {
				spvrp.PhaseKey = value.String
			}
		case subscriptionpatchvalueremovephase.FieldShiftBehavior:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field shift_behavior", values[i])
			} else if value.Valid {
				spvrp.ShiftBehavior = int(value.Int64)
			}
		default:
			spvrp.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the SubscriptionPatchValueRemovePhase.
// This includes values selected through modifiers, order, etc.
func (spvrp *SubscriptionPatchValueRemovePhase) Value(name string) (ent.Value, error) {
	return spvrp.selectValues.Get(name)
}

// QuerySubscriptionPatch queries the "subscription_patch" edge of the SubscriptionPatchValueRemovePhase entity.
func (spvrp *SubscriptionPatchValueRemovePhase) QuerySubscriptionPatch() *SubscriptionPatchQuery {
	return NewSubscriptionPatchValueRemovePhaseClient(spvrp.config).QuerySubscriptionPatch(spvrp)
}

// Update returns a builder for updating this SubscriptionPatchValueRemovePhase.
// Note that you need to call SubscriptionPatchValueRemovePhase.Unwrap() before calling this method if this SubscriptionPatchValueRemovePhase
// was returned from a transaction, and the transaction was committed or rolled back.
func (spvrp *SubscriptionPatchValueRemovePhase) Update() *SubscriptionPatchValueRemovePhaseUpdateOne {
	return NewSubscriptionPatchValueRemovePhaseClient(spvrp.config).UpdateOne(spvrp)
}

// Unwrap unwraps the SubscriptionPatchValueRemovePhase entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (spvrp *SubscriptionPatchValueRemovePhase) Unwrap() *SubscriptionPatchValueRemovePhase {
	_tx, ok := spvrp.config.driver.(*txDriver)
	if !ok {
		panic("db: SubscriptionPatchValueRemovePhase is not a transactional entity")
	}
	spvrp.config.driver = _tx.drv
	return spvrp
}

// String implements the fmt.Stringer.
func (spvrp *SubscriptionPatchValueRemovePhase) String() string {
	var builder strings.Builder
	builder.WriteString("SubscriptionPatchValueRemovePhase(")
	builder.WriteString(fmt.Sprintf("id=%v, ", spvrp.ID))
	builder.WriteString("namespace=")
	builder.WriteString(spvrp.Namespace)
	builder.WriteString(", ")
	builder.WriteString("subscription_patch_id=")
	builder.WriteString(spvrp.SubscriptionPatchID)
	builder.WriteString(", ")
	builder.WriteString("phase_key=")
	builder.WriteString(spvrp.PhaseKey)
	builder.WriteString(", ")
	builder.WriteString("shift_behavior=")
	builder.WriteString(fmt.Sprintf("%v", spvrp.ShiftBehavior))
	builder.WriteByte(')')
	return builder.String()
}

// SubscriptionPatchValueRemovePhases is a parsable slice of SubscriptionPatchValueRemovePhase.
type SubscriptionPatchValueRemovePhases []*SubscriptionPatchValueRemovePhase