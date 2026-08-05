package meter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJSONPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "simple member", path: "$.value"},
		{name: "nested member", path: "$.foo.bar"},
		{name: "underscore member", path: "$.duration_ms"},
		{name: "bare root", path: "$"},
		{name: "bracket single quoted", path: `$['my-key']`},
		{name: "bracket double quoted", path: `$["my key"]`},
		{name: "array index", path: "$.items[0].price"},
		{name: "star member", path: "$.*"},
		{name: "star in path", path: "$.a[*].b"},
		{name: "clickhouse range", path: "$[0 to 2, 4]"},
		{name: "numeric member", path: "$.2"},

		// go-sqlbuilder argument-reference metacharacters.
		{name: "positional arg reference", path: "$2", wantErr: true},
		{name: "named arg reference", path: "${foo}", wantErr: true},
		{name: "successive arg reference", path: "$?", wantErr: true},

		{name: "dollar after root", path: "$.a$1", wantErr: true},
		{name: "semicolon", path: "$.a; DROP TABLE x", wantErr: true},
		{name: "backslash", path: `$.a\'`, wantErr: true},
		{name: "newline control char", path: "$.a\nx", wantErr: true},
		{name: "empty", path: "", wantErr: true},
		{name: "no root", path: "value", wantErr: true},
		{name: "too long", path: "$." + strings.Repeat("a", maxJSONPathLength), wantErr: true},
		{name: "invalid utf8", path: "$.\xff\xfe", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSONPath(tt.path)

			if tt.wantErr {
				require.Error(t, err, "expected %q to be rejected", tt.path)
			} else {
				require.NoError(t, err, "expected %q to be accepted", tt.path)
			}
		})
	}
}

func TestValidateGroupByKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{name: "simple identifier", key: "region"},
		{name: "leading underscore", key: "_region"},
		{name: "alphanumeric", key: "region_2"},
		{name: "empty", key: "", wantErr: true},
		{name: "whitespace only", key: "   ", wantErr: true},
		{name: "hyphen", key: "my-key", wantErr: true},
		{name: "leading digit", key: "2region", wantErr: true},
		{name: "sql injection shaped", key: "g) FROM system.numbers --", wantErr: true},
		{name: "quote", key: "g'", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGroupByKey(tt.key)

			if tt.wantErr {
				assert.Error(t, err, "expected key %q to be rejected", tt.key)
			} else {
				assert.NoError(t, err, "expected key %q to be accepted", tt.key)
			}
		})
	}
}
