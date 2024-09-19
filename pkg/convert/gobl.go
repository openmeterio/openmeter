package convert

import (
	"github.com/invopop/gobl/cbc"
	"github.com/samber/lo"
)

func MetadataToGOBLMeta(meta map[string]string) cbc.Meta {
	return lo.MapEntries(meta, func(key string, value string) (cbc.Key, string) {
		return cbc.Key(key), value
	})
}
