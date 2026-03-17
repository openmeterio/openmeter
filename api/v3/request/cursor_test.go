package request_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/request"
)

func TestEncodeCursor(t *testing.T) {
	t.Run("empty id returns ErrInvalidCursor", func(t *testing.T) {
		_, err := request.EncodeCursor(request.DefaultCipherKey, "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, request.ErrInvalidCursor))
	})

	t.Run("non-empty id returns non-empty encoded string", func(t *testing.T) {
		c, err := request.EncodeCursor(request.DefaultCipherKey, uuid.New().String())
		require.NoError(t, err)
		assert.NotEmpty(t, c.String())
	})

	t.Run("encoded value differs from original id", func(t *testing.T) {
		id := uuid.New().String()
		c, err := request.EncodeCursor(request.DefaultCipherKey, id)
		require.NoError(t, err)
		assert.NotEqual(t, id, c.String())
	})
}

func TestDecodeCursor(t *testing.T) {
	t.Run("empty cursor returns ErrInvalidCursor", func(t *testing.T) {
		_, err := request.DecodeCursor(request.DefaultCipherKey, "", false)
		require.Error(t, err)
		assert.True(t, errors.Is(err, request.ErrInvalidCursor))
	})

	t.Run("non-base64 input returns ErrInvalidCursor", func(t *testing.T) {
		_, err := request.DecodeCursor(request.DefaultCipherKey, "not-valid-base64!!!", false)
		require.Error(t, err)
		assert.True(t, errors.Is(err, request.ErrInvalidCursor))
	})

	t.Run("valid base64 with wrong content returns ErrInvalidCursor", func(t *testing.T) {
		// base64("hello") — valid base64 but not a valid cursor
		_, err := request.DecodeCursor(request.DefaultCipherKey, "aGVsbG8=", false)
		require.Error(t, err)
		assert.True(t, errors.Is(err, request.ErrInvalidCursor))
	})
}

func TestCursorRoundtrip(t *testing.T) {
	t.Run("single uuid roundtrip", func(t *testing.T) {
		id := uuid.New().String()
		encoded, err := request.EncodeCursor(request.DefaultCipherKey, id)
		require.NoError(t, err)

		decoded, err := request.DecodeCursor(request.DefaultCipherKey, encoded.String(), false)
		require.NoError(t, err)
		assert.Equal(t, id, decoded.ID())
	})

	t.Run("compound uuid roundtrip", func(t *testing.T) {
		id := uuid.New().String() + ":" + uuid.New().String()
		encoded, err := request.EncodeCursor(request.DefaultCipherKey, id)
		require.NoError(t, err)

		decoded, err := request.DecodeCursor(request.DefaultCipherKey, encoded.String(), false)
		require.NoError(t, err)
		assert.Equal(t, id, decoded.ID())
	})

	t.Run("custom cipher key roundtrip", func(t *testing.T) {
		const key = "custom-cipher-key-long-enough-for-testing-purposes"
		id := uuid.New().String()
		encoded, err := request.EncodeCursor(key, id)
		require.NoError(t, err)

		decoded, err := request.DecodeCursor(key, encoded.String(), false)
		require.NoError(t, err)
		assert.Equal(t, id, decoded.ID())
	})

	t.Run("wrong cipher key fails to decode correctly but does not error without uuid validation", func(t *testing.T) {
		id := uuid.New().String()
		encoded, err := request.EncodeCursor(request.DefaultCipherKey, id)
		require.NoError(t, err)

		decoded, err := request.DecodeCursor("wrong-key-that-is-long-enough-for-xor-cipher-test", encoded.String(), false)
		// decodes without error but ID will differ
		require.NoError(t, err)
		assert.NotEqual(t, id, decoded.ID())
	})
}

func TestCursorUUIDValidation(t *testing.T) {
	t.Run("valid uuid passes validation", func(t *testing.T) {
		id := uuid.New().String()
		encoded, err := request.EncodeCursor(request.DefaultCipherKey, id)
		require.NoError(t, err)

		decoded, err := request.DecodeCursor(request.DefaultCipherKey, encoded.String(), true)
		require.NoError(t, err)
		assert.Equal(t, id, decoded.ID())
	})

	t.Run("non-uuid value fails validation", func(t *testing.T) {
		encoded, err := request.EncodeCursor(request.DefaultCipherKey, "not-a-uuid")
		require.NoError(t, err)

		_, err = request.DecodeCursor(request.DefaultCipherKey, encoded.String(), true)
		require.Error(t, err)
		assert.True(t, errors.Is(err, request.ErrInvalidCursor))
	})

	t.Run("compound valid uuids pass validation", func(t *testing.T) {
		id := uuid.New().String() + ":" + uuid.New().String()
		encoded, err := request.EncodeCursor(request.DefaultCipherKey, id)
		require.NoError(t, err)

		decoded, err := request.DecodeCursor(request.DefaultCipherKey, encoded.String(), true)
		require.NoError(t, err)
		assert.Equal(t, id, decoded.ID())
	})

	t.Run("compound with one invalid part fails validation", func(t *testing.T) {
		id := uuid.New().String() + ":not-a-uuid"
		encoded, err := request.EncodeCursor(request.DefaultCipherKey, id)
		require.NoError(t, err)

		_, err = request.DecodeCursor(request.DefaultCipherKey, encoded.String(), true)
		require.Error(t, err)
		assert.True(t, errors.Is(err, request.ErrInvalidCursor))
	})
}
