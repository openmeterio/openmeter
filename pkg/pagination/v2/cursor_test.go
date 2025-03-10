package pagination

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCursor(t *testing.T) {
	// Test that time is converted to UTC
	loc, err := time.LoadLocation("America/New_York")
	assert.NoError(t, err)

	nyTime := time.Date(2023, 5, 15, 12, 30, 0, 0, loc)
	cursor := NewCursor(nyTime, "test-id")

	assert.Equal(t, nyTime.UTC(), cursor.Time, "Time should be stored in UTC")
	assert.Equal(t, "test-id", cursor.ID, "ID should be stored as provided")
}

func TestCursorEncodeDecode(t *testing.T) {
	tests := []struct {
		name       string
		time       time.Time
		id         string
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Basic cursor",
			time:    time.Date(2023, 5, 15, 12, 30, 0, 0, time.UTC),
			id:      "test-id",
			wantErr: false,
		},
		{
			name:    "Empty ID",
			time:    time.Date(2023, 5, 15, 12, 30, 0, 0, time.UTC),
			id:      "",
			wantErr: false,
		},
		{
			name:    "ID with delimiter (comma)",
			time:    time.Date(2023, 5, 15, 12, 30, 0, 0, time.UTC),
			id:      "id,with,commas",
			wantErr: false,
		},
		{
			name:    "ID with special characters",
			time:    time.Date(2023, 5, 15, 12, 30, 0, 0, time.UTC),
			id:      "id|with!special@chars#$%^&*()",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create and encode a cursor
			cursor := NewCursor(tt.time, tt.id)
			encoded := cursor.Encode()

			// Decode the cursor
			decoded, err := DecodeCursor(encoded)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Equal(t, tt.errMessage, err.Error(), "error message should match expected")
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, cursor.Time.Unix(), decoded.Time.Unix(), "Decoded time should match original")
			assert.Equal(t, cursor.ID, decoded.ID, "Decoded ID should match original")
		})
	}
}

func TestTimeEncodingIsConsistent(t *testing.T) {
	// Test that different time zones are normalized in encoding
	utcTime := time.Date(2023, 5, 15, 12, 30, 0, 0, time.UTC)

	// Create a time with a different zone but same instant
	est, err := time.LoadLocation("America/New_York")
	assert.NoError(t, err)
	estTime := utcTime.In(est)

	// Create cursors with the same time but different zones
	cursorUTC := NewCursor(utcTime, "id")
	cursorEST := NewCursor(estTime, "id")

	// The encodings should be identical
	assert.Equal(t, cursorUTC.Encode(), cursorEST.Encode(),
		"Cursors with same time instant in different zones should encode identically")
}

func TestDecodeCursorWithInvalidInput(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		errMessage string
	}{
		{
			name:       "Not base64",
			input:      "not-base64",
			errMessage: "decode cursor: illegal base64 data at input byte 3",
		},
		{
			name:       "Base64 but no delimiter",
			input:      "MjAyMy0wNS0xNVQxMjozMDowMFo=", // "2023-05-15T12:30:00Z" in base64
			errMessage: "cursor is invalid: no delimiter found",
		},
		{
			name:       "Base64 but invalid time format",
			input:      "aW52YWxpZC10aW1lLGlk", // "invalid-time,id" in base64
			errMessage: "parse cursor timestamp: parsing time \"invalid-time\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"invalid-time\" as \"2006\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := DecodeCursor(tt.input)
			assert.Error(t, err)
			assert.Equal(t, tt.errMessage, err.Error(), "error message should match expected")
			assert.Nil(t, decoded)
		})
	}
}

func TestRoundTripWithDifferentTimes(t *testing.T) {
	// Test different time representations
	times := []time.Time{
		time.Date(2023, 5, 15, 12, 30, 0, 0, time.UTC),           // Standard date
		time.Date(2023, 5, 15, 12, 30, 0, 123456789, time.UTC),   // With nanoseconds
		time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC), // Far future
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),              // Unix epoch
		time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC),              // Very old date
	}

	for i, tm := range times {
		t.Run(tm.String(), func(t *testing.T) {
			cursor := NewCursor(tm, "test-id")
			encoded := cursor.Encode()

			decoded, err := DecodeCursor(encoded)

			assert.NoError(t, err)
			assert.Equal(t, cursor.Time.Format(time.RFC3339), decoded.Time.Format(time.RFC3339),
				"Times should match after round trip (case %d)", i)
		})
	}
}

func TestTextMarshalUnmarshal(t *testing.T) {
	// Create a cursor with a specific time and ID
	timeStr := "2023-05-15T12:30:00Z"
	tm, err := time.Parse(time.RFC3339, timeStr)
	assert.NoError(t, err)

	id := "test-id-with,comma"
	cursor := NewCursor(tm, id)

	// Test MarshalText
	text, err := cursor.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, cursor.Encode(), string(text), "MarshalText should return the same as Encode()")

	// Test UnmarshalText
	var newCursor Cursor
	err = newCursor.UnmarshalText(text)
	assert.NoError(t, err)

	// Verify the unmarshaled cursor matches the original
	assert.Equal(t, cursor.Time.UTC().Format(time.RFC3339), newCursor.Time.Format(time.RFC3339),
		"Time should be preserved through marshal/unmarshal")
	assert.Equal(t, cursor.ID, newCursor.ID, "ID should be preserved through marshal/unmarshal")

	// Test with invalid text
	var invalidCursor Cursor
	err = invalidCursor.UnmarshalText([]byte("invalid-text"))
	assert.Error(t, err, "UnmarshalText should return an error for invalid text")

	// Test with nil text (should not panic)
	var nilCursor Cursor
	err = nilCursor.UnmarshalText(nil)
	assert.Error(t, err, "UnmarshalText should return an error for nil text")
}
