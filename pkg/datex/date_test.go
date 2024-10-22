package datex_test

import (
	"testing"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/stretchr/testify/assert"
)

func TestISOOperations(t *testing.T) {
	t.Run("Parse", func(t *testing.T) {
		isoDuration := "P1Y2M3DT4H5M6S"

		period, err := datex.ISOString(isoDuration).Parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		now := testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z")

		expected := testutils.GetRFC3339Time(t, "2021-03-04T04:05:06Z")
		actual, precise := period.AddTo(now)
		assert.True(t, precise)
		assert.Equal(t, expected, actual)
	})

	t.Run("ParseError", func(t *testing.T) {
		isoDuration := "P1Y2M3DT4H5M6SX"

		_, err := datex.ISOString(isoDuration).Parse()
		assert.NotNil(t, err)
	})

	t.Run("Works with 0 duration", func(t *testing.T) {
		isoDuration := "PT0S"

		period, err := datex.ISOString(isoDuration).Parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		now := testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z")

		expected := testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z")
		actual, precise := period.AddTo(now)
		assert.True(t, precise)
		assert.Equal(t, expected, actual)
	})

	t.Run("Adding periods", func(t *testing.T) {
		isoDuration1 := "PT5M"
		isoDuration2 := "PT1M1S"

		period1, err := datex.ISOString(isoDuration1).Parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		period2, err := datex.ISOString(isoDuration2).Parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedS := "PT6M1S"
		expected, err := datex.ISOString(expectedS).Parse()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		actual, err := period1.Add(period2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		assert.Equal(t, expected, actual)
	})
}
