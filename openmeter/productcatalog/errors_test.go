package productcatalog

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnwrapErrors(t *testing.T) {
	planResource := Resource{
		Key:  "premium",
		Kind: "plan",
		Attributes: map[string]any{
			"version": 3,
		},
	}

	addonResource := Resource{
		Key:  "storage",
		Kind: "addon",
		Attributes: map[string]any{
			"version": 1,
		},
	}

	tests := []struct {
		name           string
		err            error
		expectedErrors []InvalidResourceError
	}{
		{
			name:           "nil error returns empty slice",
			err:            nil,
			expectedErrors: []InvalidResourceError{},
		},
		{
			name: "single InvalidResourceError",
			err: InvalidResourceError{
				Resource: planResource,
				Field:    "currency",
				Detail:   "invalid currency value",
			},
			expectedErrors: []InvalidResourceError{
				{
					Resource: planResource,
					Field:    "currency",
					Detail:   "invalid currency value",
				},
			},
		},
		{
			name: "wrapped single InvalidResourceError with fmt.Errorf",
			err: fmt.Errorf("validation failed: %w",
				InvalidResourceError{
					Resource: planResource,
					Field:    "status",
					Detail:   "invalid status",
				},
			),
			expectedErrors: []InvalidResourceError{
				{
					Resource: planResource,
					Field:    "status",
					Detail:   "invalid status",
				},
			},
		},
		{
			name: "wrapped with errors.Join containing InvalidResourceError",
			err: errors.Join(
				errors.New("some other error"),
				InvalidResourceError{
					Resource: planResource,
					Field:    "name",
					Detail:   "name is required",
				},
			),
			expectedErrors: []InvalidResourceError{
				{
					Resource: planResource,
					Field:    "name",
					Detail:   "name is required",
				},
			},
		},
		{
			name: "wrapped with errors.Join containing multiple InvalidResourceErrors",
			err: errors.Join(
				InvalidResourceError{
					Resource: planResource,
					Field:    "currency",
					Detail:   "invalid currency",
				},
				errors.New("some validation error"),
				InvalidResourceError{
					Resource: addonResource,
					Field:    "maxQuantity",
					Detail:   "must be positive",
				},
			),
			expectedErrors: []InvalidResourceError{
				{
					Resource: planResource,
					Field:    "currency",
					Detail:   "invalid currency",
				},
				{
					Resource: addonResource,
					Field:    "maxQuantity",
					Detail:   "must be positive",
				},
			},
		},
		{
			name: "deeply nested errors",
			err: fmt.Errorf("outer error: %w",
				errors.Join(
					errors.New("regular error"),
					fmt.Errorf("middle error: %w",
						errors.Join(
							InvalidResourceError{
								Resource: planResource,
								Field:    "key",
								Detail:   "key must be provided",
							},
							errors.New("another error"),
							InvalidResourceError{
								Resource: addonResource,
								Field:    "fromPlanPhase",
								Detail:   "plan phase not found",
							},
						),
					),
				),
			),
			expectedErrors: []InvalidResourceError{
				{
					Resource: planResource,
					Field:    "key",
					Detail:   "key must be provided",
				},
				{
					Resource: addonResource,
					Field:    "fromPlanPhase",
					Detail:   "plan phase not found",
				},
			},
		},
		{
			name:           "error without InvalidResourceError",
			err:            errors.New("just a regular error"),
			expectedErrors: []InvalidResourceError{},
		},
		{
			name: "multiple levels of wrapped InvalidResourceErrors",
			err: errors.Join(
				fmt.Errorf("level 1: %w",
					InvalidResourceError{
						Resource: planResource,
						Field:    "version",
						Detail:   "invalid version",
					},
				),
				fmt.Errorf("level 2: %w",
					errors.Join(
						InvalidResourceError{
							Resource: addonResource,
							Field:    "currency",
							Detail:   "currency mismatch",
						},
						errors.New("standard error"),
					),
				),
			),
			expectedErrors: []InvalidResourceError{
				{
					Resource: planResource,
					Field:    "version",
					Detail:   "invalid version",
				},
				{
					Resource: addonResource,
					Field:    "currency",
					Detail:   "currency mismatch",
				},
			},
		},
		{
			name: "nested parent resources in InvalidResourceError",
			err: InvalidResourceError{
				Resource: Resource{
					Parent:     &planResource,
					Key:        "addon-phase",
					Kind:       "phase",
					Attributes: nil,
				},
				Field:  "duration",
				Detail: "invalid duration",
			},
			expectedErrors: []InvalidResourceError{
				{
					Resource: Resource{
						Parent:     &planResource,
						Key:        "addon-phase",
						Kind:       "phase",
						Attributes: nil,
					},
					Field:  "duration",
					Detail: "invalid duration",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnwrapErrors(tt.err)

			// Check that we have the right number of errors
			assert.Equal(t, len(tt.expectedErrors), len(result),
				"Expected %d InvalidResourceErrors, got %d",
				len(tt.expectedErrors), len(result))

			// Check each error matches what we expect
			if len(tt.expectedErrors) > 0 {
				for i, expectedErr := range tt.expectedErrors {
					// Check basic fields
					assert.Equal(t, expectedErr.Field, result[i].Field)
					assert.Equal(t, expectedErr.Detail, result[i].Detail)

					// Check resource fields
					assert.Equal(t, expectedErr.Resource.Key, result[i].Resource.Key)
					assert.Equal(t, expectedErr.Resource.Kind, result[i].Resource.Kind)

					// If there's a parent, check it too
					if expectedErr.Resource.Parent != nil {
						assert.NotNil(t, result[i].Resource.Parent)
						assert.Equal(t, expectedErr.Resource.Parent.Key, result[i].Resource.Parent.Key)
						assert.Equal(t, expectedErr.Resource.Parent.Kind, result[i].Resource.Parent.Kind)
					}

					// Check attributes if they exist
					if expectedErr.Resource.Attributes != nil {
						assert.Equal(t, expectedErr.Resource.Attributes, result[i].Resource.Attributes)
					}
				}
			}
		})
	}
}
