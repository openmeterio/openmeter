package serializer

import (
	_ "embed"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
)

type JSONSerializer = serializer.JSONSerializer

func NewJSONSerializer() JSONSerializer {
	return serializer.NewJSONSerializer()
}
