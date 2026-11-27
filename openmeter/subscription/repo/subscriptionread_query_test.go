package repo_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
)

var errQueryCaptured = errors.New("query captured")

type subscriptionQueryRecorder struct {
	dialect.Driver
	query string
	args  any
}

func (*subscriptionQueryRecorder) Dialect() string { return dialect.Postgres }

func (r *subscriptionQueryRecorder) Query(_ context.Context, query string, args, _ any) error {
	r.query, r.args = query, args
	return errQueryCaptured
}

func (r *subscriptionQueryRecorder) requireReadTemplate(t *testing.T, expected string, args ...any) {
	t.Helper()

	// Ent owns the column projection. Pin the tables, predicates and parameter bindings.
	_, from, ok := strings.Cut(r.query, " FROM ")
	require.True(t, ok)
	require.Equal(t, strings.Join(strings.Fields(expected), " "), "FROM "+from)
	require.Equal(t, args, r.args)
}

func TestGetForSubscriptionsAt_SQLTemplate(t *testing.T) {
	recorder := &subscriptionQueryRecorder{}
	client := db.NewClient(db.Driver(recorder))
	at := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	inputs := []subscription.GetForSubscriptionAtInput{
		{Namespace: "ns", SubscriptionID: "sub-1", At: at},
		{Namespace: "ns", SubscriptionID: "sub-2", At: at},
	}

	t.Run("items", func(t *testing.T) {
		_, err := subscriptionrepo.NewSubscriptionItemRepo(client).GetForSubscriptionsAt(t.Context(), inputs)
		require.ErrorIs(t, err, errQueryCaptured)

		// A shared cutoff needs one phase lookup, even when the page contains multiple subscriptions.
		expected := `
FROM "subscription_items"
WHERE "subscription_items"."namespace" = $1
    AND EXISTS (SELECT "subscription_phases"."id" FROM "subscription_phases"
        WHERE "subscription_items"."phase_id" = "subscription_phases"."id"
        AND ("subscription_phases"."subscription_id" IN ($2, $3) AND "subscription_phases"."namespace" = $4
            AND ("subscription_phases"."deleted_at" IS NULL OR "subscription_phases"."deleted_at" > $5)))
    AND ("subscription_items"."deleted_at" IS NULL OR "subscription_items"."deleted_at" > $6)`
		recorder.requireReadTemplate(t, expected, "ns", "sub-1", "sub-2", "ns", at, at)
	})

	t.Run("phases", func(t *testing.T) {
		_, err := subscriptionrepo.NewSubscriptionPhaseRepo(client).GetForSubscriptionsAt(t.Context(), inputs)
		require.ErrorIs(t, err, errQueryCaptured)

		expected := `
FROM "subscription_phases"
WHERE "subscription_phases"."subscription_id" IN ($1, $2) AND "subscription_phases"."namespace" = $3
    AND ("subscription_phases"."deleted_at" IS NULL OR "subscription_phases"."deleted_at" > $4)`
		recorder.requireReadTemplate(t, expected, "sub-1", "sub-2", "ns", at)
	})
}
