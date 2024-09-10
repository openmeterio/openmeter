package modelref

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	TypeValueSeparator = ":"
	VersionSeparator   = "@"
)

type Ref interface {
	Value() string
}

type VersionedKeyRef struct {
	Key     string
	Version int
}

func (r VersionedKeyRef) Value() string {
	return fmt.Sprintf("%s%s%d", r.Key, VersionSeparator, r.Version)
}

var _ json.Marshaler = VersionedKeyRef{}

func (r VersionedKeyRef) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.Value() + `"`), nil
}

// Using mixed references

func SeparatedKeyParser(key string) (typ string, val string, err error) {
	_, err = fmt.Sscanf(strings.Replace(key, TypeValueSeparator, " ", 1), "%s%s", &typ, &val)
	if err != nil {
		err = fmt.Errorf("failed to parse key: %w", err)
	}
	return
}

func SeparatedKeyToValue(typ string, val string) string {
	return fmt.Sprintf("%s%s%s", typ, TypeValueSeparator, val)
}

type IdOrKeyRefType string

const (
	IdRefType  IdOrKeyRefType = "id"
	KeyRefType IdOrKeyRefType = "key"
)

// IdOrKeyRef is a reference to an object by either its ID or its Key.
// This makes sense for any any Versioned and Archiveable keyed resources (currently Features)
// where you might want to reference either an exact version (by ID) or the latest version (by Key).
type IdOrKeyRef struct {
	typ IdOrKeyRefType
	val string
}

// Value returns a formatted representation of the reference.
// By convention, type always precedes value.
func (r IdOrKeyRef) Value() string {
	return SeparatedKeyToValue(string(r.typ), r.val)
}

func (r IdOrKeyRef) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.Value() + `"`), nil
}

func (r IdOrKeyRef) ByType() (string, IdOrKeyRefType) {
	return r.val, r.typ
}

func NewIdOrKeyRefFromValue(value string) (IdOrKeyRef, error) {
	typ, val, err := SeparatedKeyParser(value)
	if err != nil {
		return IdOrKeyRef{}, err
	}

	// TODO: validate typ

	return IdOrKeyRef{typ: IdOrKeyRefType(typ), val: val}, nil
}

func NewIdOrKeyRef(typ IdOrKeyRefType, val string) IdOrKeyRef {
	return IdOrKeyRef{typ: typ, val: val}
}
