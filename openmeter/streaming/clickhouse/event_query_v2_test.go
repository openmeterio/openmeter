package clickhouse

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func TestQueryEventsTableV2_ToSQL(t *testing.T) {
	now := time.Now()
	limit := 50
	cursorTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cursorID := "event-123"

	tests := []struct {
		name     string
		query    queryEventsTableV2
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			name: "basic query with namespace only",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
				},
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id FROM openmeter.om_events WHERE namespace = ? ORDER BY time DESC, id DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", 100},
		},
		{
			name: "query with ID filter",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
					ID: &filter.FilterString{
						Eq: lo.ToPtr("event-123"),
					},
				},
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id FROM openmeter.om_events WHERE namespace = ? AND id = ? ORDER BY time DESC, id DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", "event-123", 100},
		},
		{
			name: "query with subject filter",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
					Subject: &filter.FilterString{
						Like: lo.ToPtr("%customer%"),
					},
				},
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id FROM openmeter.om_events WHERE namespace = ? AND subject LIKE ? ORDER BY time DESC, id DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", "%customer%", 100},
		},
		{
			name: "query with time filter",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
					Time: &filter.FilterTime{
						Gte: &now,
					},
				},
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id FROM openmeter.om_events WHERE namespace = ? AND time >= ? ORDER BY time DESC, id DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", now, 100},
		},
		{
			name: "query with cursor and custom limit",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
					Cursor: &pagination.Cursor{
						Time: cursorTime,
						ID:   cursorID,
					},
					Limit: &limit,
				},
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id FROM openmeter.om_events WHERE namespace = ? AND time <= ? AND (time < ? OR id < ?) ORDER BY time DESC, id DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", cursorTime.Unix(), cursorTime.Unix(), cursorID, 50},
		},
		{
			name: "query with ingested_at filter",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
					IngestedAt: &filter.FilterTime{
						Gte: &now,
					},
				},
			},
			wantSQL:  "SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id FROM openmeter.om_events WHERE namespace = ? AND ingested_at >= ? ORDER BY ingested_at DESC, id DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", now, 100},
		},
		{
			name: "query with customer filter",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
					Customers: &[]streaming.Customer{
						customer.Customer{
							ManagedResource: models.ManagedResource{
								NamespacedModel: models.NamespacedModel{
									Namespace: "my_namespace",
								},
								ID: "customer1-id",
							},
							Key: lo.ToPtr("customer1-key"),
							UsageAttribution: &customer.CustomerUsageAttribution{
								SubjectKeys: []string{"customer1-subject1", "customer1-subject2"},
							},
						},
						customer.Customer{
							ManagedResource: models.ManagedResource{
								NamespacedModel: models.NamespacedModel{
									Namespace: "my_namespace",
								},
								ID: "customer2-id",
							},
							Key: lo.ToPtr("customer2-key"),
							UsageAttribution: &customer.CustomerUsageAttribution{
								SubjectKeys: []string{"customer2-subject1", "customer2-subject2"},
							},
						},
					},
				},
			},
			wantSQL:  "WITH map('customer1-key', 'customer1-id', 'customer1-subject1', 'customer1-id', 'customer1-subject2', 'customer1-id', 'customer2-key', 'customer2-id', 'customer2-subject1', 'customer2-id', 'customer2-subject2', 'customer2-id') as subject_to_customer_id SELECT id, type, subject, source, time, data, ingested_at, stored_at, store_row_id, subject_to_customer_id[om_events.subject] AS customer_id FROM openmeter.om_events WHERE namespace = ? AND openmeter.om_events.subject IN (?) ORDER BY time DESC, id DESC LIMIT ?",
			wantArgs: []interface{}{"my_namespace", []string{"customer1-key", "customer1-subject1", "customer1-subject2", "customer2-key", "customer2-subject1", "customer2-subject2"}, 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := tt.query.toSQL()
			assert.Equal(t, tt.wantSQL, gotSQL)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}

func TestQueryEventsTableV2_ToCountRowSQL(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		query    queryEventsTableV2
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			name: "basic count query",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
				},
			},
			wantSQL:  "SELECT count() as total FROM openmeter.om_events WHERE namespace = ?",
			wantArgs: []interface{}{"my_namespace"},
		},
		{
			name: "count query with type filter",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
					Type: &filter.FilterString{
						Eq: lo.ToPtr("api-calls"),
					},
				},
			},
			wantSQL:  "SELECT count() as total FROM openmeter.om_events WHERE namespace = ? AND type = ?",
			wantArgs: []interface{}{"my_namespace", "api-calls"},
		},
		{
			name: "count query with time filter",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
					Time: &filter.FilterTime{
						Gte: &now,
					},
				},
			},
			wantSQL:  "SELECT count() as total FROM openmeter.om_events WHERE namespace = ? AND time >= ?",
			wantArgs: []interface{}{"my_namespace", now},
		},
		{
			name: "count query with subject filter",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
					Subject: &filter.FilterString{
						Like: lo.ToPtr("%customer%"),
					},
				},
			},
			wantSQL:  "SELECT count() as total FROM openmeter.om_events WHERE namespace = ? AND subject LIKE ?",
			wantArgs: []interface{}{"my_namespace", "%customer%"},
		},
		{
			name: "count query with customer filter",
			query: queryEventsTableV2{
				Database:        "openmeter",
				EventsTableName: "om_events",
				Params: streaming.ListEventsV2Params{
					Namespace: "my_namespace",
					Customers: &[]streaming.Customer{
						customer.Customer{
							ManagedResource: models.ManagedResource{
								NamespacedModel: models.NamespacedModel{
									Namespace: "my_namespace",
								},
								ID: "customer1",
							},
							UsageAttribution: &customer.CustomerUsageAttribution{
								SubjectKeys: []string{"subject1"},
							},
						},
					},
				},
			},
			wantSQL:  "SELECT count() as total FROM openmeter.om_events WHERE namespace = ? AND openmeter.om_events.subject IN (?)",
			wantArgs: []interface{}{"my_namespace", []string{"subject1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL, gotArgs := tt.query.toCountRowSQL()
			assert.Equal(t, tt.wantSQL, gotSQL)
			assert.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}
