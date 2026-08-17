package viewgen

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderRecreateSQL(t *testing.T) {
	// given:
	// - two Ent-managed views in creation order
	// when:
	// - replacement SQL is rendered
	// then:
	// - existing views are dropped in reverse order before both are recreated
	views := []ViewDef{
		{Name: "first_view", Query: "SELECT 1 AS value"},
		{Name: "second_view", Query: "SELECT value FROM first_view"},
	}

	actual := string(RenderRecreateSQL(views))

	require.Equal(t, `-- Recreate Ent-managed views in a disposable desired-state database.
DROP VIEW IF EXISTS "second_view";
DROP VIEW IF EXISTS "first_view";

CREATE VIEW "first_view" AS
SELECT 1 AS value;

CREATE VIEW "second_view" AS
SELECT value FROM first_view;
`, actual)
}
