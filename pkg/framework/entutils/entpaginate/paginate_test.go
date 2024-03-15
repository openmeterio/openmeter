// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package entpaginate_test

import (
	"context"
	"fmt"
	"testing"

	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db"
	db_example "github.com/openmeterio/openmeter/pkg/framework/entutils/testutils/ent1/db/example1"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestPaginate(t *testing.T) {
	assert := assert.New(t)

	// create isolated pg db for tests
	driver := testutils.InitPostgresDB(t)
	defer driver.PGDriver.Close()

	// build db clients
	dbClient := db.NewClient(db.Driver(driver.EntDriver.Driver()))
	defer dbClient.Close()

	if err := dbClient.Schema.Create(context.Background()); err != nil {
		t.Fatalf("failed to migrate database %s", err)
	}

	// insert items
	ctx := context.Background()
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

	t.Run("Should return empty page", func(t *testing.T) {
		paged, err := dbClient.Example1.Query().Order(db_example.ByID(sql.OrderAsc())).Paginate(ctx, pagination.Page{
			PageSize:   3,
			PageNumber: 10,
		})
		if err != nil {
			t.Fatalf("failed to paginate: %s", err)
		}

		assert.Equal(0, len(paged.Items))
		assert.Equal(10, paged.TotalCount)
	})

	t.Run("Should return all items", func(t *testing.T) {
		paged, err := dbClient.Example1.Query().Order(db_example.ByID(sql.OrderAsc())).Paginate(ctx, pagination.Page{})
		if err != nil {
			t.Fatalf("failed to paginate: %s", err)
		}

		assert.Equal(10, len(paged.Items))
		assert.Equal(10, paged.TotalCount)
	})
}
