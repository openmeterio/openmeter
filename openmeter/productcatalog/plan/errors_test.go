package plan

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		Name          string
		Error         error
		ExpectedError bool
	}{
		{
			Name: "Valid",
			Error: NewNotFoundError(NotFoundErrorParams{
				Namespace: "test",
				ID:        "test",
			}),
			ExpectedError: true,
		},
		{
			Name: "Wrapped",
			Error: errors.Join(
				fmt.Errorf("wrapped: %w", NewNotFoundError(NotFoundErrorParams{
					Namespace: "test",
					ID:        "test",
				})),
			),
			ExpectedError: true,
		},
		{
			Name:          "Invalid",
			Error:         errors.New("test error"),
			ExpectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.ExpectedError, IsNotFound(test.Error))
		})
	}
}

func TestIsNotFoundError_String(t *testing.T) {
	tests := []struct {
		Name          string
		Error         error
		ExpectedError string
	}{
		{
			Name: "ID",
			Error: NewNotFoundError(NotFoundErrorParams{
				Namespace: "namespace",
				ID:        "id",
			}),
			ExpectedError: "not found error: plan not found. [namespace=namespace id=id]",
		},
		{
			Name: "Key",
			Error: NewNotFoundError(NotFoundErrorParams{
				Namespace: "namespace",
				Key:       "key",
			}),
			ExpectedError: "not found error: plan not found. [namespace=namespace key=key]",
		},
		{
			Name: "KeyVersion",
			Error: NewNotFoundError(NotFoundErrorParams{
				Namespace: "namespace",
				Key:       "key",
				Version:   1,
			}),
			ExpectedError: "not found error: plan not found. [namespace=namespace key=key version=1]",
		},
		{
			Name:          "Default",
			Error:         NewNotFoundError(NotFoundErrorParams{}),
			ExpectedError: "not found error: plan not found",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			t.Logf("%v", test.Error)

			assert.Equal(t, test.ExpectedError, test.Error.Error())
		})
	}
}
