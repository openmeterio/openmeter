package errorsx

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithPrefix(t *testing.T) {
	const prefix = "prefix"

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "single error",
			err:      errors.New("error"),
			expected: "prefix: error",
		},
		{
			name: "multiple errors",
			err: errors.Join(
				errors.New("error 1"),
				errors.New("error 2"),
			),
			expected: "prefix: error 1\nprefix: error 2",
		},

		{
			name: "multiple errors (not top-level)",
			err: fmt.Errorf("%w", errors.Join(
				errors.New("error 1"),
				errors.New("error 2"),
			)),
			expected: "prefix: error 1\nerror 2",
		},
		{
			name: "multiple errors",
			err: errors.Join(
				WithPrefix(
					errors.Join(

						errors.New("error 1.1"),
						errors.New("error 1.2"),
					),
					"subprefix",
				),
				errors.New("error 2"),
			),
			expected: "prefix: subprefix: error 1.1\nprefix: subprefix: error 1.2\nprefix: error 2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := WithPrefix(test.err, prefix).Error()

			assert.Equal(t, test.expected, actual)
		})
	}
}
