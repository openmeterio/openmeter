package e2e

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func TestEventSubjectsV3(t *testing.T) {
	c := newV3Client(t)

	// Unique subject prefix so re-runs against a shared DB don't collide and
	// the key filter only matches this test's fixtures.
	prefix := uniqueKey("subj")
	keys := []string{prefix + "_a", prefix + "_b", prefix + "_c"}

	now := time.Now()
	events := make([]apiv3.MeteringEvent, 0, len(keys))
	for _, key := range keys {
		events = append(events, apiv3.MeteringEvent{
			Id:          gofakeit.UUID(),
			Source:      "e2e",
			Specversion: "1.0",
			Type:        "ingest",
			Subject:     key,
			Time:        nullable.NewNullableWithValue[apiv3.DateTime](now),
			Data:        nullable.NewNullableWithValue(map[string]interface{}{"duration_ms": "100"}),
		})
	}

	status, problem := c.IngestEvents(events)
	require.Nil(t, problem, "ingest problem: %+v", problem)
	require.Equal(t, http.StatusAccepted, status)

	// Attribute the middle subject to a customer so the attributed filter has
	// both kinds of subjects to work with.
	status, _, problem = c.CreateCustomer(apiv3.CreateCustomerRequest{
		Key:  uniqueKey("subj_customer"),
		Name: "Event Subjects Test Customer",
		UsageAttribution: &apiv3.BillingCustomerUsageAttribution{
			SubjectKeys: []apiv3.UsageAttributionSubjectKey{keys[1]},
		},
	})
	require.Nil(t, problem, "create customer problem: %+v", problem)
	require.Equal(t, http.StatusCreated, status)

	filterQuery := "?filter[key][contains]=" + url.QueryEscape(prefix)

	subjectKeys := func(resp *apiv3.SubjectPaginatedResponse) []string {
		return lo.Map(resp.Data, func(s apiv3.MeteringEventSubject, _ int) string { return s.Key })
	}

	// Wait for the sink worker to land the events in ClickHouse.
	assert.EventuallyWithT(t, func(t *assert.CollectT) {
		status, resp, problem := c.ListEventSubjects(filterQuery)
		require.Nil(t, problem, "list problem: %+v", problem)
		require.Equal(t, http.StatusOK, status)
		require.NotNil(t, resp)

		// Subjects are ordered by key ascending.
		assert.Equal(t, keys, subjectKeys(resp))
	}, time.Minute, time.Second)

	t.Run("CursorPagination", func(t *testing.T) {
		status, page1, problem := c.ListEventSubjects(filterQuery + "&page[size]=2")
		require.Nil(t, problem, "list problem: %+v", problem)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, keys[:2], subjectKeys(page1))

		next, err := page1.Meta.Page.Next.Get()
		require.NoError(t, err, "full first page must carry a next cursor")

		status, page2, problem := c.ListEventSubjects(filterQuery + "&page[size]=2&page[after]=" + url.QueryEscape(next))
		require.Nil(t, problem, "list problem: %+v", problem)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, keys[2:], subjectKeys(page2))

		// Only an absent next cursor signals exhaustion; a short page does not.
		_, err = page2.Meta.Page.Next.Get()
		require.Error(t, err, "exhausted listing must not carry a next cursor")
	})

	t.Run("InvalidParams", func(t *testing.T) {
		for _, query := range []string{
			"?page[before]=" + url.QueryEscape(pagination.NewCursor(time.Time{}, "x").Encode()),
			"?page[after]=not-a-cursor",
			"?page[size]=0",
			"?page[size]=101",
			"?filter[attributed][eq]=notabool",
		} {
			status, _, problem := c.ListEventSubjects(query)
			require.Equal(t, http.StatusBadRequest, status, "query %q must be rejected: %+v", query, problem)
		}
	})

	t.Run("AttributedFilter", func(t *testing.T) {
		status, resp, problem := c.ListEventSubjects(filterQuery + "&filter[attributed][eq]=true")
		require.Nil(t, problem, "list problem: %+v", problem)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, []string{keys[1]}, subjectKeys(resp))
	})

	t.Run("UnattributedFilter", func(t *testing.T) {
		status, resp, problem := c.ListEventSubjects(filterQuery + "&filter[attributed][eq]=false")
		require.Nil(t, problem, "list problem: %+v", problem)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, []string{keys[0], keys[2]}, subjectKeys(resp))
	})

	t.Run("KeyFilterNoMatch", func(t *testing.T) {
		status, resp, problem := c.ListEventSubjects("?filter[key][contains]=" + url.QueryEscape(prefix+"_no_such_subject"))
		require.Nil(t, problem, "list problem: %+v", problem)
		require.Equal(t, http.StatusOK, status)
		require.Empty(t, resp.Data)
	})
}
