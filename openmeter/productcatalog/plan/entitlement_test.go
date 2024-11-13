package plan

import (
	"testing"

	json "github.com/json-iterator/go"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/datex"
)

func TestEntitlementTemplate_JSON(t *testing.T) {
	tests := []struct {
		Name                string
		EntitlementTemplate EntitlementTemplate
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
				UsagePeriod:             datex.MustParse(t, "P1M"),
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

			d := EntitlementTemplate{}
			err = json.Unmarshal(b, &d)
			require.NoError(t, err)

			assert.Equal(t, test.EntitlementTemplate, d)
		})
	}
}
