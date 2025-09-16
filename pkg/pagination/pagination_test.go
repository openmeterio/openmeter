package pagination_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestShouldFlattenPageInfoWhenMarshalling(t *testing.T) {
	assert := assert.New(t)
	pagedRes := pagination.Result[int]{
		Items:      []int{1, 2, 3},
		TotalCount: 3,
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
	}

	expected := `{"pageSize":10,"page":1,"totalCount":3,"items":[1,2,3]}
`

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)

	if err := enc.Encode(pagedRes); err != nil {
		t.Fatalf("failed to marshal paged response: %v", err)
	}

	assert.JSONEq(expected, buf.String())
	// enforces ordering
	assert.Equal(expected, buf.String())
}
