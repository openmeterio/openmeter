package modelref

import (
	"encoding/json"
	"fmt"
)

type Ref interface {
	Value() string
}

type VersionedKeyRef struct {
	Key     string
	Version int
}

func (r VersionedKeyRef) Value() string {
	return fmt.Sprintf("%s@%d", r.Key, r.Version)
}

var _ json.Marshaler = VersionedKeyRef{}

func (r VersionedKeyRef) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.Value() + `"`), nil
}
