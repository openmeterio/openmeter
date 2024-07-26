package strcase_test

import (
	"testing"

	"github.com/openmeterio/openmeter/pkg/strcase"
)

func TestSnakeToCamel(t *testing.T) {
	tt := []struct {
		name  string
		snake string
		camel string
	}{
		{
			name:  "empty",
			snake: "",
			camel: "",
		},
		{
			name:  "single",
			snake: "a",
			camel: "a",
		},
		{
			name:  "two",
			snake: "a_b",
			camel: "aB",
		},
		{
			name:  "three",
			snake: "a_b_c",
			camel: "aBC",
		},
		{
			name:  "long",
			snake: "abc_def",
			camel: "abcDef",
		},
		{
			name:  "withUppers",
			snake: "aBc_dEf_gHi",
			camel: "aBcDEfGHi",
		},
		{
			name:  "withSpecial",
			snake: "a_b-c_d/e",
			camel: "aB-cD/e",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			camel := strcase.SnakeToCamel(tc.snake)
			if camel != tc.camel {
				t.Errorf("expected %q, got %q", tc.camel, camel)
			}
		})
	}
}

func TestCamelToSnake(t *testing.T) {
	tt := []struct {
		name  string
		camel string
		snake string
	}{
		{
			name:  "empty",
			camel: "",
			snake: "",
		},
		{
			name:  "single",
			camel: "a",
			snake: "a",
		},
		{
			name:  "two",
			camel: "aB",
			snake: "a_b",
		},
		{
			name:  "three",
			camel: "aBC",
			snake: "a_b_c",
		},
		{
			name:  "long",
			camel: "abcDef",
			snake: "abc_def",
		},
		{
			name:  "withSpecial",
			camel: "aB-cD/e",
			snake: "a_b-c_d/e",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			snake := strcase.CamelToSnake(tc.camel)
			if snake != tc.snake {
				t.Errorf("expected %q, got %q", tc.snake, snake)
			}
		})
	}
}

func TestCamelToSnakeToCamel(t *testing.T) {
	tt := []string{
		"",
		"a",
		"aB",
		"aBC",
		"abcDef",
		"aBcDEfGHi",
		"aB-cD/e",
	}

	for _, camel := range tt {
		t.Run(camel, func(t *testing.T) {
			snake := strcase.CamelToSnake(camel)
			camel2 := strcase.SnakeToCamel(snake)
			if camel != camel2 {
				t.Errorf("expected %q, got %q", camel, camel2)
			}
		})
	}
}
