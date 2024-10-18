// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionpatch"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionpatchvalueadditem"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionpatchvalueaddphase"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionpatchvalueextendphase"
)

// SubscriptionPatch is the model entity for the SubscriptionPatch schema.
type SubscriptionPatch struct {
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
	// Metadata holds the value of the "metadata" field.
	Metadata map[string]string `json:"metadata,omitempty"`
	// SubscriptionID holds the value of the "subscription_id" field.
	SubscriptionID string `json:"subscription_id,omitempty"`
	// AppliedAt holds the value of the "applied_at" field.
	AppliedAt time.Time `json:"applied_at,omitempty"`
	// BatchIndex holds the value of the "batch_index" field.
	BatchIndex int `json:"batch_index,omitempty"`
	// Operation holds the value of the "operation" field.
	Operation string `json:"operation,omitempty"`
	// Path holds the value of the "path" field.
	Path string `json:"path,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the SubscriptionPatchQuery when eager-loading is set.
	Edges        SubscriptionPatchEdges `json:"edges"`
	selectValues sql.SelectValues
}

// SubscriptionPatchEdges holds the relations/edges for other nodes in the graph.
type SubscriptionPatchEdges struct {
	// Subscription holds the value of the subscription edge.
	Subscription *Subscription `json:"subscription,omitempty"`
	// ValueAddItem holds the value of the value_add_item edge.
	ValueAddItem *SubscriptionPatchValueAddItem `json:"value_add_item,omitempty"`
	// ValueAddPhase holds the value of the value_add_phase edge.
	ValueAddPhase *SubscriptionPatchValueAddPhase `json:"value_add_phase,omitempty"`
	// ValueExtendPhase holds the value of the value_extend_phase edge.
	ValueExtendPhase *SubscriptionPatchValueExtendPhase `json:"value_extend_phase,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [4]bool
}

// SubscriptionOrErr returns the Subscription value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e SubscriptionPatchEdges) SubscriptionOrErr() (*Subscription, error) {
	if e.Subscription != nil {
		return e.Subscription, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: subscription.Label}
	}
	return nil, &NotLoadedError{edge: "subscription"}
}

// ValueAddItemOrErr returns the ValueAddItem value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e SubscriptionPatchEdges) ValueAddItemOrErr() (*SubscriptionPatchValueAddItem, error) {
	if e.ValueAddItem != nil {
		return e.ValueAddItem, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: subscriptionpatchvalueadditem.Label}
	}
	return nil, &NotLoadedError{edge: "value_add_item"}
}

// ValueAddPhaseOrErr returns the ValueAddPhase value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e SubscriptionPatchEdges) ValueAddPhaseOrErr() (*SubscriptionPatchValueAddPhase, error) {
	if e.ValueAddPhase != nil {
		return e.ValueAddPhase, nil
	} else if e.loadedTypes[2] {
		return nil, &NotFoundError{label: subscriptionpatchvalueaddphase.Label}
	}
	return nil, &NotLoadedError{edge: "value_add_phase"}
}

// ValueExtendPhaseOrErr returns the ValueExtendPhase value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e SubscriptionPatchEdges) ValueExtendPhaseOrErr() (*SubscriptionPatchValueExtendPhase, error) {
	if e.ValueExtendPhase != nil {
		return e.ValueExtendPhase, nil
	} else if e.loadedTypes[3] {
		return nil, &NotFoundError{label: subscriptionpatchvalueextendphase.Label}
	}
	return nil, &NotLoadedError{edge: "value_extend_phase"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*SubscriptionPatch) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case subscriptionpatch.FieldMetadata:
			values[i] = new([]byte)
		case subscriptionpatch.FieldBatchIndex:
			values[i] = new(sql.NullInt64)
		case subscriptionpatch.FieldID, subscriptionpatch.FieldNamespace, subscriptionpatch.FieldSubscriptionID, subscriptionpatch.FieldOperation, subscriptionpatch.FieldPath:
			values[i] = new(sql.NullString)
		case subscriptionpatch.FieldCreatedAt, subscriptionpatch.FieldUpdatedAt, subscriptionpatch.FieldDeletedAt, subscriptionpatch.FieldAppliedAt:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the SubscriptionPatch fields.
func (sp *SubscriptionPatch) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case subscriptionpatch.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				sp.ID = value.String
			}
		case subscriptionpatch.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				sp.Namespace = value.String
			}
		case subscriptionpatch.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				sp.CreatedAt = value.Time
			}
		case subscriptionpatch.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				sp.UpdatedAt = value.Time
			}
		case subscriptionpatch.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				sp.DeletedAt = new(time.Time)
				*sp.DeletedAt = value.Time
			}
		case subscriptionpatch.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &sp.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case subscriptionpatch.FieldSubscriptionID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field subscription_id", values[i])
			} else if value.Valid {
				sp.SubscriptionID = value.String
			}
		case subscriptionpatch.FieldAppliedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field applied_at", values[i])
			} else if value.Valid {
				sp.AppliedAt = value.Time
			}
		case subscriptionpatch.FieldBatchIndex:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field batch_index", values[i])
			} else if value.Valid {
				sp.BatchIndex = int(value.Int64)
			}
		case subscriptionpatch.FieldOperation:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field operation", values[i])
			} else if value.Valid {
				sp.Operation = value.String
			}
		case subscriptionpatch.FieldPath:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field path", values[i])
			} else if value.Valid {
				sp.Path = value.String
			}
		default:
			sp.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the SubscriptionPatch.
// This includes values selected through modifiers, order, etc.
func (sp *SubscriptionPatch) Value(name string) (ent.Value, error) {
	return sp.selectValues.Get(name)
}

// QuerySubscription queries the "subscription" edge of the SubscriptionPatch entity.
func (sp *SubscriptionPatch) QuerySubscription() *SubscriptionQuery {
	return NewSubscriptionPatchClient(sp.config).QuerySubscription(sp)
}

// QueryValueAddItem queries the "value_add_item" edge of the SubscriptionPatch entity.
func (sp *SubscriptionPatch) QueryValueAddItem() *SubscriptionPatchValueAddItemQuery {
	return NewSubscriptionPatchClient(sp.config).QueryValueAddItem(sp)
}

// QueryValueAddPhase queries the "value_add_phase" edge of the SubscriptionPatch entity.
func (sp *SubscriptionPatch) QueryValueAddPhase() *SubscriptionPatchValueAddPhaseQuery {
	return NewSubscriptionPatchClient(sp.config).QueryValueAddPhase(sp)
}

// QueryValueExtendPhase queries the "value_extend_phase" edge of the SubscriptionPatch entity.
func (sp *SubscriptionPatch) QueryValueExtendPhase() *SubscriptionPatchValueExtendPhaseQuery {
	return NewSubscriptionPatchClient(sp.config).QueryValueExtendPhase(sp)
}

// Update returns a builder for updating this SubscriptionPatch.
// Note that you need to call SubscriptionPatch.Unwrap() before calling this method if this SubscriptionPatch
// was returned from a transaction, and the transaction was committed or rolled back.
func (sp *SubscriptionPatch) Update() *SubscriptionPatchUpdateOne {
	return NewSubscriptionPatchClient(sp.config).UpdateOne(sp)
}

// Unwrap unwraps the SubscriptionPatch entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (sp *SubscriptionPatch) Unwrap() *SubscriptionPatch {
	_tx, ok := sp.config.driver.(*txDriver)
	if !ok {
		panic("db: SubscriptionPatch is not a transactional entity")
	}
	sp.config.driver = _tx.drv
	return sp
}

// String implements the fmt.Stringer.
func (sp *SubscriptionPatch) String() string {
	var builder strings.Builder
	builder.WriteString("SubscriptionPatch(")
	builder.WriteString(fmt.Sprintf("id=%v, ", sp.ID))
	builder.WriteString("namespace=")
	builder.WriteString(sp.Namespace)
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(sp.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(sp.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := sp.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", sp.Metadata))
	builder.WriteString(", ")
	builder.WriteString("subscription_id=")
	builder.WriteString(sp.SubscriptionID)
	builder.WriteString(", ")
	builder.WriteString("applied_at=")
	builder.WriteString(sp.AppliedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("batch_index=")
	builder.WriteString(fmt.Sprintf("%v", sp.BatchIndex))
	builder.WriteString(", ")
	builder.WriteString("operation=")
	builder.WriteString(sp.Operation)
	builder.WriteString(", ")
	builder.WriteString("path=")
	builder.WriteString(sp.Path)
	builder.WriteByte(')')
	return builder.String()
}

// SubscriptionPatches is a parsable slice of SubscriptionPatch.
type SubscriptionPatches []*SubscriptionPatch
