package meter

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// maxJSONPathLength is a sanity ceiling, not a functional limit.
const maxJSONPathLength = 512

// validateJSONPath rejects byte sequences no valid ClickHouse JSONPath can contain, so
// it can run on already-stored paths without locking anyone out. It is a floor, not a
// grammar check; ClickHouse itself is authoritative. The load-bearing rule is that only
// '.', '[' or '*' may follow the '$' root, which rejects "$2" and "${foo}" — the
// sequences go-sqlbuilder reads as argument references.
func validateJSONPath(path string) error {
	if path == "" {
		return errors.New("must not be empty")
	}

	if len(path) > maxJSONPathLength {
		return fmt.Errorf("must not be longer than %d characters", maxJSONPathLength)
	}

	if !utf8.ValidString(path) {
		return errors.New("must be valid UTF-8")
	}

	if path[0] != '$' {
		return errors.New("must start with $")
	}

	if len(path) > 1 && path[1] != '.' && path[1] != '[' && path[1] != '*' {
		return fmt.Errorf("must continue with '.', '[' or '*' after $, got %q", path[1])
	}

	if strings.ContainsRune(path[1:], '$') {
		return errors.New("must not contain $ other than the leading root token")
	}

	for offset, r := range path {
		switch {
		case r < 0x20 || r == 0x7f:
			return fmt.Errorf("must not contain control characters (offset %d)", offset)
		case r == '\\':
			return fmt.Errorf("must not contain backslashes (offset %d)", offset)
		case r == ';':
			return fmt.Errorf("must not contain semicolons (offset %d)", offset)
		}
	}

	return nil
}

// ValidateGroupByKey reports whether a group-by key is a safe SQL identifier. The key
// becomes a column alias and GROUP BY term and cannot be bound, so it is the one
// meter-controlled value that must be constrained rather than parameterised. Shared by
// write-time and read-time validation so they cannot drift.
func ValidateGroupByKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("meter group by key cannot be empty")
	}

	if !groupByKeyRegExp.MatchString(key) {
		return fmt.Errorf("meter group by key %s is invalid, only alphanumeric and underscore characters are allowed", key)
	}

	return nil
}
