package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisabledWorkerExitsBeforeInitialization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`rte:
  enabled: false
  pollInterval: -1s
credits:
  enabled: false
`), 0o600))

	require.NoError(t, runService(t.Context(), []string{"--config", path}))
	require.NoError(t, runService(t.Context(), []string{"--config", path, "--validate"}))
}
