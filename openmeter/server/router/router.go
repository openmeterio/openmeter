package router

import (
	"github.com/openmeterio/openmeter/internal/server/router"
)

type IngestHandler = router.IngestHandler

type Config = router.Config

type Router = router.Router

func NewRouter(config Config) (*Router, error) {
	return router.NewRouter(config)
}
