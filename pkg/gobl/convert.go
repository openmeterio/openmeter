package gobl

import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/cbc"
	"github.com/invopop/gobl/num"
	"github.com/samber/lo"
)

func MetadataToGOBLMeta(meta map[string]string) cbc.Meta {
	return lo.MapEntries(meta, func(key string, value string) (cbc.Key, string) {
		return cbc.Key(key), value
	})
}

func DecimalToAmount(d alpacadecimal.Decimal) (num.Amount, error) {
	// TODO[OM-930]: this is terribly slow, let's map it some other way on the long run
	return num.AmountFromString(d.String())
}
