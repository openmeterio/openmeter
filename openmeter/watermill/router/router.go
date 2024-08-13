package router

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/internal/watermill/router"
)

type (
	Options = router.Options
)

func NewDefaultRouter(opts Options) (*message.Router, error) {
	return router.NewDefaultRouter(opts)
}
