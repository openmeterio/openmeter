package pagination

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
)

type Cursor struct {
	Time time.Time
	ID   string
}

const cursorDelimiter = ","

// NewCursor creates a new Cursor object with the given time and ID.
// The time is converted to UTC before being stored.
func NewCursor(t time.Time, id string) Cursor {
	return Cursor{
		Time: t.UTC(),
		ID:   id,
	}
}

func (c Cursor) Validate() error {
	var errs []error

	if c.Time.IsZero() {
		errs = append(errs, fmt.Errorf("cursor time is zero"))
	}

	if c.ID == "" {
		errs = append(errs, fmt.Errorf("cursor id is empty"))
	}

	return errors.Join(errs...)
}

// DecodeCursor decodes a base64-encoded cursor string into a Cursor object.
func DecodeCursor(s string) (*Cursor, error) {
	var cursor Cursor

	err := cursor.UnmarshalText([]byte(s))
	if err != nil {
		return nil, err
	}

	return &cursor, nil
}

// Encode converts the cursor to a base64-encoded string representation.
// The encoded string is formatted as <RFC3339 time>,<ID>.
func (c Cursor) Encode() string {
	// Ensure time is in UTC
	t := c.Time.UTC()

	s := fmt.Sprintf("%s%s%s", t.Format(time.RFC3339), cursorDelimiter, c.ID)

	return base64.StdEncoding.EncodeToString([]byte(s))
}

func (c *Cursor) EncodePtr() *string {
	if c == nil {
		return nil
	}

	return lo.ToPtr(c.Encode())
}

// MarshalText implements the encoding.TextMarshaler interface.
// It encodes the cursor into a text form.
func (c Cursor) MarshalText() ([]byte, error) {
	return []byte(c.Encode()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// It decodes the cursor from its text form.
func (c *Cursor) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return fmt.Errorf("text is empty")
	}

	b, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return fmt.Errorf("decode cursor: %w", err)
	}

	parts := strings.SplitN(string(b), cursorDelimiter, 2)

	if len(parts) != 2 {
		return fmt.Errorf("cursor is invalid: no delimiter found")
	}

	// Parse the time
	timestamp, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return fmt.Errorf("parse cursor timestamp: %w", err)
	}

	id := parts[1]

	*c = Cursor{
		Time: timestamp,
		ID:   id,
	}

	return nil
}
