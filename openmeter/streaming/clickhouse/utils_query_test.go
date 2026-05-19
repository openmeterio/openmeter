package clickhouse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ValidateJsonPathQuery(t *testing.T) {
	query, args := validateJsonPathQuery{
		jsonPath: "$.foo.bar",
	}.toSQL()

	assert.Equal(t, `SELECT JSON_VALUE('{}', ?)`, query)
	assert.Equal(t, []interface{}{
		"$.foo.bar",
	}, args)
}
