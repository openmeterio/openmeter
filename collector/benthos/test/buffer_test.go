package test

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	_ "github.com/redpanda-data/benthos/v4/public/components/io"
	_ "github.com/redpanda-data/benthos/v4/public/components/pure"
	_ "github.com/redpanda-data/benthos/v4/public/components/pure/extended"
	"github.com/redpanda-data/benthos/v4/public/service"

	_ "github.com/openmeterio/openmeter/collector/benthos/bloblang" // import bloblang plugins
	_ "github.com/openmeterio/openmeter/collector/benthos/input"    // import input plugins
	_ "github.com/openmeterio/openmeter/collector/benthos/output"   // import output plugins
)

//go:embed testdata/buffer/config.yaml
var config []byte

type cliArgs struct {
	LogLevel   string
	Values     map[string]string
	ConfigPath string
}

func (a cliArgs) Args() []string {
	args := []string{
		"benthos",
		"run",
	}

	if len(a.Values) > 0 {
		for k, v := range a.Values {
			args = append(args, "--set", fmt.Sprintf("%s=%s", k, v))
		}
	}

	if a.LogLevel == "" {
		args = append(args, "--log.level", "info")
	} else {
		args = append(args, "--log.level", a.LogLevel)
	}

	args = append(args, a.ConfigPath)

	return args
}

func getTCPListenAddress() (*net.TCPAddr, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = l.Close()
	}()

	addr, err = net.ResolveTCPAddr("tcp", l.Addr().String())
	if err != nil {
		return nil, err
	}

	return addr, nil
}

func TestCLIWithBuffer(t *testing.T) {
	var err error

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err = os.WriteFile(configPath, config, 0700)
	require.NoError(t, err, "failed to write config")

	t.Cleanup(func() {
		t.Logf("Cleaning up test directory: %s", tmpDir)
		if err = os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temporary directory: %v", err)
		}
	})

	netAddr, err := getTCPListenAddress()
	require.NoError(t, err, "Failed to get TCP listen address")

	args := cliArgs{
		LogLevel: "debug",
		Values: map[string]string{
			"http.address": netAddr.String(),
		},
		ConfigPath: configPath,
	}

	var summary atomic.Pointer[service.RunningStreamSummary]

	stopChan := make(chan struct{})
	stopChanCloser := sync.OnceFunc(func() {
		t.Log("Closing control channel")
		close(stopChan)
	})
	t.Cleanup(stopChanCloser)

	var exitCode int
	go func() {
		exitCode, err = service.RunCLIToCode(ctx,
			service.CLIOptSetArgs(args.Args()...),
			service.CLIOptOnStreamStart(func(s *service.RunningStreamSummary) error {
				summary.Store(s)
				return nil
			}),
			service.CLIOptAddTeeLogger(slog.Default()),
			service.CLIOptSetBinaryName(t.Name()),
			service.CLIOptSetVersion("test", time.Now().Format(time.RFC3339)),
		)

		stopChanCloser()
	}()

	t.Cleanup(func() {
		t.Logf("Exit code: %d", exitCode)
		// FIXME: log error
	})

	t.Run("Started", func(t *testing.T) {
		var statuses []service.ConnectionStatus

		require.Eventually(t, func() bool {
			sum := summary.Load()
			if sum == nil {
				return false
			}

			statuses = sum.ConnectionStatuses()

			return true
		}, 10*time.Second, time.Second)

		for _, status := range statuses {
			require.NoError(t, status.Err(), "label", status.Label(), "path", status.Path())
		}
	})
}
