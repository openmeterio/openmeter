// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/creditentry"
	"github.com/openmeterio/openmeter/internal/credit/postgres_connector/ent/db/feature"
)

// CreditEntry is the model entity for the CreditEntry schema.
type CreditEntry struct {
	config `json:"-"`
	// ID of the ent.
	ID string `json:"id,omitempty"`
	// CreatedAt holds the value of the "created_at" field.
	CreatedAt time.Time `json:"created_at,omitempty"`
	// UpdatedAt holds the value of the "updated_at" field.
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	// Namespace holds the value of the "namespace" field.
	Namespace string `json:"namespace,omitempty"`
	// LedgerID holds the value of the "ledger_id" field.
	LedgerID string `json:"ledger_id,omitempty"`
	// EntryType holds the value of the "entry_type" field.
	EntryType credit.EntryType `json:"entry_type,omitempty"`
	// Type holds the value of the "type" field.
	Type *credit.GrantType `json:"type,omitempty"`
	// FeatureID holds the value of the "feature_id" field.
	FeatureID *string `json:"feature_id,omitempty"`
	// Amount holds the value of the "amount" field.
	Amount *float64 `json:"amount,omitempty"`
	// Priority holds the value of the "priority" field.
	Priority uint8 `json:"priority,omitempty"`
	// EffectiveAt holds the value of the "effective_at" field.
	EffectiveAt time.Time `json:"effective_at,omitempty"`
	// ExpirationPeriodDuration holds the value of the "expiration_period_duration" field.
	ExpirationPeriodDuration *credit.ExpirationPeriodDuration `json:"expiration_period_duration,omitempty"`
	// ExpirationPeriodCount holds the value of the "expiration_period_count" field.
	ExpirationPeriodCount *uint8 `json:"expiration_period_count,omitempty"`
	// ExpirationAt holds the value of the "expiration_at" field.
	ExpirationAt *time.Time `json:"expiration_at,omitempty"`
	// RolloverType holds the value of the "rollover_type" field.
	RolloverType *credit.GrantRolloverType `json:"rollover_type,omitempty"`
	// RolloverMaxAmount holds the value of the "rollover_max_amount" field.
	RolloverMaxAmount *float64 `json:"rollover_max_amount,omitempty"`
	// Metadata holds the value of the "metadata" field.
	Metadata map[string]string `json:"metadata,omitempty"`
	// ParentID holds the value of the "parent_id" field.
	ParentID *string `json:"parent_id,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the CreditEntryQuery when eager-loading is set.
	Edges        CreditEntryEdges `json:"edges"`
	selectValues sql.SelectValues
}

// CreditEntryEdges holds the relations/edges for other nodes in the graph.
type CreditEntryEdges struct {
	// Parent holds the value of the parent edge.
	Parent *CreditEntry `json:"parent,omitempty"`
	// Children holds the value of the children edge.
	Children *CreditEntry `json:"children,omitempty"`
	// Feature holds the value of the feature edge.
	Feature *Feature `json:"feature,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [3]bool
}

// ParentOrErr returns the Parent value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e CreditEntryEdges) ParentOrErr() (*CreditEntry, error) {
	if e.Parent != nil {
		return e.Parent, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: creditentry.Label}
	}
	return nil, &NotLoadedError{edge: "parent"}
}

// ChildrenOrErr returns the Children value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e CreditEntryEdges) ChildrenOrErr() (*CreditEntry, error) {
	if e.Children != nil {
		return e.Children, nil
	} else if e.loadedTypes[1] {
		return nil, &NotFoundError{label: creditentry.Label}
	}
	return nil, &NotLoadedError{edge: "children"}
}

// FeatureOrErr returns the Feature value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e CreditEntryEdges) FeatureOrErr() (*Feature, error) {
	if e.Feature != nil {
		return e.Feature, nil
	} else if e.loadedTypes[2] {
		return nil, &NotFoundError{label: feature.Label}
	}
	return nil, &NotLoadedError{edge: "feature"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*CreditEntry) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case creditentry.FieldMetadata:
			values[i] = new([]byte)
		case creditentry.FieldAmount, creditentry.FieldRolloverMaxAmount:
			values[i] = new(sql.NullFloat64)
		case creditentry.FieldPriority, creditentry.FieldExpirationPeriodCount:
			values[i] = new(sql.NullInt64)
		case creditentry.FieldID, creditentry.FieldNamespace, creditentry.FieldLedgerID, creditentry.FieldEntryType, creditentry.FieldType, creditentry.FieldFeatureID, creditentry.FieldExpirationPeriodDuration, creditentry.FieldRolloverType, creditentry.FieldParentID:
			values[i] = new(sql.NullString)
		case creditentry.FieldCreatedAt, creditentry.FieldUpdatedAt, creditentry.FieldEffectiveAt, creditentry.FieldExpirationAt:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the CreditEntry fields.
func (ce *CreditEntry) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case creditentry.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				ce.ID = value.String
			}
		case creditentry.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				ce.CreatedAt = value.Time
			}
		case creditentry.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				ce.UpdatedAt = value.Time
			}
		case creditentry.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				ce.Namespace = value.String
			}
		case creditentry.FieldLedgerID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field ledger_id", values[i])
			} else if value.Valid {
				ce.LedgerID = value.String
			}
		case creditentry.FieldEntryType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field entry_type", values[i])
			} else if value.Valid {
				ce.EntryType = credit.EntryType(value.String)
			}
		case creditentry.FieldType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field type", values[i])
			} else if value.Valid {
				ce.Type = new(credit.GrantType)
				*ce.Type = credit.GrantType(value.String)
			}
		case creditentry.FieldFeatureID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field feature_id", values[i])
			} else if value.Valid {
				ce.FeatureID = new(string)
				*ce.FeatureID = value.String
			}
		case creditentry.FieldAmount:
			if value, ok := values[i].(*sql.NullFloat64); !ok {
				return fmt.Errorf("unexpected type %T for field amount", values[i])
			} else if value.Valid {
				ce.Amount = new(float64)
				*ce.Amount = value.Float64
			}
		case creditentry.FieldPriority:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field priority", values[i])
			} else if value.Valid {
				ce.Priority = uint8(value.Int64)
			}
		case creditentry.FieldEffectiveAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field effective_at", values[i])
			} else if value.Valid {
				ce.EffectiveAt = value.Time
			}
		case creditentry.FieldExpirationPeriodDuration:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field expiration_period_duration", values[i])
			} else if value.Valid {
				ce.ExpirationPeriodDuration = new(credit.ExpirationPeriodDuration)
				*ce.ExpirationPeriodDuration = credit.ExpirationPeriodDuration(value.String)
			}
		case creditentry.FieldExpirationPeriodCount:
			if value, ok := values[i].(*sql.NullInt64); !ok {
				return fmt.Errorf("unexpected type %T for field expiration_period_count", values[i])
			} else if value.Valid {
				ce.ExpirationPeriodCount = new(uint8)
				*ce.ExpirationPeriodCount = uint8(value.Int64)
			}
		case creditentry.FieldExpirationAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field expiration_at", values[i])
			} else if value.Valid {
				ce.ExpirationAt = new(time.Time)
				*ce.ExpirationAt = value.Time
			}
		case creditentry.FieldRolloverType:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field rollover_type", values[i])
			} else if value.Valid {
				ce.RolloverType = new(credit.GrantRolloverType)
				*ce.RolloverType = credit.GrantRolloverType(value.String)
			}
		case creditentry.FieldRolloverMaxAmount:
			if value, ok := values[i].(*sql.NullFloat64); !ok {
				return fmt.Errorf("unexpected type %T for field rollover_max_amount", values[i])
			} else if value.Valid {
				ce.RolloverMaxAmount = new(float64)
				*ce.RolloverMaxAmount = value.Float64
			}
		case creditentry.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &ce.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case creditentry.FieldParentID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field parent_id", values[i])
			} else if value.Valid {
				ce.ParentID = new(string)
				*ce.ParentID = value.String
			}
		default:
			ce.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the CreditEntry.
// This includes values selected through modifiers, order, etc.
func (ce *CreditEntry) Value(name string) (ent.Value, error) {
	return ce.selectValues.Get(name)
}

// QueryParent queries the "parent" edge of the CreditEntry entity.
func (ce *CreditEntry) QueryParent() *CreditEntryQuery {
	return NewCreditEntryClient(ce.config).QueryParent(ce)
}

// QueryChildren queries the "children" edge of the CreditEntry entity.
func (ce *CreditEntry) QueryChildren() *CreditEntryQuery {
	return NewCreditEntryClient(ce.config).QueryChildren(ce)
}

// QueryFeature queries the "feature" edge of the CreditEntry entity.
func (ce *CreditEntry) QueryFeature() *FeatureQuery {
	return NewCreditEntryClient(ce.config).QueryFeature(ce)
}

// Update returns a builder for updating this CreditEntry.
// Note that you need to call CreditEntry.Unwrap() before calling this method if this CreditEntry
// was returned from a transaction, and the transaction was committed or rolled back.
func (ce *CreditEntry) Update() *CreditEntryUpdateOne {
	return NewCreditEntryClient(ce.config).UpdateOne(ce)
}

// Unwrap unwraps the CreditEntry entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (ce *CreditEntry) Unwrap() *CreditEntry {
	_tx, ok := ce.config.driver.(*txDriver)
	if !ok {
		panic("db: CreditEntry is not a transactional entity")
	}
	ce.config.driver = _tx.drv
	return ce
}

// String implements the fmt.Stringer.
func (ce *CreditEntry) String() string {
	var builder strings.Builder
	builder.WriteString("CreditEntry(")
	builder.WriteString(fmt.Sprintf("id=%v, ", ce.ID))
	builder.WriteString("created_at=")
	builder.WriteString(ce.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(ce.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("namespace=")
	builder.WriteString(ce.Namespace)
	builder.WriteString(", ")
	builder.WriteString("ledger_id=")
	builder.WriteString(ce.LedgerID)
	builder.WriteString(", ")
	builder.WriteString("entry_type=")
	builder.WriteString(fmt.Sprintf("%v", ce.EntryType))
	builder.WriteString(", ")
	if v := ce.Type; v != nil {
		builder.WriteString("type=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := ce.FeatureID; v != nil {
		builder.WriteString("feature_id=")
		builder.WriteString(*v)
	}
	builder.WriteString(", ")
	if v := ce.Amount; v != nil {
		builder.WriteString("amount=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("priority=")
	builder.WriteString(fmt.Sprintf("%v", ce.Priority))
	builder.WriteString(", ")
	builder.WriteString("effective_at=")
	builder.WriteString(ce.EffectiveAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := ce.ExpirationPeriodDuration; v != nil {
		builder.WriteString("expiration_period_duration=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := ce.ExpirationPeriodCount; v != nil {
		builder.WriteString("expiration_period_count=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := ce.ExpirationAt; v != nil {
		builder.WriteString("expiration_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	if v := ce.RolloverType; v != nil {
		builder.WriteString("rollover_type=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	if v := ce.RolloverMaxAmount; v != nil {
		builder.WriteString("rollover_max_amount=")
		builder.WriteString(fmt.Sprintf("%v", *v))
	}
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", ce.Metadata))
	builder.WriteString(", ")
	if v := ce.ParentID; v != nil {
		builder.WriteString("parent_id=")
		builder.WriteString(*v)
	}
	builder.WriteByte(')')
	return builder.String()
}

// CreditEntries is a parsable slice of CreditEntry.
type CreditEntries []*CreditEntry
