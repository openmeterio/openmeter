package pagination

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPageValidate(t *testing.T) {
	for _, tc := range []struct {
		name  string
		page  Page
		valid bool
	}{
		{name: "first page", page: NewPage(1, 100), valid: true},
		{name: "zero size", page: NewPage(1, 0), valid: true},
		{name: "last page that fits", page: NewPage(math.MaxInt/1000, 1000), valid: true},
		{name: "negative size", page: NewPage(1, -1)},
		{name: "page zero", page: NewPage(0, 10)},
		{name: "end overflows", page: NewPage(math.MaxInt/1000+1, 1000)},
		{name: "huge page and size", page: NewPage(math.MaxInt, math.MaxInt)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.page.Validate()
			if !tc.valid {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.GreaterOrEqual(t, tc.page.Offset()+tc.page.Limit(), 0)
		})
	}
}
