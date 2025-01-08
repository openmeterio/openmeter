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
			Error: &NotFoundError{
				Namespace: "test",
				ID:        "test",
			},
			ExpectedError: true,
		},
		{
			Name: "Wrapped",
			Error: errors.Join(
				fmt.Errorf("wrapped: %w", &NotFoundError{
					Namespace: "test",
					ID:        "test",
				}),
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
			Error: &NotFoundError{
				Namespace: "namespace",
				ID:        "id",
			},
			ExpectedError: "plan not found. [namespace=namespace id=id]",
		},
		{
			Name: "Key",
			Error: &NotFoundError{
				Namespace: "namespace",
				Key:       "key",
			},
			ExpectedError: "plan not found. [namespace=namespace key=key]",
		},
		{
			Name: "KeyVersion",
			Error: &NotFoundError{
				Namespace: "namespace",
				Key:       "key",
				Version:   1,
			},
			ExpectedError: "plan not found. [namespace=namespace key=key version=1]",
		},
		{
			Name:          "Default",
			Error:         &NotFoundError{},
			ExpectedError: "plan not found",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			t.Logf("%v", test.Error)

			assert.Equal(t, test.ExpectedError, test.Error.Error())
		})
	}
}
