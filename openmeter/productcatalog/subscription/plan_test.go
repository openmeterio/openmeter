package plansubscription

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
)

func TestPlanInputValidate(t *testing.T) {
	t.Run("rejects when neither plan reference nor inline plan is set", func(t *testing.T) {
		var in PlanInput
		require.Error(t, in.Validate())
	})

	t.Run("rejects when both plan reference and inline plan are set", func(t *testing.T) {
		var in PlanInput
		in.FromRef(&PlanRefInput{Key: "pro"})
		in.FromInput(&plan.CreatePlanInput{})

		err := in.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only one of")
	})

	t.Run("accepts a plan reference only", func(t *testing.T) {
		var in PlanInput
		in.FromRef(&PlanRefInput{Key: "pro"})
		require.NoError(t, in.Validate())
	})

	t.Run("accepts an inline plan only", func(t *testing.T) {
		var in PlanInput
		in.FromInput(&plan.CreatePlanInput{})
		require.NoError(t, in.Validate())
	})
}
