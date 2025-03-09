package pagination

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

type Cursor struct {
	Time time.Time
	ID   string
}

const cursorDelimiter = ","

// NewCursor creates a new Cursor object with the given time and ID.
// The time is converted to UTC before being stored.
func NewCursor(t time.Time, id string) *Cursor {
	return &Cursor{
		Time: t.UTC(),
		ID:   id,
	}
}

// DecodeCursor decodes a base64-encoded cursor string into a Cursor object.
// It returns nil if the encoded cursor is nil.
func DecodeCursor(encodedCursor *string) (*Cursor, error) {
	if encodedCursor == nil {
		return nil, nil
	}

	byt, err := base64.StdEncoding.DecodeString(*encodedCursor)
	if err != nil {
		return nil, fmt.Errorf("decode cursor: %w", err)
	}

	decodedStr := string(byt)
	parts := strings.SplitN(decodedStr, cursorDelimiter, 2)

	if len(parts) != 2 {
		return nil, fmt.Errorf("cursor is invalid: no delimiter found")
	}

	timeStr := parts[0]
	id := parts[1]

	// Parse the time
	timestamp, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil, fmt.Errorf("parse cursor timestamp: %w", err)
	}

	cursor := &Cursor{
		Time: timestamp,
		ID:   id,
	}

	return cursor, nil
}

// Encode converts the cursor to a base64-encoded string representation.
// The encoded string is formatted as <RFC3339 time>,<ID>.
func (c Cursor) Encode() string {
	// Ensure time is in UTC
	utcTime := c.Time.UTC()

	encodedStr := fmt.Sprintf("%s%s%s", utcTime.Format(time.RFC3339), cursorDelimiter, c.ID)

	return base64.StdEncoding.EncodeToString([]byte(encodedStr))
}

// MarshalText implements the encoding.TextMarshaler interface.
// It encodes the cursor into a text form.
func (c Cursor) MarshalText() ([]byte, error) {
	return []byte(c.Encode()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// It decodes the cursor from its text form.
func (c *Cursor) UnmarshalText(text []byte) error {
	strText := string(text)
	decoded, err := DecodeCursor(&strText)
	if err != nil {
		return err
	}

	if decoded == nil {
		return fmt.Errorf("decoded cursor is nil")
	}

	*c = *decoded
	return nil
}
