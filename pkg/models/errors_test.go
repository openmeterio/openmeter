package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorWrappers(t *testing.T) {
	tests := []struct {
		name              string
		err               error
		expectedErrString string
	}{
		{
			name: "prefix",
			err: ErrorWithFieldPrefix(
				NewFieldSelectorGroup(
					NewFieldSelector("plan").
						WithExpression(NewMultiFieldAttrValue(
							NewFieldAttrValue("key", "pro"),
							NewFieldAttrValue("version", "1"),
						)),
				),
				ErrorWithFieldPrefix(
					NewFieldSelectorGroup(
						NewFieldSelector("phases").
							WithExpression(NewFieldAttrValue("key", "trial")),
					),
					ErrorWithFieldPrefix(
						NewFieldSelectorGroup(
							NewFieldSelector("rateCards").
								WithExpression(NewFieldAttrValue("key", "storage")),
						),
						errors.New("critical error"),
					),
				),
			),
			expectedErrString: `plan[key=pro, version=1]: phases[key=trial]: rateCards[key=storage]: critical error`,
		},
		{
			name: "component",
			err: ErrorWithComponent("openmeter",
				ErrorWithFieldPrefix(
					NewFieldSelectorGroup(NewFieldSelector("plan").
						WithExpression(NewMultiFieldAttrValue(
							NewFieldAttrValue("key", "pro"),
							NewFieldAttrValue("version", "1"),
						)),
					),
					errors.New("critical error"),
				)),
			expectedErrString: `openmeter: plan[key=pro, version=1]: critical error`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.err.Error()

			t.Logf("actual error: %s", actual)

			assert.Equalf(t, tt.expectedErrString, actual, "error string mismatch for test case %q", tt.name)
		})
	}
}
