package request

import (
	"encoding/base64"
	"errors"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

// ErrInvalidCursor is used when a cursor does not conform to the expected format.
var ErrInvalidCursor = errors.New("invalid pagination cursor provided")

const (
	DefaultCipherKey = "Oh hai there! I was originally written in Koko, the super awesome control plane for Kong Gateway! AND VOILA"
	cursorVersion    = "1"
)

// Cursor is a representation of an object's ID, as a cursor should
// be opaque and that its format should not be relied upon.
//
// A cursor is used to implement keyset pagination within a database.
//
// Under the hood, this uses a simple XOR cipher.
type Cursor struct{ encoded, decoded, version string }

// String implements fmt.Stringer & returns the encoded cursor representation of the ID.
func (c *Cursor) String() string { return c.encoded }

// ID decodes the provided cursor (during instantiation) and returns its representation
// (usually set to a UUID or [UUID:]UUID). If there is no cursor, an empty string is returned.
func (c *Cursor) ID() string {
	if c == nil {
		return ""
	}
	return c.decoded
}

// xorText is a simple XOR cipher implementation.
// Read more: https://en.wikipedia.org/wiki/XOR_cipher
func xorText(cipherKey, input string) string {
	var output string
	keyLen := len(cipherKey)
	for i := range input {
		output += string(input[i] ^ cipherKey[i%keyLen])
	}
	return output
}

// EncodeCursor instantiates a new Cursor from an object's ID. A cursor ID can
// be one or more UUIDs separated by the colon char.
//
// You must only provide an ASCII string. If UTF-8 is used (e.g.: graphics), decoding will
// not function properly. There are no error checks for this due to performance reasons.
//
// ErrInvalidCursor be returned when the provided ID is empty.
func EncodeCursor(cipherKey, id string) (*Cursor, error) {
	if id == "" {
		return nil, ErrInvalidCursor
	}

	c := Cursor{
		encoded: xorText(cipherKey, id),
		decoded: id,
		version: cursorVersion,
	}

	// Inject the version into the XOR'ed string.
	versionIdx := getVersionIdx(c.encoded)
	c.encoded = c.encoded[:versionIdx] + c.version + c.encoded[versionIdx:]

	// Base64 encoding for ASCII compatibility.
	c.encoded = base64.StdEncoding.EncodeToString([]byte(c.encoded))

	// we need to pass the base64 encoded string through QueryEscape as the encoded string will contain
	// certain characters which are not valid in a URL parser.
	// Example : when the id input is `01960a8d-eccd-72c0-935e-a87370362c2a`
	// 			 the base64 encoded string is `f1kZXlEIGBBFABEGRQ1+EhRRMV4ZXEcMSghWVl9bSRNBQApGFQ==`
	//           The + char in the above string is not valid in a URL parser.
	c.encoded = url.QueryEscape(c.encoded)

	return &c, nil
}

// DecodeCursor instantiates a new Cursor from an encoded cursor value with query escaped value.
//
// Returns ErrInvalidCursor when an invalid cursor is provided (and optionally
// validates the decoded value as a UUID when validateAsUUID is true).
func DecodeCursor(cipherKey, cursor string, validateAsUUID bool) (*Cursor, error) {
	if cursor == "" {
		return nil, ErrInvalidCursor
	}

	unescapeCursor, err := url.QueryUnescape(cursor)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	return decodeCursorAfterQueryUnescape(cipherKey, unescapeCursor, validateAsUUID)
}

// decodeCursorAfterQueryUnescape instantiates a new Cursor from already unescaped cursor value.
// This function is needed because GetAipAttributes function already
// unescapes the query param so we need not unescape it again.
//
// Returns ErrInvalidCursor when an invalid cursor is provided (and optionally
// validates the decoded value as a UUID when validateAsUUID is true).
func decodeCursorAfterQueryUnescape(cipherKey, cursor string, validateAsUUID bool) (*Cursor, error) {
	if cursor == "" {
		return nil, ErrInvalidCursor
	}

	// All cursors should always be base64 encoded (see store.EncodeCursor).
	v, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	c := Cursor{encoded: cursor, decoded: string(v)}

	// We're adding a single-character version ID to the cursor, in case we ever change this
	// implementation. As such, we need to account for it & remove it before XOR'ing the input.
	versionIdx := getVersionIdx(c.decoded[1:])
	if c.version = string(c.decoded[versionIdx]); c.version != cursorVersion {
		return nil, ErrInvalidCursor
	}
	c.decoded = xorText(cipherKey, c.decoded[:versionIdx]+c.decoded[versionIdx+1:])

	if validateAsUUID {
		ids := strings.Split(c.decoded, ":")
		if !lo.EveryBy(ids, func(id string) bool { _, err := uuid.Parse(id); return err == nil }) {
			return nil, ErrInvalidCursor
		}
	}

	return &c, err
}

// getVersionIdx returns the index where the single character is, representing the version of this implementation.
func getVersionIdx(input string) int {
	// The version is injected halfway in the string to not introduce too much predictability.
	return int(float64(len(input) / 2)) //nolint:gomnd
}
