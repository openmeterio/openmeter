package config

import "strings"

// AddPrefix returns string with "<prefix>." prepended to key.
// If returns key unmodified if prefix is empty or key already has the prefix added.
func AddPrefix(prefix, key string) string {
	if prefix == "" || strings.HasPrefix(key, prefix+".") {
		return key
	}

	return prefix + "." + key
}
