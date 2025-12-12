package streamingretry

import (
	"errors"
	"io"
	"testing"
	"time"

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
