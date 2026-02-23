package entcursor_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db"
	db_example "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db/example1"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func TestCursor(t *testing.T) {
	// create isolated pg db for tests
	driver := testutils.InitPostgresDB(t)
	defer driver.PGDriver.Close()

	// build db clients
	dbClient := db.NewClient(db.Driver(driver.EntDriver.Driver()))
	defer dbClient.Close()

	if err := dbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}

	ctx := context.Background()
	baseTime := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)

	_, err := dbClient.Example1.Create().SetID("001").SetExampleValue1("v1").SetCreatedAt(baseTime).Save(ctx)
	require.NoError(t, err)
	_, err = dbClient.Example1.Create().SetID("002").SetExampleValue1("v2").SetCreatedAt(baseTime).Save(ctx)
	require.NoError(t, err)
	_, err = dbClient.Example1.Create().SetID("003").SetExampleValue1("v3").SetCreatedAt(baseTime.Add(time.Second)).Save(ctx)
	require.NoError(t, err)

	t.Run("Returns first page and next cursor", func(t *testing.T) {
		res, err := dbClient.Example1.Query().Limit(2).Cursor(ctx, nil)
		require.NoError(t, err)

		require.Len(t, res.Items, 2)
		assert.Equal(t, "001", res.Items[0].ID)
		assert.Equal(t, "002", res.Items[1].ID)
		require.NotNil(t, res.NextCursor)
		assert.Equal(t, baseTime, res.NextCursor.Time)
		assert.Equal(t, "002", res.NextCursor.ID)
	})

	t.Run("Applies cursor and returns next page", func(t *testing.T) {
		firstPage, err := dbClient.Example1.Query().Limit(2).Cursor(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, firstPage.NextCursor)

		secondPage, err := dbClient.Example1.Query().Limit(2).Cursor(ctx, firstPage.NextCursor)
		require.NoError(t, err)

		require.Len(t, secondPage.Items, 1)
		assert.Equal(t, "003", secondPage.Items[0].ID)
		require.NotNil(t, secondPage.NextCursor)
		assert.Equal(t, "003", secondPage.NextCursor.ID)
	})

	t.Run("Returns validation error for invalid cursor", func(t *testing.T) {
		_, err := dbClient.Example1.Query().Limit(2).Cursor(ctx, &pagination.Cursor{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cursor")
	})

	t.Run("Returns empty items for empty result", func(t *testing.T) {
		res, err := dbClient.Example1.Query().Where(db_example.ID("not-found")).Cursor(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, res.Items)
		assert.Len(t, res.Items, 0)
		assert.Nil(t, res.NextCursor)
	})
}
