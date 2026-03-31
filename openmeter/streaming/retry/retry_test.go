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
)

type MockRetryable struct {
	mock.Mock
}

func (m *MockRetryable) Action() (int, error) {
	callArgs := m.Called()
	return callArgs.Get(0).(int), callArgs.Error(1)
}

func TestRetry(t *testing.T) {
	require := require.New(t)

	connector := &Connector{
		maxTries:          2,
		retryWaitDuration: 10 * time.Millisecond,
		logger:            slog.Default(),
	}

	t.Run("should return the result of the action", func(t *testing.T) {
		mockRetryable := &MockRetryable{}
		mockRetryable.On("Action").Return(1, nil).Once()

		res, err := retry(t.Context(), connector, mockRetryable.Action)
		require.NoError(err)
		require.Equal(1, res)
	})

	t.Run("should retry on retirable errors", func(t *testing.T) {
		mockRetryable := &MockRetryable{}

		mockRetryable.On("Action").Return(1, io.ErrUnexpectedEOF).Once()
		mockRetryable.On("Action").Return(1, nil).Once()

		res, err := retry(t.Context(), connector, mockRetryable.Action)
		require.NoError(err)
		require.Equal(1, res)
	})

	t.Run("should return the error if the action fails after the maximum number of retries", func(t *testing.T) {
		mockRetryable := &MockRetryable{}
		mockRetryable.On("Action").Return(1, io.ErrUnexpectedEOF).Once()
		mockRetryable.On("Action").Return(1, io.ErrUnexpectedEOF).Once()
		mockRetryable.On("Action").Return(1, io.ErrUnexpectedEOF).Once()
		mockRetryable.On("Action").Return(1, io.ErrUnexpectedEOF).Once()

		res, err := retry(t.Context(), connector, mockRetryable.Action)
		require.Error(err)
		require.ErrorIs(err, io.ErrUnexpectedEOF)
		require.Equal(0, res)
	})

	t.Run("should return the error if the action returns a non-retirable error", func(t *testing.T) {
		mockRetryable := &MockRetryable{}
		testErr := errors.New("test error")
		mockRetryable.On("Action").Return(1, testErr).Once()

		res, err := retry(t.Context(), connector, mockRetryable.Action)
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
		logger:            slog.Default(),
	}

	ctx, cancel := context.WithCancel(t.Context())

	var attempts int
	_, err := retry(ctx, connector, func() (string, error) {
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
		logger:            slog.Default(),
	}

	attempt := 0
	var callTimes []time.Time

	start := time.Now()
	res, err := retry(t.Context(), connector, func() (string, error) {
		attempt++
		callTimes = append(callTimes, time.Now())

		// Fail the first 3 attempts with a retryable error, then succeed.
		if attempt <= 3 {
			t.Logf("attempt %d: returning error (elapsed: %v)", attempt, time.Since(start).Round(time.Millisecond))
			return "", io.ErrUnexpectedEOF
		}

		t.Logf("attempt %d: returning success (elapsed: %v)", attempt, time.Since(start).Round(time.Millisecond))
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

	// Verify exponential backoff: last gap should be larger than first.
	firstGap := callTimes[1].Sub(callTimes[0])
	lastGap := callTimes[len(callTimes)-1].Sub(callTimes[len(callTimes)-2])
	assert.Greater(t, lastGap, firstGap, "backoff should make later delays larger")
}
