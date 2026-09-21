package crediteligibility

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func (f Filters) MarshalJSON() ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	filters := f.Normalize()
	switch f.Version {
	case FiltersVersion1:
		type plain Filters
		return json.Marshal(plain(filters))
	default:
		return nil, fmt.Errorf("unsupported credit filters schema version: %d", f.Version)
	}
}

func (f *Filters) UnmarshalJSON(data []byte) error {
	var header struct {
		SchemaVersion FiltersVersion `json:"schema_version"`
	}
	// Parse the envelope first; the selected reader validates all payload fields.
	// Unmarshal also rejects trailing JSON values before version dispatch.
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var filters Filters
	switch header.SchemaVersion {
	case FiltersVersion1:
		type plain Filters
		var value plain
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		filters = Filters(value)
	default:
		return fmt.Errorf("unsupported credit filters schema version: %d", header.SchemaVersion)
	}
	if err := filters.Validate(); err != nil {
		return err
	}
	*f = filters.Normalize()
	return nil
}
