package eventtypes

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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

	list := func(t *testing.T, params ListNotificationEventTypesParams) (int, response.PagePaginationResponse[apiv3models.BillingNotificationEventTypeSummary]) {
		t.Helper()

		rec := httptest.NewRecorder()
		h.ListNotificationEventTypes().With(params).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		var body response.PagePaginationResponse[apiv3models.BillingNotificationEventTypeSummary]
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
		require.Equal(t, 4, lo.FromPtr(body.Meta.Page.Total))

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

	t.Run("pages", func(t *testing.T) {
		code, body := list(t, ListNotificationEventTypesParams{
			Page: &apiv3models.PagePaginationQuery{Number: lo.ToPtr(2), Size: lo.ToPtr(3)},
		})
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, []string{"v1.entitlement.usage_period.reset"}, types(body.Data))
		require.Equal(t, 4, lo.FromPtr(body.Meta.Page.Total))

		// A page past the end is empty, not an error.
		code, body = list(t, ListNotificationEventTypesParams{
			Page: &apiv3models.PagePaginationQuery{Number: lo.ToPtr(3), Size: lo.ToPtr(3)},
		})
		require.Equal(t, http.StatusOK, code)
		require.Empty(t, body.Data)
	})

	t.Run("invalid page", func(t *testing.T) {
		code, _ := list(t, ListNotificationEventTypesParams{
			Page: &apiv3models.PagePaginationQuery{Number: lo.ToPtr(0)},
		})
		require.Equal(t, http.StatusBadRequest, code)
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
