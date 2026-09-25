package service_test

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestListRules_RequiresNamespace(t *testing.T) {
	env := newServiceTestEnv(t)

	_, err := env.service.ListRules(t.Context(), notification.ListRulesInput{
		Page: pagination.NewPage(1, 20),
	})
	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err), "expected a models.GenericValidationError, got: %v", err)
}

func TestListRules_InvalidFilterCombinationRejected(t *testing.T) {
	env := newServiceTestEnv(t)

	_, err := env.service.ListRules(t.Context(), notification.ListRulesInput{
		Namespaces: []string{ulid.Make().String()},
		Page:       pagination.NewPage(1, 20),
		Name: &filter.FilterString{
			Eq:       lo.ToPtr("a"),
			Contains: lo.ToPtr("b"),
		},
	})
	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err), "expected a models.GenericValidationError, got: %v", err)
	assert.ErrorIs(t, err, filter.ErrFilterMultipleOperators)
}
