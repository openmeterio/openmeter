package ref

import (
	"fmt"

	"github.com/oklog/ulid/v2"
)

type IDOrKey struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

func (i IDOrKey) Validate() error {
	if i.ID == "" && i.Key == "" {
		return fmt.Errorf("either id or key is required")
	}

	return nil
}

func ParseIDOrKey(s string) IDOrKey {
	n := IDOrKey{}

	_, err := ulid.Parse(s)
	if err != nil {
		n.Key = s
	} else {
		n.ID = s
	}

	return n
}
