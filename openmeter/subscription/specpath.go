package subscription

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

type SpecPath string

const (
	phasePathPrefix   = "phases"
	itemPathPrefix    = "items"
	versionPathPrefix = "idx"
)

type SpecPathType string

const (
	SpecPathTypePhase       SpecPathType = "phase"
	SpecPathTypeItem        SpecPathType = "item"
	SpecPathTypeItemVersion SpecPathType = "item_version"
)

// Lets implement JSON Unmarshaler for Path
func (p *SpecPath) UnmarshalJSON(data []byte) error {
	if err := SpecPath(data).Validate(); err != nil {
		return fmt.Errorf("path validation failed: %s", err)
	}

	*p = SpecPath(data)
	return nil
}

// Lets implement JSON Marshaler for Path
func (p SpecPath) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, p)), nil
}

func (p SpecPath) seg() []string {
	// For a properly formatted path the first segment is empty
	return strings.Split(string(p), "/")[1:]
}

// Checks whether p is a parent of other where parent means all segments of p are present and in order in other
func (p SpecPath) IsParentOf(other SpecPath) bool {
	segments := p.seg()
	otherSegments := other.seg()

	if len(otherSegments) < len(segments) {
		return false
	}

	for i, s := range segments {
		if otherSegments[i] != s {
			return false
		}
	}

	return true
}

// Lets implement validation for Path
func (p SpecPath) Validate() error {
	strVal := string(p)

	if !strings.HasPrefix(strVal, "/") {
		return &PatchValidationError{
			Msg: fmt.Sprintf("invalid path: %s, should start with /", strVal),
		}
	}

	segments := p.seg()
	if len(segments) != 2 && len(segments) != 4 && len(segments) != 6 {
		return &PatchValidationError{
			Msg: fmt.Sprintf("invalid path: %s, should have 2, 4 or 6 segments, has %d", strVal, len(segments)),
		}
	}

	if segments[0] != phasePathPrefix {
		return &PatchValidationError{Msg: fmt.Sprintf("invalid path: %s, first segment should be %s", strVal, phasePathPrefix)}
	}

	if len(segments) == 4 && segments[2] != itemPathPrefix {
		return &PatchValidationError{Msg: fmt.Sprintf("invalid path: %s, third segment should be %s", strVal, itemPathPrefix)}
	}

	if len(segments) == 6 && segments[4] != versionPathPrefix {
		return &PatchValidationError{Msg: fmt.Sprintf("invalid path: %s, fifth segment should be %s", strVal, versionPathPrefix)}
	}

	if lo.SomeBy(segments, func(s string) bool { return s == "" }) {
		return &PatchValidationError{Msg: fmt.Sprintf("invalid path: %s, segments should not be empty", strVal)}
	}

	return nil
}

func (p SpecPath) Type() SpecPathType {
	if len(p.seg()) == 6 {
		return SpecPathTypeItemVersion
	}

	if len(p.seg()) == 4 {
		return SpecPathTypeItem
	}

	// As validity is not guaranteed, Phase is the placeholder type for invalid values
	return SpecPathTypePhase
}

func (p SpecPath) PhaseKey() string {
	return p.seg()[1]
}

func (p SpecPath) ItemKey() string {
	if p.Type() != SpecPathTypeItem {
		return ""
	}

	return p.seg()[3]
}

func (p SpecPath) ItemVersion() int {
	if p.Type() != SpecPathTypeItemVersion {
		return -1
	}

	idxStr := p.seg()[5]
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return -1
	}

	return idx
}

func NewPhasePath(phaseKey string) SpecPath {
	return SpecPath(fmt.Sprintf("/%s/%s", phasePathPrefix, phaseKey))
}

func NewItemPath(phaseKey, itemKey string) SpecPath {
	phasePath := NewPhasePath(phaseKey)
	return SpecPath(fmt.Sprintf("%s/%s/%s", phasePath, itemPathPrefix, itemKey))
}

func NewItemVersionPath(phaseKey, itemKey string, idx int) SpecPath {
	itemPath := NewItemPath(phaseKey, itemKey)
	return SpecPath(fmt.Sprintf("%s/%s/%d", itemPath, versionPathPrefix, idx))
}
