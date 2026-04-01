package streamingretry

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/streaming"
)

// noopConnector satisfies streaming.Connector for config/constructor tests.
type noopConnector struct {
	streaming.Connector
}

func validConfig() Config {
	return Config{
		DownstreamConnector: &noopConnector{},
		Logger:              slog.Default(),
		RetryWaitDuration:   100 * time.Millisecond,
		MaxTries:            3,
	}
}

func TestConfigValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		assert.NoError(t, validConfig().Validate())
	})

	t.Run("missing downstream connector", func(t *testing.T) {
		cfg := validConfig()
		cfg.DownstreamConnector = nil
		assert.ErrorContains(t, cfg.Validate(), "downstream connector is required")
	})

	t.Run("missing logger", func(t *testing.T) {
		cfg := validConfig()
		cfg.Logger = nil
		assert.ErrorContains(t, cfg.Validate(), "logger is required")
	})

	t.Run("zero retry wait duration", func(t *testing.T) {
		cfg := validConfig()
		cfg.RetryWaitDuration = 0
		assert.ErrorContains(t, cfg.Validate(), "retry wait duration must be greater than 0")
	})

	t.Run("negative retry wait duration", func(t *testing.T) {
		cfg := validConfig()
		cfg.RetryWaitDuration = -1
		assert.ErrorContains(t, cfg.Validate(), "retry wait duration must be greater than 0")
	})

	t.Run("max tries of 1 is valid", func(t *testing.T) {
		cfg := validConfig()
		cfg.MaxTries = 1
		assert.NoError(t, cfg.Validate())
	})

	t.Run("max tries of 0 is invalid", func(t *testing.T) {
		cfg := validConfig()
		cfg.MaxTries = 0
		assert.ErrorContains(t, cfg.Validate(), "max tries must be at least 1")
	})

	t.Run("negative max tries is invalid", func(t *testing.T) {
		cfg := validConfig()
		cfg.MaxTries = -1
		assert.ErrorContains(t, cfg.Validate(), "max tries must be at least 1")
	})

	t.Run("multiple errors returned together", func(t *testing.T) {
		cfg := Config{}
		err := cfg.Validate()
		assert.ErrorContains(t, err, "downstream connector is required")
		assert.ErrorContains(t, err, "logger is required")
		assert.ErrorContains(t, err, "retry wait duration must be greater than 0")
		assert.ErrorContains(t, err, "max tries must be at least 1")
	})
}

func TestNewConnector(t *testing.T) {
	t.Run("invalid config returns error", func(t *testing.T) {
		_, err := New(Config{})
		assert.Error(t, err)
	})

	t.Run("max delay is preserved when set", func(t *testing.T) {
		cfg := validConfig()
		cfg.MaxDelay = 30 * time.Second

		c, err := New(cfg)
		require.NoError(t, err)
		assert.Equal(t, 30*time.Second, c.maxDelay)
	})
}

type MockRetryable struct {
	mock.Mock
}

func (m *MockRetryable) Action() (int, error) {
	callArgs := m.Called()
	return callArgs.Get(0).(int), callArgs.Error(1)
}

func TestRetry(t *testing.T) {
	connector := &Connector{
		maxTries:          2,
		retryWaitDuration: 10 * time.Millisecond,
		maxDelay:          1 * time.Second,
		logger:            slog.Default(),
	}

	t.Run("should return the result of the action", func(t *testing.T) {
		require := require.New(t)

		mockRetryable := &MockRetryable{}
		mockRetryable.On("Action").Return(1, nil).Once()

		res, err := withRetry(t.Context(), connector, mockRetryable.Action)
		require.NoError(err)
		require.Equal(1, res)
	})

	t.Run("should retry on retirable errors", func(t *testing.T) {
		require := require.New(t)

		mockRetryable := &MockRetryable{}
		mockRetryable.On("Action").Return(1, io.ErrUnexpectedEOF).Once()
		mockRetryable.On("Action").Return(1, nil).Once()

		res, err := withRetry(t.Context(), connector, mockRetryable.Action)
		require.NoError(err)
		require.Equal(1, res)
	})

	t.Run("should return the error if the action fails after the maximum number of retries", func(t *testing.T) {
		require := require.New(t)

		mockRetryable := &MockRetryable{}
		mockRetryable.On("Action").Return(1, io.ErrUnexpectedEOF).Once()
		mockRetryable.On("Action").Return(1, io.ErrUnexpectedEOF).Once()
		mockRetryable.On("Action").Return(1, io.ErrUnexpectedEOF).Once()
		mockRetryable.On("Action").Return(1, io.ErrUnexpectedEOF).Once()

		res, err := withRetry(t.Context(), connector, mockRetryable.Action)
		require.Error(err)
		require.ErrorIs(err, io.ErrUnexpectedEOF)
		require.Equal(0, res)
	})

	t.Run("should return the error if the action returns a non-retirable error", func(t *testing.T) {
		require := require.New(t)

		mockRetryable := &MockRetryable{}
		testErr := errors.New("test error")
		mockRetryable.On("Action").Return(1, testErr).Once()

		res, err := withRetry(t.Context(), connector, mockRetryable.Action)
		require.Error(err)
		require.ErrorIs(err, testErr)
		require.Equal(0, res)
	})
}

func TestRetryContextCancellation(t *testing.T) {
	maxTries := 10
	connector := &Connector{
		maxTries:          maxTries,
		retryWaitDuration: 100 * time.Millisecond,
		maxDelay:          5 * time.Second,
		logger:            slog.Default(),
	}

	ctx, cancel := context.WithCancel(t.Context())

	var attempts int
	_, err := withRetry(ctx, connector, func() (string, error) {
		attempts++
		if attempts == 2 {
			cancel()
		}
		return "", io.ErrUnexpectedEOF
	})

	require.Error(t, err)
	assert.Less(t, attempts, maxTries, "should have stopped retrying after context cancellation")
}

func TestRetryBackoffAndJitter(t *testing.T) {
	baseDelay := 50 * time.Millisecond
	connector := &Connector{
		maxTries:          5,
		retryWaitDuration: baseDelay,
		maxDelay:          5 * time.Second,
		logger:            slog.Default(),
	}

	attempt := 0
	var callTimes []time.Time

	res, err := withRetry(t.Context(), connector, func() (string, error) {
		attempt++
		callTimes = append(callTimes, time.Now())

		// Fail the first 3 attempts with a retryable error, then succeed.
		if attempt <= 3 {
			return "", io.ErrUnexpectedEOF
		}

		return "ok", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", res)
	assert.Equal(t, 4, attempt, "should have taken 4 attempts")

	// Log the delay between each retry to show backoff + jitter.
	for i := 1; i < len(callTimes); i++ {
		gap := callTimes[i].Sub(callTimes[i-1])
		t.Logf("gap %d→%d: %v", i, i+1, gap.Round(time.Millisecond))
	}

	totalElapsed := callTimes[len(callTimes)-1].Sub(callTimes[0])
	flatTotal := 3 * baseDelay
	assert.Greater(t, totalElapsed, flatTotal, "backoff total delay should exceed flat retry delay")
}
