package productcatalog

import (
	"testing"

	json "github.com/json-iterator/go"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/datetime"
)

func TestEntitlementTemplate_JSON(t *testing.T) {
	tests := []struct {
		Name                string
		EntitlementTemplate *EntitlementTemplate
		ExpectedError       bool
	}{
		{
			Name: "Metered",
			EntitlementTemplate: NewEntitlementTemplateFrom(MeteredEntitlementTemplate{
				Metadata: map[string]string{
					"key": "value",
				},
				IsSoftLimit:             true,
				IssueAfterReset:         lo.ToPtr(500.0),
				IssueAfterResetPriority: lo.ToPtr[uint8](1),
				PreserveOverageAtReset:  lo.ToPtr(true),
				UsagePeriod:             datetime.MustParseDuration(t, "P1M"),
			}),
		},
		{
			Name: "Static",
			EntitlementTemplate: NewEntitlementTemplateFrom(StaticEntitlementTemplate{
				Metadata: map[string]string{
					"key": "value",
				},
				Config: []byte(`{"key":"value"}`),
			}),
		},
		{
			Name: "Boolean",
			EntitlementTemplate: NewEntitlementTemplateFrom(BooleanEntitlementTemplate{
				Metadata: map[string]string{
					"key": "value",
				},
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			b, err := json.Marshal(&test.EntitlementTemplate)
			require.NoError(t, err)

			t.Logf("Serialized EntitlementTemplate: %s", string(b))

			d := &EntitlementTemplate{}
			err = json.Unmarshal(b, d)
			require.NoError(t, err)

			assert.Equal(t, test.EntitlementTemplate, d)
		})
	}
}

func TestEntitlementTemplateEqual(t *testing.T) {
	tests := []struct {
		Name string

		Left  *EntitlementTemplate
		Right *EntitlementTemplate

		ExpectedResult bool
	}{
		{
			Name: "Equal",
			Left: NewEntitlementTemplateFrom(
				MeteredEntitlementTemplate{
					Metadata:                map[string]string{"name": "metered"},
					IsSoftLimit:             true,
					IssueAfterReset:         lo.ToPtr(1000.0),
					IssueAfterResetPriority: lo.ToPtr[uint8](5),
					PreserveOverageAtReset:  lo.ToPtr(true),
					UsagePeriod:             datetime.MustParseDuration(t, "P1M"),
				},
			),
			Right: NewEntitlementTemplateFrom(
				MeteredEntitlementTemplate{
					Metadata:                map[string]string{"name": "metered"},
					IsSoftLimit:             true,
					IssueAfterReset:         lo.ToPtr(1000.0),
					IssueAfterResetPriority: lo.ToPtr[uint8](5),
					PreserveOverageAtReset:  lo.ToPtr(true),
					UsagePeriod:             datetime.MustParseDuration(t, "P1M"),
				},
			),
			ExpectedResult: true,
		},
		{
			Name: "ContentMismatch",
			Left: NewEntitlementTemplateFrom(
				MeteredEntitlementTemplate{
					Metadata:                map[string]string{"name": "metered1"},
					IsSoftLimit:             true,
					IssueAfterReset:         lo.ToPtr(1000.0),
					IssueAfterResetPriority: lo.ToPtr[uint8](5),
					PreserveOverageAtReset:  lo.ToPtr(true),
					UsagePeriod:             datetime.MustParseDuration(t, "P1M"),
				},
			),
			Right: NewEntitlementTemplateFrom(
				MeteredEntitlementTemplate{
					Metadata:                map[string]string{"name": "metered2"},
					IsSoftLimit:             false,
					IssueAfterReset:         lo.ToPtr(2000.0),
					IssueAfterResetPriority: lo.ToPtr[uint8](1),
					PreserveOverageAtReset:  lo.ToPtr(false),
					UsagePeriod:             datetime.MustParseDuration(t, "P3M"),
				},
			),
			ExpectedResult: false,
		},
		{
			Name: "TypeMismatch",
			Left: NewEntitlementTemplateFrom(
				MeteredEntitlementTemplate{
					Metadata:                map[string]string{"name": "metered1"},
					IsSoftLimit:             true,
					IssueAfterReset:         lo.ToPtr(1000.0),
					IssueAfterResetPriority: lo.ToPtr[uint8](5),
					PreserveOverageAtReset:  lo.ToPtr(true),
					UsagePeriod:             datetime.MustParseDuration(t, "P1M"),
				},
			),
			Right: NewEntitlementTemplateFrom(
				StaticEntitlementTemplate{
					Metadata: map[string]string{"name": "metered2"},
					Config:   []byte(`"name": "metered1"`),
				},
			),
			ExpectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			match := test.Left.Equal(test.Right)
			assert.Equal(t, test.ExpectedResult, match)
		})
	}
}
