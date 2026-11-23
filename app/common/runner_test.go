package common

import (
	"fmt"
	"log/slog"
	"net/http"
	"syscall"
	"testing"

	"github.com/oklog/run"
	"github.com/stretchr/testify/require"
)

func TestRunnerReturnsUnexpectedFailures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		actorErr error
		wantErr  error
	}{
		{name: "telemetry bind failure", actorErr: fmt.Errorf("listen: %w", syscall.EADDRINUSE), wantErr: syscall.EADDRINUSE},
		{name: "signal shutdown", actorErr: &run.SignalError{Signal: syscall.SIGTERM}},
		{name: "closed server", actorErr: http.ErrServerClosed},
		{name: "normal exit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := Runner{Logger: slog.New(slog.DiscardHandler)}
			runner.Group.Add(func() error { return tc.actorErr }, func(error) {})
			require.ErrorIs(t, runner.RunWithError(), tc.wantErr)
		})
	}
}
