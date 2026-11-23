package eventtypes

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/filters"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/response"
	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/openmeter/notification/v4/definitions"
)

func newTestHandler(t *testing.T) Handler {
	t.Helper()

	registry, err := v4.NewEventRegistry(v4.EventRegistryConfig{
		Logger:      slog.New(slog.DiscardHandler),
		Definitions: definitions.All(),
	})
	require.NoError(t, err)

	return New(registry)
}

func TestListNotificationEventTypes(t *testing.T) {
	h := newTestHandler(t)

	list := func(t *testing.T, params ListNotificationEventTypesParams) (int, response.CursorPaginationResponse[apiv3models.BillingNotificationEventTypeSummary]) {
		t.Helper()

		rec := httptest.NewRecorder()
		h.ListNotificationEventTypes().With(params).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		var body response.CursorPaginationResponse[apiv3models.BillingNotificationEventTypeSummary]
		if rec.Code == http.StatusOK {
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		}

		return rec.Code, body
	}

	types := func(items []apiv3models.BillingNotificationEventTypeSummary) []string {
		return lo.Map(items, func(item apiv3models.BillingNotificationEventTypeSummary, _ int) string {
			return item.Type
		})
	}

	t.Run("all, in version then name order", func(t *testing.T) {
		code, body := list(t, ListNotificationEventTypesParams{})
		require.Equal(t, http.StatusOK, code)

		require.Equal(t, []string{
			"v1.billing.invoice.created",
			"v1.billing.invoice.updated",
			"v1.entitlement.balance_threshold.triggered",
			"v1.entitlement.usage_period.reset",
		}, types(body.Data))
		require.True(t, body.Meta.Page.Next.IsNull(), "a single page has no next page")
		require.True(t, body.Meta.Page.Previous.IsNull())

		// The descriptions name the legacy type each one replaces.
		require.Contains(t, body.Data[0].Description, "`invoice.created`")
		require.Contains(t, body.Data[3].Description, "`entitlements.reset`")
		require.Nil(t, body.Data[0].DeprecatedBy)
	})

	t.Run("filter by type", func(t *testing.T) {
		code, body := list(t, ListNotificationEventTypesParams{
			Filter: &apiv3models.ListNotificationEventTypesParamsFilter{
				Type: &filters.FilterString{Contains: lo.ToPtr("entitlement")},
			},
		})
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, []string{
			"v1.entitlement.balance_threshold.triggered",
			"v1.entitlement.usage_period.reset",
		}, types(body.Data))

		code, body = list(t, ListNotificationEventTypesParams{
			Filter: &apiv3models.ListNotificationEventTypesParamsFilter{
				Type: &filters.FilterString{Eq: lo.ToPtr("v1.billing.invoice.updated")},
			},
		})
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, []string{"v1.billing.invoice.updated"}, types(body.Data))
	})

	cursorOf := func(t *testing.T, c nullable.Nullable[string]) string {
		t.Helper()

		value, err := c.Get()
		require.NoError(t, err, "the cursor must be set")

		return value
	}

	t.Run("pages forward and back", func(t *testing.T) {
		code, first := list(t, ListNotificationEventTypesParams{
			Page: &apiv3models.CursorPaginationQuery{Size: lo.ToPtr(3)},
		})
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, []string{
			"v1.billing.invoice.created",
			"v1.billing.invoice.updated",
			"v1.entitlement.balance_threshold.triggered",
		}, types(first.Data))
		require.True(t, first.Meta.Page.Previous.IsNull())

		code, second := list(t, ListNotificationEventTypesParams{
			Page: &apiv3models.CursorPaginationQuery{Size: lo.ToPtr(3), After: lo.ToPtr(cursorOf(t, first.Meta.Page.Next))},
		})
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, []string{"v1.entitlement.usage_period.reset"}, types(second.Data))
		require.True(t, second.Meta.Page.Next.IsNull())

		// Going back from the second page returns the first page again.
		code, back := list(t, ListNotificationEventTypesParams{
			Page: &apiv3models.CursorPaginationQuery{Size: lo.ToPtr(3), Before: lo.ToPtr(cursorOf(t, second.Meta.Page.Previous))},
		})
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, types(first.Data), types(back.Data))
		require.True(t, back.Meta.Page.Previous.IsNull())
		require.Equal(t, cursorOf(t, first.Meta.Page.Next), cursorOf(t, back.Meta.Page.Next))
	})

	t.Run("pages with a filter", func(t *testing.T) {
		filter := &apiv3models.ListNotificationEventTypesParamsFilter{
			Type: &filters.FilterString{Contains: lo.ToPtr("entitlement")},
		}

		code, first := list(t, ListNotificationEventTypesParams{
			Page:   &apiv3models.CursorPaginationQuery{Size: lo.ToPtr(1)},
			Filter: filter,
		})
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, []string{"v1.entitlement.balance_threshold.triggered"}, types(first.Data))

		code, second := list(t, ListNotificationEventTypesParams{
			Page:   &apiv3models.CursorPaginationQuery{Size: lo.ToPtr(1), After: lo.ToPtr(cursorOf(t, first.Meta.Page.Next))},
			Filter: filter,
		})
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, []string{"v1.entitlement.usage_period.reset"}, types(second.Data))
		require.True(t, second.Meta.Page.Next.IsNull())
	})

	t.Run("a filter change keeps next and previous to pages with matches", func(t *testing.T) {
		cursor := func(eventType string) *string {
			return lo.ToPtr(base64.StdEncoding.EncodeToString([]byte(eventType)))
		}

		// Before the first entitlement type, only invoice types match "billing",
		// and none comes after the cursor.
		code, body := list(t, ListNotificationEventTypesParams{
			Page: &apiv3models.CursorPaginationQuery{Before: cursor("v1.entitlement.balance_threshold.triggered")},
			Filter: &apiv3models.ListNotificationEventTypesParamsFilter{
				Type: &filters.FilterString{Contains: lo.ToPtr("billing")},
			},
		})
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, []string{"v1.billing.invoice.created", "v1.billing.invoice.updated"}, types(body.Data))
		require.True(t, body.Meta.Page.Next.IsNull(), "no billing type comes after the cursor")
		require.True(t, body.Meta.Page.Previous.IsNull())

		// After the last invoice type, only entitlement types match
		// "entitlement", and none comes before the cursor.
		code, body = list(t, ListNotificationEventTypesParams{
			Page: &apiv3models.CursorPaginationQuery{After: cursor("v1.billing.invoice.updated")},
			Filter: &apiv3models.ListNotificationEventTypesParamsFilter{
				Type: &filters.FilterString{Contains: lo.ToPtr("entitlement")},
			},
		})
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, []string{"v1.entitlement.balance_threshold.triggered", "v1.entitlement.usage_period.reset"}, types(body.Data))
		require.True(t, body.Meta.Page.Previous.IsNull(), "no entitlement type comes before the cursor")
		require.True(t, body.Meta.Page.Next.IsNull())
	})

	t.Run("invalid cursors", func(t *testing.T) {
		for name, page := range map[string]*apiv3models.CursorPaginationQuery{
			"not base64":         {After: lo.ToPtr("not base64!")},
			"not an event type":  {Before: lo.ToPtr(base64.StdEncoding.EncodeToString([]byte("invoice.created")))},
			"unknown event type": {After: lo.ToPtr(base64.StdEncoding.EncodeToString([]byte("v1.unknown.type")))},
			"after and before": {
				After:  lo.ToPtr(base64.StdEncoding.EncodeToString([]byte("v1.billing.invoice.created"))),
				Before: lo.ToPtr(base64.StdEncoding.EncodeToString([]byte("v1.billing.invoice.updated"))),
			},
		} {
			t.Run(name, func(t *testing.T) {
				code, _ := list(t, ListNotificationEventTypesParams{Page: page})
				require.Equal(t, http.StatusBadRequest, code)
			})
		}
	})
}

func TestGetNotificationEventType(t *testing.T) {
	h := newTestHandler(t)

	get := func(t *testing.T, eventType string) *httptest.ResponseRecorder {
		t.Helper()

		rec := httptest.NewRecorder()
		h.GetNotificationEventType().With(eventType).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		return rec
	}

	t.Run("with schemas", func(t *testing.T) {
		rec := get(t, "v1.entitlement.balance_threshold.triggered")
		require.Equal(t, http.StatusOK, rec.Code)

		var body apiv3models.BillingNotificationEventTypeDefinition
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

		require.Equal(t, "v1.entitlement.balance_threshold.triggered", body.Type)
		require.Contains(t, body.Description, "`entitlements.balance.threshold`")

		// The schemas are returned as they are registered.
		var payloadSchema, ruleConfigSchema map[string]any
		require.NoError(t, json.Unmarshal(definitions.EntitlementBalanceThresholdTriggeredV1PayloadJSONSchema, &payloadSchema))
		require.NoError(t, json.Unmarshal(definitions.EntitlementBalanceThresholdTriggeredV1RuleConfigJSONSchema, &ruleConfigSchema))
		require.Equal(t, payloadSchema, body.PayloadJsonSchema)
		require.NotNil(t, body.RuleConfigJsonSchema)
		require.Equal(t, ruleConfigSchema, *body.RuleConfigJsonSchema)
	})

	t.Run("unknown", func(t *testing.T) {
		rec := get(t, "v9.billing.invoice.created")
		require.Equal(t, http.StatusNotFound, rec.Code)

		// The same problem type as the other v3 errors, not about:blank.
		var problem struct {
			Type string `json:"type"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
		require.NotEqual(t, "about:blank", problem.Type)
	})

	t.Run("not canonical", func(t *testing.T) {
		require.Equal(t, http.StatusBadRequest, get(t, "invoice.created").Code)
	})
}

func TestToAPIBillingNotificationEventTypeSummaryDeprecated(t *testing.T) {
	replacement := v4.EventType{Name: "billing.invoice.created", Version: 2}

	got := ToAPIBillingNotificationEventTypeSummary(v4.EventTypeDefinition{
		EventType:    v4.EventType{Name: "billing.invoice.created", Version: 1},
		Description:  "An invoice was created.",
		DeprecatedBy: &replacement,
	})

	require.Equal(t, apiv3models.BillingNotificationEventTypeSummary{
		Type:         "v1.billing.invoice.created",
		Description:  "An invoice was created.",
		DeprecatedBy: lo.ToPtr("v2.billing.invoice.created"),
	}, got)
}
