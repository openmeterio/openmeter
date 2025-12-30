package streaming

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCustomerUsageAttributionValidate(t *testing.T) {
	key := "test-key"

	tests := []struct {
		name      string
		attrib    CustomerUsageAttribution
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid with key only",
			attrib: CustomerUsageAttribution{
				ID:          "customer-1",
				Key:         &key,
				SubjectKeys: nil,
			},
			wantError: false,
		},
		{
			name: "valid with subject keys only",
			attrib: CustomerUsageAttribution{
				ID:          "customer-2",
				Key:         nil,
				SubjectKeys: []string{"key1", "key2"},
			},
			wantError: false,
		},
		{
			name: "valid with both key and subject keys",
			attrib: CustomerUsageAttribution{
				ID:          "customer-3",
				Key:         &key,
				SubjectKeys: []string{"key1", "key2"},
			},
			wantError: false,
		},
		{
			name: "valid with empty subject keys slice",
			attrib: CustomerUsageAttribution{
				ID:          "customer-4",
				Key:         &key,
				SubjectKeys: []string{},
			},
			wantError: false,
		},
		{
			name: "invalid with empty ID",
			attrib: CustomerUsageAttribution{
				ID:          "",
				Key:         &key,
				SubjectKeys: []string{"key1"},
			},
			wantError: true,
			errorMsg:  "usage attribution must have an id",
		},
		{
			name: "invalid with no key and no subject keys",
			attrib: CustomerUsageAttribution{
				ID:          "customer-5",
				Key:         nil,
				SubjectKeys: nil,
			},
			wantError: true,
			errorMsg:  "usage attribution must have a key or subject keys",
		},
		{
			name: "invalid with no key and empty subject keys",
			attrib: CustomerUsageAttribution{
				ID:          "customer-6",
				Key:         nil,
				SubjectKeys: []string{},
			},
			wantError: true,
			errorMsg:  "usage attribution must have a key or subject keys",
		},
		{
			name: "invalid with empty subject key",
			attrib: CustomerUsageAttribution{
				ID:          "customer-7",
				Key:         nil,
				SubjectKeys: []string{"key1", "", "key2"},
			},
			wantError: true,
			errorMsg:  "subject key cannot be empty",
		},
		{
			name: "invalid with only empty subject key",
			attrib: CustomerUsageAttribution{
				ID:          "customer-8",
				Key:         nil,
				SubjectKeys: []string{""},
			},
			wantError: true,
			errorMsg:  "subject key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.attrib.Validate()
			if tt.wantError {
				assert.Error(t, err)
				assert.True(t, models.IsGenericValidationError(err))
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCustomerUsageAttributionGetValues(t *testing.T) {
	key := "test-key"
	key2 := "test-key-2"

	tests := []struct {
		name     string
		attrib   CustomerUsageAttribution
		expected []string
	}{
		{
			name: "key only",
			attrib: CustomerUsageAttribution{
				ID:          "customer-1",
				Key:         &key,
				SubjectKeys: nil,
			},
			expected: []string{"test-key"},
		},
		{
			name: "subject keys only",
			attrib: CustomerUsageAttribution{
				ID:          "customer-2",
				Key:         nil,
				SubjectKeys: []string{"key1", "key2", "key3"},
			},
			expected: []string{"key1", "key2", "key3"},
		},
		{
			name: "both key and subject keys",
			attrib: CustomerUsageAttribution{
				ID:          "customer-3",
				Key:         &key,
				SubjectKeys: []string{"key1", "key2"},
			},
			expected: []string{"test-key", "key1", "key2"},
		},
		{
			name: "key with empty subject keys",
			attrib: CustomerUsageAttribution{
				ID:          "customer-4",
				Key:         &key,
				SubjectKeys: []string{},
			},
			expected: []string{"test-key"},
		},
		{
			name: "nil key with empty subject keys",
			attrib: CustomerUsageAttribution{
				ID:          "customer-5",
				Key:         nil,
				SubjectKeys: []string{},
			},
			expected: []string{},
		},
		{
			name: "nil key with nil subject keys",
			attrib: CustomerUsageAttribution{
				ID:          "customer-6",
				Key:         nil,
				SubjectKeys: nil,
			},
			expected: []string{},
		},
		{
			name: "multiple subject keys",
			attrib: CustomerUsageAttribution{
				ID:          "customer-7",
				Key:         &key2,
				SubjectKeys: []string{"sub1", "sub2", "sub3", "sub4"},
			},
			expected: []string{"test-key-2", "sub1", "sub2", "sub3", "sub4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.attrib.GetValues()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCustomerUsageAttributionEqual(t *testing.T) {
	key1 := "key-1"
	key2 := "key-2"

	tests := []struct {
		name     string
		attrib1  CustomerUsageAttribution
		attrib2  CustomerUsageAttribution
		expected bool
	}{
		{
			name: "equal with same key and same subject keys",
			attrib1: CustomerUsageAttribution{
				ID:          "customer-1",
				Key:         &key1,
				SubjectKeys: []string{"sub1", "sub2"},
			},
			attrib2: CustomerUsageAttribution{
				ID:          "customer-1",
				Key:         &key1,
				SubjectKeys: []string{"sub1", "sub2"},
			},
			expected: true,
		},
		{
			name: "equal with same key and nil subject keys",
			attrib1: CustomerUsageAttribution{
				ID:          "customer-2",
				Key:         &key1,
				SubjectKeys: nil,
			},
			attrib2: CustomerUsageAttribution{
				ID:          "customer-2",
				Key:         &key1,
				SubjectKeys: nil,
			},
			expected: true,
		},
		{
			name: "equal with same key and empty subject keys",
			attrib1: CustomerUsageAttribution{
				ID:          "customer-3",
				Key:         &key1,
				SubjectKeys: []string{},
			},
			attrib2: CustomerUsageAttribution{
				ID:          "customer-3",
				Key:         &key1,
				SubjectKeys: []string{},
			},
			expected: true,
		},
		{
			name: "equal with nil key and same subject keys",
			attrib1: CustomerUsageAttribution{
				ID:          "customer-4",
				Key:         nil,
				SubjectKeys: []string{"sub1", "sub2"},
			},
			attrib2: CustomerUsageAttribution{
				ID:          "customer-4",
				Key:         nil,
				SubjectKeys: []string{"sub1", "sub2"},
			},
			expected: true,
		},
		{
			name: "equal with nil subject keys vs empty subject keys",
			attrib1: CustomerUsageAttribution{
				ID:          "customer-11",
				Key:         &key1,
				SubjectKeys: nil,
			},
			attrib2: CustomerUsageAttribution{
				ID:          "customer-11",
				Key:         &key1,
				SubjectKeys: []string{},
			},
			expected: true,
		},
		{
			name: "not equal with different ID",
			attrib1: CustomerUsageAttribution{
				ID:          "customer-5",
				Key:         &key1,
				SubjectKeys: []string{"sub1"},
			},
			attrib2: CustomerUsageAttribution{
				ID:          "customer-6",
				Key:         &key1,
				SubjectKeys: []string{"sub1"},
			},
			expected: false,
		},
		{
			name: "not equal with different key",
			attrib1: CustomerUsageAttribution{
				ID:          "customer-7",
				Key:         &key1,
				SubjectKeys: []string{"sub1"},
			},
			attrib2: CustomerUsageAttribution{
				ID:          "customer-7",
				Key:         &key2,
				SubjectKeys: []string{"sub1"},
			},
			expected: false,
		},
		{
			name: "not equal with one nil key and one non-nil key",
			attrib1: CustomerUsageAttribution{
				ID:          "customer-8",
				Key:         nil,
				SubjectKeys: []string{"sub1"},
			},
			attrib2: CustomerUsageAttribution{
				ID:          "customer-8",
				Key:         &key1,
				SubjectKeys: []string{"sub1"},
			},
			expected: false,
		},
		{
			name: "not equal with different subject keys",
			attrib1: CustomerUsageAttribution{
				ID:          "customer-9",
				Key:         &key1,
				SubjectKeys: []string{"sub1", "sub2"},
			},
			attrib2: CustomerUsageAttribution{
				ID:          "customer-9",
				Key:         &key1,
				SubjectKeys: []string{"sub1", "sub3"},
			},
			expected: false,
		},
		{
			name: "not equal with different subject keys length",
			attrib1: CustomerUsageAttribution{
				ID:          "customer-10",
				Key:         &key1,
				SubjectKeys: []string{"sub1", "sub2"},
			},
			attrib2: CustomerUsageAttribution{
				ID:          "customer-10",
				Key:         &key1,
				SubjectKeys: []string{"sub1"},
			},
			expected: false,
		},
		{
			name: "not equal with different subject keys order",
			attrib1: CustomerUsageAttribution{
				ID:          "customer-12",
				Key:         &key1,
				SubjectKeys: []string{"sub1", "sub2"},
			},
			attrib2: CustomerUsageAttribution{
				ID:          "customer-12",
				Key:         &key1,
				SubjectKeys: []string{"sub2", "sub1"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.attrib1.Equal(tt.attrib2)
			assert.Equal(t, tt.expected, got)
		})
	}
}
