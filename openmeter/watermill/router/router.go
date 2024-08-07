package router

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/openmeterio/openmeter/internal/watermill/router"
)

type (
	Options = router.Options
)

func NewDefaultRouter(opts Options, dlqHandler message.NoPublishHandlerFunc) (*message.Router, error) {
	return router.NewDefaultRouter(opts, dlqHandler)
}
