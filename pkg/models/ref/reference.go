package modelref

import (
	"encoding/json"
	"fmt"
)

type Ref interface {
	Value() string
}

type IDRef string

func (r IDRef) Value() string {
	return string(r)
}

var _ json.Marshaler = IDRef("")

func (r IDRef) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r + `"`), nil
}

type KeyRef string

func (r KeyRef) Value() string {
	return string(r)
}

var _ json.Marshaler = KeyRef("")

func (r KeyRef) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r + `"`), nil
}

type VersionedKeyRef struct {
	Key     KeyRef
	Version int
}

func (r VersionedKeyRef) Value() string {
	return fmt.Sprintf("%s@%d", r.Key, r.Version)
}

var _ json.Marshaler = VersionedKeyRef{}

func (r VersionedKeyRef) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.Value() + `"`), nil
}

type RefType int

const (
	RefTypeID RefType = iota + 1
	RefTypeKey
)

type IDOrKeyRef struct {
	typ    RefType
	idRef  IDRef
	keyRef KeyRef
}

// AsIDRef returns the IDRef if the reference is of type ID, otherwise false.
//
// Example:
//
//	ref := FromID("123")
//	idRef, ok := ref.AsIDRef()
func (ref IDOrKeyRef) AsIDRef() (IDRef, bool) {
	if ref.typ == RefTypeID {
		return ref.idRef, true
	}
	return IDRef(""), false
}

// AsKeyRef returns the KeyRef if the reference is of type Key, otherwise false.
//
// Example:
//
//	ref := &IDOrKeyRef{typ: refTypeKey, keyRef: &KeyRef{Key: "123"}}
//	keyRef, ok := ref.AsKeyRef()
func (ref IDOrKeyRef) AsKeyRef() (KeyRef, bool) {
	if ref.typ == RefTypeKey {
		return ref.keyRef, true
	}
	return KeyRef(""), false
}

// Ref returns the reference as a Ref interface.
func (ref IDOrKeyRef) Ref() (Ref, bool) {
	if ref.typ == RefTypeID {
		return ref.idRef, true
	}
	if ref.typ == RefTypeKey {
		return ref.keyRef, true
	}
	return nil, false
}

func (ref IDOrKeyRef) Type() RefType {
	return ref.typ
}

var _ json.Marshaler = IDOrKeyRef{}

func (ref IDOrKeyRef) MarshalJSON() ([]byte, error) {
	switch ref.typ {
	case RefTypeID:
		return ref.idRef.MarshalJSON()
	case RefTypeKey:
		return ref.keyRef.MarshalJSON()
	}
	return nil, fmt.Errorf("unknown reference type: %d", ref.typ)
}

// FromID creates a new IDOrKeyRef with the given ID.
func FromID[T ~string](id T) IDOrKeyRef {
	return IDOrKeyRef{
		idRef: IDRef(id),
		typ:   RefTypeID,
	}
}

var (
	_ IDOrKeyRef = FromID("")
	_ IDOrKeyRef = FromID(IDRef(""))
)

// FromKey creates a new IDOrKeyRef with the given Key.
func FromKey[T ~string](key T) IDOrKeyRef {
	return IDOrKeyRef{
		keyRef: KeyRef(key),
		typ:    RefTypeKey,
	}
}

var (
	_ IDOrKeyRef = FromKey("")
	_ IDOrKeyRef = FromKey(KeyRef(""))
)
