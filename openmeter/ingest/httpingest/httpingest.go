package httpingest

import (
	"github.com/openmeterio/openmeter/internal/ingest/httpingest"
)

// Handler receives an event in CloudEvents format and forwards it to a {Collector}.
type Handler = httpingest.Handler

type HandlerConfig = httpingest.HandlerConfig

func NewHandler(config HandlerConfig) (*Handler, error) {
	return httpingest.NewHandler(config)
}
