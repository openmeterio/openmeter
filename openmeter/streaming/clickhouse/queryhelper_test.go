package clickhouse

import (
	"testing"

	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSelectCustomerIDColumnBindsUsageAttribution(t *testing.T) {
	maliciousCustomerKey := "customer', (SELECT sleep(3)), 'key"
	maliciousSubjectKey := "subject', (SELECT sleep(3)), 'key"

	query := sqlbuilder.ClickHouse.NewSelectBuilder()
	query.Select("id").From("openmeter.om_events")
	query = selectCustomerIdColumn("om_events", []streaming.Customer{
		customer.Customer{
			ManagedResource: models.ManagedResource{ID: "customer-1-id"},
			Key:             lo.ToPtr(maliciousCustomerKey),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"customer-1-subject-1", "customer-1-subject-2"},
			},
		},
		customer.Customer{
			ManagedResource: models.ManagedResource{ID: "customer-2-id"},
			Key:             lo.ToPtr("customer-2-key"),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"customer-2-subject-1", maliciousSubjectKey},
			},
		},
	}, query)

	gotSQL, gotArgs := query.Build()

	require.Equal(t, "SELECT id, mapFromArrays(?, ?)[om_events.subject] AS customer_id FROM openmeter.om_events", gotSQL)
	require.NotContains(t, gotSQL, maliciousCustomerKey)
	require.NotContains(t, gotSQL, maliciousSubjectKey)
	require.Equal(t, []interface{}{
		[]string{
			maliciousCustomerKey,
			"customer-1-subject-1",
			"customer-1-subject-2",
			"customer-2-key",
			"customer-2-subject-1",
			maliciousSubjectKey,
		},
		[]string{
			"customer-1-id",
			"customer-1-id",
			"customer-1-id",
			"customer-2-id",
			"customer-2-id",
			"customer-2-id",
		},
	}, gotArgs)
}
