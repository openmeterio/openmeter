package entpaginate_test

import (
	"context"
	"fmt"
	"testing"

	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db"
	db_example "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db/example1"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestPaginate(t *testing.T) {
	assert := assert.New(t)

	// create isolated pg db for tests
	driver := testutils.InitPostgresDB(t, testutils.PostgresDBStateEmpty)
	defer driver.PGDriver.Close()

	// build db clients
	var queries []string
	dbClient := db.NewClient(
		db.Driver(driver.EntDriver.Driver()),
		db.Debug(),
		db.Log(func(args ...any) {
			queries = append(queries, fmt.Sprint(args...))
		}),
	)
	defer dbClient.Close()

	if err := dbClient.Schema.Create(t.Context()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}

	// insert items
	ctx := t.Context()
	for i := 0; i < 10; i++ {
		_, err := dbClient.Example1.Create().
			SetID(fmt.Sprintf("%v", i)).
			SetExampleValue1(fmt.Sprintf("%v", i)).
			Save(ctx)
		if err != nil {
			t.Fatalf("failed to insert item %d: %s", i, err)
		}
	}

	// total
	total, err := dbClient.Example1.Query().Count(ctx)
	if err != nil {
		t.Fatalf("failed to count: %s", err)
	}

	assert.Equal(10, total)

	t.Run("Should return first item", func(t *testing.T) {
		paged, err := dbClient.Example1.Query().Order(db_example.ByID(sql.OrderAsc())).Paginate(ctx, pagination.Page{
			PageSize:   1,
			PageNumber: 1,
		})
		if err != nil {
			t.Fatalf("failed to paginate: %s", err)
		}

		assert.Equal(1, len(paged.Items))
		assert.Equal(10, paged.TotalCount)
		assert.Equal("0", paged.Items[0].ID)
	})

	t.Run("Should respect ordering", func(t *testing.T) {
		paged, err := dbClient.Example1.Query().Order(db_example.ByID(sql.OrderDesc())).Paginate(ctx, pagination.Page{
			PageSize:   1,
			PageNumber: 1,
		})
		if err != nil {
			t.Fatalf("failed to paginate: %s", err)
		}

		assert.Equal(1, len(paged.Items))
		assert.Equal(10, paged.TotalCount)
		assert.Equal("9", paged.Items[0].ID)
	})

	t.Run("Should respect filtering", func(t *testing.T) {
		paged, err := dbClient.Example1.Query().Where(db_example.IDContainsFold("1")).Paginate(ctx, pagination.Page{
			PageSize:   1,
			PageNumber: 1,
		})
		if err != nil {
			t.Fatalf("failed to paginate: %s", err)
		}

		assert.Equal(1, len(paged.Items))
		assert.Equal(1, paged.TotalCount)
		assert.Equal("1", paged.Items[0].ID)
	})

	t.Run("Should page", func(t *testing.T) {
		paged, err := dbClient.Example1.Query().Order(db_example.ByID(sql.OrderAsc())).Paginate(ctx, pagination.Page{
			PageSize:   3,
			PageNumber: 2,
		})
		if err != nil {
			t.Fatalf("failed to paginate: %s", err)
		}

		assert.Equal(3, len(paged.Items))
		assert.Equal(10, paged.TotalCount)
		assert.Equal(3, paged.Page.PageSize)
		assert.Equal(2, paged.Page.PageNumber)
		assert.Equal("3", paged.Items[0].ID)
		assert.Equal("4", paged.Items[1].ID)
		assert.Equal("5", paged.Items[2].ID)
	})

	t.Run("Should preserve the total on a partial last page", func(t *testing.T) {
		queries = nil
		page := pagination.NewPage(4, 3)
		paged, err := dbClient.Example1.Query().
			Order(db_example.ByID(sql.OrderAsc())).
			Paginate(t.Context(), page)
		require.NoError(t, err)
		require.Len(t, paged.Items, 1)
		require.Equal(t, "9", paged.Items[0].ID)
		require.Equal(t, 10, paged.TotalCount)
		require.Equal(t, page, paged.Page)
		require.Len(t, queries, 2)
	})

	t.Run("Should skip fetching an explicitly paginated empty collection", func(t *testing.T) {
		queries = nil
		page := pagination.NewPage(1, 3)
		paged, err := dbClient.Example1.Query().
			Where(db_example.IDEQ("missing")).
			Paginate(t.Context(), page)
		require.NoError(t, err)
		require.NotNil(t, paged.Items)
		require.Empty(t, paged.Items)
		require.Zero(t, paged.TotalCount)
		require.Equal(t, page, paged.Page)
		require.Len(t, queries, 1)
		require.Contains(t, queries[0], "COUNT(")
	})

	t.Run("Should return empty page", func(t *testing.T) {
		queries = nil
		paged, err := dbClient.Example1.Query().Order(db_example.ByID(sql.OrderAsc())).Paginate(ctx, pagination.Page{
			PageSize:   3,
			PageNumber: 10,
		})
		if err != nil {
			t.Fatalf("failed to paginate: %s", err)
		}

		assert.Equal(0, len(paged.Items))
		assert.Equal(10, paged.TotalCount)
		require.Len(t, queries, 2)
	})

	t.Run("Should return all items", func(t *testing.T) {
		queries = nil
		paged, err := dbClient.Example1.Query().Order(db_example.ByID(sql.OrderAsc())).Paginate(ctx, pagination.Page{})
		if err != nil {
			t.Fatalf("failed to paginate: %s", err)
		}

		assert.Equal(10, len(paged.Items))
		assert.Equal(10, paged.TotalCount)
		require.Equal(t, pagination.Page{}, paged.Page)
		require.Len(t, queries, 1)
	})

	t.Run("Should clear pagination while preserving filtering and ordering", func(t *testing.T) {
		queries = nil
		paged, err := dbClient.Example1.Query().
			Where(db_example.IDIn("2", "4", "8")).
			Order(db_example.ByID(sql.OrderDesc())).
			Limit(1).
			Offset(1).
			Paginate(t.Context(), pagination.Page{})
		require.NoError(t, err)
		require.Len(t, paged.Items, 3)
		require.Equal(t, 3, paged.TotalCount)
		require.Equal(t, "8", paged.Items[0].ID)
		require.Equal(t, "4", paged.Items[1].ID)
		require.Equal(t, "2", paged.Items[2].ID)
		require.Equal(t, pagination.Page{}, paged.Page)
		require.Len(t, queries, 1)
	})

	t.Run("Should return an empty list without a page", func(t *testing.T) {
		queries = nil
		paged, err := dbClient.Example1.Query().
			Where(db_example.IDEQ("missing")).
			Paginate(t.Context(), pagination.Page{})
		require.NoError(t, err)
		require.NotNil(t, paged.Items)
		require.Empty(t, paged.Items)
		require.Zero(t, paged.TotalCount)
		require.Equal(t, pagination.Page{}, paged.Page)
		require.Len(t, queries, 1)
	})

	t.Run("Should propagate fetch errors without a page", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := dbClient.Example1.Query().Paginate(cancelledCtx, pagination.Page{})
		require.ErrorIs(t, err, context.Canceled)
	})
}
