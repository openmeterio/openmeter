package e2e

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func TestEventSubjectsV3(t *testing.T) {
	c := newV3Client(t)
	ctx := t.Context()

	// Unique subject prefix so re-runs against a shared DB don't collide and
	// the key filter only matches this test's fixtures.
	prefix := uniqueKey("subj")
	keys := []string{prefix + "_a", prefix + "_b", prefix + "_c"}

	now := time.Now()
	events := make([]v3sdk.EventInput, 0, len(keys))
	for _, key := range keys {
		events = append(events, v3sdk.EventInput{
			ID:          gofakeit.UUID(),
			Source:      "e2e",
			Specversion: lo.ToPtr("1.0"),
			Type:        "ingest",
			Subject:     key,
			Time:        v3sdk.NullableValue(now),
			Data:        v3sdk.NullableValue(map[string]any{"duration_ms": "100"}),
		})
	}

	c.requireStatus(http.StatusAccepted, c.Events.IngestEventsJSON(ctx, v3sdk.Many(events)))

	// Attribute the middle subject via usage attribution subject keys and the
	// last subject via the customer's own key, so the attributed filter
	// exercises both attribution legs and leaves one subject unattributed.
	_, err := c.Customers.Create(ctx, v3sdk.CreateCustomerRequest{
		Key:  uniqueKey("subj_customer"),
		Name: "Event Subjects Test Customer",
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{keys[1]},
		},
	})
	c.requireStatus(http.StatusCreated, err)

	_, err = c.Customers.Create(ctx, v3sdk.CreateCustomerRequest{
		Key:  keys[2],
		Name: "Event Subjects Test Customer By Key",
	})
	c.requireStatus(http.StatusCreated, err)

	keyFilter := &v3sdk.EventSubjectFilter{
		Key: &v3sdk.StringFilter{Contains: lo.ToPtr(prefix)},
	}

	subjectKeys := func(resp *v3sdk.EventSubjectPaginatedResponse) []string {
		return lo.Map(resp.Data, func(s v3sdk.EventSubject, _ int) string { return s.Key })
	}

	// Wait for the sink worker to land the events in ClickHouse.
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		resp, err := c.Events.ListSubjects(ctx, v3sdk.EventSubjectListParams{Filter: keyFilter})
		require.NoError(collect, err)

		// Subjects are ordered by key ascending.
		assert.Equal(collect, keys, subjectKeys(resp))
	}, time.Minute, time.Second)

	t.Run("CursorPagination", func(t *testing.T) {
		page1, err := c.Events.ListSubjects(ctx, v3sdk.EventSubjectListParams{
			Page:   &v3sdk.CursorPageParams{Size: lo.ToPtr(2)},
			Filter: keyFilter,
		})
		c.requireStatus(http.StatusOK, err)
		require.Equal(t, keys[:2], subjectKeys(page1))
		require.Equal(t, int64(2), page1.Meta.Page.Size)

		// First and last carry the item cursors bounding the page.
		first, err := pagination.DecodeCursor(lo.FromPtr(page1.Meta.Page.First))
		require.NoError(t, err)
		require.Equal(t, keys[0], first.ID)

		last, err := pagination.DecodeCursor(lo.FromPtr(page1.Meta.Page.Last))
		require.NoError(t, err)
		require.Equal(t, keys[1], last.ID)

		next, err := page1.Meta.Page.Next.Get()
		require.NoError(t, err, "full first page must carry a next cursor")

		page2, err := c.Events.ListSubjects(ctx, v3sdk.EventSubjectListParams{
			Page:   &v3sdk.CursorPageParams{Size: lo.ToPtr(2), After: lo.ToPtr(next)},
			Filter: keyFilter,
		})
		c.requireStatus(http.StatusOK, err)
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
			status, _, problem := c.doMalformedRequest(http.MethodGet, "/events/subjects"+query, nil)
			require.Equal(t, http.StatusBadRequest, status, "query %q must be rejected: %+v", query, problem)
		}
	})

	t.Run("AttributedFilter", func(t *testing.T) {
		// keys[1] is attributed via usage attribution subject keys, keys[2]
		// via the customer's own key.
		resp, err := c.Events.ListSubjects(ctx, v3sdk.EventSubjectListParams{
			Filter: &v3sdk.EventSubjectFilter{
				Key:        keyFilter.Key,
				Attributed: &v3sdk.BooleanFilter{Eq: lo.ToPtr(true)},
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.Equal(t, []string{keys[1], keys[2]}, subjectKeys(resp))
	})

	t.Run("UnattributedFilter", func(t *testing.T) {
		resp, err := c.Events.ListSubjects(ctx, v3sdk.EventSubjectListParams{
			Filter: &v3sdk.EventSubjectFilter{
				Key:        keyFilter.Key,
				Attributed: &v3sdk.BooleanFilter{Eq: lo.ToPtr(false)},
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.Equal(t, []string{keys[0]}, subjectKeys(resp))
	})

	t.Run("KeyFilterNoMatch", func(t *testing.T) {
		resp, err := c.Events.ListSubjects(ctx, v3sdk.EventSubjectListParams{
			Filter: &v3sdk.EventSubjectFilter{
				Key: &v3sdk.StringFilter{Contains: lo.ToPtr(prefix + "_no_such_subject")},
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.Empty(t, resp.Data)
		require.Nil(t, resp.Meta.Page.First, "empty result must not carry page bounds")
		require.Nil(t, resp.Meta.Page.Last, "empty result must not carry page bounds")
	})
}
