package e2e

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestV3CustomerCreditBalanceAsOfFilterParsing(t *testing.T) {
	c := newV3Client(t)
	customerID := ulid.Make().String()

	t.Run("valid as_of reaches endpoint handling", func(t *testing.T) {
		query := url.Values{
			"filter[as_of]": {time.Date(2026, 5, 11, 10, 30, 0, 0, time.UTC).Format(time.RFC3339)},
		}

		status, _, problem := c.do(http.MethodGet, "/customers/"+customerID+"/credits/balance?"+query.Encode(), nil)
		require.NotEqual(t, http.StatusBadRequest, status, "problem: %+v", problem)
	})

	t.Run("invalid as_of is rejected by query parsing", func(t *testing.T) {
		query := url.Values{
			"filter[as_of]": {"not-a-date"},
		}

		status, _, problem := c.do(http.MethodGet, "/customers/"+customerID+"/credits/balance?"+query.Encode(), nil)
		require.Equal(t, http.StatusBadRequest, status)
		require.NotNil(t, problem)
		assert.Contains(t, problem.Detail, "filter")
	})
}
