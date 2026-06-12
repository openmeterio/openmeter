package adapter

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// fakeStreamingConnector serves ListSubjectsV2 from an in-memory sorted key
// list, honoring the keyset cursor and limit like the ClickHouse query does.
type fakeStreamingConnector struct {
	streaming.Connector

	keys  []string
	calls int
}

func (f *fakeStreamingConnector) ListSubjectsV2(_ context.Context, params streaming.ListSubjectsV2Params) ([]string, error) {
	f.calls++

	limit := lo.FromPtr(params.Limit)

	page := []string{}
	for _, key := range f.keys {
		if params.Cursor != nil && key <= params.Cursor.ID {
			continue
		}
		if len(page) == limit {
			break
		}
		page = append(page, key)
	}

	return page, nil
}

// fakeCustomerService resolves attribution from subject key → customer ID and
// customer key → customer ID maps, mirroring the two lookups
// attributedSubjectKeys performs: customers whose usage attribution subject
// keys contain the subject key, and customers whose own key is the subject key.
type fakeCustomerService struct {
	customer.Service

	customerIDBySubjectKey  map[string]string
	customerIDByCustomerKey map[string]string
}

func (f *fakeCustomerService) ListCustomers(_ context.Context, input customer.ListCustomersInput) (pagination.Result[customer.Customer], error) {
	var items []customer.Customer

	if input.UsageAttributionSubjectKey != nil && input.UsageAttributionSubjectKey.In != nil {
		for _, key := range *input.UsageAttributionSubjectKey.In {
			if customerID, ok := f.customerIDBySubjectKey[key]; ok {
				items = append(items, customer.Customer{
					ManagedResource: models.ManagedResource{ID: customerID},
					UsageAttribution: &customer.CustomerUsageAttribution{
						SubjectKeys: []string{key},
					},
				})
			}
		}
	}

	if input.Key != nil && input.Key.In != nil {
		for _, key := range *input.Key.In {
			if customerID, ok := f.customerIDByCustomerKey[key]; ok {
				items = append(items, customer.Customer{
					ManagedResource: models.ManagedResource{ID: customerID},
					Key:             lo.ToPtr(key),
				})
			}
		}
	}

	return pagination.Result[customer.Customer]{Items: items, TotalCount: len(items)}, nil
}

func TestListSubjects(t *testing.T) {
	newAdapter := func(keys []string, attribution map[string]string) (*adapter, *fakeStreamingConnector) {
		connector := &fakeStreamingConnector{keys: keys}
		return &adapter{
			streamingConnector: connector,
			customerService:    &fakeCustomerService{customerIDBySubjectKey: attribution},
		}, connector
	}

	t.Run("WithoutAttributedFilterSingleBatch", func(t *testing.T) {
		// given:
		// - three subjects, the middle one attributed to a customer
		// when:
		// - listing without the attributed filter
		// then:
		// - all subjects are returned in one streaming batch, no next cursor
		a, connector := newAdapter([]string{"s1", "s2", "s3"}, map[string]string{"s2": "customer-2"})

		result, err := a.ListSubjects(t.Context(), meterevent.ListSubjectsParams{
			Namespace: "ns",
			Limit:     lo.ToPtr(10),
		})
		require.NoError(t, err)

		require.Equal(t, []meterevent.Subject{
			{Key: "s1"},
			{Key: "s2"},
			{Key: "s3"},
		}, result.Items)
		require.Nil(t, result.NextCursor)
		require.Equal(t, 1, connector.calls)
	})

	t.Run("AttributedByCustomerKey", func(t *testing.T) {
		// given:
		// - a customer attributed by its own key, with no usage attribution subject keys
		// when:
		// - listing attributed subjects
		// then:
		// - the subject matching the customer key is attributed
		connector := &fakeStreamingConnector{keys: []string{"s1", "s2"}}
		a := &adapter{
			streamingConnector: connector,
			customerService:    &fakeCustomerService{customerIDByCustomerKey: map[string]string{"s2": "customer-2"}},
		}

		result, err := a.ListSubjects(t.Context(), meterevent.ListSubjectsParams{
			Namespace:  "ns",
			Limit:      lo.ToPtr(10),
			Attributed: lo.ToPtr(true),
		})
		require.NoError(t, err)

		require.Equal(t, []meterevent.Subject{{Key: "s2"}}, result.Items)
		require.Nil(t, result.NextCursor)
	})

	t.Run("FullPageEmitsNextCursor", func(t *testing.T) {
		a, _ := newAdapter([]string{"s1", "s2", "s3"}, nil)

		result, err := a.ListSubjects(t.Context(), meterevent.ListSubjectsParams{
			Namespace: "ns",
			Limit:     lo.ToPtr(2),
		})
		require.NoError(t, err)

		require.Len(t, result.Items, 2)
		require.NotNil(t, result.NextCursor)
		require.Equal(t, "s2", result.NextCursor.ID)

		// The follow-up page returns the remainder without a next cursor.
		result, err = a.ListSubjects(t.Context(), meterevent.ListSubjectsParams{
			Namespace: "ns",
			Limit:     lo.ToPtr(2),
			Cursor:    result.NextCursor,
		})
		require.NoError(t, err)
		require.Equal(t, []meterevent.Subject{{Key: "s3"}}, result.Items)
		require.Nil(t, result.NextCursor)
	})

	t.Run("AttributedFilterScansAcrossBatches", func(t *testing.T) {
		// given:
		// - 250 subjects spanning multiple streaming batches, every tenth one attributed
		keys := make([]string, 0, 250)
		attribution := map[string]string{}
		for i := range 250 {
			key := fmt.Sprintf("s%03d", i)
			keys = append(keys, key)
			if i%10 == 0 {
				attribution[key] = "customer-" + key
			}
		}

		a, connector := newAdapter(keys, attribution)

		// when:
		// - listing attributed subjects with a page size larger than one batch yields
		// then:
		// - the scan crosses batches until the page fills and continues via the cursor
		result, err := a.ListSubjects(t.Context(), meterevent.ListSubjectsParams{
			Namespace:  "ns",
			Limit:      lo.ToPtr(15),
			Attributed: lo.ToPtr(true),
		})
		require.NoError(t, err)

		require.Len(t, result.Items, 15)
		for _, subject := range result.Items {
			require.Contains(t, attribution, subject.Key, "subject %s must be attributed", subject.Key)
		}
		require.Greater(t, connector.calls, 1)
		require.NotNil(t, result.NextCursor)
		require.Equal(t, "s140", result.NextCursor.ID)

		// The follow-up page returns the remaining 10 attributed subjects and
		// reports exhaustion.
		result, err = a.ListSubjects(t.Context(), meterevent.ListSubjectsParams{
			Namespace:  "ns",
			Limit:      lo.ToPtr(15),
			Attributed: lo.ToPtr(true),
			Cursor:     result.NextCursor,
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 10)
		require.Nil(t, result.NextCursor)
	})

	t.Run("UnattributedFilterDropsAttributedSubjects", func(t *testing.T) {
		a, _ := newAdapter([]string{"s1", "s2", "s3"}, map[string]string{"s2": "customer-2"})

		result, err := a.ListSubjects(t.Context(), meterevent.ListSubjectsParams{
			Namespace:  "ns",
			Limit:      lo.ToPtr(10),
			Attributed: lo.ToPtr(false),
		})
		require.NoError(t, err)

		require.Equal(t, []meterevent.Subject{{Key: "s1"}, {Key: "s3"}}, result.Items)
		require.Nil(t, result.NextCursor)
	})

	t.Run("ScanRoundCapReturnsContinuationCursor", func(t *testing.T) {
		// given:
		// - more fully-attributed subjects than the scan cap covers (10 rounds x 100)
		keys := make([]string, 0, 1100)
		attribution := map[string]string{}
		for i := range 1100 {
			key := fmt.Sprintf("s%04d", i)
			keys = append(keys, key)
			attribution[key] = "customer-" + key
		}

		a, connector := newAdapter(keys, attribution)

		// when:
		// - listing unattributed subjects, which never match
		// then:
		// - the scan stops at the round cap with an empty page and a continuation cursor
		result, err := a.ListSubjects(t.Context(), meterevent.ListSubjectsParams{
			Namespace:  "ns",
			Limit:      lo.ToPtr(10),
			Attributed: lo.ToPtr(false),
		})
		require.NoError(t, err)

		require.Empty(t, result.Items)
		require.Equal(t, listSubjectsMaxScanRounds, connector.calls)
		require.NotNil(t, result.NextCursor)
		require.Equal(t, "s0999", result.NextCursor.ID)
	})
}
