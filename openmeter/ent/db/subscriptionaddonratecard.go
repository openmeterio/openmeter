// Code generated by ent, DO NOT EDIT.

package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/openmeterio/openmeter/openmeter/ent/db/addonratecard"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddon"
	"github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddonratecard"
)

// SubscriptionAddonRateCard is the model entity for the SubscriptionAddonRateCard schema.
type SubscriptionAddonRateCard struct {
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
	// SubscriptionAddonID holds the value of the "subscription_addon_id" field.
	SubscriptionAddonID string `json:"subscription_addon_id,omitempty"`
	// AddonRatecardID holds the value of the "addon_ratecard_id" field.
	AddonRatecardID string `json:"addon_ratecard_id,omitempty"`
	// Edges holds the relations/edges for other nodes in the graph.
	// The values are being populated by the SubscriptionAddonRateCardQuery when eager-loading is set.
	Edges        SubscriptionAddonRateCardEdges `json:"edges"`
	selectValues sql.SelectValues
}

// SubscriptionAddonRateCardEdges holds the relations/edges for other nodes in the graph.
type SubscriptionAddonRateCardEdges struct {
	// SubscriptionAddon holds the value of the subscription_addon edge.
	SubscriptionAddon *SubscriptionAddon `json:"subscription_addon,omitempty"`
	// Items holds the value of the items edge.
	Items []*SubscriptionAddonRateCardItemLink `json:"items,omitempty"`
	// AddonRatecard holds the value of the addon_ratecard edge.
	AddonRatecard *AddonRateCard `json:"addon_ratecard,omitempty"`
	// loadedTypes holds the information for reporting if a
	// type was loaded (or requested) in eager-loading or not.
	loadedTypes [3]bool
}

// SubscriptionAddonOrErr returns the SubscriptionAddon value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e SubscriptionAddonRateCardEdges) SubscriptionAddonOrErr() (*SubscriptionAddon, error) {
	if e.SubscriptionAddon != nil {
		return e.SubscriptionAddon, nil
	} else if e.loadedTypes[0] {
		return nil, &NotFoundError{label: subscriptionaddon.Label}
	}
	return nil, &NotLoadedError{edge: "subscription_addon"}
}

// ItemsOrErr returns the Items value or an error if the edge
// was not loaded in eager-loading.
func (e SubscriptionAddonRateCardEdges) ItemsOrErr() ([]*SubscriptionAddonRateCardItemLink, error) {
	if e.loadedTypes[1] {
		return e.Items, nil
	}
	return nil, &NotLoadedError{edge: "items"}
}

// AddonRatecardOrErr returns the AddonRatecard value or an error if the edge
// was not loaded in eager-loading, or loaded but was not found.
func (e SubscriptionAddonRateCardEdges) AddonRatecardOrErr() (*AddonRateCard, error) {
	if e.AddonRatecard != nil {
		return e.AddonRatecard, nil
	} else if e.loadedTypes[2] {
		return nil, &NotFoundError{label: addonratecard.Label}
	}
	return nil, &NotLoadedError{edge: "addon_ratecard"}
}

// scanValues returns the types for scanning values from sql.Rows.
func (*SubscriptionAddonRateCard) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case subscriptionaddonratecard.FieldMetadata:
			values[i] = new([]byte)
		case subscriptionaddonratecard.FieldID, subscriptionaddonratecard.FieldNamespace, subscriptionaddonratecard.FieldSubscriptionAddonID, subscriptionaddonratecard.FieldAddonRatecardID:
			values[i] = new(sql.NullString)
		case subscriptionaddonratecard.FieldCreatedAt, subscriptionaddonratecard.FieldUpdatedAt, subscriptionaddonratecard.FieldDeletedAt:
			values[i] = new(sql.NullTime)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the SubscriptionAddonRateCard fields.
func (sarc *SubscriptionAddonRateCard) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case subscriptionaddonratecard.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				sarc.ID = value.String
			}
		case subscriptionaddonratecard.FieldNamespace:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field namespace", values[i])
			} else if value.Valid {
				sarc.Namespace = value.String
			}
		case subscriptionaddonratecard.FieldCreatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field created_at", values[i])
			} else if value.Valid {
				sarc.CreatedAt = value.Time
			}
		case subscriptionaddonratecard.FieldUpdatedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field updated_at", values[i])
			} else if value.Valid {
				sarc.UpdatedAt = value.Time
			}
		case subscriptionaddonratecard.FieldDeletedAt:
			if value, ok := values[i].(*sql.NullTime); !ok {
				return fmt.Errorf("unexpected type %T for field deleted_at", values[i])
			} else if value.Valid {
				sarc.DeletedAt = new(time.Time)
				*sarc.DeletedAt = value.Time
			}
		case subscriptionaddonratecard.FieldMetadata:
			if value, ok := values[i].(*[]byte); !ok {
				return fmt.Errorf("unexpected type %T for field metadata", values[i])
			} else if value != nil && len(*value) > 0 {
				if err := json.Unmarshal(*value, &sarc.Metadata); err != nil {
					return fmt.Errorf("unmarshal field metadata: %w", err)
				}
			}
		case subscriptionaddonratecard.FieldSubscriptionAddonID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field subscription_addon_id", values[i])
			} else if value.Valid {
				sarc.SubscriptionAddonID = value.String
			}
		case subscriptionaddonratecard.FieldAddonRatecardID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field addon_ratecard_id", values[i])
			} else if value.Valid {
				sarc.AddonRatecardID = value.String
			}
		default:
			sarc.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the SubscriptionAddonRateCard.
// This includes values selected through modifiers, order, etc.
func (sarc *SubscriptionAddonRateCard) Value(name string) (ent.Value, error) {
	return sarc.selectValues.Get(name)
}

// QuerySubscriptionAddon queries the "subscription_addon" edge of the SubscriptionAddonRateCard entity.
func (sarc *SubscriptionAddonRateCard) QuerySubscriptionAddon() *SubscriptionAddonQuery {
	return NewSubscriptionAddonRateCardClient(sarc.config).QuerySubscriptionAddon(sarc)
}

// QueryItems queries the "items" edge of the SubscriptionAddonRateCard entity.
func (sarc *SubscriptionAddonRateCard) QueryItems() *SubscriptionAddonRateCardItemLinkQuery {
	return NewSubscriptionAddonRateCardClient(sarc.config).QueryItems(sarc)
}

// QueryAddonRatecard queries the "addon_ratecard" edge of the SubscriptionAddonRateCard entity.
func (sarc *SubscriptionAddonRateCard) QueryAddonRatecard() *AddonRateCardQuery {
	return NewSubscriptionAddonRateCardClient(sarc.config).QueryAddonRatecard(sarc)
}

// Update returns a builder for updating this SubscriptionAddonRateCard.
// Note that you need to call SubscriptionAddonRateCard.Unwrap() before calling this method if this SubscriptionAddonRateCard
// was returned from a transaction, and the transaction was committed or rolled back.
func (sarc *SubscriptionAddonRateCard) Update() *SubscriptionAddonRateCardUpdateOne {
	return NewSubscriptionAddonRateCardClient(sarc.config).UpdateOne(sarc)
}

// Unwrap unwraps the SubscriptionAddonRateCard entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (sarc *SubscriptionAddonRateCard) Unwrap() *SubscriptionAddonRateCard {
	_tx, ok := sarc.config.driver.(*txDriver)
	if !ok {
		panic("db: SubscriptionAddonRateCard is not a transactional entity")
	}
	sarc.config.driver = _tx.drv
	return sarc
}

// String implements the fmt.Stringer.
func (sarc *SubscriptionAddonRateCard) String() string {
	var builder strings.Builder
	builder.WriteString("SubscriptionAddonRateCard(")
	builder.WriteString(fmt.Sprintf("id=%v, ", sarc.ID))
	builder.WriteString("namespace=")
	builder.WriteString(sarc.Namespace)
	builder.WriteString(", ")
	builder.WriteString("created_at=")
	builder.WriteString(sarc.CreatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	builder.WriteString("updated_at=")
	builder.WriteString(sarc.UpdatedAt.Format(time.ANSIC))
	builder.WriteString(", ")
	if v := sarc.DeletedAt; v != nil {
		builder.WriteString("deleted_at=")
		builder.WriteString(v.Format(time.ANSIC))
	}
	builder.WriteString(", ")
	builder.WriteString("metadata=")
	builder.WriteString(fmt.Sprintf("%v", sarc.Metadata))
	builder.WriteString(", ")
	builder.WriteString("subscription_addon_id=")
	builder.WriteString(sarc.SubscriptionAddonID)
	builder.WriteString(", ")
	builder.WriteString("addon_ratecard_id=")
	builder.WriteString(sarc.AddonRatecardID)
	builder.WriteByte(')')
	return builder.String()
}

// SubscriptionAddonRateCards is a parsable slice of SubscriptionAddonRateCard.
type SubscriptionAddonRateCards []*SubscriptionAddonRateCard
