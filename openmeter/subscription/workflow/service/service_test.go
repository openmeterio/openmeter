package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewWorkflowServiceValidatesConfig(t *testing.T) {
	_, err := NewWorkflowService(WorkflowServiceConfig{})
	require.Error(t, err)
	require.ErrorContains(t, err, "subscription service is required")
	require.ErrorContains(t, err, "subscription add-on service is required")
	require.ErrorContains(t, err, "customer service is required")
	require.ErrorContains(t, err, "transaction manager is required")
	require.ErrorContains(t, err, "logger is required")
	require.ErrorContains(t, err, "locker is required")
	require.ErrorContains(t, err, "feature flags service is required")
}
