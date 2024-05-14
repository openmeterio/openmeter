package jsonx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONMerge(t *testing.T) {
	res, err := Merge(
		[]byte(`{"a":1, "b": "c"}`),
		[]byte(`{"b": 2, "d": 3, "o": {"a": 1}}`),
	)

	assert.NoError(t, err)
	assert.Equal(t, `{"a":1,"b":2,"d":3,"o":{"a":1}}`, string(res))

	_, err = Merge(
		[]byte(`{"a":1, "b": "c"}`),
		[]byte(`null`),
	)

	assert.Error(t, err)
}
