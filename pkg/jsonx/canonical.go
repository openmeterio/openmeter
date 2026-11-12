// Package jsonx holds JSON helpers that encoding/json does not provide.
package jsonx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// Canonical re-encodes a JSON document with sorted object keys and no
// whitespace. Numbers keep their original text and arrays keep their order,
// so schemas that only mean the same thing, like 1 and 1.0 or reordered enum
// values, stay different. For an immutability check a false "changed" is the
// safe mistake.
func Canonical(doc []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()

	var value any
	if err := dec.Decode(&value); err != nil {
		return nil, err
	}

	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("unexpected data after JSON document")
	}

	var out bytes.Buffer

	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(value); err != nil {
		return nil, err
	}

	return bytes.TrimSuffix(out.Bytes(), []byte("\n")), nil
}
