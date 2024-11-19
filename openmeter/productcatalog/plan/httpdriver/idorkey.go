package httpdriver

import (
	"fmt"

	"github.com/oklog/ulid/v2"
)

var _ fmt.Stringer = (*IDOrKey)(nil)

// IDOrKey is a container which either contains an ID in ULID format or a Key.
type IDOrKey struct {
	// ID is a globally unique identifier in ULID format.
	ID string `json:"id,omitempty"`

	// Key is a human-readable identifier which is unique in scope of a namespace
	Key string `json:"key,omitempty"`
}

func (i *IDOrKey) IsEmpty() bool {
	return i.ID == "" && i.Key == ""
}

// Parse returns an IDOrKey where the ID attribute is set to s in case it is a valid ULID otherwise s is assumed to be a Key.
func (i *IDOrKey) Parse(s string) {
	n := IDOrKey{}

	_, err := ulid.Parse(s)
	if err != nil {
		n.Key = s
	} else {
		n.ID = s
	}

	*i = n
}

func (i *IDOrKey) String() string {
	if i.ID != "" {
		return i.ID
	} else {
		return i.Key
	}
}

func NewIDOrKey(s string) *IDOrKey {
	i := &IDOrKey{}
	i.Parse(s)

	return i
}
