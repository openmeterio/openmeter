package crediteligibility

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type FiltersVersion int

const FiltersVersion1 FiltersVersion = 1

// StoredFilters preserves the encoding version separately from the dimensions
// used for matching. Re-encoding it keeps the explicitly selected version.
type StoredFilters struct {
	Version FiltersVersion
	Filters Filters
}

// filtersV1 is the frozen feature-only storage format. New dimensions belong in
// a new format so existing readers never silently lose restrictions.
type filtersV1 struct {
	SchemaVersion FiltersVersion `json:"schema_version"`
	Features      []string       `json:"features,omitempty"`
}

func (s StoredFilters) MarshalJSON() ([]byte, error) {
	if err := s.Filters.Validate(); err != nil {
		return nil, err
	}
	filters := s.Filters.Normalize()
	switch s.Version {
	case FiltersVersion1:
		return json.Marshal(filtersV1{SchemaVersion: s.Version, Features: filters.Features})
	default:
		return nil, fmt.Errorf("unsupported credit filters schema version: %d", s.Version)
	}
}

func (s *StoredFilters) UnmarshalJSON(data []byte) error {
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
		var value filtersV1
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		filters = Filters{Features: value.Features}
	default:
		return fmt.Errorf("unsupported credit filters schema version: %d", header.SchemaVersion)
	}
	if err := filters.Validate(); err != nil {
		return err
	}
	*s = StoredFilters{Version: header.SchemaVersion, Filters: filters.Normalize()}
	return nil
}

// MarshalJSON chooses the oldest format that represents all dimensions. This
// keeps feature-only writes readable by older releases during a rolling upgrade.
func (f Filters) MarshalJSON() ([]byte, error) {
	return json.Marshal(StoredFilters{Version: FiltersVersion1, Filters: f})
}

// UnmarshalJSON maps supported storage formats into common matching dimensions.
// Use StoredFilters when the caller also needs to preserve the encoding version.
func (f *Filters) UnmarshalJSON(data []byte) error {
	var stored StoredFilters
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}
	*f = stored.Filters
	return nil
}
